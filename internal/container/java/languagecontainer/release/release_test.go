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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/container/java/message"
)

func TestReadReleaseStageRequest(t *testing.T) {
	want := &message.ReleaseStageRequest{
		Libraries: []*message.Library{
			{
				ID:      "google-cloud-secretmanager-v1",
				Version: "1.3.0",
				Changes: []*message.Change{
					{
						Type:          "feat",
						Subject:       "add new UpdateRepository API",
						Body:          "This adds the ability to update a repository's properties.",
						PiperCLNumber: "786353207",
						CommitHash:    "9461532e7d19c8d71709ec3b502e5d81340fb661",
					},
					{
						Type:          "docs",
						Subject:       "fix typo in BranchRule comment",
						Body:          "",
						PiperCLNumber: "786353207",
						CommitHash:    "9461532e7d19c8d71709ec3b502e5d81340fb661",
					},
				},
				APIs: []message.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
					{
						Path: "google/cloud/secretmanager/v1beta",
					},
				},
				SourcePaths: []string{
					"secretmanager",
					"other/location/secretmanager",
				},
				ReleaseTriggered: true,
			},
		},
	}
	bytes, err := os.ReadFile(filepath.Join("..", "testdata", "release-stage-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	got := &message.ReleaseStageRequest{}
	if err := json.Unmarshal(bytes, got); err != nil {
		t.Fatal(err)
	}
	// We can't compare the entire struct because the testdata file has more fields
	// than the want struct. Instead, we'll just compare the fields we care about.
	if len(got.Libraries) != 1 {
		t.Fatalf("got %d libraries, want %d", len(got.Libraries), 1)
	}
	if diff := cmp.Diff(want.Libraries[0], got.Libraries[0]); diff != "" {
		t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
	}
}

func TestReleaseStage(t *testing.T) {
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
			expected:  "    <version>2.0.0<!-- {x-version-update:google-cloud-java:current} --> </version>",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			// Copy the testdata pom.xml to the temporary directory.
			inputPath := filepath.Join("testdata", "pom.xml")
			outputPath := filepath.Join(tmpDir, "pom.xml")
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("failed to read input file: %v", err)
			}
			if err := os.WriteFile(outputPath, input, 0644); err != nil {
				t.Fatalf("failed to write output file: %v", err)
			}

			request := &message.ReleaseStageRequest{
				Libraries: []*message.Library{
					{
						ID:      test.libraryID,
						Version: test.version,
					},
				},
			}
			response := &message.ReleaseStageResponse{}

			// Change the current working directory to the temporary directory.
			// This is important because UpdateVersions walks the current directory.
			t.Chdir(tmpDir)

			ReleaseStage(request, response)

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
				if !strings.Contains(string(content), test.expected) {
					t.Errorf("expected file to contain %q, got %q", test.expected, string(content))
				}
			}
		})
	}
}
