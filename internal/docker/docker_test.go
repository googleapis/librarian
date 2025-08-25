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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
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
		mockImage            = "mockImage"
		testAPIRoot          = "testAPIRoot"
		testImage            = "testImage"
		testLibraryID        = "testLibraryID"
		testOutput           = "testOutput"
		simulateDockerErrMsg = "simulate docker command failure for testing"
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
		wantErrMsg string
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
				"-v", fmt.Sprintf("%s/.librarian:/librarian", repoDir),
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
			want:       []string{},
			wantErr:    true,
			wantErrMsg: "failed to make directory",
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
			want:       []string{},
			wantErr:    true,
			wantErrMsg: simulateDockerErrMsg,
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
				"-v", fmt.Sprintf("%s/.librarian:/librarian", repoDir),
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
				"-v", fmt.Sprintf("%s/.librarian:/librarian", repoDir),
				"-v", fmt.Sprintf("%s:/repo", repoDir),
				testImage,
				string(CommandBuild),
				"--librarian=/librarian",
				"--repo=/repo",
			},
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
			want:       []string{},
			wantErr:    true,
			wantErrMsg: "failed to make directory",
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
			want:       []string{},
			wantErr:    true,
			wantErrMsg: simulateDockerErrMsg,
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
							SourceRoots: []string{
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
			want:       []string{},
			wantErr:    true,
			wantErrMsg: "failed to make directory",
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
			want:       []string{},
			wantErr:    true,
			wantErrMsg: simulateDockerErrMsg,
		},
		{
			name: "Release init for all libraries",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				partialRepoDir := filepath.Join(repoDir, "release-init-all-libraries")
				if err := os.MkdirAll(filepath.Join(repoDir, config.LibrarianDir), 0755); err != nil {
					t.Fatal(err)
				}

				releaseInitRequest := &ReleaseInitRequest{
					Cfg: &config.Config{
						Repo: repoDir,
					},
					State:           state,
					Output:          testOutput,
					LibrarianConfig: &config.LibrarianConfig{},
					PartialRepoDir:  partialRepoDir,
				}

				defer os.RemoveAll(partialRepoDir)

				return d.ReleaseInit(ctx, releaseInitRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian", filepath.Join(repoDir, "release-init-all-libraries")),
				"-v", fmt.Sprintf("%s:/repo:ro", filepath.Join(repoDir, "release-init-all-libraries")),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				testImage,
				string(CommandReleaseInit),
				"--librarian=/librarian",
				"--repo=/repo",
				"--output=/output",
			},
		},
		{
			name: "Release init returns error",
			docker: &Docker{
				Image: mockImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				partialRepoDir := filepath.Join(repoDir, "release-init-returns-error")
				if err := os.MkdirAll(filepath.Join(repoDir, config.LibrarianDir), 0755); err != nil {
					t.Fatal(err)
				}

				releaseInitRequest := &ReleaseInitRequest{
					Cfg: &config.Config{
						Repo: repoDir,
					},
					State:           state,
					PartialRepoDir:  partialRepoDir,
					Output:          testOutput,
					LibrarianConfig: &config.LibrarianConfig{},
				}
				defer os.RemoveAll(partialRepoDir)

				return d.ReleaseInit(ctx, releaseInitRequest)
			},
			wantErr:    true,
			wantErrMsg: simulateDockerErrMsg,
		},
		{
			name: "Release init with invalid partial repo dir",
			docker: &Docker{
				Image: mockImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				releaseInitRequest := &ReleaseInitRequest{
					Cfg: &config.Config{
						Repo: os.TempDir(),
					},
					State:          state,
					PartialRepoDir: "/non-exist-dir",
					Output:         testOutput,
				}

				return d.ReleaseInit(ctx, releaseInitRequest)
			},
			wantErr:    true,
			wantErrMsg: "failed to make directory",
		},
		{
			name: "Release init for one library",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				partialRepoDir := filepath.Join(repoDir, "release-init-one-library")
				if err := os.MkdirAll(filepath.Join(repoDir, config.LibrarianDir), 0755); err != nil {
					t.Fatal(err)
				}
				releaseInitRequest := &ReleaseInitRequest{
					Cfg: &config.Config{
						Repo: repoDir,
					},
					State:           state,
					PartialRepoDir:  partialRepoDir,
					Output:          testOutput,
					LibraryID:       testLibraryID,
					LibrarianConfig: &config.LibrarianConfig{},
				}
				defer os.RemoveAll(partialRepoDir)

				return d.ReleaseInit(ctx, releaseInitRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian", filepath.Join(repoDir, "release-init-one-library")),
				"-v", fmt.Sprintf("%s:/repo:ro", filepath.Join(repoDir, "release-init-one-library")),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				testImage,
				string(CommandReleaseInit),
				"--librarian=/librarian",
				"--repo=/repo",
				"--output=/output",
				fmt.Sprintf("--library=%s", testLibraryID),
			},
		},
		{
			name: "Release init for one library with version",
			docker: &Docker{
				Image: testImage,
			},
			runCommand: func(ctx context.Context, d *Docker) error {
				partialRepoDir := filepath.Join(repoDir, "release-init-one-library-with-version")
				if err := os.MkdirAll(filepath.Join(repoDir, config.LibrarianDir), 0755); err != nil {
					t.Fatal(err)
				}

				releaseInitRequest := &ReleaseInitRequest{
					Cfg: &config.Config{
						Repo: os.TempDir(),
					},
					State:           state,
					PartialRepoDir:  partialRepoDir,
					Output:          testOutput,
					LibraryID:       testLibraryID,
					LibraryVersion:  "1.2.3",
					LibrarianConfig: &config.LibrarianConfig{},
				}
				defer os.RemoveAll(partialRepoDir)

				return d.ReleaseInit(ctx, releaseInitRequest)
			},
			want: []string{
				"run", "--rm",
				"-v", fmt.Sprintf("%s/.librarian:/librarian", filepath.Join(repoDir, "release-init-one-library-with-version")),
				"-v", fmt.Sprintf("%s:/repo:ro", filepath.Join(repoDir, "release-init-one-library-with-version")),
				"-v", fmt.Sprintf("%s:/output", testOutput),
				testImage,
				string(CommandReleaseInit),
				"--librarian=/librarian",
				"--repo=/repo",
				"--output=/output",
				fmt.Sprintf("--library=%s", testLibraryID),
				"--library-version=1.2.3",
			},
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
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWriteLibraryState(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		state      *config.LibrarianState
		path       string
		filename   string
		wantFile   string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "write library state to file",
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
								Status:        "new",
							},
						},
						SourceRoots: []string{
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
								Status:        "existing",
							},
						},
					},
				},
			},
			path:     os.TempDir(),
			filename: "a-library-example.json",
			wantFile: "successful-marshaling-and-writing.json",
		},
		{
			name:     "empty library state",
			state:    &config.LibrarianState{},
			path:     os.TempDir(),
			filename: "another-library-example.json",
			wantFile: "empty-library-state.json",
		},
		{
			name:       "nonexistent directory",
			state:      &config.LibrarianState{},
			path:       "/nonexistent_dir_for_test",
			filename:   "example.json",
			wantErr:    true,
			wantErrMsg: "failed to make directory",
		},
		{
			name:       "invalid file name",
			state:      &config.LibrarianState{},
			path:       os.TempDir(),
			filename:   "my\u0000file.json",
			wantErr:    true,
			wantErrMsg: "failed to create JSON file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			filePath := filepath.Join(test.path, test.filename)
			err := writeLibraryState(test.state, "google-cloud-go", filePath)

			if test.wantErr {
				if err == nil {
					t.Errorf("writeLibraryState() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}

			// Verify the file content
			gotBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			wantBytes, readErr := os.ReadFile(filepath.Join("../..", "testdata", "test-write-library-state", test.wantFile))
			if readErr != nil {
				t.Fatalf("Failed to read expected state for comparison: %v", readErr)
			}

			if diff := cmp.Diff(strings.TrimSpace(string(wantBytes)), string(gotBytes)); diff != "" {
				t.Errorf("Generated JSON mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteLibrarianState(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		state      *config.LibrarianState
		path       string
		filename   string
		wantFile   string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "write to a json file",
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
								Status:        "existing",
							},
						},
						SourceRoots: []string{
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
								Status:        "existing",
							},
						},
					},
				},
			},
			path:     os.TempDir(),
			filename: "a-librarian-example.json",
			wantFile: "write-librarian-state-example.json",
		},
		{
			name:     "empty librarian state",
			state:    &config.LibrarianState{},
			path:     os.TempDir(),
			filename: "another-librarian-example.json",
			wantFile: "empty-librarian-state.json",
		},
		{
			name:       "nonexistent directory",
			state:      &config.LibrarianState{},
			path:       "/nonexistent_dir_for_test",
			filename:   "example.json",
			wantErr:    true,
			wantErrMsg: "failed to make directory",
		},
		{
			name:       "invalid file name",
			state:      &config.LibrarianState{},
			path:       os.TempDir(),
			filename:   "my\u0000file.json",
			wantErr:    true,
			wantErrMsg: "failed to create JSON file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			filePath := filepath.Join(test.path, test.filename)
			err := writeLibrarianState(test.state, filePath)
			if test.wantErr {
				if err == nil {
					t.Errorf("writeLibrarianState() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}

			// Verify the file content
			gotBytes, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			wantBytes, readErr := os.ReadFile(filepath.Join("../..", "testdata", "test-write-librarian-state", test.wantFile))
			if readErr != nil {
				t.Fatalf("Failed to read expected state for comparison: %v", readErr)
			}

			if diff := cmp.Diff(strings.TrimSpace(string(wantBytes)), string(gotBytes)); diff != "" {
				t.Errorf("Generated JSON mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDocker_runCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmdName string
		args    []string
		wantErr bool
	}{
		{
			name:    "success",
			cmdName: "echo",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "failure",
			cmdName: "some-non-existent-command",
			args:    []string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Docker{}
			if err := c.runCommand(tt.cmdName, tt.args...); (err != nil) != tt.wantErr {
				t.Errorf("Docker.runCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
