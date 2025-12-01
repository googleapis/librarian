// Copyright 2025 Google LLC
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

package legacylibrarian

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
	"gopkg.in/yaml.v3"
)

const (
	librarianConfigFile = "legacyconfig.yaml"
	librarianStateFile  = "state.yaml"
	serviceConfigType   = "type"
	serviceConfigValue  = "google.api.Service"
)

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoState(repo *legacygitrepo.LocalRepository, source string) (*legacyconfig.LibrarianState, error) {
	if repo == nil {
		slog.Info("repo is nil, skipping state loading")
		return nil, nil
	}
	path := filepath.Join(repo.Dir, legacyconfig.LibrarianDir, librarianStateFile)
	return parseLibrarianState(path, source)
}

func loadRepoStateFromGitHub(ctx context.Context, ghClient GitHubClient, branch string) (*legacyconfig.LibrarianState, error) {
	content, err := ghClient.GetRawContent(ctx, path.Join(legacyconfig.LibrarianDir, legacyconfig.LibrarianStateFile), branch)
	if err != nil {
		return nil, err
	}
	state, err := loadLibrarianStateFromBytes(content, "")
	if err != nil {
		return nil, err
	}
	return state, nil
}

func loadLibrarianConfig(repo *legacygitrepo.LocalRepository) (*legacyconfig.LibrarianConfig, error) {
	if repo == nil {
		slog.Info("repo is nil, skipping state loading")
		return nil, nil
	}
	path := filepath.Join(repo.Dir, legacyconfig.LibrarianDir, librarianConfigFile)
	return parseLibrarianConfig(path)
}

func loadLibrarianConfigFromGitHub(ctx context.Context, ghClient GitHubClient, branch string) (*legacyconfig.LibrarianConfig, error) {
	content, err := ghClient.GetRawContent(ctx, path.Join(legacyconfig.LibrarianDir, legacyconfig.LibrarianConfigFile), branch)
	if err != nil {
		return nil, err
	}
	cfg, err := loadLibrarianConfigFromBytes(content)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseLibrarianState(path, source string) (*legacyconfig.LibrarianState, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return loadLibrarianStateFromBytes(bytes, source)
}

func loadLibrarianStateFromBytes(data []byte, source string) (*legacyconfig.LibrarianState, error) {
	var s legacyconfig.LibrarianState
	if err := yaml.Unmarshal(data, &s); err != nil {
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

func parseLibrarianConfig(path string) (*legacyconfig.LibrarianConfig, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("legacyconfig.yaml not found, proceeding")
			return nil, nil
		}
		return nil, err
	}
	return loadLibrarianConfigFromBytes(bytes)
}

func loadLibrarianConfigFromBytes(data []byte) (*legacyconfig.LibrarianConfig, error) {
	var lc legacyconfig.LibrarianConfig
	if err := yaml.Unmarshal(data, &lc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal global config: %w", err)
	}
	if err := lc.Validate(); err != nil {
		return nil, fmt.Errorf("invalid global config: %w", err)
	}
	return &lc, nil
}

func populateServiceConfigIfEmpty(state *legacyconfig.LibrarianState, source string) error {
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

	slog.Info("no service config found; assuming proto-only package", "path", path)
	return "", nil
}

func saveLibrarianState(repoDir string, state *legacyconfig.LibrarianState) error {
	sortByLibraryID(state)
	stateFile := filepath.Join(repoDir, legacyconfig.LibrarianDir, librarianStateFile)
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	err := encoder.Encode(state)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, buffer.Bytes(), 0644)
}

// sortByLibraryID sorts legacyconfig.LibraryState with respect to ID.
func sortByLibraryID(state *legacyconfig.LibrarianState) {
	sort.Slice(state.Libraries, func(i, j int) bool {
		return state.Libraries[i].ID < state.Libraries[j].ID
	})
}

// readLibraryState reads the library state from a container response, if it exists.
// If the response file does not exist, readLibraryState succeeds but returns a nil pointer.
//
// The response file is removed afterward.
func readLibraryState(jsonFilePath string) (*legacyconfig.LibraryState, error) {
	data, err := os.ReadFile(jsonFilePath)
	defer func() {
		if b, err := os.ReadFile(jsonFilePath); err == nil {
			slog.Debug("container response", "content", string(b))
		}
		if err := os.Remove(jsonFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("fail to remove file", slog.String("name", jsonFilePath), slog.Any("err", err))
		}
	}()
	if err != nil {
		// If we only failed to read the file because it didn't exist, just succeed
		// with a nil pointer.

		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read response file, path: %s, error: %w", jsonFilePath, err)
	}

	var libraryState *legacyconfig.LibraryState

	if err := json.Unmarshal(data, &libraryState); err != nil {
		return nil, fmt.Errorf("failed to load file, %s, to state: %w", jsonFilePath, err)
	}

	if libraryState.ErrorMessage != "" {
		return nil, fmt.Errorf("failed with error message: %s", libraryState.ErrorMessage)
	}

	return libraryState, nil
}
