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

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated error")
}

func TestRecordedRequest_Redact(t *testing.T) {
	testCases := []struct {
		name           string
		request        RecordedRequest
		secrets        []string
		expectedRequest RecordedRequest
	}{
		{
			name: "Redact secret in request line",
			request: RecordedRequest{
				Request: "GET /path/with/secret/abc HTTP/1.1",
				Header:  http.Header{},
				Body:    []byte{},
			},
			secrets: []string{"abc"},
			expectedRequest: RecordedRequest{
				Request: "GET /path/with/secret/REDACTED HTTP/1.1",
				Header:  http.Header{},
				Body:    []byte{},
			},
		},
		{
			name: "Redact secret in header",
			request: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header:  http.Header{"Authorization": []string{"Bearer secret_token_123"}},
				Body:    []byte{},
			},
			secrets: []string{"secret_token_123"},
			expectedRequest: RecordedRequest{
				Request: "GET / HTTP/1.1",
				Header:  http.Header{"Authorization": []string{"Bearer REDACTED"}},
				Body:    []byte{},
			},
		},
		{
			name: "Redact secret in body",
			request: RecordedRequest{
				Request: "POST /data HTTP/1.1",
				Header:  http.Header{},
				Body:    []byte("{\"token\": \"secret_value_456\"}"),
			},
			secrets: []string{"secret_value_456"},
			expectedRequest: RecordedRequest{
				Request: "POST /data HTTP/1.1",
				Header:  http.Header{},
				Body:    []byte("{\"token\": \"REDACTED\"}"),
			},
		},
		{
			name: "Redact multiple secrets",
			request: RecordedRequest{
				Request: "GET /path/abc?token=123 HTTP/1.1",
				Header:  http.Header{"X-Api-Key": []string{"key_value_xyz"}},
				Body:    []byte("user=test&password=password123"),
			},
			secrets: []string{"abc", "123", "key_value_xyz", "password123"},
			expectedRequest: RecordedRequest{
				Request: "GET /path/REDACTED?token=REDACTED HTTP/1.1",
				Header:  http.Header{"X-Api-Key": []string{"REDACTED"}},
				Body:    []byte("user=test&password=REDACTED"),
			},
		},
		{
			name: "No secrets to redact",
			request: RecordedRequest{
				Request: "GET /path HTTP/1.1",
				Header:  http.Header{"X-Api-Key": []string{"some_value"}},
				Body:    []byte("user=test"),
			},
			secrets: []string{},
			expectedRequest: RecordedRequest{
				Request: "GET /path HTTP/1.1",
				Header:  http.Header{"X-Api-Key": []string{"some_value"}},
				Body:    []byte("user=test"),
			},
		},
		{
			name: "Empty secret in list",
			request: RecordedRequest{
				Request: "GET /path/abc HTTP/1.1",
				Header:  http.Header{},
				Body:    []byte{},
			},
			secrets: []string{"", "abc"},
			expectedRequest: RecordedRequest{
				Request: "GET /path/REDACTED HTTP/1.1",
				Header:  http.Header{},
				Body:    []byte{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.request.Redact(tc.secrets)
			require.Equal(t, tc.expectedRequest.Request, tc.request.Request)
			require.Equal(t, tc.expectedRequest.Header, tc.request.Header)
			require.Equal(t, tc.expectedRequest.Body, tc.request.Body)
		})
	}
}

func TestRecordedResponse_Redact(t *testing.T) {
	testCases := []struct {
		name            string
		response        RecordedResponse
		secrets         []string
		expectedResponse RecordedResponse
	}{
		{
			name: "Redact secret in header",
			response: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"Set-Cookie": []string{"sessionid=secret_session_id_789"}},
				Body:       []byte{},
			},
			secrets: []string{"secret_session_id_789"},
			expectedResponse: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"Set-Cookie": []string{"sessionid=REDACTED"}},
				Body:       []byte{},
			},
		},
		{
			name: "Redact secret in body",
			response: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       []byte("{\"user_token\": \"secret_user_token_abc\"}"),
			},
			secrets: []string{"secret_user_token_abc"},
			expectedResponse: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{},
				Body:       []byte("{\"user_token\": \"REDACTED\"}"),
			},
		},
		{
			name: "Redact multiple secrets",
			response: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"X-Response-Secret": []string{"resp_secret_1"}},
				Body:       []byte("token=resp_secret_2&id=123"),
			},
			secrets: []string{"resp_secret_1", "resp_secret_2"},
			expectedResponse: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"X-Response-Secret": []string{"REDACTED"}},
				Body:       []byte("token=REDACTED&id=123"),
			},
		},
		{
			name: "No secrets to redact",
			response: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"X-Response-Secret": []string{"some_value"}},
				Body:       []byte("user=test"),
			},
			secrets: []string{},
			expectedResponse: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"X-Response-Secret": []string{"some_value"}},
				Body:       []byte("user=test"),
			},
		},
		{
			name: "Empty secret in list",
			response: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"Set-Cookie": []string{"sessionid=secret_session_id_789"}},
				Body:       []byte{},
			},
			secrets: []string{"", "secret_session_id_789"},
			expectedResponse: RecordedResponse{
				StatusCode: 200,
				Header:     http.Header{"Set-Cookie": []string{"sessionid=REDACTED"}},
				Body:       []byte{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.response.Redact(tc.secrets)
			require.Equal(t, tc.expectedResponse.StatusCode, tc.response.StatusCode)
			require.Equal(t, tc.expectedResponse.Header, tc.response.Header)
			require.Equal(t, tc.expectedResponse.Body, tc.response.Body)
		})
	}
}

