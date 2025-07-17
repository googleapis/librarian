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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// LibrarianState defines the contract for the state.yaml file.
type LibrarianState struct {
	// The name and tag of the generator image to use. tag is required.
	Image string `yaml:"image"`
	// A list of library configurations.
	Libraries []*LibraryState `yaml:"libraries"`
}

// Validate checks that the LibrarianState is valid.
func (s *LibrarianState) Validate() error {
	if s.Image == "" {
		return fmt.Errorf("image is required")
	}
	if !isValidImage(s.Image) {
		return fmt.Errorf("invalid image: %q", s.Image)
	}
	if len(s.Libraries) == 0 {
		return fmt.Errorf("libraries cannot be empty")
	}
	for i, l := range s.Libraries {
		if l == nil {
			return fmt.Errorf("library at index %d cannot be nil", i)
		}
		if err := l.Validate(); err != nil {
			return fmt.Errorf("invalid library at index %d: %w", i, err)
		}
	}
	return nil
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

// LibraryState represents the state of a single library within state.yaml.
type LibraryState struct {
	// A unique identifier for the library, in a language-specific format.
	// A valid ID should not be empty and only contains alphanumeric characters, slashes, periods, underscores, and hyphens.
	ID string `yaml:"id"`
	// The last released version of the library, following SemVer.
	Version string `yaml:"version"`
	// The commit hash from the API definition repository at which the library was last generated.
	LastGeneratedCommit string `yaml:"last_generated_commit"`
	// A list of APIs that are part of this library.
	APIs []*API `yaml:"apis"`
	// A list of directories in the language repository where Librarian contributes code.
	SourcePaths []string `yaml:"source_paths"`
	// A list of regular expressions for files and directories to preserve during the copy and remove process.
	PreserveRegex []string `yaml:"preserve_regex"`
	// A list of regular expressions for files and directories to remove before copying generated code.
	// If not set, this defaults to the `source_paths`.
	// A more specific `preserve_regex` takes precedence.
	RemoveRegex []string `yaml:"remove_regex"`
}

var (
	libraryIDRegex = regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`)
	semverRegex    = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)
	hexRegex       = regexp.MustCompile("^[a-fA-F0-9]+$")
)

// Validate checks that the Library is valid.
func (l *LibraryState) Validate() error {
	if l.ID == "" {
		return fmt.Errorf("id is required")
	}
	if l.ID == "." || l.ID == ".." {
		return fmt.Errorf(`id cannot be "." or ".." only`)
	}
	if !libraryIDRegex.MatchString(l.ID) {
		return fmt.Errorf("invalid id: %q", l.ID)
	}
	if l.Version != "" && !semverRegex.MatchString(l.Version) {
		return fmt.Errorf("invalid version: %q", l.Version)
	}
	if l.LastGeneratedCommit != "" {
		if !hexRegex.MatchString(l.LastGeneratedCommit) {
			return fmt.Errorf("last_generated_commit must be a hex string")
		}
		if len(l.LastGeneratedCommit) != 40 {
			return fmt.Errorf("last_generated_commit must be 40 characters")
		}
	}
	if len(l.APIs) == 0 {
		return fmt.Errorf("apis cannot be empty")
	}
	for i, a := range l.APIs {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("invalid api at index %d: %w", i, err)
		}
	}
	if len(l.SourcePaths) == 0 {
		return fmt.Errorf("source_paths cannot be empty")
	}
	for i, p := range l.SourcePaths {
		if !isValidDirPath(p) {
			return fmt.Errorf("invalid source_path at index %d: %q", i, p)
		}
	}
	for i, r := range l.PreserveRegex {
		if _, err := regexp.Compile(r); err != nil {
			return fmt.Errorf("invalid preserve_regex at index %d: %w", i, err)
		}
	}
	for i, r := range l.RemoveRegex {
		if _, err := regexp.Compile(r); err != nil {
			return fmt.Errorf("invalid remove_regex at index %d: %w", i, err)
		}
	}
	return nil
}

// API represents an API that is part of a library.
type API struct {
	// The path to the API, relative to the root of the API definition repository (e.g., "google/storage/v1").
	Path string `yaml:"path"`
	// The name of the service config file, relative to the API `path`.
	ServiceConfig string `yaml:"service_config"`
}

// Validate checks that the API is valid.
func (a *API) Validate() error {
	if !isValidDirPath(a.Path) {
		return fmt.Errorf("invalid path: %q", a.Path)
	}
	return nil
}

// invalidPathChars contains characters that are invalid in path components,
// plus path separators and the null byte.
const invalidPathChars = `<>:"|?*\/\\x00`

func isValidDirPath(pathString string) bool {
	if pathString == "" {
		return false
	}

	// The paths are expected to be relative and use the OS-specific path separator.
	// We clean the path to resolve ".." and check that it doesn't try to
	// escape the root.
	cleaned := filepath.Clean(pathString)
	if filepath.IsAbs(pathString) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return false
	}

	// A single dot is a valid relative path, but likely not the intended input.
	if cleaned == "." {
		return false
	}

	// Each path component must not contain invalid characters.
	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if strings.ContainsAny(component, invalidPathChars) {
			return false
		}
	}
	return true
}

// isValidImage checks if a string is a valid container image name with a required tag.
// It validates that the image string contains a tag, separated by a colon, and has no whitespace.
// It correctly distinguishes between a tag and a port number in the registry host.
func isValidImage(image string) bool {
	// Basic validation: no whitespace.
	if strings.ContainsAny(image, " \t\n\r") {
		return false
	}

	ref, tag := parseImage(image)

	return ref != "" && tag != ""
}
