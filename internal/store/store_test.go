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
	"fmt"
	"net/http"
	"testing"

	"github.com/google/test-server/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRecordedRequest_Serialize(t *testing.T) {
	testCases := []struct {
		name     string
		request  RecordedRequest
		expected string
	}{
		{
			name: "Empty request",
			request: RecordedRequest{
				Request:         "",
				Headers:         map[string]string{},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: "{\n  \"previousRequest\": \"b4d6e60a9b97e7b98c63df9308728c5c88c0b40c398046772c63447b94608b4d\"\n}",
		},
		{
			name: "Request with headers",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Accept":       "application/xml",
					"Content-Type": "application/json",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: "{\n  \"request\": \"GET / HTTP/1.1\",\n  \"headers\": {\n    \"Accept\": \"application/xml\",\n    \"Content-Type\": \"application/json\"\n  },\n  \"previousRequest\": \"b4d6e60a9b97e7b98c63df9308728c5c88c0b40c398046772c63447b94608b4d\"\n}",
		},
		{
			name: "Request with body",
			request: RecordedRequest{
				Request:         "POST /data HTTP/1.1",
				Headers:         map[string]string{},
				BodySegments:    []map[string]any{{"key": "value"}},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: "{\n  \"request\": \"POST /data HTTP/1.1\",\n  \"bodySegments\": [\n    {\n      \"key\": \"value\"\n    }\n  ],\n  \"previousRequest\": \"b4d6e60a9b97e7b98c63df9308728c5c88c0b40c398046772c63447b94608b4d\"\n}",
		},
		{
			name: "Request with previous request SHA256 sum",
			request: RecordedRequest{
				Request:         "GET / HTTP/1.1",
				Headers:         map[string]string{},
				BodySegments:    []map[string]any{},
				PreviousRequest: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: "{\n  \"request\": \"GET / HTTP/1.1\",\n  \"previousRequest\": \"0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20\"\n}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.request.Serialize()
			require.Equal(t, tc.expected, actual, "Serialize() result mismatch")
		})
	}
}

func TestNewRecordedRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *http.Request
		cfg         config.EndpointConfig
		expected    *RecordedRequest
		expectedErr bool
	}{
		{
			name: "Test with body",
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "http://example.com/test", bytes.NewBuffer([]byte("{\"test body\": \"\"}")))
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			cfg: config.EndpointConfig{
				TargetHost: "example.com",
				TargetPort: 443,
				TargetType: "https",
			},
			expected: &RecordedRequest{
				Request:         "POST http://example.com/test HTTP/1.1",
				Headers:         map[string]string{"Content-Type": "application/json"},
				BodySegments:    []map[string]any{{"test body": ""}},
				PreviousRequest: HeadSHA,
				ServerAddress:   "example.com",
				Port:            443,
				Protocol:        "https",
			},
			expectedErr: false,
		},
		{
			name: "Test without body",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://example.com/test", nil)
				return req
			}(),
			cfg: config.EndpointConfig{
				TargetHost: "example.com",
				TargetPort: 443,
				TargetType: "https",
			},
			expected: &RecordedRequest{
				Request:         "GET http://example.com/test HTTP/1.1",
				Headers:         map[string]string{},
				BodySegments:    []map[string]any{{}},
				PreviousRequest: HeadSHA,
				ServerAddress:   "example.com",
				Port:            443,
				Protocol:        "https",
			},
			expectedErr: false,
		},
		{
			name: "Test with error reading body",
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "http://example.com/test", &errorReader{})
				return req
			}(),
			cfg: config.EndpointConfig{
				TargetHost: "example.com",
				TargetPort: 443,
				TargetType: "https",
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recordedRequest, err := NewRecordedRequest(tc.request, HeadSHA, tc.cfg)

			if tc.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected.Request, recordedRequest.Request)
			require.Equal(t, tc.expected.Headers, recordedRequest.Headers)
			require.Equal(t, tc.expected.BodySegments, recordedRequest.BodySegments)
			require.Equal(t, tc.expected.PreviousRequest, recordedRequest.PreviousRequest)
		})
	}
}

func TestRecordedRequest_RedactHeaders(t *testing.T) {
	testCases := []struct {
		name            string
		request         RecordedRequest
		headersToRedact []string
		expectedHeaders map[string]string
	}{
		{
			name: "Redact single header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Accept":       "application/xml",
					"Content-Type": "application/json",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Content-Type"},
			expectedHeaders: map[string]string{
				"Accept": "application/xml",
			},
		},
		{
			name: "Redact multiple headers",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Accept":        "application/xml",
					"Content-Type":  "application/json",
					"Authorization": "Bearer token",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Content-Type", "Authorization"},
			expectedHeaders: map[string]string{
				"Accept": "application/xml",
			},
		},
		{
			name: "Redact non-existent header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Accept": "application/xml",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Non-Existent"},
			expectedHeaders: map[string]string{
				"Accept": "application/xml",
			},
		},
		{
			name: "Redact all headers",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Accept":       "application/xml",
					"Content-Type": "application/json",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Accept", "Content-Type"},
			expectedHeaders: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.request.RedactHeaders(tc.headersToRedact)
			require.Equal(t, tc.expectedHeaders, tc.request.Headers, "RedactHeaders() result mismatch")
		})
	}
}

func TestRecordedRequest_GetRecordFileName(t *testing.T) {
	testCases := []struct {
		name        string
		request     RecordedRequest
		expected    string
		expectedErr bool
	}{
		{
			name: "Request with test name header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Test-Name": "random test name",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected:    "random_test_name",
			expectedErr: false,
		},
		{
			name: "Request with empty test name header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Test-Name": "",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected:    "28f9ed309209353577523abbe4224d54aacf62c8f7cb2368b66a35088d830f4d",
			expectedErr: false,
		},
		{
			name: "Request with invalid test name header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Test-Name": "../invalid_name",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected:    "",
			expectedErr: true,
		},
		{
			name: "Request without test name header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Headers: map[string]string{
					"Accept":       "application/xml",
					"Content-Type": "application/json",
				},
				BodySegments:    []map[string]any{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected:    "cf125193d9ada2cc07b684455524cc5d61e39269892178db1a8046273f3268d1",
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.request.GetRecordingFileName()
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tc.expected, actual, "GetRecordFileName() result mismatch")
		})
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated error")
}
