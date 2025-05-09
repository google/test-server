/*
Copyright 2025 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package record

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
	"github.com/google/test-server/internal/store"
	"github.com/gorilla/websocket"
)

type RecordingHTTPSProxy struct {
	prevRequestSHA string
	config         *config.EndpointConfig
	recordingDir   string
	redactor       *redact.Redact
}

func NewRecordingHTTPSProxy(cfg *config.EndpointConfig, recordingDir string, redactor *redact.Redact) *RecordingHTTPSProxy {
	return &RecordingHTTPSProxy{
		prevRequestSHA: store.HeadSHA,
		config:         cfg,
		recordingDir:   recordingDir,
		redactor:       redactor,
	}
}

func (r *RecordingHTTPSProxy) ResetChain() {
	r.prevRequestSHA = store.HeadSHA
}

func (r *RecordingHTTPSProxy) Start() error {
	addr := fmt.Sprintf(":%d", r.config.SourcePort)
	server := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(r.handleRequest),
	}
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
	return nil
}

func (r *RecordingHTTPSProxy) handleRequest(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("Recording request: %s %s\n", req.Method, req.URL.String())

	reqHash, err := r.recordRequest(req)
	if err != nil {
		fmt.Printf("Error recording request: %v\n", err)
		http.Error(w, fmt.Sprintf("Error recording request: %v", err), http.StatusInternalServerError)
		return
	}

	if req.Header.Get("Upgrade") == "websocket" {
		fmt.Printf("Upgrading connection to websocket...\n")
		r.proxyWebsocket(w, req, reqHash)
		return
	}

	resp, respBody, err := r.proxyRequest(w, req)
	if err != nil {
		fmt.Printf("Error proxying request: %v\n", err)
		http.Error(w, fmt.Sprintf("Error proxying request: %v", err), http.StatusInternalServerError)
		return
	}

	err = r.recordResponse(resp, reqHash, respBody)

	if err != nil {
		fmt.Printf("Error recording response: %v\n", err)
		http.Error(w, fmt.Sprintf("Error recording response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (r *RecordingHTTPSProxy) recordRequest(req *http.Request) (string, error) {
	recordedRequest, err := store.NewRecordedRequest(req, r.prevRequestSHA, *r.config)
	if err != nil {
		return "", err
	}

	// Redact headers by key
	recordedRequest.RedactHeaders(r.config.RedactRequestHeaders)
	// Redacts secrets from header values
	r.redactor.Headers(recordedRequest.Header)
	recordedRequest.Request = r.redactor.String(recordedRequest.Request)
	recordedRequest.Body = r.redactor.Bytes(recordedRequest.Body)

	reqHash, err := recordedRequest.ComputeSum()
	if err != nil {
		return "", err
	}

	recordPath := filepath.Join(r.recordingDir, reqHash+".req")
	err = os.WriteFile(recordPath, []byte(recordedRequest.Serialize()), 0644)
	if err != nil {
		return "", err
	}
	return reqHash, nil
}

func (r *RecordingHTTPSProxy) proxyRequest(w http.ResponseWriter, req *http.Request) (*http.Response, []byte, error) {
	url := fmt.Sprintf("https://%s:%d%s", r.config.TargetHost, r.config.TargetPort, req.URL.Path)
	if req.URL.RawQuery != "" {
		url += "?" + req.URL.RawQuery
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, err
	}
	req.Body.Close()

	proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, nil, err
	}

	for name, values := range req.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		return nil, nil, err
	}

	r.applyResponseHeaderReplacements(resp.Header)

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	w.Write(respBodyBytes) // Send original (compressed) body to client
	return resp, respBodyBytes, nil
}

func (r *RecordingHTTPSProxy) recordResponse(resp *http.Response, reqHash string, body []byte) error {
	recordedResponse, err := store.NewRecordedResponse(resp, body)
	if err != nil {
		return err
	}

	recordedResponse.Body = r.redactor.Bytes(recordedResponse.Body)

	recordPath := filepath.Join(r.recordingDir, reqHash+".resp")
	fmt.Printf("Writing response to: %s\n", recordPath)
	err = os.WriteFile(recordPath, []byte(recordedResponse.Serialize()), 0644)
	if err != nil {
		return err
	}

	return nil
}

// applyResponseHeaderReplacements applies the header replacements defined in the EndpointConfig to the request headers.
func (r *RecordingHTTPSProxy) applyResponseHeaderReplacements(headers http.Header) {
	for _, replacement := range r.config.ResponseHeaderReplacements {
		if values, ok := headers[replacement.Header]; ok {
			for i, value := range values {
				headers[replacement.Header][i] = replaceRegex(value, replacement.Regex, replacement.Replace)
			}
		}
	}
}

func replaceRegex(s, regex, replacement string) string {
	// Compile the regex
	re := regexp.MustCompile(regex)

	// Replace all matches
	return re.ReplaceAllString(s, replacement)
}

func (r *RecordingHTTPSProxy) proxyWebsocket(w http.ResponseWriter, req *http.Request, reqHash string) {
	conn, clientConn, err := r.upgradeConnectionToWebsocket(w, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error proxying websocket: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	defer clientConn.Close()

	c := make(chan []byte)
	quit := make(chan int)

	go pumpWebsocket(clientConn, conn, c, quit, ">")
	go pumpWebsocket(conn, clientConn, c, quit, "<")

	recordPath := filepath.Join(r.recordingDir, reqHash+".websocket")
	f, err := os.Create(recordPath)
	if err != nil {
		fmt.Printf("Error creating websocket recording file: %v\n", err)
		http.Error(w, fmt.Sprintf("Error proxying websocket: %v", err), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	quitCount := 0
	for {
		select {
		case buf := <-c:
			_, err := f.Write(buf)
			if err != nil {
				panic(fmt.Sprintf("Error writing to websocket recording file: %v\n", err))
			}
		case <-quit:
			quitCount += 1
			if quitCount == 2 {
				return
			}
		}
	}
}

func pumpWebsocket(src, dst *websocket.Conn, c chan []byte, quit chan int, prepend string) {
	for {
		msgType, buf, err := src.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				quit <- 0
				return
			}
			fmt.Printf("Error reading from websocket\n")
			quit <- 1
			return
		}
		prefix := fmt.Sprintf("%s%d", prepend, cap(buf))
		c <- append([]byte(prefix), buf...)
		err = dst.WriteMessage(msgType, buf)
		if err != nil {
			fmt.Printf("Error writing to websocket: %v\n", err)
			quit <- 1
			return
		}
	}
}

func (r *RecordingHTTPSProxy) upgradeConnectionToWebsocket(w http.ResponseWriter, req *http.Request) (*websocket.Conn, *websocket.Conn, error) {
	url := fmt.Sprintf("wss://%s:%d%s", r.config.TargetHost, r.config.TargetPort, req.URL.Path)
	if req.URL.RawQuery != "" {
		url += "?" + req.URL.RawQuery
	}

	dialHeaders := http.Header{}
	excludedHeaders := map[string]bool{
		"Sec-Websocket-Version":    true,
		"Sec-Websocket-Key":        true,
		"Sec-Websocket-Extensions": true,
		"Connection":               true,
		"Upgrade":                  true,
	}
	for k, v := range req.Header {
		if _, ok := excludedHeaders[k]; ok {
			continue
		}
		dialHeaders[k] = v
	}

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, dialHeaders)
	if err != nil {
		return nil, nil, err
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins
		},
	}

	clientConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return nil, nil, err
	}
	return conn, clientConn, err
}
