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
	"regexp"
	"strings"
)

// REDACTED is the string used to replace redacted secrets.
const REDACTED = "REDACTED"

// Redact holds the compiled regex for redacting secrets.
type Redact struct {
	regex *regexp.Regexp
}

// NewRedact creates a new Redact instance with the given secrets.
func NewRedact(secrets []string) (*Redact, error) {
	filteredSecrets := []string{}
	for _, secret := range secrets {
		if secret != "" {
			filteredSecrets = append(filteredSecrets, regexp.QuoteMeta(secret))
		}
	}

	if len(filteredSecrets) == 0 {
		return &Redact{regex: nil}, nil // No secrets to redact
	}

	regexPattern := strings.Join(filteredSecrets, "|")
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, err
	}

	return &Redact{regex: re}, nil
}

// Headers redacts the secrets in the values of the http.Header.
func (r *Redact) Headers(headers map[string][]string) {
	if r == nil || r.regex == nil {
		return // No redactor or no secrets configured
	}
	for name, values := range headers {
		for i, value := range values {
			headers[name][i] = r.regex.ReplaceAllString(value, REDACTED)
		}
	}
}

// String redacts the secrets in the input string.
func (r *Redact) String(input string) string {
	if r == nil || r.regex == nil {
		return input // No redactor or no secrets configured
	}
	return r.regex.ReplaceAllString(input, REDACTED)
}

// Bytes redacts the secrets in the input byte slice.
func (r *Redact) Bytes(input []byte) []byte {
	if r == nil || r.regex == nil {
		return input // No redactor or no secrets configured
	}
	if input == nil {
		return nil // Return nil if input is nil
	}
	return r.regex.ReplaceAll(input, []byte(REDACTED))
}
