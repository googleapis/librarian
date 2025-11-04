// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
		version     string
		expected    string
		expectError bool
	}{
		{
			name:      "happy path",
			libraryID: "google-cloud-foo",
			version:   "2.0.0",
			expected:  "<version>2.0.0</version><!-- {x-version-update:google-cloud-java:current} -->",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			inputPath := filepath.Join("testdata", "java-foo")

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
							ID:      test.libraryID,
							Version: test.version,
						},
					},
				},
			}

			response, err := Stage(context.Background(), cfg)
			if err != nil {
				if !test.expectError {
					t.Fatalf("Stage() got unexpected error: %v", err)
				}
			}

			if test.expectError {
				if response.Error == "" {
					t.Errorf("expected error, got success")
				}
			} else {
				if response.Error != "" {
					t.Errorf("expected success, got error: %s", response.Error)
				}
				content, err := os.ReadFile(filepath.Join(outputDir, "pom.xml"))
				if err != nil {
					t.Fatalf("failed to read output file: %v", err)
				}
				hasExpectedVersion := strings.Contains(string(content), "<version>"+test.version+"</version>")
				hasAnnotation := strings.Contains(string(content), "<!-- {x-version-update:google-cloud-foo:current} -->")
				if !hasExpectedVersion || !hasAnnotation {
					t.Errorf("expected file to contain version %q and comment, got %q", test.version, string(content))
				}
			}
		})
	}
}
