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
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// BuildRequest is a JSON file that describes which library to build/test.
	BuildRequest = "build-request.json"
	// BuildResponse is a JSON file that describes which library to change after
	// built/test.
	BuildResponse = "build-response.json"
	// ConfigureRequest is a JSON file that describes which library to configure.
	ConfigureRequest = "configure-request.json"
	// ConfigureResponse is a JSON file that describes which library to change
	// after initial configuration.
	ConfigureResponse = "configure-response.json"
	// GeneratorInputDir is the default directory to store files that generator
	// needs to regenerate libraries from an empty directory.
	GeneratorInputDir = ".librarian/generator-input"
	// GenerateRequest is a JSON file that describes which library to generate.
	GenerateRequest = "generate-request.json"
	// GenerateResponse is a JSON file that describes which library to change
	// after re-generation.
	GenerateResponse = "generate-response.json"
	// LibrarianDir is the default directory to store librarian state/config files,
	// along with any additional configuration.
	LibrarianDir = ".librarian"
	// ReleaseStageRequest is a JSON file that describes which library to release.
	ReleaseStageRequest = "release-stage-request.json"
	// ReleaseStageResponse is a JSON file that describes which library to change
	// after release.
	ReleaseStageResponse = "release-stage-response.json"
	// LibrarianStateFile is the name of the pipeline state file.
	LibrarianStateFile = "state.yaml"
	// LibrarianConfigFile is the name of the language-repository config file.
	LibrarianConfigFile = "config.yaml"
	// LibrarianGithubToken is the name of the env var used to store the GitHub token.
	LibrarianGithubToken = "LIBRARIAN_GITHUB_TOKEN"
)

// are variables so it can be replaced during testing.
var (
	tempDir     = os.TempDir
	currentUser = user.Current
)

var (
	// pullRequestRegexp is regular expression that describes an uri of a pull request.
	pullRequestRegexp = regexp.MustCompile(`^https://github\.com/([a-zA-Z0-9-._]+)/([a-zA-Z0-9-._]+)/pull/([0-9]+)$`)
)

// Config holds all configuration values parsed from flags or environment
// variables. When adding members to this struct, please keep them in
// alphabetical order.
type Config struct {
	// API is the path to the API to be configured or generated,
	// relative to the root of the googleapis repository. It is a directory
	// name as far as (and including) the version (v1, v2, v1alpha etc.). It
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

	// APISourceDepth controls the depth of the repository closing **IF**
	// APISource is a GitHub repository, and it is cloned.
	APISourceDepth int

	// Branch is the remote branch of the language repository to use.
	// This is the branch which is cloned when Repo is a URL, and also used
	// as the base reference for any pull requests created by the command.
	// By default, the branch "main" is used.
	Branch string

	// Build determines whether to build the generated library, and is only
	// used in the generate command.
	//
	// Build is specified with the -build flag.
	Build bool

	// CI is the type of Continuous Integration (CI) environment in which
	// the tool is executing.
	CI string

	// CommandName is the name of the command being executed.
	//
	// commandName is populated automatically after flag parsing. No user setup is
	// expected.
	CommandName string

	// Commit determines whether to create a commit for the release but not create
	// a pull request.
	//
	// This flag is ignored if Push is set to true.
	Commit bool

	// GenerateUnchanged determines whether to generate libraries where none of
	// the associated APIs have changed since the commit at which they were last
	// generated. Note that this does not override any configuration indicating
	// that the library should not be automatically generated.
	GenerateUnchanged bool

	// GitHubAPIEndpoint is the GitHub API endpoint to use for all GitHub API
	// operations.
	//
	// This is intended for testing and should not be used in production.
	GitHubAPIEndpoint string

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

	// Library is the library ID to generate (e.g. secretmanager).
	// This usually corresponds to a releasable language unit -- for Go this would
	// be a Go module or for dotnet the name of a NuGet package. If neither this nor
	// api is specified all currently managed libraries will be regenerated.
	Library string

	// LibraryVersion is the library version to release.
	//
	// Overrides the automatic semantic version calculation and forces a specific
	// version for a library.
	// This is intended for exceptional cases, such as applying a backport patch
	// or forcing a major version bump.
	//
	// Requires the --library flag to be specified.
	LibraryVersion string

	// PullRequest to target and operate one in the context of a release.
	//
	// The pull request should be in the format `https://github.com/{owner}/{repo}/pull/{number}`.
	// Setting this field for `tag` means librarian will only attempt
	// to process this exact pull request and not search for other pull requests
	// that may be ready for tagging and releasing.
	PullRequest string

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
func New(cmdName string) *Config {
	return &Config{
		CommandName: cmdName,
		GitHubToken: os.Getenv(LibrarianGithubToken),
	}
}

// setupUser performs late initialization of user-specific configuration,
// determining the current user. This is in a separate method as it
// can fail, and is called after flag parsing.
func (c *Config) setupUser() error {
	user, err := currentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	c.UserUID = user.Uid
	c.UserGID = user.Gid
	return nil
}

func (c *Config) createWorkRoot() error {
	if c.WorkRoot != "" {
		slog.Info("using specified working directory", "dir", c.WorkRoot)
		return nil
	}
	path, err := os.MkdirTemp(tempDir(), "librarian-*")
	if err != nil {
		return err
	}

	slog.Info("temporary working directory", "dir", path)
	c.WorkRoot = path
	return nil
}

func (c *Config) deriveRepo() error {
	if c.Repo != "" {
		slog.Debug("repo value provided by user", "repo", c.Repo)
		return nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	stateFile := filepath.Join(wd, LibrarianDir, LibrarianStateFile)
	if _, err := os.Stat(stateFile); err != nil {
		return fmt.Errorf("repo flag not specified and no state file found in current working directory: %w", err)
	}
	slog.Info("repo not specified, using current working directory as repo root", "path", wd)
	c.Repo = wd
	return nil
}

// IsValid ensures the values contained in a Config are valid.
func (c *Config) IsValid() (bool, error) {
	if c.Push && c.GitHubToken == "" {
		return false, errors.New("no GitHub token supplied for push")
	}

	if c.Library == "" && c.LibraryVersion != "" {
		return false, errors.New("specified library version without library id")
	}

	if c.PullRequest != "" {
		matched := pullRequestRegexp.MatchString(c.PullRequest)
		if !matched {
			return false, errors.New("pull request URL is not valid")
		}
	}

	if _, err := validateHostMount(c.HostMount, ""); err != nil {
		return false, err
	}

	if c.Repo == "" {
		return false, errors.New("language repository not specified or detected")
	}

	return true, nil
}

// SetDefaults initializes values not set directly by the user.
func (c *Config) SetDefaults() error {
	if err := c.setupUser(); err != nil {
		return err
	}
	if err := c.createWorkRoot(); err != nil {
		return err
	}
	if err := c.deriveRepo(); err != nil {
		return err
	}
	return nil
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
