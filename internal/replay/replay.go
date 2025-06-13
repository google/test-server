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
	"context"
	"fmt"
	"os"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
	"golang.org/x/sync/errgroup"
)

// Replay serves recorded responses for HTTP requests
func Replay(ctx context.Context, cfg *config.TestServerConfig, recordingDir string, redactor *redact.Redact) error {
	// Validate recording directory exists
	if _, err := os.Stat(recordingDir); os.IsNotExist(err) {
		return fmt.Errorf("recording directory does not exist: %s", recordingDir)
	}

	fmt.Printf("Replaying from directory: %s\n", recordingDir)

	errGroup, errCtx := errgroup.WithContext(ctx)

	for _, endpoint := range cfg.Endpoints {
		ep := endpoint // Capture range variable
		errGroup.Go(func() error {
			server := NewReplayHTTPServer(&endpoint, recordingDir, redactor)
			err := server.Start(errCtx)
			if err != nil {
				return fmt.Errorf("replay error for %s:%d: %w",
					ep.TargetHost, ep.TargetPort, err)
			}
			return nil
		})
	}

	return errGroup.Wait()
}
