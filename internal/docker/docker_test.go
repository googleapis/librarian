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
	"encoding/json"
	"errors"
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
		testWorkRoot = "testWorkRoot"
		testImage    = "testImage"
		testUID      = "1000"
		testGID      = "1001"
	)
	d, err := New(testWorkRoot, testImage, testUID, testGID)
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
	if d.run == nil {
		t.Error("d.run is nil")
	}
}

func TestDockerRun(t *testing.T) {
	const (
		mockImage     = "mockImage"
		testAPIRoot   = "testAPIRoot"
		testImage     = "testImage"
		testLibraryID = "testLibraryID"
		testOutput    = "testOutput"
	)

	state := &config.LibrarianState{}
	cfg := &config.Config{}
	cfgInDocker := &config.Config{
		HostMount: "hostDir:localDir",
	}
	repoDir := filepath.Join(os.TempDir())
	for _, test := range []struct {
		name       string
		docker     *Docker
		runCommand func(ctx context.Context, d *Docker) error
		want       []string
		wantErr    bool
	}{
		{
			name: "Generate",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				generateRequest := &GenerateRequest{
					Cfg:       cfg,
					State:     state,
					RepoDir:   repoDir,
					ApiRoot:   testAPIRoot,
					Output:    testOutput,
					LibraryID: testLibraryID,
				}

				return d.Generate(ctx, generateRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian:ro", repoDir),
				"-v", fmt.Sprintf("%s/.librarian/generator-input:/input", repoDir),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				"-v", fmt.Sprintf("%s:/source:ro", testAPIRoot),
				testImage,
				string(CommandGenerate),
				"--librarian=/librarian",
				"--input=/input",
				"--output=/output",
				"--source=/source",
			},
			wantErr: false,
		},
		{
			name: "Generate with invalid repo root",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				generateRequest := &GenerateRequest{
					Cfg:       cfg,
					State:     state,
					RepoDir:   "/non-existed-dir",
					ApiRoot:   testAPIRoot,
					Output:    testOutput,
					LibraryID: testLibraryID,
				}
				return d.Generate(ctx, generateRequest)
			},
			want:    []string{},
			wantErr: true,
		},
		{
			name: "Generate with mock image",
			docker: &Docker{
				Image: mockImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				generateRequest := &GenerateRequest{
					Cfg:       cfg,
					State:     state,
					RepoDir:   repoDir,
					ApiRoot:   testAPIRoot,
					Output:    testOutput,
					LibraryID: testLibraryID,
				}

				return d.Generate(ctx, generateRequest)
			},
			want:    []string{},
			wantErr: true,
		},
		{
			name: "Generate runs in docker",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				generateRequest := &GenerateRequest{
					Cfg:       cfgInDocker,
					State:     state,
					RepoDir:   repoDir,
					ApiRoot:   testAPIRoot,
					Output:    "hostDir",
					LibraryID: testLibraryID,
				}

				return d.Generate(ctx, generateRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian:ro", repoDir),
				"-v", fmt.Sprintf("%s/.librarian/generator-input:/input", repoDir),
				"-v", "localDir:/output",
				"-v", fmt.Sprintf("%s:/source:ro", testAPIRoot),
				testImage,
				string(CommandGenerate),
				"--librarian=/librarian",
				"--input=/input",
				"--output=/output",
				"--source=/source",
			},
			wantErr: false,
		},
		{
			name: "Build",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				buildRequest := &BuildRequest{
					Cfg:       cfg,
					State:     state,
					LibraryID: testLibraryID,
					RepoDir:   repoDir,
				}

				return d.Build(ctx, buildRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian:ro", repoDir),
				"-v", fmt.Sprintf("%s:/repo", repoDir),
				testImage,
				string(CommandBuild),
				"--librarian=/librarian",
				"--repo=/repo",
			},
			wantErr: false,
		},
		{
			name: "Build with invalid repo dir",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				buildRequest := &BuildRequest{
					Cfg:       cfg,
					State:     state,
					LibraryID: testLibraryID,
					RepoDir:   "/non-exist-dir",
				}
				return d.Build(ctx, buildRequest)
			},
			want:    []string{},
			wantErr: true,
		},
		{
			name: "Build with mock image",
			docker: &Docker{
				Image: mockImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				buildRequest := &BuildRequest{
					Cfg:       cfg,
					State:     state,
					LibraryID: testLibraryID,
					RepoDir:   repoDir,
				}

				return d.Build(ctx, buildRequest)
			},
			want:    []string{},
			wantErr: true,
		},
		{
			name: "Configure",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				configureRequest := &ConfigureRequest{
					Cfg:       cfg,
					State:     state,
					LibraryID: testLibraryID,
					RepoDir:   repoDir,
					ApiRoot:   testAPIRoot,
				}
				jsonData, _ := json.MarshalIndent(&config.LibraryState{}, "", "  ")
				if err := os.MkdirAll(filepath.Join(configureRequest.RepoDir, config.LibrarianDir), 0755); err != nil {
					return err
				}
				jsonFilePath := filepath.Join(configureRequest.RepoDir, config.LibrarianDir, config.ConfigureResponse)
				if err := os.WriteFile(jsonFilePath, jsonData, 0644); err != nil {
					return err
				}

				_, err := d.Configure(ctx, configureRequest)

				return err
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian", repoDir),
				"-v", fmt.Sprintf("%s/.librarian/generator-input:/input", repoDir),
				"-v", fmt.Sprintf("%s:/source:ro", testAPIRoot),
				testImage,
				string(CommandConfigure),
				"--librarian=/librarian",
				"--input=/input",
				"--source=/source",
			},
			wantErr: false,
		},
		{
			name: "Configure with multiple libraries in librarian state",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				curState := &config.LibrarianState{
					Image: testImage,
					Libraries: []*config.LibraryState{
						{
							ID: testLibraryID,
							APIs: []*config.API{
								{
									Path: "example/path/v1",
								},
							},
						},
						{
							ID: "another-example-library",
							APIs: []*config.API{
								{
									Path:          "another/example/path/v1",
									ServiceConfig: "another_v1.yaml",
								},
							},
							SourcePaths: []string{
								"another-example-source-path",
							},
						},
					},
				}
				configureRequest := &ConfigureRequest{
					Cfg:       cfg,
					State:     curState,
					LibraryID: testLibraryID,
					RepoDir:   repoDir,
					ApiRoot:   testAPIRoot,
				}
				jsonData, _ := json.MarshalIndent(&config.LibraryState{
					ID: testLibraryID,
					APIs: []*config.API{
						{
							Path:          "example/path/v1",
							ServiceConfig: "generated_example_v1.yaml",
						},
					},
				}, "", "  ")

				if err := os.MkdirAll(filepath.Join(configureRequest.RepoDir, config.LibrarianDir), 0755); err != nil {
					return err
				}

				jsonFilePath := filepath.Join(configureRequest.RepoDir, config.LibrarianDir, config.ConfigureResponse)
				if err := os.WriteFile(jsonFilePath, jsonData, 0644); err != nil {
					return err
				}

				configuredLibrary, err := d.Configure(ctx, configureRequest)
				if configuredLibrary != testLibraryID {
					return errors.New("configured library, " + configuredLibrary + " is wrong")
				}

				return err
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian", repoDir),
				"-v", fmt.Sprintf("%s/.librarian/generator-input:/input", repoDir),
				"-v", fmt.Sprintf("%s:/source:ro", testAPIRoot),
				testImage,
				string(CommandConfigure),
				"--librarian=/librarian",
				"--input=/input",
				"--source=/source",
			},
			wantErr: false,
		},
		{
			name: "Configure with invalid repo dir",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				configureRequest := &ConfigureRequest{
					Cfg:       cfg,
					State:     state,
					LibraryID: testLibraryID,
					RepoDir:   "/non-exist-dir",
					ApiRoot:   testAPIRoot,
				}
				_, err := d.Configure(ctx, configureRequest)
				return err
			},
			want:    []string{},
			wantErr: true,
		},
		{
			name: "Configure with mock image",
			docker: &Docker{
				Image: mockImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				configureRequest := &ConfigureRequest{
					Cfg:       cfg,
					State:     state,
					LibraryID: testLibraryID,
					RepoDir:   repoDir,
					ApiRoot:   testAPIRoot,
				}

				_, err := d.Configure(ctx, configureRequest)

				return err
			},
			want:    []string{},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.docker.run = func(args ...string) error {
				if test.docker.Image == mockImage {
					return errors.New("simulate docker command failure for testing")
				}
				if diff := cmp.Diff(test.want, args); diff != "" {
					t.Errorf("mismatch(-want +got):\n%s", diff)
				}
				return nil
			}
			ctx := t.Context()
			err := test.runCommand(ctx, test.docker)

			if test.wantErr {
				if err == nil {
					t.Errorf("%s should return error", test.name)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestToGenerateRequestJSON(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		state     *config.LibrarianState
		expectErr bool
	}{
		{
			name: "successful-marshaling-and-writing",
			state: &config.LibrarianState{
				Image: "v1.0.0",
				Libraries: []*config.LibraryState{
					{
						ID:                  "google-cloud-go",
						Version:             "1.0.0",
						LastGeneratedCommit: "abcd123",
						APIs: []*config.API{
							{
								Path:          "google/cloud/compute/v1",
								ServiceConfig: "example_service_config.yaml",
							},
						},
						SourcePaths: []string{
							"src/example/path",
						},
						PreserveRegex: []string{
							"example-preserve-regex",
						},
						RemoveRegex: []string{
							"example-remove-regex",
						},
					},
					{
						ID:      "google-cloud-storage",
						Version: "1.2.3",
						APIs: []*config.API{
							{
								Path:          "google/storage/v1",
								ServiceConfig: "storage_service_config.yaml",
							},
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name:      "empty-pipelineState",
			state:     &config.LibrarianState{},
			expectErr: false,
		},
		{
			name:      "nonexistent_dir_for_test",
			state:     &config.LibrarianState{},
			expectErr: true,
		},
		{
			name:      "invalid_file_name",
			state:     &config.LibrarianState{},
			expectErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if test.name == "invalid_file_name" {
				filePath := filepath.Join(tempDir, "my\x00file.json")
				err := writeRequest(test.state, "google-cloud-go", filePath)
				if err == nil {
					t.Errorf("writeGenerateRequest() expected an error but got nil")
				}
				return
			} else if test.expectErr {
				filePath := filepath.Join("/non-exist-dir", "generate-request.json")
				err := writeRequest(test.state, "google-cloud-go", filePath)
				if err == nil {
					t.Errorf("writeGenerateRequest() expected an error but got nil")
				}
				return
			}

			filePath := filepath.Join(tempDir, "generate-request.json")
			err := writeRequest(test.state, "google-cloud-go", filePath)

			if err != nil {
				t.Fatalf("writeGenerateRequest() unexpected error: %v", err)
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
