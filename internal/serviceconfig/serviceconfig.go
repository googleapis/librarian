// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package serviceconfig reads and parses API service config files.
package serviceconfig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/yaml"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/encoding/protojson"
)

// Type aliases for genproto service config types.
type (
	Service            = serviceconfig.Service
	Documentation      = serviceconfig.Documentation
	DocumentationRule  = serviceconfig.DocumentationRule
	Backend            = serviceconfig.Backend
	BackendRule        = serviceconfig.BackendRule
	Authentication     = serviceconfig.Authentication
	AuthenticationRule = serviceconfig.AuthenticationRule
	OAuthRequirements  = serviceconfig.OAuthRequirements
)

// Read reads a service config from a YAML file and returns it as a Service
// proto. The file is parsed as YAML, converted to JSON, and then unmarshaled
// into a Service proto.
func Read(serviceConfigPath string) (*Service, error) {
	y, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error reading service config [%s]: %w", serviceConfigPath, err)
	}

	yamlData, err := yaml.Unmarshal[any](y)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML [%s]: %w", serviceConfigPath, err)
	}
	j, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("error converting YAML to JSON [%s]: %w", serviceConfigPath, err)
	}

	cfg := &Service{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(j, cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling service config [%s]: %w", serviceConfigPath, err)
	}

	// An API Service Config will always have a `name` so if it is not populated,
	// it's an invalid config.
	if cfg.GetName() == "" {
		return nil, fmt.Errorf("missing name in service config file [%s]", serviceConfigPath)
	}
	return cfg, nil
}

// Find looks up the service config path and title override for a given API path.
// It first checks the API allowlist for overrides, then searches for YAML files
// containing "type: google.api.Service", skipping any files ending in _gapic.yaml.
//
// The path should be relative to googleapisDir (e.g., "google/cloud/secretmanager/v1").
// Returns an API struct with Path, ServiceConfig, and Title fields populated.
// ServiceConfig and Title may be empty strings if not found or not configured.
func Find(googleapisDir, path string) (*API, error) {
	result := &API{Path: path}

	// Check allowlist for overrides
	for _, api := range APIs {
		if api.Path == path {
			result.ServiceConfig = api.ServiceConfig
			result.Title = api.Title
			result.Discovery = api.Discovery
			break
		}
	}

	// If service config is overridden in allowlist, use it
	if result.ServiceConfig != "" {
		return result, nil
	}

	// Search filesystem for service config
	dir := filepath.Join(googleapisDir, path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		if strings.HasSuffix(name, "_gapic.yaml") {
			continue
		}

		filePath := filepath.Join(dir, name)
		isServiceConfig, err := isServiceConfigFile(filePath)
		if err != nil {
			return nil, err
		}
		if isServiceConfig {
			result.ServiceConfig = filepath.Join(path, name)
			return result, nil
		}
	}
	return result, nil
}

// isServiceConfigFile checks if the file contains "type: google.api.Service".
func isServiceConfigFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for i := 0; i < 20 && scanner.Scan(); i++ {
		if strings.TrimSpace(scanner.Text()) == "type: google.api.Service" {
			return true, nil
		}
	}
	return false, scanner.Err()
}
