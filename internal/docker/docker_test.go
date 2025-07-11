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

package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"

	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	const (
		testWorkRoot       = "testWorkRoot"
		testImage          = "testImage"
		testSecretsProject = "testSecretsProject"
		testUID            = "1000"
		testGID            = "1001"
	)
	pipelineConfig := &config.PipelineConfig{}
	d, err := New(testWorkRoot, testImage, testSecretsProject, testUID, testGID, pipelineConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if d.Image != testImage {
		t.Errorf("d.Image = %q, want %q", d.Image, testImage)
	}
	if d.uid != testUID {
		t.Errorf("d.uid = %q, want %q", d.uid, testUID)
	}
	if d.gid != testGID {
		t.Errorf("d.gid = %q, want %q", d.gid, testGID)
	}
	if d.env == nil {
		t.Error("d.env is nil")
	}
	if d.run == nil {
		t.Error("d.run is nil")
	}
}

func TestDockerRun(t *testing.T) {
	const (
		testAPIPath         = "testAPIPath"
		testAPIRoot         = "testAPIRoot"
		testGenerateRequest = "testGenerateRequest"
		testGeneratorInput  = "testGeneratorInput"
		testImage           = "testImage"
		testLibraryID       = "testLibraryID"
		testOutput          = "testOutput"
		testRepoRoot        = "testRepoRoot"
	)

	state := &config.PipelineState{}
	cfg := &config.Config{}
	cfgInDocker := &config.Config{
		HostMount: "hostDir:localDir",
	}
	for _, test := range []struct {
		name       string
		docker     *Docker
		runCommand func(ctx context.Context, d *Docker) error
		want       []string
	}{
		{
			name: "Generate",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				generateRequest := NewGenerateRequest(cfgInDocker, state, testAPIRoot, testOutput, testGenerateRequest, testGeneratorInput, testLibraryID)
				return d.Generate(ctx, generateRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/librarian:ro", testGenerateRequest),
				"-v", fmt.Sprintf("%s:/input", testGeneratorInput),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				"-v", fmt.Sprintf("%s:/source:ro", testAPIRoot),
				testImage,
				string(CommandGenerate),
				"--librarian=/librarian",
				"--input=/input",
				"--output=/output",
				"--source=/source",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "Generate runs in docker",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				generateRequest := NewGenerateRequest(cfgInDocker, state, testAPIRoot, "hostDir", testGenerateRequest, testGeneratorInput, testLibraryID)
				return d.Generate(ctx, generateRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/librarian:ro", testGenerateRequest),
				"-v", fmt.Sprintf("%s:/input", testGeneratorInput),
				"-v", "localDir:/output",
				"-v", fmt.Sprintf("%s:/source:ro", testAPIRoot),
				testImage,
				string(CommandGenerate),
				"--librarian=/librarian",
				"--input=/input",
				"--output=/output",
				"--source=/source",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "Build",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.Build(ctx, cfg, testRepoRoot, testLibraryID)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/repo", testRepoRoot),
				testImage,
				string(CommandBuild),
				"--repo-root=/repo",
				"--test=true",
				fmt.Sprintf("--library-id=%s", testLibraryID),
			},
		},
		{
			name: "Configure",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				return d.Configure(ctx, cfg, testAPIRoot, testAPIPath, testGeneratorInput)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s:/apis", testAPIRoot),
				"-v", fmt.Sprintf("%s:/.librarian/generator-input", testGeneratorInput),
				testImage,
				string(CommandConfigure),
				"--source=/apis",
				"--.librarian/generator-input=/.librarian/generator-input",
				fmt.Sprintf("--api=%s", testAPIPath),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.docker.run = func(args ...string) error {
				if diff := cmp.Diff(test.want, args); diff != "" {
					t.Errorf("mismatch(-want +got):\n%s", diff)
				}
				return nil
			}
			ctx := t.Context()
			if err := test.runCommand(ctx, test.docker); err != nil {
				t.Fatal(err)
			}
			os.Remove(testGenerateRequest)
		})
	}
}

func TestToGenerateRequestJSON(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
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
		{
			name:      "nonexistent_dir_for_test",
			state:     &config.PipelineState{},
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.expectErr {
				filePath := filepath.Join("/non-exist-dir", "generate-request.json")
				err := toGenerateRequestJSON(test.state, filePath)
				if err == nil {
					t.Errorf("toGenerateRequestJSON() expected an error but got nil")
				}
				return
			}

			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "generate-request.json")
			err := toGenerateRequestJSON(test.state, filePath)

			if err != nil {
				t.Fatalf("toGenerateRequestJSON() unexpected error: %v", err)
			}

			// Verify the file content
			gotBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			fileName := fmt.Sprintf("%s.json", test.name)
			wantBytes, readErr := os.ReadFile(filepath.Join("..", "..", "testdata", fileName))
			if readErr != nil {
				t.Fatalf("Failed to read expected state for comparison: %v", readErr)
			}

			if diff := cmp.Diff(strings.TrimSpace(string(wantBytes)), string(gotBytes)); diff != "" {
				t.Errorf("Generated JSON mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
