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
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/containers/image/v5/docker/reference"
	"github.com/go-playground/validator/v10"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"

	"github.com/googleapis/librarian/internal/gitrepo"
	"gopkg.in/yaml.v3"
)

const pipelineStateFile = "state.yaml"
const pipelineConfigFile = "pipeline-config.json"

// Utility functions for saving and loading pipeline state and config from various places.

func loadRepoStateAndConfig(languageRepo *gitrepo.Repository) (*LibrarianState, *config.PipelineConfig, error) {
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

func loadLibrarianState(repo *gitrepo.Repository) (*LibrarianState, error) {
	if repo == nil {
		return nil, nil
	}
	path := filepath.Join(repo.Dir, config.GeneratorInputDir, pipelineStateFile)
	return parseLibrarianState(func() ([]byte, error) { return os.ReadFile(path) })
}

func loadLibrarianStateFile(path string) (*LibrarianState, error) {
	return parseLibrarianState(func() ([]byte, error) { return os.ReadFile(path) })
}

func loadRepoPipelineConfig(languageRepo *gitrepo.Repository) (*config.PipelineConfig, error) {
	path := filepath.Join(languageRepo.Dir, config.GeneratorInputDir, "pipeline-config.json")
	return loadPipelineConfigFile(path)
}

func loadPipelineConfigFile(path string) (*config.PipelineConfig, error) {
	return parsePipelineConfig(func() ([]byte, error) { return os.ReadFile(path) })
}

func fetchRemoteLibrarianState(ctx context.Context, repo *github.Repository, ref string, gitHubToken string) (*LibrarianState, error) {
	ghClient, err := github.NewClient(gitHubToken)
	if err != nil {
		return nil, err
	}
	return parseLibrarianState(func() ([]byte, error) {
		return ghClient.GetRawContent(ctx, repo, config.GeneratorInputDir+"/"+pipelineStateFile, ref)
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

func saveLibrarianState(repo *gitrepo.Repository, s *LibrarianState) error {
	path := filepath.Join(repo.Dir, config.GeneratorInputDir, pipelineStateFile)
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling librarian state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func parseLibrarianState(contentLoader func() ([]byte, error)) (*LibrarianState, error) {
	bytes, err := contentLoader()
	if err != nil {
		return nil, err
	}
	var s LibrarianState
	if err := yaml.Unmarshal(bytes, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling librarian state: %w", err)
	}

	validate := validator.New()
	if err := validate.RegisterValidation("is-regexp", validateRegexp); err != nil {
		return nil, fmt.Errorf("registering regexp validator: %w", err)
	}
	if err := validate.RegisterValidation("is-dirpath", validateDirPath); err != nil {
		return nil, fmt.Errorf("registering dirpath validator: %w", err)
	}
	if err := validate.RegisterValidation("is-image", validateImage); err != nil {
		return nil, fmt.Errorf("registering image validator: %w", err)
	}
	if err := validate.RegisterValidation("is-library-id", validateLibraryID); err != nil {
		return nil, fmt.Errorf("registering library ID validator: %w", err)
	}
	if err := validate.Struct(&s); err != nil {
		return nil, fmt.Errorf("validating librarian state: %w", err)
	}
	return &s, nil
}

func validateRegexp(fl validator.FieldLevel) bool {
	_, err := regexp.Compile(fl.Field().String())
	return err == nil
}

// invalidPathChars contains characters that are invalid in path components,
// plus path separators and the null byte.
const invalidPathChars = `<>:"|?*\/\\\x00`

func validateDirPath(fl validator.FieldLevel) bool {
	pathString := fl.Field().String()
	if pathString == "" {
		return false
	}

	// The paths are expected to be relative and use forward slashes.
	// We clean the path to resolve ".." and check that it doesn't try to
	// escape the root.
	cleaned := path.Clean(pathString)
	if path.IsAbs(pathString) || strings.HasPrefix(cleaned, "..") || cleaned == ".." {
		return false
	}

	// A single dot is a valid relative path, but likely not the intended input.
	if cleaned == "." {
		return false
	}

	// Each path component must not contain invalid characters.
	for _, component := range strings.Split(cleaned, "/") {
		if strings.ContainsAny(component, invalidPathChars) {
			return false
		}
	}
	return true
}

// validateImage checks if a string is a valid, normalized container image name.
func validateImage(fl validator.FieldLevel) bool {
	_, err := reference.ParseNormalizedNamed(fl.Field().String())
	return err == nil
}

var libraryIDRegexp = regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`)

func validateLibraryID(fl validator.FieldLevel) bool {
	id := fl.Field().String()
	if id == "" {
		// This is caught by 'required' tag, but good to be defensive.
		return false
	}
	// The ID should not be "." or "..".
	if id == "." || id == ".." {
		return false
	}
	return libraryIDRegexp.MatchString(id)
}
