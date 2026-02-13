// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package snippetmetadata provides cross-language functionality for working
// with metadata files associated with generated sample code.
package snippetmetadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrNoClientLibraryField is the error returned if a snippet metadata file
	// has no clientLibrary field at the top level.
	ErrNoClientLibraryField     = errors.New("no clientLibrary field at the top level")
	errSnippetMetadataDirectory = errors.New("expected file; was a directory")
	errSnippetMetadataLink      = errors.New("expected regular file; was a link")
)

// readMetadata reads and parses the file at the given path as a JSON file.
func readMetadata(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading snippet metadata file %s: %w", path, err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(content, &metadata); err != nil {
		return nil, fmt.Errorf("error parsing snippet metadata file %s: %w", path, err)
	}
	return metadata, nil
}

// writeMetadata formats and writes the given metadata as a JSON file at the
// given path.
func writeMetadata(path string, metadata map[string]any) error {
	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding snippet metadata file %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("error writing snippet metadata file %s: %w", path, err)
	}
	return nil
}

// updateLibraryVersion updates the client library version for a single file.
func updateLibraryVersion(path, version string) error {
	metadata, err := readMetadata(path)
	if err != nil {
		return err
	}
	clientLibrary, ok := metadata["clientLibrary"].(map[string]any)
	if !ok {
		return fmt.Errorf("error updating snippet metadata file %s: %w", path, ErrNoClientLibraryField)
	}
	clientLibrary["version"] = version
	return writeMetadata(path, metadata)
}

// reformat reads a JSON file with the given path, and reformats it.
func reformat(path string) error {
	metadata, err := readMetadata(path)
	if err != nil {
		return err
	}
	return writeMetadata(path, metadata)
}

// findAll finds all snippet metadata files (filenames starting with
// "snippet_metadata" and ending with ".json") under the given directory
// (including subdirectories).
func findAll(dir string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if !strings.HasPrefix(name, "snippet_metadata") || !strings.HasSuffix(name, ".json") {
			return nil
		}
		if d.IsDir() {
			return fmt.Errorf("error for possible snippet metadata file %s: %w", path, errSnippetMetadataDirectory)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("error for possible snippet metadata file %s: %w", path, errSnippetMetadataLink)
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// ReformatAll reformats all snippet metadata files (filenames starting with
// "snippet_metadata" and ending with ".json") under the given directory
// (including subdirectories).
func ReformatAll(dir string) error {
	files, err := findAll(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := reformat(file); err != nil {
			return err
		}
	}
	return nil
}

// UpdateAllLibraryVersions updates the clientLibrary.version field of all
// snippet metadata files (filenames starting with "snippet_metadata" and
// ending with ".json") under the given directory (including subdirectories).
func UpdateAllLibraryVersions(dir, version string) error {
	files, err := findAll(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := updateLibraryVersion(file, version); err != nil {
			return err
		}
	}
	return nil
}
