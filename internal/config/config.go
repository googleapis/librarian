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
	// BuildRequest is a JSON file that describes which library to build/test.
	BuildRequest string = "build-request.json"
	// ConfigureRequest is a JSON file that describes which library to configure.
	ConfigureRequest string = "configure-request.json"
	// GeneratorInputDir is the default directory to store files that generator
	// needs to regenerate libraries from an empty directory.
	GeneratorInputDir string = ".librarian/generator-input"
	// GenerateRequest is a JSON file that describes which library to generate.
	GenerateRequest string = "generate-request.json"
	// LibrarianDir is the default directory to store librarian state/config files,
	// along with any additional configuration.
	LibrarianDir string = ".librarian"
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

	// APISource is the path to the root of the googleapis repository.
	// When this is not specified, the googleapis repository is cloned
	// automatically.
	//
	// APISource is used by generate, update-apis and configure
	// commands.
	//
	// APISource is specified with the -api-source flag.
	APISource string

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

	// Library is the library ID to generate (e.g. google-cloud-secretmanager-v1 ).
	// This usually corresponds to a releasable language unit -- for Go this would
	// be a Go module or for dotnet the name of a NuGet package. If neither this nor
	// api is specified all currently managed libraries will be regenerated.
	Library string

	// Push determines whether to push changes to GitHub. It is used in
	// all commands that create commits in a language repository:
	// configure and update-apis.
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
	// create-release-pr, configure and update-apis.
	//
	// PushConfig is optional. If unspecified, commits will use a default name of
	// "Google Cloud SDK" and a default email of noreply-cloudsdk@google.com.
	//
	// PushConfig is specified with the -push-config flag.
	PushConfig string

	// Repo specifies the language repository to use, as either a local root directory
	// or a URL to clone from. If a local directory is specified, it can
	// be relative to the current working directory. The repository must
	// be in a clean state (i.e. git status should report no changes) to avoid mixing
	// Librarian-created changes with other changes.
	//
	// Repo is used by all commands which operate on a language repository:
	// configure, generate, update-apis.
	//
	// When a local directory is specified for the generate command, the repo is checked to
	// determine whether the specified API path is configured as a library. See the generate
	// command documentation for more details.
	// For all commands other than generate, omitting Repo is equivalent to
	// specifying Repo as https://github.com/googleapis/google-cloud-{Language}.
	//
	// Repo is specified with the -repo flag.
	Repo string

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
		PushConfig:  "",
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

	if _, err := validatePushConfig(c.PushConfig, ""); err != nil {
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
