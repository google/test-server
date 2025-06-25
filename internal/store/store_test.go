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
				Header:          http.Header{},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: HeadSHA + "\nServer Address: \nPort: 0\nProtocol: \n********************************************************************************\n\n\n\n",
		},
		{
			name: "Request with headers",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header: http.Header{
					"Accept":       []string{"application/xml"},
					"Content-Type": []string{"application/json"},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: HeadSHA + "\nServer Address: \nPort: 0\nProtocol: \n********************************************************************************\nGET / HTTP/1.1\nAccept: application/xml\nContent-Type: application/json\n\n\n",
		},
		{
			name: "Request with body",
			request: RecordedRequest{
				Request:         "POST /data HTTP/1.1",
				Header:          http.Header{},
				Body:            []byte("{\"key\": \"value\"}"),
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: HeadSHA + "\nServer Address: \nPort: 0\nProtocol: \n********************************************************************************\nPOST /data HTTP/1.1\n\n\n{\"key\": \"value\"}",
		},
		{
			name: "Request with previous request SHA256 sum",
			request: RecordedRequest{
				Request:         "GET / HTTP/1.1",
				Header:          http.Header{},
				Body:            []byte{},
				PreviousRequest: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20\nServer Address: \nPort: 0\nProtocol: \n********************************************************************************\nGET / HTTP/1.1\n\n\n",
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
				req, _ := http.NewRequest("POST", "http://example.com/test", bytes.NewBuffer([]byte("test body")))
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
				Header:          http.Header{"Content-Type": []string{"application/json"}},
				Body:            []byte("test body"),
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
				Header:          http.Header{},
				Body:            []byte{},
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
			require.Equal(t, tc.expected.Header, recordedRequest.Header)
			require.Equal(t, tc.expected.Body, recordedRequest.Body)
			require.Equal(t, tc.expected.PreviousRequest, recordedRequest.PreviousRequest)
		})
	}
}

func TestRecordedRequest_RedactHeaders(t *testing.T) {
	testCases := []struct {
		name            string
		request         RecordedRequest
		headersToRedact []string
		expectedHeaders http.Header
	}{
		{
			name: "Redact single header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header: http.Header{
					"Accept":       []string{"application/xml"},
					"Content-Type": []string{"application/json"},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Content-Type"},
			expectedHeaders: http.Header{
				"Accept": []string{"application/xml"},
			},
		},
		{
			name: "Redact multiple headers",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header: http.Header{
					"Accept":        []string{"application/xml"},
					"Content-Type":  []string{"application/json"},
					"Authorization": []string{"Bearer token"},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Content-Type", "Authorization"},
			expectedHeaders: http.Header{
				"Accept": []string{"application/xml"},
			},
		},
		{
			name: "Redact non-existent header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header: http.Header{
					"Accept": []string{"application/xml"},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Non-Existent"},
			expectedHeaders: http.Header{
				"Accept": []string{"application/xml"},
			},
		},
		{
			name: "Redact all headers",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header: http.Header{
					"Accept":       []string{"application/xml"},
					"Content-Type": []string{"application/json"},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			headersToRedact: []string{"Accept", "Content-Type"},
			expectedHeaders: http.Header{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.request.RedactHeaders(tc.headersToRedact)
			require.Equal(t, tc.expectedHeaders, tc.request.Header, "RedactHeaders() result mismatch")
		})
	}
}

func TestRecordedRequest_Deserialize(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    *RecordedRequest
		expectedErr bool
	}{
		{
			name:  "Valid serialized request",
			input: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20\nServer Address: example.com\nPort: 8080\nProtocol: http\n********************************************************************************\nGET / HTTP/1.1\nAccept: application/xml\nContent-Type: application/json\n\n\n{\"key\": \"value\"}",
			expected: &RecordedRequest{
				Request:         "GET / HTTP/1.1",
				Header:          http.Header{"Accept": []string{"application/xml"}, "Content-Type": []string{"application/json"}},
				Body:            []byte("{\"key\": \"value\"}"),
				PreviousRequest: "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
				ServerAddress:   "example.com",
				Port:            8080,
				Protocol:        "http",
			},
			expectedErr: false,
		},
		{
			name:        "Invalid serialized request - missing separator",
			input:       "GET / HTTP/1.1\nAccept: application/xml",
			expected:    nil,
			expectedErr: true,
		},
		{
			name:        "Empty input",
			input:       "",
			expected:    nil,
			expectedErr: true,
		},
		{
			name:        "Invalid serialized request - invalid port",
			input:       "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20\nServer Address: example.com\nPort: invalid\nProtocol: http\n********************************************************************************\nGET / HTTP/1.1\nAccept: application/xml\nContent-Type: application/json\n\n\n{\"key\": \"value\"}",
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := Deserialize(tc.input)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
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
				Header: http.Header{
					"Test-Name": []string{"random test name"},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected:    "random test name",
			expectedErr: false,
		},
		{
			name: "Request with empty test name header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header: http.Header{
					"Test-Name": []string{""},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected:    "f824dd099907ed4549822de827b075a7578baadebf08c5bc7303ead90a8f9ff7",
			expectedErr: false,
		},
		{
			name: "Request with invalid test name header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header: http.Header{
					"Test-Name": []string{"../invalid_name"},
				},
				Body:            []byte{},
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
				Header: http.Header{
					"Accept":       []string{"application/xml"},
					"Content-Type": []string{"application/json"},
				},
				Body:            []byte{},
				PreviousRequest: HeadSHA,
				ServerAddress:   "",
				Port:            0,
				Protocol:        "",
			},
			expected:    "fc060aea9a2bf35da16ed18c6be577ca64d0f91d681d5db385082df61ecf4ccf",
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
