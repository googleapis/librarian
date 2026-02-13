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

// Find looks up the service config path and title override for a given API path,
// and validates that the API is allowed for the specified language.
//
// It first checks the API list for overrides and language restrictions,
// then searches for YAML files containing "type: google.api.Service",
// skipping any files ending in _gapic.yaml.
//
// The path should be relative to googleapisDir (e.g., "google/cloud/secretmanager/v1").
// Returns an API struct with Path, ServiceConfig, and Title fields populated.
// ServiceConfig and Title may be empty strings if not found or not configured.
//
// The Showcase API ("schema/google/showcase/v1beta1") is a special case:
// it does not live under https://github.com/googleapis/googleapis.
// For this API only, googleapisDir should point to showcase source dir instead.
func Find(googleapisDir, path, language string) (*API, error) {
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

	var err error
	result, err = validateAPI(path, language, result)
	if err != nil {
		return nil, err
	}

	// Find the service config if it hasn't been specified.
	if result.ServiceConfig == "" {
		serviceConfigPath, err := findServiceConfig(googleapisDir, result.Path)
		if err != nil {
			return nil, fmt.Errorf("error when finding service config for %s: %w", result.Path, err)
		}
		result.ServiceConfig = serviceConfigPath
	}

	// Populate API fields that haven't been explicitly specified, if we have
	// a service config.
	if result.ServiceConfig != "" {
		serviceConfig, err := Read(filepath.Join(googleapisDir, result.ServiceConfig))
		if err != nil {
			return nil, err
		}
		result = populateFromServiceConfig(result, serviceConfig)
	}
	return result, nil
}

// findServiceConfig searches the filesystem for a service config file under the
// given directory. An empty string is returned if no service config is found;
// otherwise, the location of the service config relative to the googleapis
// directory is returned.
func findServiceConfig(googleapisDir, path string) (string, error) {
	dir := filepath.Join(googleapisDir, path)
	_, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
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
			return "", err
		}
		if isServiceConfig {
			return filepath.Join(path, name), nil
		}
	}
	return "", nil
}

func populateFromServiceConfig(api *API, cfg *Service) *API {
	if api.Description == "" && cfg.GetDocumentation() != nil {
		api.Description = strings.TrimSpace(cfg.GetDocumentation().GetSummary())
	}
	if api.ServiceName == "" {
		api.ServiceName = cfg.GetName()
	}
	if api.ShortName == "" {
		api.ShortName = defaultShortName(api.ServiceName)
	}
	if api.Title == "" {
		api.Title = cfg.GetTitle()
	}
	api.ShortName = defaultShortName(api.ServiceName)
	publishing := cfg.GetPublishing()
	if publishing != nil {
		if api.NewIssueURI == "" {
			api.NewIssueURI = publishing.GetNewIssueUri()
		}
		if api.DocumentationURI == "" {
			api.DocumentationURI = publishing.GetDocumentationUri()
		}
		if publishing.GetApiShortName() != "" {
			api.ShortName = publishing.GetApiShortName()
		}
	}
	return api
}

// validateAPI checks if the given API path is allowed for the specified language.
//
// API paths starting with "google/cloud/" are allowed for all languages by default.
// If such a path is explicitly included in the allowlist, it must satisfy any
// language restrictions defined there.
//
// API paths not starting with "google/cloud/" must be explicitly included in the
// allowlist and satisfy its language restrictions.
func validateAPI(path, language string, api *API) (*API, error) {
	if api == nil && strings.HasPrefix(path, "google/cloud/") {
		return &API{Path: path}, nil
	}
	if api == nil {
		return nil, fmt.Errorf("API %s is not in allowlist", path)
	}
	if len(api.Languages) == 0 {
		return api, nil
	}
	for _, l := range api.Languages {
		if l == language {
			return api, nil
		}
	}
	return nil, fmt.Errorf("API %s is not allowed for language %s", path, language)
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

// defaultShortName returns the default short name from serviceName.
func defaultShortName(serviceName string) string {
	return strings.Split(serviceName, ".")[0]
}

// FindGRPCServiceConfig searches for gRPC service config files in the given
// API directory. It returns the path relative to googleapisDir for use with
// protoc's retry-config option. Returns empty string if no config is found.
// Returns an error if multiple matching files exist.
func FindGRPCServiceConfig(googleapisDir, path string) (string, error) {
	pattern := filepath.Join(googleapisDir, path, "*_grpc_service_config.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple gRPC service config files found in %q", path)
	}
	return filepath.Rel(googleapisDir, matches[0])
}
