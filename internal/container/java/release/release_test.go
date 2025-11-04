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
			libraryID: "google-cloud-java",
			version:   "2.0.0",
			expected:  "    <version>2.0.0</version><!-- {x-version-update:google-cloud-java:current} -->",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			// Copy the testdata pom.xml to the temporary directory.
			inputPath := filepath.Join("..", "languagecontainer", "release", "testdata", "pom.xml")
			// The testdata directory does not exist for the new test file, so we need to create it.
			if err := os.MkdirAll(filepath.Dir(inputPath), 0755); err != nil {
				t.Fatalf("failed to create testdata directory: %v", err)
			}
			outputPath := filepath.Join(tmpDir, "pom.xml")
			// This is a simple pom.xml that is sufficient for this test.
			pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.google.cloud</groupId>
  <artifactId>google-cloud-java</artifactId>
  <version>1.0.0</version><!-- {x-version-update:google-cloud-java:current} -->
  <packaging>pom</packaging>
</project>
`
			if err := os.WriteFile(outputPath, []byte(pomContent), 0644); err != nil {
				t.Fatalf("failed to write output file: %v", err)
			}

			cfg := &release.Config{
				Context: &release.Context{
					RepoDir: tmpDir,
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
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("failed to read output file: %v", err)
				}
				if !strings.Contains(string(content), "<version>"+test.version+"</version>") || !strings.Contains(string(content), "<!-- {x-version-update:google-cloud-java:current} -->") {
					t.Errorf("expected file to contain version %q and comment, got %q", test.version, string(content))
				}
			}
		})
	}
}
