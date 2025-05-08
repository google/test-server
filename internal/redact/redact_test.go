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

package redact

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedact_String(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		secrets        []string
		expectedOutput string
	}{
		{
			name:           "Redact single secret",
			input:          "This is a secret: abc",
			secrets:        []string{"abc"},
			expectedOutput: "This is a secret: REDACTED",
		},
		{
			name:           "Redact multiple secrets",
			input:          "Secret1: 123, Secret2: xyz",
			secrets:        []string{"123", "xyz"},
			expectedOutput: "Secret1: REDACTED, Secret2: REDACTED",
		},
		{
			name:           "No secrets to redact",
			input:          "No secrets here",
			secrets:        []string{},
			expectedOutput: "No secrets here",
		},
		{
			name:           "Empty input string",
			input:          "",
			secrets:        []string{"abc"},
			expectedOutput: "",
		},
		{
			name:           "Empty secret in list",
			input:          "This is a secret: abc",
			secrets:        []string{"", "abc"},
			expectedOutput: "This is a secret: REDACTED",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			redactor, err := NewRedact(tc.secrets)
			require.NoError(t, err)
			actualOutput := redactor.String(tc.input)
			require.Equal(t, tc.expectedOutput, actualOutput)
		})
	}
}

func TestRedact_Bytes(t *testing.T) {
	testCases := []struct {
		name           string
		input          []byte
		secrets        []string
		expectedOutput []byte
	}{
		{
			name:           "Redact single secret",
			input:          []byte("This is a secret: abc"),
			secrets:        []string{"abc"},
			expectedOutput: []byte("This is a secret: REDACTED"),
		},
		{
			name:           "Redact multiple secrets",
			input:          []byte("Secret1: 123, Secret2: xyz"),
			secrets:        []string{"123", "xyz"},
			expectedOutput: []byte("Secret1: REDACTED, Secret2: REDACTED"),
		},
		{
			name:           "No secrets to redact",
			input:          []byte("No secrets here"),
			secrets:        []string{},
			expectedOutput: []byte("No secrets here")},
		{
			name:           "Empty input bytes",
			input:          []byte{},
			secrets:        []string{"abc"},
			expectedOutput: nil,
		},
		{
			name:           "Empty secret in list",
			input:          []byte("This is a secret: abc"),
			secrets:        []string{"", "abc"},
			expectedOutput: []byte("This is a secret: REDACTED"),
		},
		{
			name:           "Nil input bytes",
			input:          nil,
			secrets:        []string{"abc"},
			expectedOutput: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			redactor, err := NewRedact(tc.secrets)
			require.NoError(t, err)
			actualOutput := redactor.Bytes(tc.input)
			require.Equal(t, tc.expectedOutput, actualOutput)
		})
	}
}

func TestRedact_Headers(t *testing.T) {
	testCases := []struct {
		name            string
		headers         http.Header
		secrets         []string
		expectedHeaders http.Header
	}{
		{
			name: "Redact secret in single header value",
			headers: http.Header{
				"Authorization": []string{"Bearer secret_token_123"},
				"Content-Type":  []string{"application/json"},
			},
			secrets: []string{"secret_token_123"},
			expectedHeaders: http.Header{
				"Authorization": []string{"Bearer REDACTED"},
				"Content-Type":  []string{"application/json"},
			},
		},
		{
			name: "Redact secret in multiple header values",
			headers: http.Header{
				"Set-Cookie": []string{"sessionid=secret_session_id_789", "other=value"},
				"X-Api-Key":  []string{"key_value_xyz"},
			},
			secrets: []string{"secret_session_id_789", "key_value_xyz"},
			expectedHeaders: http.Header{
				"Set-Cookie": []string{"sessionid=REDACTED", "other=value"},
				"X-Api-Key":  []string{"REDACTED"},
			},
		},
		{
			name: "No secrets to redact",
			headers: http.Header{
				"Authorization": []string{"Bearer token"},
			},
			secrets: []string{},
			expectedHeaders: http.Header{
				"Authorization": []string{"Bearer token"},
			},
		},
		{
			name: "Empty secret in list",
			headers: http.Header{
				"Authorization": []string{"Bearer secret_token_123"},
			},
			secrets: []string{"", "secret_token_123"},
			expectedHeaders: http.Header{
				"Authorization": []string{"Bearer REDACTED"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			redactor, err := NewRedact(tc.secrets)
			require.NoError(t, err)
			// Create a copy of the headers to avoid modifying the original test case data
			headersCopy := make(http.Header)
			for k, v := range tc.headers {
				headersCopy[k] = append([]string{}, v...)
			}
			redactor.Headers(headersCopy)
			require.Equal(t, tc.expectedHeaders, headersCopy)
		})
	}
}
