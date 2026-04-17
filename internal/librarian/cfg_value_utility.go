// Copyright 2026 Google LLC
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

package librarian

import (
	"errors"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

var (
	// errUnsupportedPath is returned when a dot-notation path is not supported.
	errUnsupportedPath = errors.New("unsupported config path")

	// errSourceNotConfigured is returned when attempting to access a source that is not configured.
	errSourceNotConfigured = errors.New("source not configured")
)

// setConfigValue sets a value at a specific path within the configuration.
// It only supports a limited set of paths for now.
func setConfigValue(cfg *config.Config, path string, value string) (*config.Config, error) {
	switch path {
	case "version":
		cfg.Version = value
	case "sources.googleapis.commit":
		endpoints := &fetch.Endpoints{
			API:      githubAPI,
			Download: githubDownload,
		}
		repo := sourceRepos["googleapis"]
		repo.Branch = value
		sha256, err := fetch.CommitChecksum(endpoints, &repo, value)
		if err != nil {
			return nil, fmt.Errorf("fetching checksum for %s: %w", value, err)
		}

		if cfg.Sources == nil {
			cfg.Sources = &config.Sources{}
		}
		if cfg.Sources.Googleapis == nil {
			cfg.Sources.Googleapis = &config.Source{}
		}
		cfg.Sources.Googleapis.Commit = value
		cfg.Sources.Googleapis.SHA256 = sha256
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedPath, path)
	}
	return cfg, nil
}

// getConfigValue returns the value at a specific path within the configuration.
// It only supports a limited set of paths for now.
func getConfigValue(cfg *config.Config, path string) (string, error) {
	switch path {
	case "version":
		return cfg.Version, nil
	case "sources.googleapis.commit":
		if cfg.Sources == nil || cfg.Sources.Googleapis == nil {
			return "", fmt.Errorf("%w: googleapis", errSourceNotConfigured)
		}
		return cfg.Sources.Googleapis.Commit, nil
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedPath, path)
	}
}
