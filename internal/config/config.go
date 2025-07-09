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
	"fmt"
	"os"
	"os/user"
	"strings"
)

const (
	// GeneratorInputDir is the default directory to store files that generator
	// needs to regenerate libraries from an empty directory.
	GeneratorInputDir string = "generator-input"
	DefaultPushConfig string = "noreply-cloudsdk@google.com,Google Cloud SDK"
)

// Config holds all configuration values parsed from flags or environment
// variables. When adding members to this struct, please keep them in
// alphabetical order.
type Config struct {
	// API is the path to the API to be configured or generated,
	// relative to the root of the googleapis repository. It is a directory
	// name as far as (and including) the version (v1, v2, v1alpha etc). It
	// is expected to contain a service config YAML file.
	// Example: "google/cloud/functions/v2"
	//
	// API is used by generate and configure commands.
	//
	// API Path is specified with the -api flag.
	API string

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

	// CI is the type of Continuous Integration (CI) environment in which
	// the tool is executing.
	CI string

	// GitHubToken is the access token to use for all operations involving
	// GitHub.
	//
	// GitHubToken is used by to configure, update-apis and update-image-tag commands,
	// when Push is true.
	//
	// GitHubToken is not specified by a flag, as flags are logged and the
	// access token is sensitive information. Instead, it is fetched from the
	// LIBRARIAN_GITHUB_TOKEN environment variable.
	GitHubToken string

	// HostMount is used to remap Docker mount paths when running in environments
	// where Docker containers are siblings (e.g., Kokoro).
	// It specifies a mount point from the Docker host into the Docker container.
	// The format is "{host-dir}:{local-dir}".
	//
	// HostMount is specified with the -host-mount flag.
	HostMount string

	// Image is the language-specific container image to use for language-specific
	// operations. It is primarily used for testing Librarian and/or new images.
	//
	// Image is used by all commands which perform language-specific operations.
	// If this is set via the -image flag, it is expected to be used directly
	// (potentially including a repository and/or tag). If the -image flag is not
	// set, use an image configured in the `config.yaml`.
	//
	// Image is specified with the -image flag.
	Image string

	// LibrarianRepository specifies the repository where Librarian-related assets
	// are stored.
	//
	// LibrarianRepository is fetched from the LIBRARIAN_REPOSITORY environment
	// variable.
	LibrarianRepository string

	// Project is the Google Cloud project containing Secret Manager secrets to
	// provide to the language-specific container commands via environment variables.
	//
	// Project is used by all commands which perform language-specific operations.
	// (This covers all commands other than merge-release-pr.)
	// If no value is set, any language-specific operations which include an
	// environment variable based on a secret will act as if the secret name
	// wasn't set (so will just use a host environment variable or default value,
	// if any).
	//
	// Project is specified with the -project flag.
	Project string

	// Push determines whether to push changes to GitHub. It is used in
	// all commands that create commits in a language repository:
	// configure, update-apis and update-image-tag.
	// These commands all create pull requests if they
	//
	// By default (when Push isn't explicitly specified), commits are created in
	// the language repo (whether a fresh clone or one specified with RepoRoot)
	// but no pull request is created. In this situation, the description of the
	// pull request that would have been created is displayed in the output of
	// the command.
	//
	// When Push is true, GitHubToken must also be specified.
	//
	// Push is specified with the -push flag. No value is required.
	Push bool

	// PushConfig specifies the email address and display name used in Git commits,
	// in the format "email,name".
	//
	// PushConfig is used in all commands that create commits in a language repository:
	// create-release-pr, configure, update-apis and update-image-tag.
	//
	// PushConfig is optional. If unspecified, commits will use a default name of
	// "Google Cloud SDK" and a default email of noreply-cloudsdk@google.com.
	//
	// PushConfig is specified with the -push-config flag.
	PushConfig string

	// ReleaseID is the identifier of a release PR. Each release PR created by
	// Librarian has a release ID, which is included in both the PR description and
	// the commit message of every commit within the release PR. This is effectively
	// used for internal bookkeeping, to collate all library releases within a single
	// release flow. The format of ReleaseID is effectively opaque; it is currently
	// timestamp-based but could change to a UUID or similar in the future.
	//
	// ReleaseID is required for the create-release-artifacts command, and is only
	// used by this command.
	//
	// ReleaseID is specified with the -release-id flag.
	ReleaseID string

	// Repo specifies the language repository to use, as either a local root directory
	// or a URL to clone from. If a local directory is specified, it can
	// be relative to the current working directory. The repository must
	// be in a clean state (i.e. git status should report no changes) to avoid mixing
	// Librarian-created changes with other changes.
	//
	// Repo is used by all commands which operate on a language repository:
	// configure, create-release-artifacts, generate, update-apis,
	// update-image-tag.
	//
	// When a local directory is specified for the generate command, the repo is checked to
	// determine whether the specified API path is configured as a library. See the generate
	// command documentation for more details.
	// For all commands other than generate, omitting Repo is equivalent to
	// specifying Repo as https://github.com/googleapis/google-cloud-{Language}.
	//
	// Repo is specified with the -repo flag.
	Repo string

	// SkipIntegrationTests is used by the create-release-pr and create-release-artifacts
	// commands, and disables integration tests if it is set to a non-empty value.

	// SkipIntegrationTests is used by the create-release-artifacts
	// command, and disables integration tests if it is set to a non-empty value.
	// The value must reference a bug (e.g., b/12345).
	//
	// SkipIntegrationTests is specified with the -skip-integration-tests flag.
	SkipIntegrationTests string

	// Source is the path to the root of the googleapis repository.
	// When this is not specified, the googleapis repository is cloned
	// automatically.
	//
	// Source is used by generate, update-apis, update-image-tag and configure
	// commands.
	//
	// Source is specified with the -source flag.
	Source string

	// Tag is the new tag for the language-specific Docker image, used only by the
	// update-image-tag command. All operations within update-image-tag are performed
	// using the new tag.
	//
	// Tag is specified with the -tag flag.
	Tag string

	// UserGID is the group ID of the current user. It is used to run Docker
	// containers with the same user, so that created files have the correct
	// ownership.
	//
	// This is populated automatically after flag parsing. No user setup is
	// expected.
	UserGID string

	// UserUID is the user ID of the current user. It is used to run Docker
	// containers with the same user, so that created files have the correct
	// ownership.
	//
	// This is populated automatically after flag parsing. No user setup is
	// expected.
	UserUID string

	// WorkRoot is the root directory used for temporary working files, including
	// any repositories that are cloned. By default, this is created in /tmp with
	// a timestamped directory name (e.g. /tmp/librarian-20250617T083548Z) but
	// can be specified with the -output flag.
	//
	// WorkRoot is used by all librarian commands.
	WorkRoot string
}

// New returns a new Config populated with environment variables.
func New() *Config {
	return &Config{
		GitHubToken: os.Getenv("LIBRARIAN_GITHUB_TOKEN"),
		PushConfig:  DefaultPushConfig,
	}
}

// currentUser is a variable, so it can be replaced during testing.
var currentUser = user.Current

// SetupUser performs late initialization of user-specific configuration,
// determining the current user. This is in a separate method as it
// can fail, and is called after flag parsing.
func (c *Config) SetupUser() error {
	user, err := currentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	c.UserUID = user.Uid
	c.UserGID = user.Gid
	return nil
}

// IsValid ensures the values contained in a Config are valid.
func (c *Config) IsValid() (bool, error) {
	if c.Push && c.GitHubToken == "" {
		return false, errors.New("no GitHub token supplied for push")
	}

	if _, err := validatePushConfig(c.PushConfig, DefaultPushConfig); err != nil {
		return false, err
	}

	if _, err := validateHostMount(c.HostMount, ""); err != nil {
		return false, err
	}

	return true, nil
}

func validateHostMount(hostMount, defaultValue string) (bool, error) {
	if hostMount == defaultValue {
		return true, nil
	}

	parts := strings.Split(hostMount, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false, errors.New("unable to parse host mount")
	}

	return true, nil
}

func validatePushConfig(pushConfig, defaultValue string) (bool, error) {
	if pushConfig == defaultValue {
		return true, nil
	}

	parts := strings.Split(pushConfig, ",")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false, errors.New("unable to parse push config")
	}

	return true, nil
}
