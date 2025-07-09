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
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"

	"github.com/googleapis/librarian/internal/gitrepo"
)

const pipelineStateFile = "pipeline-state.json"
const pipelineConfigFile = "pipeline-config.json"

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoStateAndConfig(languageRepo *gitrepo.Repository) (*config.PipelineState, *config.PipelineConfig, error) {
	if languageRepo == nil {
		return nil, nil, nil
	}
	state, err := loadRepoPipelineState(languageRepo)
	if err != nil {
		return nil, nil, err
	}
	config, err := loadRepoPipelineConfig(languageRepo)
	if err != nil {
		return nil, nil, err
	}
	return state, config, nil
}

func loadRepoPipelineState(languageRepo *gitrepo.Repository) (*config.PipelineState, error) {
	path := filepath.Join(languageRepo.Dir, config.GeneratorInputDir, pipelineStateFile)
	return loadPipelineStateFile(path)
}

func loadPipelineStateFile(path string) (*config.PipelineState, error) {
	return parsePipelineState(func() ([]byte, error) { return os.ReadFile(path) })
}

func loadRepoPipelineConfig(languageRepo *gitrepo.Repository) (*config.PipelineConfig, error) {
	path := filepath.Join(languageRepo.Dir, config.GeneratorInputDir, "pipeline-config.json")
	return loadPipelineConfigFile(path)
}

func loadPipelineConfigFile(path string) (*config.PipelineConfig, error) {
	return parsePipelineConfig(func() ([]byte, error) { return os.ReadFile(path) })
}

func fetchRemotePipelineState(ctx context.Context, repo *github.Repository, ref string, gitHubToken string) (*config.PipelineState, error) {
	ghClient, err := github.NewClient(gitHubToken, repo)
	if err != nil {
		return nil, err
	}
	return parsePipelineState(func() ([]byte, error) {
		return ghClient.GetRawContent(ctx, config.GeneratorInputDir+"/"+pipelineStateFile, ref)
	})
}

func parsePipelineState(contentLoader func() ([]byte, error)) (*config.PipelineState, error) {
	bytes, err := contentLoader()
	if err != nil {
		return nil, err
	}
	state := &config.PipelineState{}
	if err := json.Unmarshal(bytes, state); err != nil {
		return nil, err
	}
	return state, nil
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
