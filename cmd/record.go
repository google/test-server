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
	"github.com/google/test-server/internal/record"
	"github.com/google/test-server/internal/redact"
	"github.com/spf13/cobra"
)

var recordingDir string

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Run test-server in record mode",
	Long: `Runs test-server in record mode, all request will be proxies to the
target server, and all requests and responses will be recorded.`,
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

		err = record.Record(config, recordingDir, redactor)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(recordCmd)
	recordCmd.Flags().StringVar(&recordingDir, "recording-dir", "recordings", "Directory to store recorded requests and responses")
}
