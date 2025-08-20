// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"
)

const (
	librarianConfigFile = "config.yaml"
	librarianStateFile  = "state.yaml"
	serviceConfigType   = "type"
	serviceConfigValue  = "google.api.Service"
)

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoState(repo *gitrepo.LocalRepository, source string) (*config.LibrarianState, error) {
	if repo == nil {
		slog.Info("repo is nil, skipping state loading")
		return nil, nil
	}
	path := filepath.Join(repo.Dir, config.LibrarianDir, librarianStateFile)
	return parseLibrarianState(path, source)
}

func loadLibrarianConfig(repo *gitrepo.LocalRepository) (*config.LibrarianConfig, error) {
	if repo == nil {
		slog.Info("repo is nil, skipping state loading")
		return nil, nil
	}
	path := filepath.Join(repo.Dir, config.LibrarianDir, librarianConfigFile)
	return parseLibrarianConfig(path)
}

func parseLibrarianState(path, source string) (*config.LibrarianState, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s config.LibrarianState
	if err := yaml.Unmarshal(bytes, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling librarian state: %w", err)
	}
	if err := populateServiceConfigIfEmpty(&s, source); err != nil {
		return nil, fmt.Errorf("populating service config: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("validating librarian state: %w", err)
	}
	return &s, nil
}

func parseLibrarianConfig(path string) (*config.LibrarianConfig, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("config.yaml not found, proceeding")
			return nil, nil
		}
		return nil, err
	}
	var lc config.LibrarianConfig
	if err := yaml.Unmarshal(bytes, &lc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal global config: %w", err)
	}
	if err := lc.Validate(); err != nil {
		return nil, fmt.Errorf("invalid global config: %w", err)
	}
	return &lc, nil
}

func populateServiceConfigIfEmpty(state *config.LibrarianState, source string) error {
	if source == "" {
		slog.Info("source not specified, skipping service config population")
		return nil
	}
	for i, library := range state.Libraries {
		for j, api := range library.APIs {
			if api.ServiceConfig != "" {
				// Do not change API if the service config has already been set.
				continue
			}
			apiPath := filepath.Join(source, api.Path)
			serviceConfig, err := findServiceConfigIn(apiPath)
			if err != nil {
				return err
			}
			state.Libraries[i].APIs[j].ServiceConfig = serviceConfig
		}
	}

	return nil
}

// findServiceConfigIn detects the service config in a given path.
//
// Returns the file name (relative to the given path) if the following criteria
// are met:
//
// 1. the file ends with `.yaml` and it is a valid yaml file.
//
// 2. the file contains `type: google.api.Service`.
func findServiceConfigIn(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read dir %q: %w", path, err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		bytes, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return "", err
		}
		var configMap map[string]interface{}
		if err := yaml.Unmarshal(bytes, &configMap); err != nil {
			return "", err
		}
		if value, ok := configMap[serviceConfigType].(string); ok && value == serviceConfigValue {
			return entry.Name(), nil
		}
	}

	return "", errors.New("could not find service config in " + path)
}

func saveLibrarianState(repoDir string, state *config.LibrarianState) error {
	path := filepath.Join(repoDir, config.LibrarianDir, librarianStateFile)
	bytes, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

// readLibraryState reads the library state from a container response.
//
// The response file is removed afterwards.
func readLibraryState(jsonFilePath string) (*config.LibraryState, error) {
	data, err := os.ReadFile(jsonFilePath)
	defer func() {
		if err := os.Remove(jsonFilePath); err != nil {
			slog.Warn("fail to remove file", slog.String("name", jsonFilePath), slog.Any("err", err))
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to read response file, path: %s, error: %w", jsonFilePath, err)
	}

	var libraryState *config.LibraryState

	if err := json.Unmarshal(data, &libraryState); err != nil {
		return nil, fmt.Errorf("failed to load file, %s, to state: %w", jsonFilePath, err)
	}

	if libraryState.ErrorMessage != "" {
		return nil, fmt.Errorf("failed with error message: %s", libraryState.ErrorMessage)
	}

	return libraryState, nil
}
