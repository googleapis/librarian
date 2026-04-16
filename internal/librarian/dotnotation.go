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
func setConfigValue(currentConfig *config.Config, path string, value string) error {
	switch path {
	case "version":
		currentConfig.Version = value
	case "sources.googleapis.commit":
		if currentConfig.Sources == nil {
			currentConfig.Sources = &config.Sources{}
		}
		if currentConfig.Sources.Googleapis == nil {
			currentConfig.Sources.Googleapis = &config.Source{}
		}
		currentConfig.Sources.Googleapis.Commit = value
		endpoints := &fetch.Endpoints{
			API:      githubAPI,
			Download: githubDownload,
		}
		repo := &fetch.RepoRef{
			Org:    "googleapis",
			Name:   "googleapis",
			Branch: value,
		}
		_, sha256, err := fetch.LatestCommitAndChecksum(endpoints, repo)
		if err != nil {
			return fmt.Errorf("failed to fetch checksum for commit %s: %w", value, err)
		}
		currentConfig.Sources.Googleapis.SHA256 = sha256
	default:
		return fmt.Errorf("%w: %s", errUnsupportedPath, path)
	}
	return nil
}

// getConfigValue returns the value at a specific path within the configuration.
// It only supports a limited set of paths for now.
func getConfigValue(currentConfig *config.Config, path string) (string, error) {
	switch path {
	case "version":
		return currentConfig.Version, nil
	case "sources.googleapis.commit":
		if currentConfig.Sources == nil || currentConfig.Sources.Googleapis == nil {
			return "", fmt.Errorf("%w: googleapis", errSourceNotConfigured)
		}
		return currentConfig.Sources.Googleapis.Commit, nil
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedPath, path)
	}
}
