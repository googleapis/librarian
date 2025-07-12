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
	"fmt"
	"os"

	"path/filepath"

	"github.com/googleapis/librarian/internal/config"

	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"
)

const pipelineStateFile = "state.yaml"
const pipelineConfigFile = "pipeline-config.json"

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoStateAndConfig(languageRepo *gitrepo.Repository) (*config.LibrarianState, *config.PipelineConfig, error) {
	if languageRepo == nil {
		return nil, nil, nil
	}
	state, err := loadLibrarianState(languageRepo)
	if err != nil {
		return nil, nil, err
	}
	config, err := loadRepoPipelineConfig(languageRepo)
	if err != nil {
		return nil, nil, err
	}
	return state, config, nil
}

func loadLibrarianState(languageRepo *gitrepo.Repository) (*config.LibrarianState, error) {
	if languageRepo == nil {
		return nil, nil
	}
	path := filepath.Join(languageRepo.Dir, config.LibrarianDir, pipelineStateFile)
	return parseLibrarianState(func() ([]byte, error) { return os.ReadFile(path) })
}

func loadLibrarianStateFile(path string) (*config.LibrarianState, error) {
	return parseLibrarianState(func() ([]byte, error) { return os.ReadFile(path) })
}

func loadRepoPipelineConfig(languageRepo *gitrepo.Repository) (*config.PipelineConfig, error) {
	path := filepath.Join(languageRepo.Dir, config.LibrarianDir, pipelineConfigFile)
	return loadPipelineConfigFile(path)
}

func loadPipelineConfigFile(path string) (*config.PipelineConfig, error) {
	return parsePipelineConfig(func() ([]byte, error) { return os.ReadFile(path) })
}

func fetchRemoteLibrarianState(ctx context.Context, client GitHubClient, ref string) (*config.LibrarianState, error) {
	return parseLibrarianState(func() ([]byte, error) {
		return client.GetRawContent(ctx, config.GeneratorInputDir+"/"+pipelineStateFile, ref)
	})
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

func parseLibrarianState(contentLoader func() ([]byte, error)) (*config.LibrarianState, error) {
	bytes, err := contentLoader()
	if err != nil {
		return nil, err
	}
	var s config.LibrarianState
	if err := yaml.Unmarshal(bytes, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling librarian state: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("validating librarian state: %w", err)
	}
	return &s, nil
}
