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

// AutomationLevel is the degree of automation to use when generating/releasing.
type AutomationLevel int32

const (
	// AutomationLevelNone is not used.
	AutomationLevelNone AutomationLevel = 0
	// AutomationLevelBlocked is for when automation is blocked: this API/library should be skipped.
	AutomationLevelBlocked AutomationLevel = 1
	// AutomationLevelManualReview is for when automation can generate changes/releases,
	// but they need to be reviewed.
	AutomationLevelManualReview AutomationLevel = 2
	// AutomationLevelAutomatic is for when automation can generated changes/releases which can
	// proceed without further review if all tests pass.
	AutomationLevelAutomatic AutomationLevel = 3
)

// PipelineState is the overall state of the generation and release pipeline. This is expected
// to be stored in each language repo as generator-input/pipeline-state.json.
type PipelineState struct {
	// The image tag that the CLI should use when invoking the
	// language-specific tooling. The image name is assumed by convention, or
	// overridden in PipelineConfig.
	ImageTag string `json:"image_tag,omitempty"`
	// The state of each library which is released within this repository.
	Libraries []*LibraryState `json:"libraries,omitempty"`
	// Paths to files/directories which can trigger
	// a release in all libraries.
	CommonLibrarySourcePaths []string `json:"common_library_source_paths,omitempty"`
	// API paths which are deliberately not configured. (Ideally this would
	// be empty for all languages, but there may be temporary reasons not to configure
	// an API.)
	IgnoredAPIPaths []string `json:"ignored_api_paths,omitempty"`
}

// LibraryState is the generation state of a single library.
type LibraryState struct {
	// The library identifier (language-specific format)
	ID string `json:"id,omitempty"`
	// The last version that was released, if any.
	CurrentVersion string `json:"current_version,omitempty"`
	// The next version to release (to force a specific version number).
	// This should usually be unset.
	NextVersion string `json:"next_version,omitempty"`
	// The automation level for generation for this library.
	GenerationAutomationLevel AutomationLevel `json:"generation_automation_level,omitempty"`
	// The automation level for releases for this library.
	ReleaseAutomationLevel AutomationLevel `json:"release_automation_level,omitempty"`
	// The timestamp of the latest release. (This is typically
	// some timestamp within the process of generating the release
	// PR for the library. Importantly, it's not just a date as
	// there may be reasons to release a library multiple times
	// within a day.) This is unset if the library has not yet been
	// released.
	ReleaseTimestamp string `json:"release_timestamp,omitempty"`
	// The commit hash (within the API definition repo) at which
	// the library was last generated. This is empty if the library
	// has not yet been generated.
	LastGeneratedCommit string `json:"last_generated_commit,omitempty"`
	// The last-generated commit hash (within the API definition repo)
	// at the point of the last/in-progress release. (This is taken
	// from the generation state at the time of the release.) This
	// is empty if the library has not yet been released.
	LastReleasedCommit string `json:"last_released_commit,omitempty"`
	// The API paths included in this library, relative to the root
	// of the API definition repo, e.g. "google/cloud/functions/v2".
	APIPaths []string `json:"api_paths,omitempty"`
	// Paths to files or directories contributing to this library.
	SourcePaths []string `json:"source_paths,omitempty"`
}

// PipelineConfig is the manually-maintained configuration for the pipeline.
type PipelineConfig struct {
	// The name of the image to use, where the convention is not
	// appropriate. The tag is specified in PipelineState.
	ImageName string `json:"image_name,omitempty"`
	// Specific configuration for each individual command.
	Commands map[string]*CommandConfig `json:"commands,omitempty"`
	// The maximum number (inclusive) of commits to create
	// in a single pull request. If this is non-positive, it is
	// ignored. If a process would generate a pull request with more
	// commits than this, excess commits are trimmed and the commits
	// which *would* have been present are described in the PR.
	MaxPullRequestCommits int32 `json:"max_pull_request_commits,omitempty"`
}

// CommandConfig is the configuration for a specific container command.
type CommandConfig struct {
	// The environment variables to populate for this command.
	EnvironmentVariables []*CommandEnvironmentVariable `json:"environment_variables,omitempty"`
}

// CommandEnvironmentVariable is an environment variable to be provided to a container command.
type CommandEnvironmentVariable struct {
	// The name of the environment variable (e.g. TEST_PROJECT).
	Name string `json:"name,omitempty"`
	// The name of the secret to be used to fetch the value of the environment
	// variable when it's not present in the host system. If this is not specified,
	// or if a Secret Manager project has not been provided to Librarian,
	// Secret Manager will not be used as a source for the environment variable.
	SecretName string `json:"secret_name,omitempty"`
	// The default value to specify if no other source is found for the environment
	// variable. If this is not provided and no other source is found, the environment
	// variable will not be passed to the container at all.
	DefaultValue string `json:"default_value,omitempty"`
}
