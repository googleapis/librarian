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

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestToGenerateRequestJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		state     *config.PipelineState
		expectErr bool
	}{
		{
			name: "successful-marshaling-and-writing",
			state: &config.PipelineState{
				ImageTag: "v1.0.0",
				Libraries: []*config.LibraryState{
					{
						ID:                        "google-cloud-go",
						CurrentVersion:            "1.0.0",
						GenerationAutomationLevel: config.AutomationLevelAutomatic,
						APIPaths:                  []string{"google/cloud/compute/v1"},
					},
					{
						ID:                        "google-cloud-storage",
						CurrentVersion:            "1.2.3",
						GenerationAutomationLevel: config.AutomationLevelManualReview,
						APIPaths:                  []string{"google/storage/v1"},
					},
				},
				IgnoredAPIPaths: []string{"google/cloud/ignored/v1"},
			},
			expectErr: false,
		},
		{
			name:      "empty-pipelineState",
			state:     &config.PipelineState{},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "generate-request.json")

			err := toGenerateRequestJSON(tc.state, filePath)

			if tc.expectErr {
				if err == nil {
					t.Errorf("toGenerateRequestJSON() expected an error but got nil")
				}
				return // Test case expects error, so no further checks
			}

			if err != nil {
				t.Fatalf("toGenerateRequestJSON() unexpected error: %v", err)
			}

			// Verify the file content
			gotBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			fileName := fmt.Sprintf("%s.json", tc.name)
			wantBytes, readErr := os.ReadFile(filepath.Join("..", "..", "testdata", fileName))
			if readErr != nil {
				t.Fatalf("Failed to read expected state for comparison: %v", readErr)
			}

			if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
				t.Errorf("Generated JSON mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("Error creating file (invalid path)", func(t *testing.T) {
		t.Parallel()
		// Attempt to write to a path where we don't have permissions or a non-existent dir
		invalidPath := filepath.Join("/nonexistent_dir_for_test", "generate-request.json")
		state := &config.PipelineState{ImageTag: "test"}
		err := toGenerateRequestJSON(state, invalidPath)
		if err == nil {
			t.Error("toGenerateRequestJSON() expected an error for invalid path, got nil")
		}
	})
}
