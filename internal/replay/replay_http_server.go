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

package replay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
	"github.com/google/test-server/internal/store"
	"github.com/gorilla/websocket"
)

type ReplayHTTPServer struct {
	prevRequestSHA string
	seenFiles      map[string]struct{}
	config         *config.EndpointConfig
	recordingDir   string
	redactor       *redact.Redact
}

func NewReplayHTTPServer(cfg *config.EndpointConfig, recordingDir string, redactor *redact.Redact) *ReplayHTTPServer {
	return &ReplayHTTPServer{
		prevRequestSHA: store.HeadSHA,
		seenFiles:      make(map[string]struct{}),
		config:         cfg,
		recordingDir:   recordingDir,
		redactor:       redactor,
	}
}

func (r *ReplayHTTPServer) Start() error {
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

func (r *ReplayHTTPServer) handleRequest(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == r.config.Health {
		w.WriteHeader(http.StatusOK)
		return
	}

	redactedReq, err := r.createRedactedRequest(req)
	if err != nil {
		fmt.Printf("Error processing request")
		http.Error(w, fmt.Sprintf("Error processing request: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Replaying request: %ss\n", redactedReq.Request)
	fileName, err := redactedReq.GetRecordingFileName()
	if err != nil {
		fmt.Printf("Invalid recording file name: %v\n", err)
		http.Error(w, fmt.Sprintf("Invalid recording file name: %v", err), http.StatusInternalServerError)
		return
	}
	if _, ok := r.seenFiles[fileName]; !ok {
		// Reset to HeadSHA when first time seen request from the given file.
		redactedReq.PreviousRequest = store.HeadSHA
	}
	if req.Header.Get("Upgrade") == "websocket" {
		fmt.Printf("Upgrading connection to websocket...\n")

		chunks, err := r.loadWebsocketChunks(fileName)
		if err != nil {
			fmt.Printf("Error loading websocket response: %v\n", err)
			http.Error(w, fmt.Sprintf("Error loading websocket response: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Printf("Replaying websocket: %s\n", fileName)
		r.proxyWebsocket(w, req, chunks)
		return
	}
	fmt.Printf("Replaying http request: %s\n", redactedReq.Request)
	shaSum := redactedReq.ComputeSum()
	resp, err := r.loadResponse(fileName, shaSum)
	if err != nil {
		fmt.Printf("Error loading response: %v\n", err)
		http.Error(w, fmt.Sprintf("Error loading response: %v", err), http.StatusInternalServerError)
		return
	}

	err = r.writeResponse(w, resp, redactedReq)
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
		panic(err)
	}
	if fileName != shaSum {
		r.prevRequestSHA = shaSum
	}
	r.seenFiles[fileName] = struct{}{}
}

func (r *ReplayHTTPServer) createRedactedRequest(req *http.Request) (*store.RecordedRequest, error) {
	recordedRequest, err := store.NewRecordedRequest(req, r.prevRequestSHA, *r.config)
	if err != nil {
		return nil, err
	}

	// Redact headers by key
	recordedRequest.RedactHeaders(r.config.RedactRequestHeaders)
	// Redacts secrets from header values
	r.redactor.Headers(recordedRequest.Headers)
	recordedRequest.Request = r.redactor.String(recordedRequest.Request)
	recordedRequest.URL = r.redactor.String(recordedRequest.URL)
	var redactedBodySegments []map[string]any
	for _, bodySegment := range recordedRequest.BodySegments {
		redactedBodySegments = append(redactedBodySegments, r.redactor.Map(bodySegment))
	}
	recordedRequest.BodySegments = redactedBodySegments
	return recordedRequest, nil
}

func (r *ReplayHTTPServer) loadResponse(fileName string, shaSum string) (*store.RecordedResponse, error) {
	// Open the replay log file for reading.
	filePath := filepath.Join(r.recordingDir, fileName+".json")
	fmt.Printf("loading response from : %s with shaSum: %s\n", filePath, shaSum)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var recordFile store.RecordFile
	err = json.Unmarshal(body, &recordFile)
	if err != nil {
		return nil, fmt.Errorf("unable to deserialize data to RecordFile: %w", err)
	}

	for _, interaction := range recordFile.Interactions {
		if interaction.SHASum == shaSum {
			return interaction.Response, nil
		}
	}

	return nil, fmt.Errorf("response with shaSum %s not found in file", shaSum)
}

func (r *ReplayHTTPServer) writeResponse(w http.ResponseWriter, resp *store.RecordedResponse, req *store.RecordedRequest) error {
	for key, value := range resp.Headers {
		if key == "Content-Length" || key == "Content-Encoding" {
			continue
		}
		w.Header().Add(key, value)
	}

	w.WriteHeader(int(resp.StatusCode))

	// When the response body is empty we return directly with the headers.
	if len(resp.BodySegments) == 0 {
		return nil
	}

	if !strings.Contains(req.URL, "alt=sse") {
		jsonBytes, err := json.Marshal(resp.BodySegments[0])
		if err != nil {
			return err
		}

		_, err = w.Write(jsonBytes)
		return err
	} else {
		for _, bodySegment := range resp.BodySegments {
			jsonBytes, err := json.Marshal(bodySegment)
			if err != nil {
				return err
			}

			line := append([]byte("data: "), jsonBytes...)
			line = append(line, []byte("\n\n")...)

			if _, err := w.Write(line); err != nil {
				return err
			}
		}
	}

	return nil
}

func extractNumber(i *int, content string) (int, error) {
	numStart := *i
	for *i < len(content) && unicode.IsDigit(rune(content[*i])) {
		*i++
	}
	numEnd := *i
	if numStart == numEnd {
		return 0, fmt.Errorf("missing chunk length after prefix at position %d", numStart-1)
	}
	numStr := content[numStart:numEnd]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid chunk length '%s': %w", numStr, err)
	}
	return num, nil
}

func (r *ReplayHTTPServer) proxyWebsocket(w http.ResponseWriter, req *http.Request, chunks []string) {
	clientConn, err := r.upgradeConnectionToWebsocket(w, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error proxying websocket: %v", err), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()
	r.replayWebsocket(clientConn, chunks)
}

func (r *ReplayHTTPServer) loadWebsocketChunks(fileName string) ([]string, error) {
	responseFile := filepath.Join(r.recordingDir, fileName+".websocket.log")
	fmt.Printf("loading websocket response from : %s\n", responseFile)
	bytes, err := os.ReadFile(responseFile)
	var chunks = make([]string, 0)
	if err != nil {
		fmt.Printf("Error loading websocket response: %v\n", err)
		return chunks, err
	}

	i := 0
	response := string(bytes)
	for i < len(response) {
		// Extracts prefix
		prefix := response[i]
		if prefix != '>' && prefix != '<' {
			return nil, fmt.Errorf("invalid message prefix at position %d: expected '>' or '<', got '%c'", i, prefix)
		}
		i++ // Move cursor past prefix.

		// Extracts chunk length number
		num, err := extractNumber(&i, response)
		i++ // Move cursor to skip the whitespace between the number and the actual chunk.
		if err != nil {
			return nil, fmt.Errorf("failed to extract number %v", err)
		}

		// Extracts chunk
		chunkStart := i
		chunkEnd := chunkStart + num
		if chunkEnd > len(response) {
			return nil, fmt.Errorf("chunk length %d at position %d exceeds response bounds", chunkEnd, chunkStart)
		}
		chunk := response[chunkStart : chunkEnd-1] // Remove the \n appended at the end of the chunk
		chunks = append(chunks, string(prefix)+chunk)
		i = chunkEnd
	}
	return chunks, nil
}

func (r *ReplayHTTPServer) replayWebsocket(conn *websocket.Conn, chunks []string) {
	for _, chunk := range chunks {
		if strings.HasPrefix(chunk, ">") {
			_, buf, err := conn.ReadMessage()
			reqChunk := r.redactor.String(string(buf))
			if err != nil {
				fmt.Printf("Error reading from websocket: %v\n", err)
				return
			}

			runes := []rune(chunk)
			recChunk := string(runes[1:])
			if reqChunk != recChunk {
				fmt.Printf("input chunk mismatch\n Input chunk: %s\n Recorded chunk: %s\n", reqChunk, recChunk)
				writeError(conn, "input chunk mismatch")
				return
			}
		} else if strings.HasPrefix(chunk, "<") {
			runes := []rune(chunk)
			recChunk := string(runes[1:])
			// Write binary message. (messageType=2)
			err := conn.WriteMessage(2, []byte(recChunk))
			if err != nil {
				fmt.Printf("Error writing to websocket: %v\n", err)
				return
			}
		} else {
			fmt.Printf("Unreconginized chunk: %s", chunk)
			return
		}
	}
}

func writeError(conn *websocket.Conn, errMsg string) {
	closeMessage := websocket.FormatCloseMessage(
		websocket.CloseInternalServerErr,
		errMsg,
	)
	err := conn.WriteMessage(websocket.CloseMessage, closeMessage)
	if err != nil {
		fmt.Printf("Failed to write error: %v\n", err)
	}
}

func (r *ReplayHTTPServer) upgradeConnectionToWebsocket(w http.ResponseWriter, req *http.Request) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins
		},
	}

	clientConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return nil, err
	}
	return clientConn, err
}
