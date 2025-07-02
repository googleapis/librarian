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
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"

	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/protobuf/encoding/protojson"
)

const pipelineStateFile = "pipeline-state.json"
const pipelineConfigFile = "pipeline-config.json"

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoStateAndConfig(
	languageRepo *gitrepo.Repository,
) (*statepb.PipelineState, *statepb.PipelineConfig, error) {
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

func loadRepoPipelineState(languageRepo *gitrepo.Repository) (*statepb.PipelineState, error) {
	path := filepath.Join(languageRepo.Dir, config.GeneratorInputDir, pipelineStateFile)
	return loadPipelineStateFile(path)
}

func loadPipelineStateFile(path string) (*statepb.PipelineState, error) {
	return parsePipelineState(func() ([]byte, error) { return os.ReadFile(path) })
}

func loadRepoPipelineConfig(languageRepo *gitrepo.Repository) (*statepb.PipelineConfig, error) {
	path := filepath.Join(languageRepo.Dir, config.GeneratorInputDir, "pipeline-config.json")
	return loadPipelineConfigFile(path)
}

func loadPipelineConfigFile(path string) (*statepb.PipelineConfig, error) {
	return parsePipelineConfig(func() ([]byte, error) { return os.ReadFile(path) })
}

func savePipelineState(state *commandState) error {
	path := filepath.Join(state.languageRepo.Dir, config.GeneratorInputDir, pipelineStateFile)
	// Marshal the protobuf message as JSON...
	unformatted, err := protojson.Marshal(state.pipelineState)
	if err != nil {
		return err
	}
	// ... then reformat it
	var formatted bytes.Buffer
	err = json.Indent(&formatted, unformatted, "", "    ")
	if err != nil {
		return err
	}
	// The file mode is likely to be irrelevant, given that the permissions aren't changed
	// if the file exists, which we expect it to anyway.
	err = os.WriteFile(path, formatted.Bytes(), os.FileMode(0644))
	return err
}

func fetchRemotePipelineState(
	ctx context.Context,
	repo *github.Repository,
	ref string,
	gitHubToken string,
) (*statepb.PipelineState, error) {
	ghClient, err := github.NewClient(gitHubToken)
	if err != nil {
		return nil, err
	}
	return parsePipelineState(func() ([]byte, error) {
		return ghClient.GetRawContent(
			ctx,
			repo,
			config.GeneratorInputDir+"/"+pipelineStateFile,
			ref,
		)
	})
}

func parsePipelineState(contentLoader func() ([]byte, error)) (*statepb.PipelineState, error) {
	bytes, err := contentLoader()
	if err != nil {
		return nil, err
	}
	state := &statepb.PipelineState{}
	if err := protojson.Unmarshal(bytes, state); err != nil {
		return nil, err
	}
	return state, nil
}

func parsePipelineConfig(contentLoader func() ([]byte, error)) (*statepb.PipelineConfig, error) {
	bytes, err := contentLoader()
	if err != nil {
		return nil, err
	}
	config := &statepb.PipelineConfig{}
	if err := protojson.Unmarshal(bytes, config); err != nil {
		return nil, err
	}
	return config, nil
}
