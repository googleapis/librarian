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

package pom

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var (
	versionRegex = regexp.MustCompile(`(<version>)([^<]+)(</version>\s*<!-- \{x-version-update:([^:]+):current\} -->)`)
)

// UpdateVersions updates the versions of all pom.xml files in a given directory.
// If the direcory is not present, it creates it.
func UpdateVersions(repoDir, sourcePath, outputDir, libraryID, version string) error {
	pomFiles, err := findPomFiles(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to find pom files: %w", err)
	}
	for _, pomFile := range pomFiles {
		relPath, err := filepath.Rel(repoDir, pomFile)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", pomFile, err)
		}
		outputPomFile := filepath.Join(outputDir, relPath)
		if err := os.MkdirAll(filepath.Dir(outputPomFile), 0755); err != nil {
			return fmt.Errorf("failed to create output directory for %s: %w", outputPomFile, err)
		}
		if err := updateVersion(pomFile, outputPomFile, libraryID, version); err != nil {
			return fmt.Errorf("failed to update version in %s: %w", pomFile, err)
		}
	}
	return nil
}

// updateVersion updates the version in a single pom.xml file.
// The directory for outputPath must already exist.
func updateVersion(inputPath, outputPath, libraryID, version string) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	newContent := versionRegex.ReplaceAllStringFunc(string(content), func(s string) string {
		matches := versionRegex.FindStringSubmatch(s)
		if len(matches) > 4 && matches[4] == libraryID {
			// matches[1] is "<version>"
			// matches[2] is the old version
			// matches[3] is " <!-- {x-version-update:libraryID:current} --> </version>"
			// matches[4] is libraryID
			return fmt.Sprintf("%s%s%s", matches[1], version, matches[3])
		}
		return s
	})

	if err := os.WriteFile(outputPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func findPomFiles(path string) ([]string, error) {
	var pomFiles []string
	// Return empty if there's no matching directory.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []string{}, nil
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "pom.xml" {
			pomFiles = append(pomFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk path: %w", err)
	}
	return pomFiles, nil
}
