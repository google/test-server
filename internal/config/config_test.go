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

package config

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestReadConfigWithFs(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		filePath    string
		wantErr     bool
		wantConfig  *TestServerConfig
	}{
		{
			name: "valid config",
			fileContent: `endpoints:
  - target_host: www.google.com
    target_port: 443
    source_port: 1443
    source_type: http
    target_type: https
    redact_request_headers:
      - X-Goog-Api-Key
    redact_response_headers:
      - X-Google-*
  - target_host: api.example.com
    target_port: 8080
    source_port: 8081
    source_type: tcp
    target_type: tcp`,
			filePath: "/test-config.yaml",
			wantErr:  false,
			wantConfig: &TestServerConfig{
				Endpoints: []EndpointConfig{
					{
						TargetHost:            "www.google.com",
						TargetPort:            443,
						SourcePort:            1443,
						SourceType:            "http",
						TargetType:            "https",
						RedactRequestHeaders:  []string{"X-Goog-Api-Key"},
						RedactResponseHeaders: []string{"X-Google-*"},
					},
					{
						TargetHost: "api.example.com",
						TargetPort: 8080,
						SourcePort: 8081,
						SourceType: "tcp",
						TargetType: "tcp",
					},
				},
			},
		},
		{
			name:        "non-existent file",
			fileContent: "",
			filePath:    "/non-existent.yaml",
			wantErr:     true,
			wantConfig:  nil,
		},
		{
			name:        "invalid yaml",
			fileContent: "invalid: - yaml: content",
			filePath:    "/invalid.yaml",
			wantErr:     true,
			wantConfig:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup in-memory filesystem
			fs := afero.NewMemMapFs()

			// Create test file if needed
			if tt.fileContent != "" {
				err := afero.WriteFile(fs, tt.filePath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			// Call the function
			got, err := ReadConfigWithFs(fs, tt.filePath)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfigWithFs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Skip config check if we expected an error
			if tt.wantErr {
				return
			}

			// Check config contents
			assert.NoError(t, err)
			assert.Equal(t, tt.wantConfig, got, "Config structs should match")
		})
	}
}
