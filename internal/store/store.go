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

package store

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/test-server/internal/config"
)

const HeadSHA = "b4d6e60a9b97e7b98c63df9308728c5c88c0b40c398046772c63447b94608b4d"

// Represents a single interaction, request and response in a replay.
type RecordInteraction struct {
	Request  *RecordedRequest  `json:"request,omitempty"`
	SHASum   string            `json:"shaSum,omitempty"`
	Response *RecordedResponse `json:"response,omitempty"`
}

// Represents a recorded session.
type RecordFile struct {
	RecordID     string               `json:"recordID,omitempty"`
	Interactions []*RecordInteraction `json:"interactions,omitempty"`
}

type RecordedRequest struct {
	Method       string            `json:"method,omitempty"`
	URL          string            `json:"url,omitempty"`
	Request      string            `json:"request,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodySegments []map[string]any  `json:"bodySegments,omitempty"`
	// The sha256 sum of the previous request in the chain.
	PreviousRequest string `json:"previousRequest,omitempty"`
	ServerAddress   string `json:"serverAddress,omitempty"`
	Port            int64  `json:"port,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
}

type RecordedResponse struct {
	StatusCode          int32             `json:"statusCode,omitempty"`
	Headers             map[string]string `json:"headers,omitempty"`
	BodySegments        []map[string]any  `json:"bodySegments,omitempty"`
	SDKResponseSegments []map[string]any  `json:"sdkResponseSegments,omitempty"`
}

// NewRecordedRequest creates a RecordedRequest from an http.Request.
func NewRecordedRequest(req *http.Request, previousRequest string, cfg config.EndpointConfig) (*RecordedRequest, error) {
	// Read the body.
	body, err := readBody(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	// Create the request string.
	request := fmt.Sprintf("%s %s %s", req.Method, req.URL.String(), req.Proto)

	// Create a copy of the headers.
	header := req.Header.Clone()

	// Create the RecordedRequest.
	recordedRequest := &RecordedRequest{
		Method:          req.Method,
		URL:             req.URL.String(),
		Request:         request,
		Headers:         GetHeadersMap(&header),
		BodySegments:    []map[string]any{body},
		PreviousRequest: previousRequest,
		ServerAddress:   cfg.TargetHost,
		Port:            cfg.TargetPort,
		Protocol:        cfg.TargetType,
	}

	return recordedRequest, nil
}

func readBody(req *http.Request) (map[string]any, error) {
	if req.Body == nil {
		return map[string]any{}, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var resultMap map[string]any
	err = json.Unmarshal(body, &resultMap)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
		return nil, err
	}
	// Restore the request body for further use.
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	return resultMap, nil
}

// ComputeSum computes the SHA256 sum of a RecordedRequest.
func (r *RecordedRequest) ComputeSum() string {
	serialized := r.Serialize()
	hash := sha256.Sum256([]byte(serialized))
	hashHex := hex.EncodeToString(hash[:])
	return hashHex
}

// GetRecordingFileName returns the recording file name.
// It prefers the value from the TEST_NAME header.
// It returns error when test name contains illegal sequence.
// If the TEST_NAME header is not present, it falls back to computed SHA256 sum.
func (r *RecordedRequest) GetRecordingFileName() (string, error) {
	testName := r.Headers["Test-Name"]
	if strings.Contains(testName, "../") {
		return "", fmt.Errorf("test name: %s contains illegal sequence '../'", testName)
	}
	if testName != "" {
		fileName := strings.ReplaceAll(testName, " ", "_")
		return fileName, nil
	}
	return r.ComputeSum(), nil
}

// Serialize the request.
func (r *RecordedRequest) Serialize() string {
	req, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Printf("unable to serialize recorded request: %s", err)
		return ""
	}

	return string(req)
}

// RedactHeaders removes the specified headers from the RecordedRequest.
func (r *RecordedRequest) RedactHeaders(headers []string) {
	for _, header := range headers {
		delete(r.Headers, header)
	}
}

func NewRecordedResponse(resp *http.Response, body []byte) (*RecordedResponse, error) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()

		// Read the uncompressed body.
		uncompressedBody := new(bytes.Buffer)
		_, err = uncompressedBody.ReadFrom(gzipReader)
		if err != nil {
			return nil, err
		}
		body = uncompressedBody.Bytes()

	}

	var bodySegment map[string]any
	err := json.Unmarshal(body, &bodySegment)
	if err != nil {
		return nil, err
	}

	recordedResponse := &RecordedResponse{
		StatusCode:   int32(resp.StatusCode),
		Headers:      GetHeadersMap(&resp.Header),
		BodySegments: []map[string]any{bodySegment},
	}
	return recordedResponse, nil
}

func GetHeadersMap(header *http.Header) map[string]string {
	// Create a new map[string]string
	headerMap := make(map[string]string)

	// Iterate over the http.Header and populate the new map
	for key, values := range *header {
		headerMap[key] = strings.Join(values, ", ")
	}

	return headerMap
}
