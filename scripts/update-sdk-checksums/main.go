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

// --- General Project Configuration ---
const (
	githubOwner = "google"
	githubRepo  = "test-server"
	projectName = "test-server"
)

// --- SDK Specific Configurations ---

// SDKConfig holds the unique properties for each SDK that needs updating.
type SDKConfig struct {
	Name              string   // e.g., "TypeScript", "Python"
	SDKDir            string   // Relative path to the SDK's directory
	InstallScriptFile []string // A list of files to update with the new version
	ChecksumsJSONFile string   // e.g., "checksums.json"
	VersionVarName    string   // The name of the version constant/variable in the install script
}

// sdksToUpdate is the list of all SDKs this script should manage.
// Add a new entry here to support another SDK.
var sdksToUpdate = []SDKConfig{
	{
		Name:              "TypeScript",
		SDKDir:            "sdks/typescript",
		InstallScriptFile: []string{"postinstall.js"},
		ChecksumsJSONFile: "checksums.json",
		VersionVarName:    "TEST_SERVER_VERSION",
	},
	{
		Name:              "Python",
		SDKDir:            "sdks/python/src/test_server_sdk",
		InstallScriptFile: []string{"install.py"},
		ChecksumsJSONFile: "checksums.json",
		VersionVarName:    "TEST_SERVER_VERSION",
	},
	{
		Name:              "Dotnet",
		SDKDir:            "sdks/dotnet",
		InstallScriptFile: []string{"BinaryInstaller.cs", "TestServerSdk.cs", "tools/installer/Program.cs"},
		ChecksumsJSONFile: "checksums.json",
		VersionVarName:    "TEST_SERVER_VERSION",
	},
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

func updateChecksumsJSON(checksumsJSONPath, newVersion string, newChecksumsMap map[string]string) error {
	allChecksums := make(map[string]map[string]string) // Reset if unmarshal fails

	if _, err := os.Stat(checksumsJSONPath); err == nil {
		existingJSON, errFileRead := os.ReadFile(checksumsJSONPath)
		if errFileRead != nil {
			return fmt.Errorf("failed to read existing %s: %w", checksumsJSONPath, errFileRead)
		}
		if len(existingJSON) > 0 {
			if errUnmarshal := json.Unmarshal(existingJSON, &allChecksums); errUnmarshal != nil {
				fmt.Printf("Warning: Could not parse existing %s, will overwrite. Error: %v\n", checksumsJSONPath, errUnmarshal)
				allChecksums = make(map[string]map[string]string)
			}
		}
	} else if !os.IsNotExist(err) { // If error is not "file does not exist", then it's a problem
		return fmt.Errorf("failed to stat %s: %w", checksumsJSONPath, err)
	}

	allChecksums[newVersion] = newChecksumsMap
	updatedJSON, err := json.MarshalIndent(allChecksums, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated checksums JSON: %w", err)
	}

	updatedJSON = append(updatedJSON, '\n')

	err = os.WriteFile(checksumsJSONPath, updatedJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated %s: %w", checksumsJSONPath, err)
	}
	fmt.Printf("Updated %s with checksums for version %s.\n", checksumsJSONPath, newVersion)
	return nil
}

func updateVersionInFile(filePath, newVersion, varName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	re := regexp.MustCompile(fmt.Sprintf(`(?m)(^\s*.*\b%s\b\s*=\s*['"]).*?(['"].*$)`, varName))

	if !re.Match(content) {
		// If the variable isn't in the file, it's not an error. Just skip it.
		fmt.Printf("Note: Did not find '%s' in %s, skipping update for this file.\n", varName, filePath)
		return nil
	}

	replacement := []byte(fmt.Sprintf(`${1}%s${2}`, newVersion))

	updatedContent := re.ReplaceAll(content, replacement)

	err = os.WriteFile(filePath, updatedContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated %s: %w", filePath, err)
	}
	fmt.Printf("Updated %s in %s to %s.\n", varName, filePath, newVersion)
	return nil
}

func main() {
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

	fmt.Printf("Fetching checksums for test-server version: %s\n", newVersion)
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

	var failedSDKs []string

	for _, sdk := range sdksToUpdate {
		fmt.Printf("\n--- Updating %s SDK ---\n", sdk.Name)

		sdkChecksumsJSONPath := filepath.Join(sdk.SDKDir, sdk.ChecksumsJSONFile)
		if err := updateChecksumsJSON(sdkChecksumsJSONPath, newVersion, newChecksumsMap); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating %s: %v\n", sdkChecksumsJSONPath, err)
			failedSDKs = append(failedSDKs, sdk.Name)
			continue
		}

		var sdkScriptUpdateFailed bool
		for _, scriptFile := range sdk.InstallScriptFile {
			sdkInstallScriptPath := filepath.Join(sdk.SDKDir, scriptFile)
			if err := updateVersionInFile(sdkInstallScriptPath, newVersion, sdk.VersionVarName); err != nil {
				fmt.Fprintf(os.Stderr, "Error updating %s: %v\n", sdkInstallScriptPath, err)
				sdkScriptUpdateFailed = true
				break
			}
		}

		if sdkScriptUpdateFailed {
			failedSDKs = append(failedSDKs, sdk.Name)
			continue // Move to the next SDK
		}
	}

	if len(failedSDKs) > 0 {
		fmt.Fprintf(os.Stderr, "\nUpdate failed for the following SDKs: %v\n", failedSDKs)
		os.Exit(1)
	}

	fmt.Println("\nSuccessfully updated all SDK checksums and versions.")
	fmt.Println("Then commit them to your repository.")
}
