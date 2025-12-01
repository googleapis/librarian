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

//go:build !mock_github

// This file contains the production implementations for functions that get
// GitHub repository details.

package legacylibrarian

import (
	"fmt"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygithub"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
)

// GetGitHubRepository determines the GitHub repository from the configuration
// or the local git remote.
var GetGitHubRepository = func(cfg *legacyconfig.Config, languageRepo legacygitrepo.Repository) (*legacygithub.Repository, error) {
	if isURL(cfg.Repo) {
		return legacygithub.ParseRemote(cfg.Repo)
	}
	return GetGitHubRepositoryFromGitRepo(languageRepo)
}

// GetGitHubRepositoryFromGitRepo determines the GitHub repository from the
// local git remote.
var GetGitHubRepositoryFromGitRepo = func(languageRepo legacygitrepo.Repository) (*legacygithub.Repository, error) {
	remotes, err := languageRepo.Remotes()
	if err != nil {
		return nil, err
	}

	for _, remote := range remotes {
		if remote.Name == "origin" {
			if len(remote.URLs) > 0 {
				return legacygithub.ParseRemote(remote.URLs[0])
			}
		}
	}

	return nil, fmt.Errorf("could not find an 'origin' remote pointing to a GitHub https URL")
}
