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
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
	"github.com/google/test-server/internal/store"
)

type ReplayHTTPServer struct {
	prevRequestSHA string
	config         *config.EndpointConfig
	recordingDir   string
	redactor       *redact.Redact
}

func NewReplayHTTPServer(cfg *config.EndpointConfig, recordingDir string, redactor *redact.Redact) *ReplayHTTPServer {
	return &ReplayHTTPServer{
		prevRequestSHA: store.HeadSHA,
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
	redactedReq, err := r.createRedactedRequest(req)
	if err != nil {
		fmt.Printf("Error processing request")
		http.Error(w, fmt.Sprintf("Error processing request: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Replaying request: %ss\n", redactedReq.Request)

	reqHash, err := redactedReq.ComputeSum()
	if err != nil {
		fmt.Printf("Error computing request sum: %v\n", err)
		http.Error(w, fmt.Sprintf("Error computing request sum: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := r.loadResponse(reqHash)
	if err != nil {
		fmt.Printf("Error loading response: %v\n", err)
		http.Error(w, fmt.Sprintf("Error loading response: %v", err), http.StatusInternalServerError)
		return
	}

	err = r.writedResponse(w, resp)
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
		panic(err)
	}
}

func (r *ReplayHTTPServer) createRedactedRequest(req *http.Request) (*store.RecordedRequest, error) {
	recordedRequest, err := store.NewRecordedRequest(req, r.prevRequestSHA, *r.config)
	if err != nil {
		return nil, err
	}

	// Redact headers by key
	recordedRequest.RedactHeaders(r.config.RedactRequestHeaders)
	// Redacts secrets from header values
	r.redactor.Headers(recordedRequest.Header)
	recordedRequest.Request = r.redactor.String(recordedRequest.Request)
	recordedRequest.Body = r.redactor.Bytes(recordedRequest.Body)

	return recordedRequest, nil
}

func (r *ReplayHTTPServer) loadResponse(sha string) (*store.RecordedResponse, error) {
	responseFile := filepath.Join(r.recordingDir, sha+".resp")
	fmt.Printf("loading response from : %s\n", responseFile)
	responseData, err := os.ReadFile(responseFile)
	if err != nil {
		return nil, err
	}
	return store.DeserializeResponse(responseData)
}

func (r *ReplayHTTPServer) writedResponse(w http.ResponseWriter, resp *store.RecordedResponse) error {
	for key, values := range resp.Header {
		for _, value := range values {
			if key == "Content-Length" || key == "Content-Encoding" {
				continue
			}
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err := w.Write(resp.Body)
	return err
}
