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
)

const (
	PermissionReadOnly  = "read-only"
	PermissionWriteOnly = "write-only"
	PermissionReadWrite = "read-write"
)

// LibrarianConfig defines the contract for the config.yaml file.
type LibrarianConfig struct {
	GlobalFilesAllowlist []*GlobalFile    `yaml:"global_files_allowlist"`
	Libraries            []*LibraryConfig `yaml:"libraries"`
	TagFormat            string           `yaml:"tag_format"`
}

// LibraryConfig defines configuration for a single library, identified by its ID.
type LibraryConfig struct {
	GenerateBlocked bool   `yaml:"generate_blocked"`
	LibraryID       string `yaml:"id"`
	NextVersion     string `yaml:"next_version"`
	ReleaseBlocked  bool   `yaml:"release_blocked"`
	TagFormat       string `yaml:"tag_format"`
	// Whether to create a GitHub release for this library.
	SkipGitHubReleaseCreation bool `yaml:"skip_github_release_creation"`
}

// GlobalFile defines the global files in language repositories.
type GlobalFile struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions"`
}

var validPermissions = map[string]bool{
	PermissionReadOnly:  true,
	PermissionWriteOnly: true,
	PermissionReadWrite: true,
}

// Validate checks that the LibrarianConfig is valid.
func (g *LibrarianConfig) Validate() error {
	for i, globalFile := range g.GlobalFilesAllowlist {
		path, permissions := globalFile.Path, globalFile.Permissions
		if !isValidRelativePath(path) {
			return fmt.Errorf("invalid global file path at index %d: %q", i, path)
		}
		if _, ok := validPermissions[permissions]; !ok {
			return fmt.Errorf("invalid global file permissions at index %d: %q", i, permissions)
		}
	}

	return nil
}

// LibraryConfigFor finds the LibraryConfig entry for a given LibraryID.
func (g *LibrarianConfig) LibraryConfigFor(LibraryID string) *LibraryConfig {
	for _, lib := range g.Libraries {
		if lib.LibraryID == LibraryID {
			return lib
		}
	}
	return nil
}

// GetGlobalFiles returns the global files defined in the librarian config.
func (g *LibrarianConfig) GetGlobalFiles() []string {
	var globalFiles []string
	for _, globalFile := range g.GlobalFilesAllowlist {
		globalFiles = append(globalFiles, globalFile.Path)
	}

	return globalFiles
}
