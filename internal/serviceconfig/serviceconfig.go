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
	"errors"
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
		return nil, fmt.Errorf("failed to read service config %q: %w", serviceConfigPath, err)
	}

	yamlData, err := yaml.Unmarshal[any](y)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %q: %w", serviceConfigPath, err)
	}
	j, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON in %q: %w", serviceConfigPath, err)
	}

	cfg := &Service{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(j, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service config %q: %w", serviceConfigPath, err)
	}

	// An API Service Config will always have a `name` so if it is not populated,
	// it's an invalid config.
	if cfg.GetName() == "" {
		return nil, fmt.Errorf("missing name in service config %q", serviceConfigPath)
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
	var result *API
	for _, api := range APIs {
		// The path for OpenAPI and discovery documents are in
		// googleapis/google-cloud-rust and
		// googleapis/discovery-artifact-manager, respectively.
		// The api.Path field is that API path in googleapis/googleapis.
		if api.Path == path || api.OpenAPI == path || api.Discovery == path {
			// Create a copy of the API struct to allow modifications to
			// result.ServiceConfig without affecting the APIs slice.
			r := api
			result = &r
			break
		}
	}

	// TODO(https://github.com/googleapis/librarian/issues/3627): all APIs
	// should be listed in the allowlist
	if result == nil {
		return &API{Path: path}, nil
	}

	// If service config is overridden in allowlist, use it
	if result.ServiceConfig != "" {
		return populateTitle(googleapisDir, result)
	}

	// Search filesystem for service config
	dir := filepath.Join(googleapisDir, result.Path)
	_, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return nil, err
	}
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
			result.ServiceConfig = filepath.Join(result.Path, name)
			return populateTitle(googleapisDir, result)
		}
	}
	return result, nil
}

func populateTitle(googleapisDir string, api *API) (*API, error) {
	if api.Title != "" || api.ServiceConfig == "" {
		return api, nil
	}
	cfg, err := Read(filepath.Join(googleapisDir, api.ServiceConfig))
	if err != nil {
		return nil, err
	}
	api.Title = cfg.GetTitle()
	return api, nil
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
