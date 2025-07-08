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
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/google/test-server/internal/config"
)

const HeadSHA = "b4d6e60a9b97e7b98c63df9308728c5c88c0b40c398046772c63447b94608b4d"

type RecordedRequest struct {
	Request         string
	Header          http.Header
	Body            []byte
	PreviousRequest string // The sha256 sum of the previous request in the chain.
	ServerAddress   string
	Port            int64
	Protocol        string
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
		Request:         request,
		Header:          header,
		Body:            body,
		PreviousRequest: previousRequest,
		ServerAddress:   cfg.TargetHost,
		Port:            cfg.TargetPort,
		Protocol:        cfg.TargetType,
	}

	return recordedRequest, nil
}

func readBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return []byte{}, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	// Restore the request body for further use.
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
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
	testName := r.Header.Get("Test-Name")
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
//
// The serialization format is as follows:
//   - The first line is the sha256 of the previous request as a hex string.
//   - Next is the server address.
//   - Next is the port.
//   - Next is the protocol.
//   - Next is a line of 80 asterisks.
//   - Next is the HTTP request.
//   - Next, a single line for each header formatted as "{key}: {value}".
//   - Next, there are 2 empty lines.
//   - The rest of the file is the body content.
func (r *RecordedRequest) Serialize() string {
	var builder strings.Builder

	// Format the SHA256 sum of the previous request.
	builder.WriteString(r.PreviousRequest)
	builder.WriteString("\n")

	builder.WriteString(fmt.Sprintf("Server Address: %s\n", r.ServerAddress))

	builder.WriteString(fmt.Sprintf("Port: %d\n", r.Port))

	builder.WriteString(fmt.Sprintf("Protocol: %s\n", r.Protocol))

	builder.WriteString(strings.Repeat("*", 80) + "\n")

	// Format the HTTP request line.
	builder.WriteString(r.Request)
	builder.WriteString("\n")

	// Format the headers in sorted order.
	keys := make([]string, 0, len(r.Header))
	for key := range r.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, value := range r.Header[key] {
			builder.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	builder.WriteString("\n\n")
	builder.WriteString(string(r.Body))

	return builder.String()
}

// Deserialize the request.
func Deserialize(data string) (*RecordedRequest, error) {
	lines := strings.Split(data, "\n")
	if len(lines) < 6 {
		return nil, fmt.Errorf("invalid serialized data: not enough lines")
	}

	previousRequest := lines[0]

	serverAddress := strings.TrimPrefix(lines[1], "Server Address: ")
	portString := strings.TrimPrefix(lines[2], "Port: ")
	protocol := strings.TrimPrefix(lines[3], "Protocol: ")

	port := 0
	if portString != "" {
		_, err := fmt.Sscan(portString, &port)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
	}

	request := lines[5]

	headerStart := 6
	bodyStart := -1
	headers := make(http.Header)

	for i := headerStart; i < len(lines); i++ {
		if lines[i] == "" && lines[i+1] == "" {
			bodyStart = i + 2
			break
		}
		parts := strings.SplitN(lines[i], ": ", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		headers.Add(key, value)
	}

	var body []byte
	if bodyStart != -1 && bodyStart < len(lines) {
		body = []byte(strings.Join(lines[bodyStart:], "\n"))
	}

	recordedRequest := &RecordedRequest{
		Request:         request,
		Header:          headers,
		Body:            body,
		PreviousRequest: previousRequest,
		ServerAddress:   serverAddress,
		Port:            int64(port),
		Protocol:        protocol,
	}

	return recordedRequest, nil
}

// RedactHeaders removes the specified headers from the RecordedRequest.
func (r *RecordedRequest) RedactHeaders(headers []string) {
	for _, header := range headers {
		r.Header.Del(header)
	}
}

type RecordedResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
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

	recordedResponse := &RecordedResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body,
	}
	return recordedResponse, nil
}

func (r *RecordedResponse) Serialize() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Status code: %d \n", r.StatusCode))
	for name, values := range r.Header {
		for _, value := range values {
			buffer.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		}
	}
	buffer.WriteString("\n")
	buffer.Write(r.Body)

	return buffer.String()
}

// DeserializeResponse deserializes the response.
func DeserializeResponse(data []byte) (*RecordedResponse, error) {
	lines := bytes.SplitN(data, []byte("\n"), 2)
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid serialized data: not enough lines")
	}

	statusCodeLine := lines[0]
	statusCode := 0

	_, err := fmt.Sscanf(string(statusCodeLine), "Status code: %d", &statusCode)
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %w", err)
	}

	headerBodySplit := bytes.SplitN(lines[1], []byte("\n\n"), 2)
	if len(headerBodySplit) < 2 {
		return nil, fmt.Errorf("invalid serialized data: no body separator")
	}

	headerLines := bytes.Split(headerBodySplit[0], []byte("\n"))
	headers := make(http.Header)

	for _, line := range headerLines {
		parts := bytes.SplitN(line, []byte(": "), 2)
		if len(parts) != 2 {
			continue
		}
		key := string(parts[0])
		value := string(parts[1])
		headers.Add(key, value)
	}

	body := headerBodySplit[1]

	recordedResponse := &RecordedResponse{
		StatusCode: statusCode,
		Header:     headers,
		Body:       body,
	}

	return recordedResponse, nil
}
