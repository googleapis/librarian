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

package release

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/container/java/languagecontainer/release"
	"github.com/googleapis/librarian/internal/container/java/message"
)

func TestStage(t *testing.T) {
	tests := []struct {
		name        string
		libraryID   string
		SourcePaths []string
		version     string
		expected    string
	}{
		{
			name:      "happy path",
			libraryID: "google-cloud-foo",
			SourcePaths: []string{
				"java-foo",
			},
			version:  "2.0.0",
			expected: "<version>2.0.0-SNAPSHOT</version><!-- {x-version-update:google-cloud-foo:current} -->",
		},
		{
			name:      "Source Paths not matching the folder",
			libraryID: "google-cloud-java",
			SourcePaths: []string{
				"java-nonexistent",
			},
			version: "2.0.0",
			// Do not expect the files updated since the source path does not exist.
			expected: "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			// This testdata is the dummy repository root.
			inputPath := filepath.Join("testdata")

			tmpDir := t.TempDir()
			outputDir := filepath.Join(tmpDir, "output")
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				t.Fatalf("failed to create output directory: %v", err)
			}
			cfg := &release.Config{
				Context: &release.Context{
					RepoDir:   inputPath,
					OutputDir: outputDir,
				},
				Request: &message.ReleaseStageRequest{
					Libraries: []*message.Library{
						{
							ID:          test.libraryID,
							Version:     test.version,
							SourcePaths: test.SourcePaths,
						},
					},
				},
			}

			response, err := Stage(context.Background(), cfg)
			if err != nil {
				t.Fatalf("Stage() got unexpected error: %v", err)
			}

			if response.Error != "" {
				t.Errorf("expected success, got error: %s", response.Error)
			}
			if test.expected != "" {
				// The file paths are relative to the repoDir.
				for _, file := range []string{"java-foo/pom.xml", "java-foo/google-cloud-foo/pom.xml"} {
					content, err := os.ReadFile(filepath.Join(outputDir, file))
					if err != nil {
						t.Fatalf("failed to read output file: %v", err)
					}

					hasExpectedVersionLineWithAnnotation := strings.Contains(string(content), test.expected)
					if !hasExpectedVersionLineWithAnnotation {
						t.Errorf("expected file to contain annotation %q and comment, got %q", test.expected, string(content))
					}
				}
			} else {
				// Expect no files in the output directory because this operation
				// does not change any files in the repodir.
				entries, err := os.ReadDir(outputDir)
				if err != nil {
					t.Fatalf("failed to read output directory: %v", err)
				}
				if len(entries) != 0 {
					t.Errorf("expected no files in output directory, got %d files", len(entries))
				}
			}
		})
	}
}
