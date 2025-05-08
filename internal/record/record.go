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

package record

import (
	"fmt"
	"os"
	"sync"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
)

func Record(cfg *config.TestServerConfig, recordingDir string, redactor *redact.Redact) error {
	// Create recording directory if it doesn't exist
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return fmt.Errorf("failed to create recording directory: %w", err)
	}

	fmt.Printf("Recording to directory: %s\n", recordingDir)
	var wg sync.WaitGroup
	errChan := make(chan error, len(cfg.Endpoints))

	// Start a proxy for each endpoint
	for _, endpoint := range cfg.Endpoints {
		wg.Add(1)
		go func(ep config.EndpointConfig) {
			defer wg.Done()

			fmt.Printf("Starting server for %v\n", endpoint)
			proxy := NewRecordingHTTPSProxy(&endpoint, recordingDir, redactor)
			err := proxy.Start()

			if err != nil {
				errChan <- fmt.Errorf("proxy error for %s:%d: %w",
					ep.TargetHost, ep.TargetPort, err)
			}
		}(endpoint)
	}

	// Wait for all proxies to complete (they shouldn't unless there's an error)
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Return the first error encountered, if any
	for err := range errChan {
		return err
	}

	// Block forever (or until interrupted)
	select {}
}
