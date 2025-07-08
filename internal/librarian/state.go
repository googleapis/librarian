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

package librarian

import (
	"github.com/containers/image/v5/docker/reference"
)

// LibrarianState defines the contract for the state.yaml file.
type LibrarianState struct {
	Image     string    `yaml:"image" validate:"required,is-image"`
	Libraries []Library `yaml:"libraries" validate:"required,gt=0,dive"`
}

// ImageRefAndTag extracts the image reference and tag from the full image string.
// For example, for "gcr.io/my-image:v1.2.3", it returns a reference to
// "gcr.io/my-image" and the tag "v1.2.3".
// It correctly handles complex image references. If no tag is present, the
// returned tag is an empty string.
func (s *LibrarianState) ImageRefAndTag() (ref reference.Named, tag string) {
	if s == nil || s.Image == "" {
		return nil, ""
	}
	named, err := reference.ParseNormalizedNamed(s.Image)
	if err != nil {
		// The validator should prevent invalid image strings, but as a safeguard:
		return nil, ""
	}
	if tagged, ok := named.(reference.Tagged); ok {
		tag = tagged.Tag()
	}
	return named, tag
}

// Library represents the state of a single library within state.yaml.
type Library struct {
	Id                  string   `yaml:"id" validate:"required,is-library-id"`
	Version             string   `yaml:"version" validate:"omitempty,semver"`
	LastGeneratedCommit string   `yaml:"last_generated_commit" validate:"omitempty,hexadecimal,len=40"`
	APIs                []API    `yaml:"apis" validate:"required,gt=0,dive"`
	SourcePaths         []string `yaml:"source_paths" validate:"required,gt=0,dive,is-dirpath"`
	PreserveRegex       []string `yaml:"preserve_regex" validate:"omitempty,dive,is-regexp"`
	RemoveRegex         []string `yaml:"remove_regex" validate:"omitempty,dive,is-regexp"`
}

// API represents an API that is part of a library.
type API struct {
	Path          string `yaml:"path" validate:"required,is-dirpath"`
	ServiceConfig string `yaml:"service_config" validate:"required"`
}
