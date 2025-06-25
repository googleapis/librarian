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

// Package config defines configuration used by the CLI.
package config

import (
	"errors"
	"os"
)

const (
	// GeneratorInputDir is the default directory to store files that generator
	// needs to regenerate libraries from an empty directory.
	GeneratorInputDir string = "generator-input"
)

// Config holds all configuration values parsed from flags or environment
// variables.
type Config struct {
	// APIPath is the path to the API to be configured or generated,
	// relative to the root of the googleapis repository. It is a directory
	// name as far as (and including) the version (v1, v2, v1alpha etc). It
	// is expected to contain a service config YAML file.
	// Example: "google/cloud/functions/v2"
	//
	// APIPath is used by the generate and configure commands.
	//
	// API Path is specified with the -api-path flag.
	APIPath string

	// APIRoot is the path to the root of the googleapis repository.
	// When this is not specified, the googleapis repository is cloned
	// automatically.
	//
	// APIRoot is used by the generate, update-apis, update-image-tag and configure
	// commands.
	//
	// APIRoot is specified with the -api-root flag.
	APIRoot string

	// ArtifactRoot is the path to previously-created release artifacts to be published.
	// It is only used by the publish-release-artifacts command, and is expected
	// to be the output directory from a previous create-release-artifacts command.
	// It is required for publish-release-artifacts.
	//
	// ArtifactRoot is specified with the -artifact-root flag.
	ArtifactRoot string

	// BaselineCommit is the commit hash of the language repo used as a
	// baseline when generating diffs as part of determining whether a release PR
	// can be merged. It is only used by the merge-release-pr command, and the
	// value is expected to be taken from the environment variable recorded in
	// env-vars.txt by the create-release-pr command. The value is the HEAD commit at
	// the start of the create-release-pr command, and it is used by merge-release-pr
	// to determine whether any source files for a library have been modified in
	// GitHub since the release PR was initiated (thereby invalidating the release of
	// that library).
	//
	// BaselineCommit is specified with the -baseline-commit flag.
	BaselineCommit string

	// Branch is the branch name to use when working with git repositories. It is
	// currently unused.
	//
	// Branch is specified with the -branch flag.
	Branch string

	// Build determines whether to build the generated library, and is only
	// used in the generate command.
	//
	// Build is specified with the -build flag.
	Build bool

	// EnvFile is the path to the file used to store environment variables, to
	// propagate information from one step to another in a CI flow. The file
	// is a list of key=value lines, where the key becomes an environment variable
	// in the next step of the CI flow. Every key must start with an underscore.
	//
	// EnvFile is only used in the create-release-pr and merge-release-pr commands.
	// It is always optional; if it is unspecified, it defaults to env-file.txt
	// within WorkRoot.
	//
	// EnvFile is specified with the -env-file flag.
	EnvFile string

	// GitHubToken is the access token to use for all operations involving
	// GitHub.
	//
	// GitHubToken is used by the create-release-pr, configure, update-apis
	// and update-image-tag commands, when Push is true. It is always used by
	// the merge-release-pr and publish-release-artifacts commands.
	//
	// GitHubToken is not specified by a flag, as flags are logged and the
	// access token is sensitive information. Instead, it is fetched from the
	// LIBRARIAN_GITHUB_TOKEN environment variable.
	GitHubToken string

	// GitUserEmail is the email address used in Git commits. It is used in
	// all commands that create commits in a language repository:
	// create-release-pr, configure, update-apis and update-image-tag.
	//
	// GitUserEmail is optional, with a default value of noreply-cloudsdk@google.com
	// being used for commits if it's unspecified.
	//
	// GitUserEmail is specified with the -git-user-email flag.
	GitUserEmail string

	// GitUserName is the display name used in Git commits. It is used in
	// all commands that create commits in a language repository:
	// create-release-pr, configure, update-apis and update-image-tag.
	//
	// GitUserName is optional, with a default value of "Google Cloud SDK"
	// being used for commits if it's unspecified.
	//
	// GitUserName is specified with the -git-user-name flag.
	GitUserName string

	// Image is the language-specific container image to use for language-specific
	// operations. It is primarily used for testing Librarian and/or new images.
	//
	// Image is used by all commands which perform language-specific operations.
	// (This covers all commands other than merge-release-pr.) If this is set via
	// the -image flag, it is expected to be used directly (potentially including a repository
	// and/or tag). If the -image flag is not set, the full image specification to
	// use is derived from the pieces of information:
	//
	// - Name: As specified in language repository configuration, or
	// a default of "google-cloud-{Language}-generator"
	// - Tag: As specified in the language repository state file, where available
	// - Repository: The LIBRARIAN_REPOSITORY environment variable, if set.
	//
	// Image is specified with the -image flag.
	Image string

	// DockerHostRootDir specifies the host view of a mount point that is
	// mounted as DockerMountRootDir from the Librarian view, when Librarian
	// is running in Docker. For example, if Librarian has been run via a command
	// such as "docker run -v /home/user/librarian:/app" then DockerHostRootDir
	// would be "/home/user/librarian" and DockerMountRootDir would be "/app".
	//
	// This information is required to enable Docker-in-Docker scenarios. When
	// creating a Docker container for language-specific operations, the methods
	// accepting directory names are all expected to be from the perspective of
	// the Librarian code - but the mount point specified when running Docker
	// from within Librarian needs to be specified from the host perspective,
	// as the new Docker container is created as a sibling of the one running
	// Librarian.
	//
	// For example, if we're in the scenario above, with
	// DockerHostRootDir=/home/user/librarian and DockerMountRootDir=/app,
	// executing a command which tries to mount the /app/work/googleapis directory
	// as /apis in the container, the eventual Docker command would need to include
	// "-v /home/user/librarian/work/googleapis:/apis"
	//
	// DockerHostRootDir and DockerMountDir are currently populated from
	// KOKORO_HOST_ROOT_DIR and KOKORO_ROOT_DIR environment variables respectively.
	// These are automatically supplied by Kokoro. Other Docker-in-Docker scenarios
	// are not currently supported, but could be implemented by populating these
	// configuration values in a similar way.
	DockerHostRootDir string

	// DockerMountRootDir specifies the Librarian view of a mount point that is
	// mounted as DockerHostRootDir from the host view, when Librarian is running in
	// Docker. See the documentation for DockerHostRootDir for more information.
	DockerMountRootDir string

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

	// SyncAuthToken provides an auth token used when polling a synchronization
	// URL at the end of the merge-release-pr command, if SyncURLPrefix has been
	// specified.
	//
	// SyncAuthToken is not specified by a flag, as flags are logged and the
	// access token is sensitive information. Instead, it is fetched from the
	// LIBRARIAN_SYNC_AUTH_TOKEN environment variable.
	SyncAuthToken string

	// SyncURLPrefix is the prefix used to build commit synchronization URLs.
	// It is only used by merge-release-pr, and is optional. When specified, a
	// full URL is constructed by appending the commit hash of the merged pull
	// request to the specified prefix, and polling until that URL can be fetched
	// via a GET request which includes the SyncAuthToken. SyncAuthToken must
	// be specified if SyncURLPrefix is specified.
	//
	// SyncURLPrefix is specified with the -sync-url-prefix flag.
	SyncURLPrefix string

	// SecretsProject is the Google Cloud project containing Secret Manager secrets
	// to provide to the language-specific container commands via environment variables.
	//
	// SecretsProject is used by all commands which perform language-specific operations.
	// (This covers all commands other than merge-release-pr.) If no value is set, any
	// language-specific operations which include an environment variable based on a secret
	// will act as if the secret name wasn't set (so will just use a host environment variable
	// or default value, if any).
	//
	// SecretsProject is specified with the -secrets-project flag.
	SecretsProject string

	// SkipIntegrationTests is used by the create-release-pr and create-release-artifacts
	// commands, and disables integration tests if it is set to a non-empty value.
	// The value must reference a bug (e.g., b/12345).
	//
	// SkipIntegrationTests is specified with the -skip-integration-tests flag.
	SkipIntegrationTests string

	// Tag is the new tag for the language-specific Docker image, used only by the
	// update-image-tag command. All operations within update-image-tag are performed
	// using the new tag.
	//
	// Tag is specified with the -tag flag.
	Tag string

	// TagRepoURL is the GitHub repository to push the tag and create a release
	// in. This is only used in the publish-release-artifacts command:
	// when all artifacts have been been published to package managers,
	// documentation sites etc, tags/releases are created on GitHub, and
	// TagRepoURL identifies this repository.
	//
	// TagRepoURL is specified with the -tag-repo-url flag.
	TagRepoURL string

	// WorkRoot is the root directory used for temporary working files, including
	// any repositories that are cloned. By default, this is created in /tmp with
	// a timestamped directory name (e.g. /tmp/librarian-20250617T083548Z) but
	// can be specified with the -work-root flag.
	//
	// WorkRoot is used by all librarian commands.
	WorkRoot string
}

// New returns a new Config populated with environment variables.
func New() *Config {
	return &Config{
		// TODO(https://github.com/googleapis/librarian/issues/507): replace
		// os.Getenv calls in other functions with these values.
		DockerHostRootDir:  os.Getenv("KOKORO_HOST_ROOT_DIR"),
		DockerMountRootDir: os.Getenv("KOKORO_ROOT_DIR"),
		GitHubToken:        os.Getenv("LIBRARIAN_GITHUB_TOKEN"),
		SyncAuthToken:      os.Getenv("LIBRARIAN_SYNC_AUTH_TOKEN"),
	}
}

// IsValid ensures the values contained in a Config are valid.
func (c *Config) IsValid() (bool, error) {
	if c.Push && c.GitHubToken == "" {
		return false, errors.New("no GitHub token supplied for push")
	}
	return true, nil
}
