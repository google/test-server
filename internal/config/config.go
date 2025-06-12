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
	"fmt"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type EndpointConfig struct {
	TargetType                 string              `yaml:"target_type"`
	TargetHost                 string              `yaml:"target_host"`
	TargetPort                 int64               `yaml:"target_port"`
	SourcePort                 int64               `yaml:"source_port"`
	SourceType                 string              `yaml:"source_type"`
	Health                     string              `yaml:"health"`
	RedactRequestHeaders       []string            `yaml:"redact_request_headers"`
	ResponseHeaderReplacements []HeaderReplacement `yaml:"response_header_replacements"`
}

type HeaderReplacement struct {
	Header  string `yaml:"header"`
	Regex   string `yaml:"regex"`
	Replace string `yaml:"replace"`
}

type TestServerConfig struct {
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

func ReadConfig(filename string) (*TestServerConfig, error) {
	return ReadConfigWithFs(afero.NewOsFs(), filename)
}

func ReadConfigWithFs(fs afero.Fs, filename string) (*TestServerConfig, error) {
	buf, err := afero.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	config := &TestServerConfig{}
	err = yaml.Unmarshal(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed parsing %s: %w", filename, err)
	}

	return config, nil
}
