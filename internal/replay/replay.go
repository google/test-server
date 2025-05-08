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
	"os"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
)

// Replay serves recorded responses for HTTP requests
func Replay(cfg *config.TestServerConfig, recordingDir string, redactor *redact.Redact) error {
	// Validate recording directory exists
	if _, err := os.Stat(recordingDir); os.IsNotExist(err) {
		return fmt.Errorf("recording directory does not exist: %s", recordingDir)
	}

	fmt.Printf("Replaying from directory: %s\n", recordingDir)

	// Start a server for each endpoint
	errChan := make(chan error, len(cfg.Endpoints))

	for _, endpoint := range cfg.Endpoints {
		go func(ep config.EndpointConfig) {
			server := NewReplayHTTPServer(&endpoint, recordingDir, redactor)
			err := server.Start()
			if err != nil {
				errChan <- fmt.Errorf("replay error for %s:%d: %w",
					ep.TargetHost, ep.TargetPort, err)
			}
		}(endpoint)
	}

	// Return the first error encountered, if any
	select {
	case err := <-errChan:
		return err
	default:
		// Block forever (or until interrupted)
		select {}
	}
}
