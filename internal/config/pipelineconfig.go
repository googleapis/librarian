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
