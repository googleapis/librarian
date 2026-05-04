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
)

// setConfigValue sets a value at a specific path within the configuration.
func setConfigValue(cfg *config.Config, path string, value string) (*config.Config, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		switch parts[0] {
		case "version":
			cfg.Version = value
			return cfg, nil
		default:
			return nil, fmt.Errorf("%w: %s", errUnsupportedPath, path)
		}
	}
	if len(parts) == 3 && parts[0] == "sources" {
		sourceName := parts[1]
		fieldName := parts[2]
		switch sourceName {
		case "conformance", "discovery", "googleapis", "protobuf", "showcase":
			switch fieldName {
			case "commit":
				commit, sha256, err := fetchSourceCommitAndChecksum("sources."+sourceName, value)
				if err != nil {
					return nil, err
				}
				src := getOrCreateSource(cfg, sourceName)
				src.Commit = commit
				src.SHA256 = sha256
				return cfg, nil
			case "sha256":
				getOrCreateSource(cfg, sourceName).SHA256 = value
				return cfg, nil
			case "dir":
				getOrCreateSource(cfg, sourceName).Dir = value
				return cfg, nil
			case "subpath":
				getOrCreateSource(cfg, sourceName).Subpath = value
				return cfg, nil
			default:
				return nil, fmt.Errorf("%w: %s", errUnsupportedPath, path)
			}
		default:
			return nil, fmt.Errorf("%w: %s", errUnsupportedPath, path)
		}
	}
	return nil, fmt.Errorf("%w: %s", errUnsupportedPath, path)
}

// getConfigValue returns the value at a specific path within the configuration.
func getConfigValue(cfg *config.Config, path string) (string, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		switch parts[0] {
		case "version":
			return cfg.Version, nil
		default:
			return "", fmt.Errorf("%w: %s", errUnsupportedPath, path)
		}
	}
	if len(parts) == 3 && parts[0] == "sources" {
		sourceName := parts[1]
		fieldName := parts[2]
		switch sourceName {
		case "conformance", "discovery", "googleapis", "protobuf", "showcase":
			src := getSource(cfg, sourceName)
			if src == nil {
				return "", nil
			}
			switch fieldName {
			case "commit":
				return src.Commit, nil
			case "sha256":
				return src.SHA256, nil
			case "dir":
				return src.Dir, nil
			case "subpath":
				return src.Subpath, nil
			default:
				return "", fmt.Errorf("%w: %s", errUnsupportedPath, path)
			}
		default:
			return "", fmt.Errorf("%w: %s", errUnsupportedPath, path)
		}
	}
	return "", fmt.Errorf("%w: %s", errUnsupportedPath, path)
}

// getOrCreateSource returns a Source pointer from the Config, initializing if needed.
func getOrCreateSource(cfg *config.Config, sourceName string) *config.Source {
	if cfg.Sources == nil {
		cfg.Sources = &config.Sources{}
	}
	switch sourceName {
	case "conformance":
		if cfg.Sources.Conformance == nil {
			cfg.Sources.Conformance = &config.Source{}
		}
		return cfg.Sources.Conformance
	case "discovery":
		if cfg.Sources.Discovery == nil {
			cfg.Sources.Discovery = &config.Source{}
		}
		return cfg.Sources.Discovery
	case "googleapis":
		if cfg.Sources.Googleapis == nil {
			cfg.Sources.Googleapis = &config.Source{}
		}
		return cfg.Sources.Googleapis
	case "protobuf":
		if cfg.Sources.ProtobufSrc == nil {
			cfg.Sources.ProtobufSrc = &config.Source{}
		}
		return cfg.Sources.ProtobufSrc
	case "showcase":
		if cfg.Sources.Showcase == nil {
			cfg.Sources.Showcase = &config.Source{}
		}
		return cfg.Sources.Showcase
	}
	return nil
}

// getSource returns a Source pointer from the Config if it exists.
func getSource(cfg *config.Config, sourceName string) *config.Source {
	if cfg.Sources == nil {
		return nil
	}
	switch sourceName {
	case "conformance":
		return cfg.Sources.Conformance
	case "discovery":
		return cfg.Sources.Discovery
	case "googleapis":
		return cfg.Sources.Googleapis
	case "protobuf":
		return cfg.Sources.ProtobufSrc
	case "showcase":
		return cfg.Sources.Showcase
	}
	return nil
}

// fetchSourceCommitAndChecksum gets the commit and checksum for a source.
func fetchSourceCommitAndChecksum(sourceName string, branch string) (string, string, error) {
	repo, ok := sourceRepos[sourceName]
	if !ok {
		return "", "", fmt.Errorf("%w: %s", errUnknownSource, sourceName)
	}
	repo.Branch = branch
	endpoints := &fetch.Endpoints{
		API:      githubAPI,
		Download: githubDownload,
	}
	return fetch.LatestCommitAndChecksum(endpoints, &repo)
}
