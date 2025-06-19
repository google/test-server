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
	"context"
	"fmt"
	"os"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
	"golang.org/x/sync/errgroup"
)

func Record(ctx context.Context, cfg *config.TestServerConfig, recordingDir string, redactor *redact.Redact) error {
	// Create recording directory if it doesn't exist
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return fmt.Errorf("failed to create recording directory: %w", err)
	}

	fmt.Printf("Recording to directory: %s\n", recordingDir)
	errGroup, errCtx := errgroup.WithContext(ctx)

	// Start a proxy for each endpoint
	for _, endpoint := range cfg.Endpoints {
		ep := endpoint
		errGroup.Go(func() error {
			fmt.Printf("Starting server for %v\n", ep)
			proxy := NewRecordingHTTPSProxy(&ep, recordingDir, redactor)
			err := proxy.Start(errCtx)
			if err != nil {
				return fmt.Errorf("proxy error for %s:%d: %w",
					ep.TargetHost, ep.TargetPort, err)
			}
			return nil
		})
	}

	return errGroup.Wait()
}
