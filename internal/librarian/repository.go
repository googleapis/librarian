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

//go:build !e2etest

// This file contains the production implementations for functions that get
// GitHub repository details.

package librarian

import (
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// GetGitHubRepository determines the GitHub repository from the configuration
// or the local git remote.
var GetGitHubRepository = func(cfg *config.Config, languageRepo gitrepo.Repository) (*github.Repository, error) {
	if isURL(cfg.Repo) {
		return github.ParseRemote(cfg.Repo)
	}
	return GetGitHubRepositoryFromGitRepo(languageRepo)
}

// GetGitHubRepositoryFromGitRepo determines the GitHub repository from the
// local git remote.
var GetGitHubRepositoryFromGitRepo = func(languageRepo gitrepo.Repository) (*github.Repository, error) {
	remotes, err := languageRepo.Remotes()
	if err != nil {
		return nil, err
	}

	for _, remote := range remotes {
		if remote.Name == "origin" {
			if len(remote.URLs) > 0 {
				return github.ParseRemote(remote.URLs[0])
			}
		}
	}

	return nil, fmt.Errorf("could not find an 'origin' remote pointing to a GitHub https URL")
}
