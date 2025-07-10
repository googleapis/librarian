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

import "strings"

// LibrarianState defines the contract for the state.yaml file.
type LibrarianState struct {
	// The name and tag of the generator image to use. tag is required.
	Image string `yaml:"image" validate:"required,is-image"`
	// A list of library configurations.
	Libraries []Library `yaml:"libraries" validate:"required,gt=0,dive"`
}

// ImageRefAndTag extracts the image reference and tag from the full image string.
// For example, for "gcr.io/my-image:v1.2.3", it returns a reference to
// "gcr.io/my-image" and the tag "v1.2.3".
// If no tag is present, the returned tag is an empty string.
func (s *LibrarianState) ImageRefAndTag() (ref string, tag string) {
	if s == nil {
		return "", ""
	}
	return parseImage(s.Image)
}

// parseImage splits an image string into its reference and tag.
// It correctly handles port numbers in the reference.
// If no tag is found, the tag part is an empty string.
func parseImage(image string) (ref string, tag string) {
	if image == "" {
		return "", ""
	}
	lastColon := strings.LastIndex(image, ":")
	if lastColon < 0 {
		return image, ""
	}
	// if there is a slash after the last colon, it's a port number, not a tag.
	if strings.Contains(image[lastColon:], "/") {
		return image, ""
	}
	return image[:lastColon], image[lastColon+1:]
}

// Library represents the state of a single library within state.yaml.
type Library struct {
	// A unique identifier for the library, in a language-specific format.
	// A valid Id should not be empty and only contains alphanumeric characters, slashes, periods, underscores, and hyphens.
	Id string `yaml:"id" validate:"required,is-library-id"`
	// The last released version of the library.
	Version string `yaml:"version" validate:"omitempty,semver"`
	// The commit hash from the API definition repository at which the library was last generated.
	LastGeneratedCommit string `yaml:"last_generated_commit" validate:"omitempty,hexadecimal,len=40"`
	// A list of APIs that are part of this library.
	APIs []API `yaml:"apis" validate:"required,gt=0,dive"`
	// A list of directories in the language repository where Librarian contributes code.
	SourcePaths []string `yaml:"source_paths" validate:"required,gt=0,dive,is-dirpath"`
	// A list of regular expressions for files and directories to preserve during the copy and remove process.
	PreserveRegex []string `yaml:"preserve_regex" validate:"omitempty,dive,is-regexp"`
	// A list of regular expressions for files and directories to remove before copying generated code.
	// If not set, this defaults to the `source_paths`.
	// A more specific `preserve_regex` takes precedence.
	RemoveRegex []string `yaml:"remove_regex" validate:"omitempty,dive,is-regexp"`
}

// API represents an API that is part of a library.
type API struct {
	// The path to the API, relative to the root of the API definition repository (e.g., "google/storage/v1").
	Path string `yaml:"path" validate:"required,is-dirpath"`
	// The name of the service config file, relative to the API `path`.
	ServiceConfig string `yaml:"service_config" validate:"omitempty"`
}
