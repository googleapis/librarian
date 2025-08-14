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
	"time"

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
				"-v", fmt.Sprintf("%s/.librarian:/librarian:ro", repoDir),
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

				releaseInitRequest := &ReleaseRequest{
					Cfg: &config.Config{
						Repo: repoDir,
					},
					State:          state,
					Output:         testOutput,
					GlobalConfig:   &config.GlobalConfig{},
					partialRepoDir: partialRepoDir,
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

				releaseInitRequest := &ReleaseRequest{
					Cfg: &config.Config{
						Repo: repoDir,
					},
					State:          state,
					partialRepoDir: partialRepoDir,
					Output:         testOutput,
					GlobalConfig:   &config.GlobalConfig{},
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
				releaseInitRequest := &ReleaseRequest{
					Cfg: &config.Config{
						Repo: os.TempDir(),
					},
					State:          state,
					partialRepoDir: "/non-exist-dir",
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
				releaseInitRequest := &ReleaseRequest{
					Cfg: &config.Config{
						Repo: repoDir,
					},
					State:          state,
					partialRepoDir: partialRepoDir,
					Output:         testOutput,
					LibraryID:      testLibraryID,
					GlobalConfig:   &config.GlobalConfig{},
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

				releaseInitRequest := &ReleaseRequest{
					Cfg: &config.Config{
						Repo: os.TempDir(),
					},
					State:          state,
					partialRepoDir: partialRepoDir,
					Output:         testOutput,
					LibraryID:      testLibraryID,
					LibraryVersion: "1.2.3",
					GlobalConfig:   &config.GlobalConfig{},
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

func TestPartialCopyRepo(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		request       *ReleaseRequest
		includedFiles []string
		excludedFiles []string
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name: "copy all libraries and required files to partial repo",
			request: &ReleaseRequest{
				Cfg: &config.Config{
					Repo: filepath.Join(os.TempDir(), "repo"),
				},
				State: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "a-library",
							SourceRoots: []string{
								"a-library/a/path",
								"a-library/another/path",
							},
						},
						{
							ID: "another-library",
							SourceRoots: []string{
								"another-library/one/path",
								"another-library/two/path",
							},
						},
					},
				},
				partialRepoDir: filepath.Join(os.TempDir(), "partial-repo"),
				GlobalConfig: &config.GlobalConfig{
					GlobalFilesAllowlist: []*config.GlobalFile{
						{
							Path:        "read/one.txt",
							Permissions: "read-only",
						},
						{
							Path:        "write/two.txt",
							Permissions: "write-only",
						},
						{
							Path:        "read-write/three.txt",
							Permissions: "read-write",
						},
					},
				},
			},
			includedFiles: []string{
				"a-library/a/path/empty.txt",
				"a-library/another/path/empty.txt",
				"another-library/one/path/empty.txt",
				"another-library/two/path/empty.txt",
				".librarian/empty.txt",
				"read/one.txt",
				"write/two.txt",
				"read-write/three.txt",
			},
			excludedFiles: []string{},
		},
		{
			name: "copy one library and required files to partial repo",
			request: &ReleaseRequest{
				Cfg: &config.Config{
					Repo: filepath.Join(os.TempDir(), "repo"),
				},
				LibraryID: "a-library",
				State: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "a-library",
							SourceRoots: []string{
								"a-library/a/path",
								"a-library/another/path",
							},
						},
						{
							ID: "another-library",
							SourceRoots: []string{
								"another-library/one/path",
								"another-library/two/path",
							},
						},
					},
				},
				partialRepoDir: filepath.Join(os.TempDir(), "partial-repo"),
				GlobalConfig: &config.GlobalConfig{
					GlobalFilesAllowlist: []*config.GlobalFile{
						{
							Path:        "read/one.txt",
							Permissions: "read-only",
						},
						{
							Path:        "write/two.txt",
							Permissions: "write-only",
						},
						{
							Path:        "read-write/three.txt",
							Permissions: "read-write",
						},
					},
				},
			},
			includedFiles: []string{
				"a-library/a/path/empty.txt",
				"a-library/another/path/empty.txt",
				".librarian/empty.txt",
				"read/one.txt",
				"write/two.txt",
				"read-write/three.txt",
			},
			excludedFiles: []string{
				"another-library/one/path/empty.txt",
				"another-library/two/path/empty.txt",
			},
		},
		{
			name: "copy one library with empty initial directory",
			request: &ReleaseRequest{
				Cfg: &config.Config{
					Repo:     filepath.Join(os.TempDir(), "repo"),
					WorkRoot: filepath.Join(os.TempDir(), time.Now().String()),
				},
				LibraryID: "a-library",
				State: &config.LibrarianState{
					Libraries: []*config.LibraryState{
						{
							ID: "a-library",
							SourceRoots: []string{
								"a-library/a/path",
								"a-library/another/path",
							},
						},
						{
							ID: "another-library",
							SourceRoots: []string{
								"another-library/one/path",
								"another-library/two/path",
							},
						},
					},
				},
				GlobalConfig: &config.GlobalConfig{
					GlobalFilesAllowlist: []*config.GlobalFile{},
				},
			},
			includedFiles: []string{
				"a-library/a/path/empty.txt",
				"a-library/another/path/empty.txt",
				".librarian/empty.txt",
			},
			excludedFiles: []string{
				"another-library/one/path/empty.txt",
				"another-library/two/path/empty.txt",
			},
		},
		{
			name: "invalid partial repo dir",
			request: &ReleaseRequest{
				Cfg: &config.Config{
					Repo: os.TempDir(),
				},
				partialRepoDir: "/invalid-path",
			},
			wantErr:    true,
			wantErrMsg: "failed to make directory",
		},
		{
			name: "invalid source repo dir",
			request: &ReleaseRequest{
				Cfg: &config.Config{
					Repo: "/non-existent-path",
				},
				State:          &config.LibrarianState{},
				partialRepoDir: os.TempDir(),
			},
			wantErr:    true,
			wantErrMsg: "failed to copy librarian dir",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !test.wantErr {
				if err := os.RemoveAll(test.request.Cfg.Repo); err != nil {
					t.Error(err)
				}
				if err := os.RemoveAll(test.request.partialRepoDir); err != nil {
					t.Error(err)
				}

				if err := prepareRepo(t, test.request); err != nil {
					t.Error(err)
				}
			}

			err := setupPartialRepo(test.request)
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
				t.Errorf("partialCopyRepo failed, error: %q", err)
			}

			for _, includedFile := range test.includedFiles {
				filename := filepath.Join(test.request.partialRepoDir, includedFile)
				if _, err := os.Stat(filename); err != nil {
					t.Error(err)
				}
			}

			for _, excludedFile := range test.excludedFiles {
				filename := filepath.Join(test.request.partialRepoDir, excludedFile)
				if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
					t.Error(err)
				}
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

func TestCopyOneLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		dst        string
		src        string
		library    *config.LibraryState
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "invalid src",
			dst:  os.TempDir(),
			src:  "/invalid-path",
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a-library/path",
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to copy",
		},
		{
			name: "invalid dst",
			dst:  "/invalid-path",
			src:  os.TempDir(),
			library: &config.LibraryState{
				ID: "example-library",
				SourceRoots: []string{
					"a-library/path",
				},
			},
			wantErr:    true,
			wantErrMsg: "failed to copy",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := copyOneLibrary(test.dst, test.src, test.library)
			if test.wantErr {
				if err == nil {
					t.Errorf("copyOneLibrary() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
			}

		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		dst         string
		src         string
		wantSrcFile bool
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:       "invalid src",
			src:        "/invalid-path/example.txt",
			wantErr:    true,
			wantErrMsg: "failed to open file",
		},
		{
			name:        "invalid dst path",
			src:         filepath.Join(os.TempDir(), "example.txt"),
			dst:         "/invalid-path/example.txt",
			wantSrcFile: true,
			wantErr:     true,
			wantErrMsg:  "failed to make directory",
		},
		{
			name:        "invalid dst file",
			src:         filepath.Join(os.TempDir(), "example.txt"),
			dst:         filepath.Join(os.TempDir(), "example\x00.txt"),
			wantSrcFile: true,
			wantErr:     true,
			wantErrMsg:  "failed to create file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.wantSrcFile {
				if err := os.MkdirAll(filepath.Dir(test.src), 0755); err != nil {
					t.Error(err)
				}
				sourceFile, err := os.Create(test.src)
				if err != nil {
					t.Error(err)
				}
				if err := sourceFile.Close(); err != nil {
					t.Error(err)
				}
			}
			err := copyFile(test.dst, test.src)
			if test.wantErr {
				if err == nil {
					t.Errorf("copyFile() shoud fail")
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}

				return
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

func prepareRepo(t *testing.T, request *ReleaseRequest) error {
	t.Helper()
	emptyFilename := "empty.txt"
	repo := request.Cfg.Repo
	// Create library files.
	for _, library := range request.State.Libraries {
		for _, sourcePath := range library.SourceRoots {
			sourcePath = filepath.Join(repo, sourcePath)
			if err := os.MkdirAll(sourcePath, 0755); err != nil {
				return err
			}
			if err := createEmptyFile(t, filepath.Join(sourcePath, emptyFilename)); err != nil {
				return err
			}
		}
	}
	// Create .librarian directory.
	librarianDir := filepath.Join(repo, ".librarian")
	if err := os.MkdirAll(librarianDir, 0755); err != nil {
		return err
	}
	if err := createEmptyFile(t, filepath.Join(librarianDir, emptyFilename)); err != nil {
		return err
	}
	// Create global files.
	for _, globalFile := range request.GlobalConfig.GlobalFilesAllowlist {
		filename := filepath.Join(repo, globalFile.Path)
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return err
		}
		if err := createEmptyFile(t, filename); err != nil {
			return err
		}
	}

	return nil
}

func createEmptyFile(t *testing.T, filename string) error {
	t.Helper()
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", filename)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %s", filename)
	}

	return nil
}
