// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestGetStagingChildDirectory(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		apiPath   string
		protoOnly bool
		expected  string
	}{
		{
			name:     "versioned path",
			apiPath:  "google/cloud/secretmanager/v1",
			expected: "v1",
		},
		{
			name:     "non-versioned path",
			apiPath:  "google/cloud/secretmanager/type",
			expected: "type-py",
		},
		{
			name:      "proto-only",
			apiPath:   "google/cloud/secretmanager/type",
			protoOnly: true,
			expected:  "type",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getStagingChildDirectory(test.apiPath, test.protoOnly)
			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("getStagingChildDirectory(%q) returned diff (-want +got):\n%s", test.apiPath, diff)
			}
		})
	}
}

func TestCreateProtocOptions(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		api      *config.API
		library  *config.Library
		expected []string
		wantErr  bool
	}{
		{
			name:    "basic case",
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,rest-numeric-enums,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "with transport",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name:      "google-cloud-secret-manager",
				Transport: "grpc",
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,rest-numeric-enums,transport=grpc,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "with python opts",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgs: []string{"opt1", "opt2"},
				},
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,opt1,opt2,rest-numeric-enums,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "with python opts by api",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"opt1", "opt2"},
						"google/cloud/secretmanager/v2": {"opt3", "opt4"},
					},
				},
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,opt1,opt2,rest-numeric-enums,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "with version",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name:    "google-cloud-secret-manager",
				Version: "1.2.3",
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,rest-numeric-enums,gapic-version=1.2.3,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "with service config",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Name: "google-cloud-secret-manager",
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,rest-numeric-enums,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "library starting google-cloud-compute does not use gRPC service config",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				// It's odd to use a Compute name for a path that's using secretmanager,
				// but it's simpler than making the test realistic by importing the
				// (huge) Compute protos etc.
				Name: "google-cloud-compute-beta",
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,rest-numeric-enums,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "rest-enumeric-enums is specified in OptArgs",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgs: []string{"rest-numeric-enums=False"},
				},
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,rest-numeric-enums=False,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "transport overridden in OptOptArgsByAPIArgs",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secret-manager",
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"transport=rest"},
					},
				},
				Transport: "grpc",
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,transport=rest,rest-numeric-enums,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "proto-only exists but doesn't include API path",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Python: &config.PythonPackage{
					ProtoOnlyAPIs: []string{"google/cloud/secretmanager/type"},
				},
			},
			expected: []string{
				"--python_gapic_out=staging",
				"--python_gapic_opt=metadata,rest-numeric-enums,retry-config=google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json,service-yaml=google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			},
		},
		{
			name: "proto-only exists and includes API path",
			api:  &config.API{Path: "google/cloud/secretmanager/type"},
			library: &config.Library{
				Python: &config.PythonPackage{
					ProtoOnlyAPIs: []string{"google/cloud/secretmanager/type"},
				},
			},
			expected: []string{
				"--python_out=staging",
				"--pyi_out=staging",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := createProtocOptions(test.api, test.library, googleapisDir, "staging")
			if (err != nil) != test.wantErr {
				t.Fatalf("createProtocOptions() error = %v, wantErr %v", err, test.wantErr)
			}

			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("createProtocOptions() returned diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStageProtoFiles(t *testing.T) {
	targetDir := t.TempDir()
	// Deliberately not including all proto files (or any non-proto) files here.
	relativeProtoPaths := []string{
		"google/cloud/gkehub/v1/feature.proto",
		"google/cloud/gkehub/v1/membership.proto",
	}
	if err := stageProtoFiles(googleapisDir, targetDir, relativeProtoPaths); err != nil {
		t.Fatal(err)
	}
	copiedFiles := []string{}
	if err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsDir() {
			relative, err := filepath.Rel(targetDir, path)
			if err != nil {
				return err
			}
			copiedFiles = append(copiedFiles, relative)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(relativeProtoPaths, copiedFiles); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestStageProtoFiles_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		relativeProtoPaths []string
		setup              func(t *testing.T, targetDir string)
		wantErr            error
	}{
		{
			name:               "path doesn't exist",
			relativeProtoPaths: []string{"google/cloud/bogus.proto"},
			wantErr:            os.ErrNotExist,
		},
		{
			name:               "can't create directory",
			relativeProtoPaths: []string{"google/cloud/gkehub/v1/feature.proto"},
			setup: func(t *testing.T, targetDir string) {
				// Create a file with the name of the directory we'd create.
				if err := os.WriteFile(filepath.Join(targetDir, "google"), []byte{}, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: syscall.ENOTDIR,
		},
		{
			name:               "can't write file",
			relativeProtoPaths: []string{"google/cloud/gkehub/v1/feature.proto"},
			setup: func(t *testing.T, targetDir string) {
				// Create a directory with the name of the file we'd create.
				if err := os.MkdirAll(filepath.Join(targetDir, "google", "cloud", "gkehub", "v1", "feature.proto"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: syscall.EISDIR,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			targetDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, targetDir)
			}
			gotErr := stageProtoFiles(googleapisDir, targetDir, test.relativeProtoPaths)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("stageProtoFiles error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestCopyReadmeToDocsDir(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name            string
		setup           func(t *testing.T, outdir string)
		expectedContent string
		expectedErr     bool
	}{
		{
			name: "no readme",
			setup: func(t *testing.T, outdir string) {
				// No setup needed
			},
		},
		{
			name: "readme is a regular file",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedContent: "hello",
		},
		{
			name: "readme is a symlink",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "REAL_README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("REAL_README.rst", filepath.Join(outdir, "README.rst")); err != nil {
					t.Fatal(err)
				}
			},
			expectedContent: "hello",
		},
		{
			name: "dest is a symlink",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(outdir, "docs"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../some/other/file", filepath.Join(outdir, "docs", "README.rst")); err != nil {
					t.Fatal(err)
				}
			},
			expectedContent: "hello",
		},
		{
			name: "unreadable readme",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0000); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					os.Chmod(filepath.Join(outdir, "README.rst"), 0644)
				})
			},
			expectedErr: true,
		},
		{
			name: "cannot create docs dir",
			setup: func(t *testing.T, outdir string) {
				if err := os.WriteFile(filepath.Join(outdir, "README.rst"), []byte("hello"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(outdir, "docs"), []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outdir := t.TempDir()
			test.setup(t, outdir)
			err := copyReadmeToDocsDir(outdir)
			if (err != nil) != test.expectedErr {
				t.Fatalf("copyReadmeToDocsDir() error = %v, wantErr %v", err, test.expectedErr)
			}

			if test.expectedContent != "" {
				content, err := os.ReadFile(filepath.Join(outdir, "docs", "README.rst"))
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(test.expectedContent, string(content)); diff != "" {
					t.Errorf("copyReadmeToDocsDir() returned diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCleanUpFilesAfterPostProcessing(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		setup       func(t *testing.T, repoRoot string)
		expectedErr bool
	}{
		{
			name: "no staging dir",
			setup: func(t *testing.T, repoRoot string) {
				// No setup needed
			},
		},
		{
			name: "staging dir exists",
			setup: func(t *testing.T, repoRoot string) {
				if err := os.MkdirAll(filepath.Join(repoRoot, "owl-bot-staging"), 0755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "error removing",
			setup: func(t *testing.T, repoRoot string) {
				stagingDir := filepath.Join(repoRoot, "owl-bot-staging")
				if err := os.MkdirAll(stagingDir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create a file in the directory
				if err := os.WriteFile(filepath.Join(stagingDir, "file"), []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				// Make the directory read-only to cause an error
				if err := os.Chmod(stagingDir, 0400); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					os.Chmod(stagingDir, 0755)
				})
			},
			expectedErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			test.setup(t, repoRoot)
			err := cleanUpFilesAfterPostProcessing(repoRoot)
			if (err != nil) != test.expectedErr {
				t.Fatalf("cleanUpFilesAfterPostProcessing() error = %v, wantErr %v", err, test.expectedErr)
			}
			if !test.expectedErr {
				if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !os.IsNotExist(err) {
					t.Errorf("owl-bot-staging should have been removed")
				}
			}
		})
	}
}

func TestRunPostProcessor(t *testing.T) {
	testhelper.RequireCommand(t, "python3")
	testhelper.RequireCommand(t, "nox")
	requireSynthtool(t)

	repoRoot := t.TempDir()
	outDir := t.TempDir()

	// Create minimal .repo-metadata.json that synthtool expects
	if err := os.WriteFile(filepath.Join(outDir, ".repo-metadata.json"), []byte(`{"default_version":"v1"}`), 0644); err != nil {
		t.Fatal(err)
	}

	err := runPostProcessor(t.Context(), repoRoot, outDir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow test: Python GAPIC code generation")
	}

	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-python_gapic")
	repoRoot := t.TempDir()
	err := generateAPI(
		t.Context(),
		&config.API{Path: "google/cloud/secretmanager/v1"},
		&config.Library{Name: "secretmanager", Output: repoRoot},
		googleapisDir,
		repoRoot,
	)
	if err != nil {
		t.Fatal(err)
	}
}

// TestGenerateLibraries performs simple testing that multiple libraries can
// be generated. Only the presence of a single expected file per library is
// performed; TestGenerate is responsible for more detailed testing of
// per-library generation.
func TestGenerateLibraries(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: Python code generation")
	}

	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-python_gapic")
	testhelper.RequireCommand(t, "python3")
	testhelper.RequireCommand(t, "nox")
	requireSynthtool(t)
	repoRoot := t.TempDir()
	outdir, err := filepath.Abs(filepath.Join(repoRoot, "packages", "secretmanager"))
	if err != nil {
		t.Fatal(err)
	}

	libraries := []*config.Library{
		{
			Name: "secretmanager",
			APIs: []*config.API{
				{
					Path: "google/cloud/secretmanager/v1",
				},
			},
		},
		{
			Name: "configdelivery",
			APIs: []*config.API{
				{
					Path: "google/cloud/configdelivery/v1",
				},
			},
		},
	}
	for _, library := range libraries {
		library.Output = filepath.Join(outdir, "packages", library.Name)
	}
	if err := GenerateLibraries(t.Context(), libraries, googleapisDir); err != nil {
		t.Fatal(err)
	}
	for _, library := range libraries {
		expectedRepoMetadata := filepath.Join(library.Output, ".repo-metadata.json")
		_, err := os.Stat(expectedRepoMetadata)
		if err != nil {
			t.Errorf("Stat(%s) returned error: %v", expectedRepoMetadata, err)
		}
	}
}

func TestGenerateLibraries_Error(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: Python code generation")
	}

	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-python_gapic")
	testhelper.RequireCommand(t, "python3")
	testhelper.RequireCommand(t, "nox")
	requireSynthtool(t)
	repoRoot := t.TempDir()
	outdir, err := filepath.Abs(filepath.Join(repoRoot, "packages", "secretmanager"))
	if err != nil {
		t.Fatal(err)
	}

	libraries := []*config.Library{
		{
			Name:   "no-apis",
			Output: filepath.Join(outdir, "packages", "no-apis"),
		},
	}
	gotErr := GenerateLibraries(t.Context(), libraries, googleapisDir)
	wantErr := errNoApis
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("GenerateLibraries error = %v, wantErr %v", gotErr, wantErr)
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("slow test: Python code generation")
	}

	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-python_gapic")
	testhelper.RequireCommand(t, "python3")
	testhelper.RequireCommand(t, "nox")
	requireSynthtool(t)
	repoRoot := t.TempDir()
	outdir, err := filepath.Abs(filepath.Join(repoRoot, "packages", "secretmanager"))
	if err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:   "secretmanager",
		Output: outdir,
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
			},
		},
	}
	if err := generate(t.Context(), library, googleapisDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outdir, ".repo-metadata.json")); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultOutputByName(t *testing.T) {
	want := "packages/google-cloud-secret-manager"
	got := DefaultOutputByName("google-cloud-secret-manager", "packages")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		api  string
		want string
	}{
		{"google/cloud/secretmanager/v1", "google-cloud-secretmanager"},
		{"google/cloud/secretmanager/v1beta2", "google-cloud-secretmanager"},
		{"google/cloud/storage/v2alpha", "google-cloud-storage"},
		{"google/maps/addressvalidation/v1", "google-maps-addressvalidation"},
		{"google/api/v1", "google-api"},
		{"google/cloud/vision", "google-cloud-vision"},
	} {
		t.Run(test.api, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func requireSynthtool(t *testing.T) {
	module := "synthtool"
	t.Helper()
	cmd := exec.Command("python3", "-c", fmt.Sprintf("import %s", module))
	if err := cmd.Run(); err != nil {
		t.Skipf("skipping test because Python module %s is not installed", module)
	}
}
