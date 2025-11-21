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

//go:build mock_github

// This file contains mock implementations of repository getters for use in
// end-to-end tests. It is compiled only when the 'mock_github' build tag is specified.

package legacylibrarian

import (
	"log/slog"
	"os"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygithub"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
)

// GetGitHubRepository returns a mock legacygithub.Repository object for e2e tests.
// It reads the LIBRARIAN_GITHUB_BASE_URL environment variable to configure
// the mock repository's BaseURL, allowing the test client to connect to a
// local httptest.Server.
var GetGitHubRepository = func(cfg *legacyconfig.Config, languageRepo legacygitrepo.Repository) (*legacygithub.Repository, error) {
	slog.Info("using mock GitHub repository for e2e test")
	baseURL := os.Getenv("LIBRARIAN_GITHUB_BASE_URL")
	return &legacygithub.Repository{Owner: "test-owner", Name: "test-repo", BaseURL: baseURL}, nil
}

// GetGitHubRepositoryFromGitRepo returns a mock legacygithub.Repository object for e2e tests.
// It reads the LIBRARIAN_GITHUB_BASE_URL environment variable to configure
// the mock repository's BaseURL.
var GetGitHubRepositoryFromGitRepo = func(languageRepo legacygitrepo.Repository) (*legacygithub.Repository, error) {
	slog.Info("using mock GitHub repository for e2e test")
	baseURL := os.Getenv("LIBRARIAN_GITHUB_BASE_URL")
	return &legacygithub.Repository{Owner: "test-owner", Name: "test-repo", BaseURL: baseURL}, nil
}
