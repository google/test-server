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
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	githubOwner = "google"
	githubRepo  = "test-server"
	projectName = "test-server"

	sdkDir               = "sdks/typescript"
	postinstallJSFile    = "postinstall.js"
	checksumsJSONFile    = "checksums.json"
	testServerVersionVar = "TEST_SERVER_VERSION"
)

var (
	sdkPostinstallPath   string
	sdkChecksumsJSONPath string
)

func initPaths() error {
	// Determine the project root. This assumes the script might be run
	// from the project root or from within its own directory 'scripts/update-sdk-checksums'.
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// If current working directory is 'scripts/update-sdk-checksums', go up two levels.
	// Otherwise, assume we are already at the project root.
	if filepath.Base(wd) == "update-sdk-checksums" && filepath.Base(filepath.Dir(wd)) == "scripts" {
		wd = filepath.Dir(filepath.Dir(wd))
	}

	sdkPostinstallPath = filepath.Join(wd, sdkDir, postinstallJSFile)
	sdkChecksumsJSONPath = filepath.Join(wd, sdkDir, checksumsJSONFile)

	// Verify postinstall.js path exists to give early feedback
	if _, err := os.Stat(sdkPostinstallPath); os.IsNotExist(err) {
		return fmt.Errorf("postinstall.js not found at %s. Ensure you are running the script from the project root, or the script needs path adjustment", sdkPostinstallPath)
	}
	// checksums.json might not exist initially, which is fine for updateChecksumsJSON.
	return nil
}

func fetchChecksumsTxt(version string) (string, error) {
	// The version in the checksums.txt filename typically does not have the 'v' prefix.
	versionForFileName := strings.TrimPrefix(version, "v")
	checksumsFileName := fmt.Sprintf("%s_%s_checksums.txt", projectName, versionForFileName)
	// The version in the download URL (tag) does have the 'v' prefix.
	checksumsURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", githubOwner, githubRepo, version, checksumsFileName)
	fmt.Printf("Downloading checksums file from %s...\n", checksumsURL)

	resp, err := http.Get(checksumsURL)
	if err != nil {
		return "", fmt.Errorf("failed to download checksums file from %s: %w", checksumsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for error message
		return "", fmt.Errorf("failed to download checksums file: status %s, body: %s", resp.Status, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(body), nil
}

func parseChecksumsTxt(checksumsText string) (map[string]string, error) {
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(checksumsText))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line) // Splits by any whitespace
		if len(parts) == 2 {
			// parts[0] is checksum, parts[1] is archive name
			checksums[parts[1]] = parts[0]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning checksums text: %w", err)
	}

	if len(checksums) == 0 {
		return nil, fmt.Errorf("no checksums could be parsed from the downloaded checksums.txt file. Is it empty or in an unexpected format?")
	}
	return checksums, nil
}

func updateChecksumsJSON(newVersion string, newChecksumsMap map[string]string) error {
	allChecksums := make(map[string]map[string]string)

	if _, err := os.Stat(sdkChecksumsJSONPath); err == nil { // Check if file exists
		existingJSON, errFileRead := os.ReadFile(sdkChecksumsJSONPath)
		if errFileRead != nil {
			return fmt.Errorf("failed to read existing %s: %w", sdkChecksumsJSONPath, errFileRead)
		}
		if len(existingJSON) > 0 { // Only unmarshal if not empty
			if errUnmarshal := json.Unmarshal(existingJSON, &allChecksums); errUnmarshal != nil {
				fmt.Printf("Warning: Could not parse existing %s, will overwrite. Error: %v\n", sdkChecksumsJSONPath, errUnmarshal)
				allChecksums = make(map[string]map[string]string) // Reset if unmarshal fails
			}
		}
	} else if !os.IsNotExist(err) { // If error is not "file does not exist", then it's a problem
		return fmt.Errorf("failed to stat %s: %w", sdkChecksumsJSONPath, err)
	}
	// If file does not exist, allChecksums remains an empty map, which is fine.

	allChecksums[newVersion] = newChecksumsMap
	updatedJSON, err := json.MarshalIndent(allChecksums, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated checksums JSON: %w", err)
	}

	// Append a newline character to match the Node script's output and common file ending.
	updatedJSON = append(updatedJSON, '\n')

	err = os.WriteFile(sdkChecksumsJSONPath, updatedJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated %s: %w", sdkChecksumsJSONPath, err)
	}
	fmt.Printf("Updated %s with checksums for version %s.\n", sdkChecksumsJSONPath, newVersion)
	return nil
}

func updatePostinstallVersion(newVersion string) error {
	content, err := os.ReadFile(sdkPostinstallPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", sdkPostinstallPath, err)
	}

	re := regexp.MustCompile(`(?m)^\s*const TEST_SERVER_VERSION = .*$`)

	if !re.Match(content) {
		return fmt.Errorf("could not find '%s' constant in %s. Pattern not matched: %s", testServerVersionVar, sdkPostinstallPath, re.String())
	}

	replacement := fmt.Sprintf("const TEST_SERVER_VERSION = '%s';", newVersion)

	updatedContent := re.ReplaceAllString(string(content), replacement)

	err = os.WriteFile(sdkPostinstallPath, []byte(updatedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated %s: %w", sdkPostinstallPath, err)
	}
	fmt.Printf("Updated %s in %s to %s.\n", testServerVersionVar, sdkPostinstallPath, newVersion)
	return nil
}

func main() {
	if err := initPaths(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing paths: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run scripts/update-sdk-checksums/main.go <version_tag>")
		fmt.Fprintln(os.Stderr, "Example: go run scripts/update-sdk-checksums/main.go v0.1.0")
		os.Exit(1)
	}
	newVersion := os.Args[1]
	if !strings.HasPrefix(newVersion, "v") {
		fmt.Fprintln(os.Stderr, "Error: version_tag must start with 'v' (e.g., v0.1.0)")
		os.Exit(1)
	}

	fmt.Printf("Updating TypeScript SDK to use test-server version: %s\n", newVersion)

	checksumsText, err := fetchChecksumsTxt(newVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError fetching checksums.txt: %v\n", err)
		os.Exit(1)
	}

	newChecksumsMap, err := parseChecksumsTxt(checksumsText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError parsing checksums.txt: %v\n", err)
		os.Exit(1)
	}

	if err := updateChecksumsJSON(newVersion, newChecksumsMap); err != nil {
		fmt.Fprintf(os.Stderr, "\nError updating %s: %v\n", sdkChecksumsJSONPath, err)
		os.Exit(1)
	}

	if err := updatePostinstallVersion(newVersion); err != nil {
		fmt.Fprintf(os.Stderr, "\nError updating %s: %v\n", sdkPostinstallPath, err)
		os.Exit(1)
	}

	fmt.Println("\nSuccessfully updated SDK checksums and version.")
	fmt.Println("Please review the changes in:")
	fmt.Printf("  - %s\n", sdkChecksumsJSONPath)
	fmt.Printf("  - %s\n", sdkPostinstallPath)
	fmt.Println("Then commit them to your repository.")
}
