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
	pipelineStateFile  = "state.yaml"
	pipelineConfigFile = "pipeline-config.json"
	serviceConfigType  = "type"
	serviceConfigValue = "google.api.Service"
)

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoState(languageRepo *gitrepo.LocalRepository, source string) (*config.LibrarianState, error) {
	if languageRepo == nil {
		return nil, nil
	}
	state, err := loadLibrarianState(languageRepo, source)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func loadLibrarianState(languageRepo *gitrepo.LocalRepository, source string) (*config.LibrarianState, error) {
	if languageRepo == nil {
		return nil, nil
	}
	path := filepath.Join(languageRepo.Dir, config.LibrarianDir, pipelineStateFile)
	return parseLibrarianState(func(file string) ([]byte, error) { return os.ReadFile(file) }, path, source)
}

func parseLibrarianState(contentLoader func(file string) ([]byte, error), path, source string) (*config.LibrarianState, error) {
	bytes, err := contentLoader(path)
	if err != nil {
		return nil, err
	}
	var s config.LibrarianState
	if err := yaml.Unmarshal(bytes, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling librarian state: %w", err)
	}
	if err := populateServiceConfigIfEmpty(&s, contentLoader, source); err != nil {
		return nil, fmt.Errorf("populating service config: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("validating librarian state: %w", err)
	}
	return &s, nil
}

func populateServiceConfigIfEmpty(state *config.LibrarianState, contentLoader func(file string) ([]byte, error), source string) error {
	for i, library := range state.Libraries {
		for j, api := range library.APIs {
			if api.ServiceConfig != "" {
				// Do not change API if the service config has already been set.
				continue
			}
			apiPath := filepath.Join(source, api.Path)
			serviceConfig, err := findServiceConfigIn(contentLoader, apiPath)
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
func findServiceConfigIn(contentLoader func(file string) ([]byte, error), path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		bytes, err := contentLoader(filepath.Join(path, entry.Name()))
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
	path := filepath.Join(repoDir, config.LibrarianDir, pipelineStateFile)
	bytes, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

// readLibraryState reads the library state from configure-response.json.
//
// The response file is removed afterwards.
func readLibraryState(contentLoader func(data []byte, state *config.LibraryState) error, jsonFilePath string) (*config.LibraryState, error) {
	data, err := os.ReadFile(jsonFilePath)
	defer func() {
		if err := os.Remove(jsonFilePath); err != nil {
			slog.Warn("fail to remove file", slog.String("name", jsonFilePath), slog.Any("err", err))
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to read response file, path: %s, error: %w", jsonFilePath, err)
	}

	libraryState := &config.LibraryState{}

	if err := contentLoader(data, libraryState); err != nil {
		return nil, fmt.Errorf("failed to load file, %s, to state: %w", jsonFilePath, err)
	}

	if libraryState.ErrorMessage != "" {
		return nil, fmt.Errorf("failed with error message: %s", libraryState.ErrorMessage)
	}

	return libraryState, nil
}
