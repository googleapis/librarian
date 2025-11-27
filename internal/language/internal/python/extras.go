// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

// This file contains code that isn't Python-specific, but which needs
// further consideration before putting it into the main codebase.
// Much of it was in internal/config in the librarianx repo.

// sourceDir fetches the given source
// TODO: dedupe from the Rust code
func sourceDir(source *config.Source, repo string) (string, error) {
	if source == nil {
		return "", nil
	}
	if source.Dir != "" {
		return source.Dir, nil
	}
	return fetch.RepoDir(repo, source.Commit, source.SHA256)
}

// prepareLibrary populates service configs, and applies defaults etc.
// This was originally in internal/config, with API filtering as well.
func prepareLibrary(cfg *config.Config, library *config.Library, googleapisDir string) error {
	// Populate service configs for the library
	if err := populateServiceConfigs(library, googleapisDir); err != nil {
		return err
	}

	// Apply version from versions map if not already set
	if library.Version == "" && cfg.Versions != nil {
		if version, ok := cfg.Versions[library.Name]; ok {
			library.Version = version
		}
	}

	if cfg.Default != nil && cfg.Default.Generate != nil {
		if library.Transport == "" {
			library.Transport = cfg.Default.Generate.Transport
		}
		if library.ReleaseLevel == "" {
			library.ReleaseLevel = cfg.Default.Generate.ReleaseLevel
		}
		if library.RestNumericEnums == nil {
			b := cfg.Default.Generate.RestNumericEnums
			library.RestNumericEnums = &b
		}
	}
	return nil
}

// populateServiceConfigs populates the APIServiceConfigs field for a library
// by looking up service configs for all its APIs.
func populateServiceConfigs(lib *config.Library, googleapisDir string) error {

	if lib.APIServiceConfigs == nil {
		lib.APIServiceConfigs = make(map[string]string)
	}

	// Get all API paths for this library
	apiPaths := lib.Channels
	if len(apiPaths) == 0 && lib.Channel != "" {
		apiPaths = []string{lib.Channel}
	}

	// Look up service config for each API
	for _, apiPath := range apiPaths {
		serviceConfigPath, err := findServiceConfigForAPI(googleapisDir, apiPath)
		if err != nil {
			return fmt.Errorf("failed to find service config for %s: %w", apiPath, err)
		}
		lib.APIServiceConfigs[apiPath] = serviceConfigPath
	}

	return nil
}

// findServiceConfigForAPI finds the service config YAML file for an API.
func findServiceConfigForAPI(googleapisDir, apiPath string) (string, error) {
	// Auto-discovery: find service config using pattern
	parts := strings.Split(apiPath, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid API path: %q", apiPath)
	}

	version := parts[len(parts)-1]
	dir := filepath.Join(googleapisDir, apiPath)

	// Pattern: *_<version>.yaml (e.g., secretmanager_v1.yaml)
	pattern := filepath.Join(dir, "*_"+version+".yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	// Filter out _gapic.yaml files
	var configs []string
	for _, m := range matches {
		if !strings.HasSuffix(m, "_gapic.yaml") {
			configs = append(configs, m)
		}
	}

	if len(configs) == 0 {
		return "", fmt.Errorf("no service config found for %q", apiPath)
	}

	if len(configs) > 1 {
		return "", fmt.Errorf("multiple service configs found for %q: %v", apiPath, configs)
	}

	return configs[0], nil
}
