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

//go:build e2etest

// This file contains mock implementations of repository getters for use in
// end-to-end tests. It is compiled only when the 'e2e-test' build tag is specified.

package librarian

import (
	"log/slog"
	"os"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// GetGitHubRepository returns a mock github.Repository object for e2e tests.
// It reads the LIBRARIAN_GITHUB_BASE_URL environment variable to configure
// the mock repository's BaseURL, allowing the test client to connect to a
// local httptest.Server.
var GetGitHubRepository = func(cfg *config.Config, languageRepo gitrepo.Repository) (*github.Repository, error) {
	slog.Info("Using mock GitHub repository for e2e test")
	baseURL := os.Getenv("LIBRARIAN_GITHUB_BASE_URL")
	return &github.Repository{Owner: "test-owner", Name: "test-repo", BaseURL: baseURL}, nil
}

// GetGitHubRepositoryFromGitRepo returns a mock github.Repository object for e2e tests.
// It reads the LIBRARIAN_GITHUB_BASE_URL environment variable to configure
// the mock repository's BaseURL.
var GetGitHubRepositoryFromGitRepo = func(languageRepo gitrepo.Repository) (*github.Repository, error) {
	slog.Info("Using mock GitHub repository for e2e test")
	baseURL := os.Getenv("LIBRARIAN_GITHUB_BASE_URL")
	return &github.Repository{Owner: "test-owner", Name: "test-repo", BaseURL: baseURL}, nil
}
