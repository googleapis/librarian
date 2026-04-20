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
	"strings"

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
func setConfigValue(cfg *config.Config, path string, value string) (*config.Config, error) {
	switch path {
	case "version":
		cfg.Version = value
		return cfg, nil
	}

	// Handle paths like "sources.<source_name>.<field>"
	parts := strings.Split(path, ".")
	if len(parts) == 3 && parts[0] == "sources" {
		sourceName := parts[1]
		field := parts[2]

		var commit, sha256 string
		var err error

		// Fallible operation first!
		if field == "commit" {
			commit, sha256, err = fetchCommitAndChecksum(sourceName, value)
			if err != nil {
				return nil, err
			}
		}

		// Now it is safe to mutate!
		if cfg.Sources == nil {
			cfg.Sources = &config.Sources{}
		}

		var source *config.Source
		switch sourceName {
		case "conformance":
			if cfg.Sources.Conformance == nil {
				cfg.Sources.Conformance = &config.Source{}
			}
			source = cfg.Sources.Conformance
		case "discovery":
			if cfg.Sources.Discovery == nil {
				cfg.Sources.Discovery = &config.Source{}
			}
			source = cfg.Sources.Discovery
		case "googleapis":
			if cfg.Sources.Googleapis == nil {
				cfg.Sources.Googleapis = &config.Source{}
			}
			source = cfg.Sources.Googleapis
		case "protobuf":
			if cfg.Sources.ProtobufSrc == nil {
				cfg.Sources.ProtobufSrc = &config.Source{}
			}
			source = cfg.Sources.ProtobufSrc
		case "showcase":
			if cfg.Sources.Showcase == nil {
				cfg.Sources.Showcase = &config.Source{}
			}
			source = cfg.Sources.Showcase
		default:
			return nil, fmt.Errorf("unsupported source: %s", sourceName)
		}

		switch field {
		case "commit":
			source.Commit = commit
			source.SHA256 = sha256
		case "branch":
			source.Branch = value
		case "dir":
			source.Dir = value
		case "sha256":
			source.SHA256 = value
		case "subpath":
			source.Subpath = value
		default:
			return nil, fmt.Errorf("unsupported field %q for source %q", field, sourceName)
		}
		return cfg, nil
	}

	return nil, fmt.Errorf("%w: %s", errUnsupportedPath, path)
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

// fetchCommitAndChecksum fetches the latest commit and checksum for the given source and branch.
func fetchCommitAndChecksum(sourceName, branch string) (commit, sha256 string, err error) {
	repoRef, ok := sourceRepos[sourceName]
	if !ok {
		return "", "", fmt.Errorf("unknown source repository for %s", sourceName)
	}
	repoRef.Branch = branch

	endpoints := &fetch.Endpoints{
		API:      githubAPI,
		Download: githubDownload,
	}

	return fetch.LatestCommitAndChecksum(endpoints, &repoRef)
}
