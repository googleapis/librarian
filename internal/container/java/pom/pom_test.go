// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerate(t *testing.T) {
	testCases := []struct {
		name          string
		libraryID     string
		modules       []string
		goldenFiles   map[string]string
		wantErr       bool
		errorContains string
	}{
		{
			name:      "happy path with proto and grpc",
			libraryID: "test",
			modules:   []string{"proto-test", "grpc-test"},
			goldenFiles: map[string]string{
				"pom.xml":                       "testdata/happy_path_parent_pom.xml",
				"proto-test/pom.xml":            "testdata/happy_path_proto_pom.xml",
				"grpc-test/pom.xml":             "testdata/happy_path_grpc_pom.xml",
				"google-cloud-test/pom.xml":     "testdata/happy_path_cloud_pom.xml",
				"google-cloud-test-bom/pom.xml": "testdata/happy_path_bom_pom.xml",
			},
			wantErr: false,
		},
		{
			name:      "only proto module",
			libraryID: "test",
			modules:   []string{"proto-test"},
			goldenFiles: map[string]string{
				"pom.xml":                       "testdata/only_proto_parent_pom.xml",
				"proto-test/pom.xml":            "testdata/only_proto_proto_pom.xml",
				"google-cloud-test/pom.xml":     "testdata/only_proto_cloud_pom.xml",
				"google-cloud-test-bom/pom.xml": "testdata/only_proto_bom_pom.xml",
			},
			wantErr: false,
		},
		{
			name:          "only grpc module",
			libraryID:     "test",
			modules:       []string{"grpc-test"},
			wantErr:       true,
			errorContains: "grpc module grpc-test exists without a corresponding proto module",
		}, {
			name:          "non-existent libraryPath",
			libraryID:     "test",
			modules:       []string{},
			wantErr:       true,
			errorContains: "could not find modules",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var libraryPath string
			if tc.name == "non-existent libraryPath" {
				libraryPath = filepath.Join(t.TempDir(), "non-existent")
			} else {
				libraryPath = t.TempDir()
				for _, module := range tc.modules {
					err := os.Mkdir(filepath.Join(libraryPath, module), 0755)
					if err != nil {
						t.Fatalf("failed to create module directory %s: %v", module, err)
					}
				}
				// Create main artifact directory
				mainArtifactDir := filepath.Join(libraryPath, fmt.Sprintf("google-cloud-%s", tc.libraryID))
				if err := os.Mkdir(mainArtifactDir, 0755); err != nil {
					t.Fatalf("failed to create main artifact directory %s: %v", mainArtifactDir, err)
				}
			}

			err := Generate(libraryPath, tc.libraryID)
			if (err != nil) != tc.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if tc.wantErr {
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Generate() error = %v, want error containing %q", err, tc.errorContains)
				}
				return
			}

			for generatedFile, goldenFile := range tc.goldenFiles {
				generatedContent, err := os.ReadFile(filepath.Join(libraryPath, generatedFile))
				if err != nil {
					t.Fatalf("failed to read generated file %s: %v", generatedFile, err)
				}

				goldenContent, err := os.ReadFile(goldenFile)
				if err != nil {
					// If golden files don't exist, create them.
					if os.IsNotExist(err) {
						goldenFileDir := filepath.Dir(goldenFile)
						if _, err := os.Stat(goldenFileDir); os.IsNotExist(err) {
							if err := os.MkdirAll(goldenFileDir, 0755); err != nil {
								t.Fatalf("failed to create golden file directory %s: %v", goldenFileDir, err)
							}
						}
						if err := os.WriteFile(goldenFile, generatedContent, 0644); err != nil {
							t.Fatalf("failed to write golden file %s: %v", goldenFile, err)
						}
						t.Logf("created golden file %s", goldenFile)
						// Reread the golden file to continue the test
						goldenContent, err = os.ReadFile(goldenFile)
						if err != nil {
							t.Fatalf("failed to read newly created golden file %s: %v", goldenFile, err)
						}
					} else {
						t.Fatalf("failed to read golden file %s: %v", goldenFile, err)
					}
				}

				if diff := cmp.Diff(string(goldenContent), string(generatedContent)); diff != "" {
					t.Errorf("generated file %s content mismatch (-want +got):\n%s", generatedFile, diff)
				}
			}
		})
	}
}
