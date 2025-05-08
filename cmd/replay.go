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

package cmd

import (
	"os"
	"strings"

	"github.com/google/test-server/internal/config"
	"github.com/google/test-server/internal/redact"
	"github.com/google/test-server/internal/replay"
	"github.com/spf13/cobra"
)

var replayRecordingDir string

// replayCmd represents the replay command
var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Replay recorded HTTP responses",
	Long: `Replay mode serves recorded HTTP responses for matching requests.
It listens on the configured source ports and returns recorded responses
when it finds a matching request. Returns a 404 error if no matching
recording is found.`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := config.ReadConfig(cfgFile)
		if err != nil {
			panic(err)
		}

		secrets := os.Getenv("TEST_SERVER_SECRETS")
		redactor, err := redact.NewRedact(strings.Split(secrets, ","))
		if err != nil {
			panic(err)
		}

		err = replay.Replay(config, replayRecordingDir, redactor)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(replayCmd)
	replayCmd.Flags().StringVar(&replayRecordingDir, "recording-dir", "recordings", "Directory containing recorded requests and responses")
}
