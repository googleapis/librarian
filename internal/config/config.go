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

package config

import (
	"os"
)

// Config holds all configuration values parsed from flags or environment
// variables.
type Config struct {
	// APIPath is the path to the API to be configured or generated.
	// Example: "google/cloud/functions/v2"
	APIPath string

	// APIRoot is the path to the root of the googleapis repository.
	APIRoot string

	// ArtifactRoot is the path to release artifacts for publishing.
	ArtifactRoot string

	// BaselineCommit is the commit hash of the language repo used as a
	// baseline when generating diffs.
	BaselineCommit string

	// Branch is the branch name to use when working with git repositories.
	Branch string

	// Build determines whether to build the generated library.
	Build bool

	// EnvFile is the path to the file storing environment variables.
	EnvFile string

	// GitUserEmail is the email address used in Git commits.
	GitUserEmail string

	// GitUserName is the display name used in Git commits.
	GitUserName string

	// Image is the language-specific container image used during generation.
	Image string

	// TODO(https://github.com/googleapis/librarian/issues/468): document usage
	KokoroHostRootDir string

	// TODO(https://github.com/googleapis/librarian/issues/468): document usage
	KokoroRootDir string

	// Language is the target language for library generation.
	Language string

	// LibraryID is the identifier of a specific library to operate on.
	LibraryID string

	// LibraryVersion is the version string used when creating a release.
	LibraryVersion string

	// Push determines whether to push changes to GitHub.
	Push bool

	// ReleaseID is the identifier of a release PR.
	ReleaseID string

	// ReleasePRURL is the URL of the release pull request.
	ReleasePRURL string

	// RepoRoot is the local root directory of the language repository.
	RepoRoot string

	// RepoURL is the URL to clone the language repository from.
	RepoURL string

	// SyncURLPrefix is the prefix used to build commit synchronization URLs.
	SyncURLPrefix string

	// SecretsProject is the Google Cloud project containing Secret Manager secrets.
	SecretsProject string

	// SkipIntegrationTests disables integration tests if set, and must
	// reference a bug (e.g., b/12345).
	SkipIntegrationTests string

	// Tag is the Docker image tag to assign when building.
	Tag string

	// TagRepoURL is the GitHub repository to push the tag and create a release
	// in.
	TagRepoURL string

	// WorkRoot is the root directory used for temporary working files.
	WorkRoot string
}

// New returns a new Config populated with environment variables.
func New() *Config {
	return &Config{
		// TODO(https://github.com/googleapis/librarian/issues/507): replace
		// os.Getenv calls in other functions with these values.
		KokoroHostRootDir: os.Getenv("KOKORO_HOST_ROOT_DIR"),
		KokoroRootDir:     os.Getenv("KOKORO_ROOT_DIR"),
	}
}
