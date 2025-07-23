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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"path/filepath"

	"github.com/googleapis/librarian/internal/config"

	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"
)

const pipelineStateFile = "state.yaml"
const pipelineConfigFile = "pipeline-config.json"
const serviceConfigType = "type"
const serviceConfigValue = "google.api.Service"

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoStateAndConfig(languageRepo *gitrepo.Repository, source string) (*config.LibrarianState, *config.PipelineConfig, error) {
	if languageRepo == nil {
		return nil, nil, nil
	}
	state, err := loadLibrarianState(languageRepo, source)
	if err != nil {
		return nil, nil, err
	}
	config, err := loadRepoPipelineConfig(languageRepo)
	if err != nil {
		return nil, nil, err
	}
	return state, config, nil
}

func loadLibrarianState(languageRepo *gitrepo.Repository, source string) (*config.LibrarianState, error) {
	if languageRepo == nil {
		return nil, nil
	}
	path := filepath.Join(languageRepo.Dir, config.LibrarianDir, pipelineStateFile)
	return parseLibrarianState(func(file string) ([]byte, error) { return os.ReadFile(file) }, path, source)
}

func loadRepoPipelineConfig(languageRepo *gitrepo.Repository) (*config.PipelineConfig, error) {
	path := filepath.Join(languageRepo.Dir, config.LibrarianDir, pipelineConfigFile)
	return loadPipelineConfigFile(path)
}

func loadPipelineConfigFile(path string) (*config.PipelineConfig, error) {
	return parsePipelineConfig(func() ([]byte, error) { return os.ReadFile(path) })
}

func fetchRemoteLibrarianState(ctx context.Context, client GitHubClient, ref, source string) (*config.LibrarianState, error) {
	return parseLibrarianState(func(file string) ([]byte, error) {
		return client.GetRawContent(ctx, file, ref)
	}, filepath.Join(config.LibrarianDir, pipelineStateFile), source)
}

func parsePipelineConfig(contentLoader func() ([]byte, error)) (*config.PipelineConfig, error) {
	bytes, err := contentLoader()
	if err != nil {
		return nil, err
	}
	config := &config.PipelineConfig{}
	if err := json.Unmarshal(bytes, config); err != nil {
		return nil, err
	}
	return config, nil
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
