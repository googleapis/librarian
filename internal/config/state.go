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

	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	StatusNew      = "new"
	StatusExisting = "existing"
)

// LibrarianState defines the contract for the state.yaml file.
type LibrarianState struct {
	// The name and tag of the generator image to use. tag is required.
	Image string `yaml:"image" json:"image"`
	// A list of library configurations.
	Libraries []*LibraryState `yaml:"libraries" json:"libraries"`
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
	seenLibraryIDs := make(map[string]bool)
	for i, l := range s.Libraries {
		if l == nil {
			return fmt.Errorf("library at index %d cannot be nil", i)
		}
		if _, exists := seenLibraryIDs[l.ID]; exists {
			return fmt.Errorf("duplicate library ID %s", l.ID)
		}
		seenLibraryIDs[l.ID] = true
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

// LibraryByID returns the library with the given ID, or nil if not found.
func (s *LibrarianState) LibraryByID(id string) *LibraryState {
	for _, lib := range s.Libraries {
		if lib.ID == id {
			return lib
		}
	}
	return nil
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
	ID string `yaml:"id" json:"id"`
	// The last released version of the library, following SemVer.
	Version string `yaml:"version" json:"version"`
	// The commit hash from the API definition repository at which the library was last generated.
	LastGeneratedCommit string `yaml:"last_generated_commit" json:"-"`
	// The changes from the language repository since the library was last released.
	// This field is ignored when writing to state.yaml.
	Changes []*Commit `yaml:"-" json:"changes,omitempty"`
	// A list of APIs that are part of this library.
	APIs []*API `yaml:"apis" json:"apis"`
	// A list of directories in the language repository where Librarian contributes code.
	SourceRoots []string `yaml:"source_roots" json:"source_roots"`
	// The previous release version, this field is only for bookkeeping.
	PreviousVersion string `yaml:"-" json:"-"`
	// A list of regular expressions for files and directories to preserve during the copy and remove process.
	PreserveRegex []string `yaml:"preserve_regex" json:"preserve_regex"`
	// A list of regular expressions for files and directories to remove before copying generated code.
	// If not set, this defaults to the `source_roots`.
	// A more specific `preserve_regex` takes precedence.
	RemoveRegex []string `yaml:"remove_regex" json:"remove_regex"`
	// A list of paths to exclude from the release.
	// Files matching these paths will not be considered part of a commit for this library.
	ReleaseExcludePaths []string `yaml:"release_exclude_paths,omitempty" json:"release_exclude_paths,omitempty"`
	// Specifying a tag format allows librarian to honor this format when creating
	// a tag for the release of the library. The replacement values of {id} and {version}
	// permitted to reference the values configured in the library. If not specified
	// the assumed format is {id}-{version}. e.g., {id}/v{version}.
	TagFormat string `yaml:"tag_format,omitempty" json:"tag_format,omitempty"`
	// Whether including this library in a release.
	// This field is ignored when writing to state.yaml.
	ReleaseTriggered bool `yaml:"-" json:"release_triggered,omitempty"`
	// An error message from the docker response.
	// This field is ignored when writing to state.yaml.
	ErrorMessage string `yaml:"-" json:"error,omitempty"`
}

// Commit represents a single commit in the release notes.
type Commit struct {
	// Type is the type of change (e.g., "feat", "fix", "docs").
	Type string `json:"type"`
	// Subject is the short summary of the change.
	Subject string `json:"subject"`
	// Body is the long-form description of the change.
	Body string `json:"body"`
	// CommitHash is the full commit hash.
	CommitHash string `json:"commit_hash,omitempty"`
	// PiperCLNumber is the Piper CL number associated with the commit.
	PiperCLNumber string `json:"piper_cl_number,omitempty"`

	// A list of library IDs associated with the commit.
	LibraryIDs string `json:"-"`
}

// IsBulkCommit returns true if the commit is associated with 10 or more
// libraries.
func (c *Commit) IsBulkCommit() bool {
	return strings.Count(c.LibraryIDs, ",") >= 9
}

var (
	libraryIDRegex = regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`)
	semverRegex    = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)
	hexRegex       = regexp.MustCompile("^[a-fA-F0-9]+$")
	tagFormatRegex = regexp.MustCompile(`{[^{}]*}`)
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
	for i, a := range l.APIs {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("invalid api at index %d: %w", i, err)
		}
	}
	if len(l.SourceRoots) == 0 {
		return fmt.Errorf("source_roots cannot be empty")
	}
	for i, p := range l.SourceRoots {
		if !isValidRelativePath(p) {
			return fmt.Errorf("invalid source_path at index %d: %q", i, p)
		}
	}
	for i, p := range l.ReleaseExcludePaths {
		if !isValidRelativePath(p) {
			return fmt.Errorf("invalid release_exclude_path at index %d: %q", i, p)
		}
	}
	if l.TagFormat != "" {
		if !strings.Contains(l.TagFormat, "{version}") {
			return fmt.Errorf("invalid tag_format: must contain {version}")
		}
		matches := tagFormatRegex.FindAllString(l.TagFormat, -1)
		for _, match := range matches {
			if match != "{id}" && match != "{version}" {
				return fmt.Errorf("invalid tag_format: placeholder %s not recognized", match)
			}
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
	Path string `yaml:"path" json:"path"`
	// The name of the service config file, relative to the API `path`.
	ServiceConfig string `yaml:"service_config" json:"service_config"`
	// The status of the API, one of "new" or "existing".
	// This field is ignored when writing to state.yaml.
	Status string `yaml:"-" json:"status"`
}

// Validate checks that the API is valid.
func (a *API) Validate() error {
	if !isValidRelativePath(a.Path) {
		return fmt.Errorf("invalid path: %q", a.Path)
	}
	return nil
}

// invalidPathChars contains characters that are invalid in path components,
// plus path separators and the null byte.
const invalidPathChars = "<>:\"|?*/\\\x00"

func isValidRelativePath(pathString string) bool {
	if pathString == "" {
		return false
	}

	if pathString == gitrepo.RootPath {
		return true
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
