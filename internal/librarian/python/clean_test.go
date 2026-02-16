// Copyright 2026 Google LLC
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
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestCleanLibrary(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		lib         *config.Library
		setupFiles  []string
		wantDeleted []string
	}{
		{
			name: "output directory doesn't exist",
			lib: &config.Library{
				Name: "test",
			},
		},
		{
			name: "no APIs",
			lib: &config.Library{
				Name: "test",
			},
			setupFiles: []string{"README.md"},
		},
		{
			name: "proto-only API",
			lib: &config.Library{
				Name: "test",
				APIs: []*config.API{
					{Path: "google/type"},
				},
				Python: &config.PythonPackage{
					ProtoOnlyAPIs: []string{"google/type"},
				},
				Keep: []string{"google/type/keep.proto"},
			},
			setupFiles: []string{
				"README.md",
				"google/type/date.proto",
				"google/type/keep.proto",
				"google/type/date_pb2.py",
				"google/type/date_pb2.pyi",
				"google/type/README.txt",
			},
			wantDeleted: []string{
				"google/type/date.proto",
				"google/type/date_pb2.py",
				"google/type/date_pb2.pyi",
			},
		},
		{
			name: "GAPIC API",
			lib: &config.Library{
				Name: "test",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{
							"{neutral-source}/delete-me.txt",
							// This is in the keep list as well, so should be kept
							"{neutral-source}/keep-me.txt",
							"docs/delete-me.txt",
							"doesn't-exist.txt",
							"delete-me-directory",
						},
					},
				},
				Keep: []string{
					"google/cloud/secretmanager/keep-me.txt",
					"google/cloud/secretmanager_v1/keep-me.txt",
				},
			},
			setupFiles: []string{
				"README.md",
				"google/cloud/secretmanager/delete-me.txt",
				"google/cloud/secretmanager/leave-me.txt",
				"google/cloud/secretmanager/keep-me.txt",
				"google/cloud/secretmanager/delete-me.txt",
				"google/cloud/secretmanager_v1/delete-me.txt",
				"google/cloud/secretmanager_v1/keep-me.txt",
				"docs/delete-me.txt",
				"delete-me-directory/a.txt",
				"delete-me-directory/subdirectory/b.txt",
			},
			wantDeleted: []string{
				"google/cloud/secretmanager/delete-me.txt",
				"google/cloud/secretmanager_v1/delete-me.txt",
				"docs/delete-me.txt",
				"delete-me-directory/a.txt",
				"delete-me-directory/subdirectory/b.txt",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			// Note: deliberately not creating the subdirectory to start with,
			// so that if we have no files to create, the directory isn't
			// created either.
			test.lib.Output = filepath.Join(dir, test.lib.Name)
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(test.lib.Output, file)
				createFileAndDirectories(t, fullPath)
			}

			if err := CleanLibrary(test.lib); err != nil {
				t.Fatal(err)
			}

			for _, file := range test.setupFiles {
				fullPath := filepath.Join(test.lib.Output, file)
				_, err := os.Stat(fullPath)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				gotDeleted := err != nil
				wantDeleted := slices.Contains(test.wantDeleted, file)
				if gotDeleted != wantDeleted {
					t.Errorf("file %s: wantDeleted=%t, gotDeleted=%t", file, wantDeleted, gotDeleted)
				}
			}
		})
	}
}

func TestCleanLibrary_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		lib     *config.Library
		setup   func(*testing.T, string)
		wantErr error
	}{
		{
			name: "cleanProtoOnly fails",
			lib: &config.Library{
				Name: "proto-only",
				APIs: []*config.API{{Path: "google/type"}},
				Python: &config.PythonPackage{
					ProtoOnlyAPIs: []string{"google/type"},
				},
			},
			setup: func(t *testing.T, dir string) {
				// This will create a file called google/type, when cleanProtoOnly
				// expects it to be a directory.
				createFileAndDirectories(t, filepath.Join(dir, "proto-only", "google", "type"))
			},
			wantErr: syscall.ENOTDIR,
		},
		{
			name: "cleanGAPIC fails",
			lib: &config.Library{
				Name: "gapic-bad-path",
				APIs: []*config.API{{Path: "google"}},
			},
			wantErr: errBadAPIPath,
		},
		{
			name: "cleanGAPICCommon fails",
			lib: &config.Library{
				Name: "google-cloud-functions",
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
			},
			wantErr: errNoCommonGAPICFilesConfig,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.lib.Output = filepath.Join(dir, test.lib.Name)
			if err := os.Mkdir(test.lib.Output, 0755); err != nil {
				t.Fatal(err)
			}
			if test.setup != nil {
				test.setup(t, dir)
			}
			gotErr := CleanLibrary(test.lib)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("CleanLibrary error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestDeriveGAPICGenerationInfo(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		lib     *config.Library
		want    *gapicGenerationInfo
	}{
		{
			name:    "no extra config",
			apiPath: "google/cloud/functions/v1",
			lib:     &config.Library{},
			want: &gapicGenerationInfo{
				RootDir:    "google/cloud",
				NeutralDir: "functions",
				VersionDir: "functions_v1",
			},
		},
		{
			name:    "no version",
			apiPath: "google/shopping/type",
			lib:     &config.Library{},
			want: &gapicGenerationInfo{
				RootDir:    "google/shopping",
				NeutralDir: "type",
			},
		},
		{
			name:    "overridden configuration",
			apiPath: "google/cloud/secrets/v1beta1",
			lib: &config.Library{
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secrets/v1beta1": {"python-gapic-namespace=google.cloud", "python-gapic-name=secretmanager"},
					},
				},
			},
			want: &gapicGenerationInfo{
				RootDir:    "google/cloud",
				NeutralDir: "secretmanager",
				VersionDir: "secretmanager_v1beta1",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &config.API{
				Path: test.apiPath,
			}
			info, err := deriveGAPICGenerationInfo(api, test.lib)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, info); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveGAPICGenerationInfo_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		lib     *config.Library
		wantErr error
	}{
		{
			name:    "no path",
			apiPath: "",
			wantErr: errBadAPIPath,
		},
		{
			name:    "single-element path",
			apiPath: "google",
			wantErr: errBadAPIPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &config.API{
				Path: test.apiPath,
			}
			_, gotErr := deriveGAPICGenerationInfo(api, test.lib)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("deriveGAPICGenerationInfo error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestFindOptArg(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *config.PythonPackage
		want string
	}{
		{
			name: "option present",
			cfg: &config.PythonPackage{
				OptArgsByAPI: map[string][]string{
					"path/to/api": {"other=x", "findme=foundvalue"},
				},
			},
			want: "foundvalue",
		},
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "nil OptArgsByAPI",
			cfg: &config.PythonPackage{
				ProtoOnlyAPIs: []string{"ignored"},
			},
		},
		{
			name: "no args for specified API",
			cfg: &config.PythonPackage{
				OptArgsByAPI: map[string][]string{
					"path/to/other": {"other=x", "findme=foundvalue"},
				},
			},
		},
		{
			name: "args for specified API, but none with given name",
			cfg: &config.PythonPackage{
				OptArgsByAPI: map[string][]string{
					"path/to/api": {"other=x"},
				},
			},
		},
		{
			name: "args for specified API, but only a prefix match",
			cfg: &config.PythonPackage{
				OptArgsByAPI: map[string][]string{
					"path/to/api": {"other=x,findmenot=xyz"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &config.API{
				Path: "path/to/api",
			}
			optName := "findme"
			got := findOptArg(api, test.cfg, optName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCleanProtoOnly(t *testing.T) {
	for _, test := range []struct {
		name        string
		setupFiles  []string
		lib         *config.Library
		wantDeleted []string
	}{
		{
			name: "common case",
			setupFiles: []string{
				"google/cloud/functions/v1/functions.proto",
				"google/cloud/functions/v1/functions_pb2.py",
				"google/cloud/functions/v1/functions_pb2.pyi",
				"google/cloud/functions/v1/handwritten.txt",
				"google/cloud/functions/v1/keep-me.proto",
				"google/cloud/functions/v1/subdir/ignored.proto",
				"README.txt",
			},
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
				Keep: []string{
					"google/cloud/functions/v1/keep-me.proto",
				},
			},
			wantDeleted: []string{
				"google/cloud/functions/v1/functions.proto",
				"google/cloud/functions/v1/functions_pb2.py",
				"google/cloud/functions/v1/functions_pb2.pyi",
			},
		},
		{
			name: "no such directory",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
				Keep: []string{
					"google/cloud/functions/v1/keep-me.proto",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				createFileAndDirectories(t, fullPath)
			}

			test.lib.Output = dir
			if err := cleanProtoOnly(test.lib.APIs[0], test.lib); err != nil {
				t.Fatal(err)
			}
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				_, err := os.Stat(fullPath)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				gotDeleted := err != nil
				wantDeleted := slices.Contains(test.wantDeleted, file)
				if gotDeleted != wantDeleted {
					t.Errorf("file %s: wantDeleted=%t, gotDeleted=%t", file, wantDeleted, gotDeleted)
				}
			}
		})
	}
}

func TestCleanGAPIC(t *testing.T) {
	for _, test := range []struct {
		name        string
		setupFiles  []string
		lib         *config.Library
		wantDeleted []string
	}{
		{
			name: "common case",
			setupFiles: []string{
				"google/cloud/functions/gapic_version.py",
				"google/cloud/functions_v1/gapic_version.py",
				"google/cloud/functions_v1/keep-me.txt",
				"docs/functions_v1/README.txt",
				"keep-me.txt",
				"noxfile.py",
				"other.txt",
			},
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
				Keep: []string{
					"keep-me.txt",
					"google/cloud/functions_v1/keep-me.txt",
				},
			},
			wantDeleted: []string{
				"google/cloud/functions_v1/gapic_version.py",
				"docs/functions_v1/README.txt",
			},
		},
		{
			name: "no version",
			setupFiles: []string{
				"google/shopping/type/gapic_version.py",
				"keep-me.txt",
				"noxfile.py",
				"other.txt",
			},
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/shopping/type"}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				createFileAndDirectories(t, fullPath)
			}

			test.lib.Output = dir
			if err := cleanGAPIC(test.lib.APIs[0], test.lib); err != nil {
				t.Fatal(err)
			}
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				_, err := os.Stat(fullPath)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				gotDeleted := err != nil
				wantDeleted := slices.Contains(test.wantDeleted, file)
				if gotDeleted != wantDeleted {
					t.Errorf("file %s: wantDeleted=%t, gotDeleted=%t", file, wantDeleted, gotDeleted)
				}
			}
		})
	}
}

func TestCleanGAPIC_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		lib     *config.Library
		wantErr error
	}{
		{
			name: "bad API path",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google"}},
			},
			wantErr: errBadAPIPath,
		},
		{
			name: "error during deletion",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
			},
			setup: func(t *testing.T, dir string) {
				sourceDirectory := filepath.Join(dir, "google", "cloud", "functions_v1")
				createFileAndDirectories(t, filepath.Join(sourceDirectory, "file.txt"))
				if err := os.Chmod(sourceDirectory, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					// Fix the permission afterwards so that it can be deleted.
					if err := os.Chmod(sourceDirectory, 0755); err != nil {
						t.Fatal(err)
					}
				})
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.lib.Output = dir
			if test.setup != nil {
				test.setup(t, dir)
			}
			gotErr := cleanGAPIC(test.lib.APIs[0], test.lib)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("CleanGAPIC error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestCleanGAPICCommon(t *testing.T) {
	for _, test := range []struct {
		name        string
		setupFiles  []string
		lib         *config.Library
		wantDeleted []string
	}{
		{
			name: "general test (covers most common cases)",
			setupFiles: []string{
				"google/cloud/functions/gapic_version.py",
				"google/cloud/functions_v1/gapic_version.py",
				"keep-me.txt",
				"noxfile.py",
				"other.txt",
			},
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{
							"{neutral-source}/gapic_version.py",
							"keep-me.txt",
							"noxfile.py",
						},
					},
				},
				Keep: []string{"keep-me.txt"},
			},
			wantDeleted: []string{
				"google/cloud/functions/gapic_version.py",
				"noxfile.py",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				createFileAndDirectories(t, fullPath)
			}

			test.lib.Output = dir
			if err := cleanGAPICCommon(test.lib); err != nil {
				t.Fatal(err)
			}
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				_, err := os.Stat(fullPath)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				gotDeleted := err != nil
				wantDeleted := slices.Contains(test.wantDeleted, file)
				if gotDeleted != wantDeleted {
					t.Errorf("file %s: wantDeleted=%t, gotDeleted=%t", file, wantDeleted, gotDeleted)
				}
			}
		})
	}
}

func TestCleanGAPICCommon_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		setup   func(*testing.T, string)
		lib     *config.Library
		wantErr error
	}{
		{
			name: "bad API path",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google"}},
			},
			wantErr: errBadAPIPath,
		},
		{
			name: "no python configuration",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
			},
			wantErr: errNoCommonGAPICFilesConfig,
		},
		{
			name: "python configuration does not contain common gapic files config",
			lib: &config.Library{
				APIs:   []*config.API{{Path: "google/cloud/functions/v1"}},
				Python: &config.PythonPackage{},
			},
			wantErr: errNoCommonGAPICFilesConfig,
		},
		{
			name: "error during deletion",
			lib: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/functions/v1"}},
				Python: &config.PythonPackage{
					PythonDefault: config.PythonDefault{
						CommonGAPICPaths: []string{"subdir"},
					},
				},
			},
			setup: func(t *testing.T, dir string) {
				createFileAndDirectories(t, filepath.Join(dir, "subdir", "file.txt"))
				subdir := filepath.Join(dir, "subdir")
				if err := os.Chmod(subdir, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					// Fix the permission afterwards so that it can be deleted.
					if err := os.Chmod(subdir, 0755); err != nil {
						t.Fatal(err)
					}
				})
			},
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			test.lib.Output = dir
			if test.setup != nil {
				test.setup(t, dir)
			}
			gotErr := cleanGAPICCommon(test.lib)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("CleanGAPICCommon error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestDeleteUnlessKept(t *testing.T) {
	for _, test := range []struct {
		name        string
		setupFiles  []string
		keep        []string
		path        string
		wantDeleted []string
	}{
		{
			name:       "path does not exist",
			setupFiles: []string{"README.txt"},
			path:       "other.txt",
		},
		{
			name:        "file is deleted",
			setupFiles:  []string{"README.txt"},
			path:        "README.txt",
			wantDeleted: []string{"README.txt"},
		},
		{
			name:       "file is kept",
			setupFiles: []string{"README.txt"},
			keep:       []string{"README.txt"},
			path:       "README.txt",
		},
		{
			name:        "subdirectory matching",
			setupFiles:  []string{"README.txt", "subdir/a.txt", "subdir/b.txt"},
			keep:        []string{"subdir/a.txt"},
			path:        "subdir",
			wantDeleted: []string{"subdir/b.txt"},
		},
		{
			name:       "subdirectory is kept",
			setupFiles: []string{"README.txt", "subdir/a.txt", "subdir/b.txt"},
			keep:       []string{"subdir"},
			path:       "subdir",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				createFileAndDirectories(t, fullPath)
			}

			lib := &config.Library{
				Output: dir,
				Keep:   test.keep,
			}
			if err := deleteUnlessKept(lib, test.path); err != nil {
				t.Fatal(err)
			}

			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				_, err := os.Stat(fullPath)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				gotDeleted := err != nil
				wantDeleted := slices.Contains(test.wantDeleted, file)
				if gotDeleted != wantDeleted {
					t.Errorf("file %s: wantDeleted=%t, gotDeleted=%t", file, wantDeleted, gotDeleted)
				}
			}
		})
	}
}

func TestDeleteUnlessKept_Error(t *testing.T) {
	for _, test := range []struct {
		name       string
		setupFiles []string
		setup      func(*testing.T, string)
		path       string
		wantErr    error
	}{
		{
			name:       "can't delete file from read-only directory",
			setupFiles: []string{"readonly.txt"},
			setup: func(t *testing.T, dir string) {
				if err := os.Chmod(dir, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					// Fix the permission afterwards so that it can be deleted.
					if err := os.Chmod(dir, 0755); err != nil {
						t.Fatal(err)
					}
				})
			},
			path:    "readonly.txt",
			wantErr: os.ErrPermission,
		},
		{
			name:       "nested error",
			setupFiles: []string{"subdir/readonly.txt"},
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "subdir")
				if err := os.Chmod(subdir, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					// Fix the permission afterwards so that it can be deleted.
					if err := os.Chmod(subdir, 0755); err != nil {
						t.Fatal(err)
					}
				})
			},
			path:    "subdir",
			wantErr: os.ErrPermission,
		},
		{
			name:       "can't stat nested file",
			setupFiles: []string{"subdir/file.txt"},
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "subdir")
				if err := os.Chmod(subdir, 0444); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					// Fix the permission afterwards so that it can be deleted.
					if err := os.Chmod(subdir, 0755); err != nil {
						t.Fatal(err)
					}
				})
			},
			path:    "subdir",
			wantErr: os.ErrPermission,
		},
		{
			name:       "can't list directory",
			setupFiles: []string{"subdir/file.txt"},
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "subdir")
				if err := os.Chmod(subdir, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					// Fix the permission afterwards so that it can be deleted.
					if err := os.Chmod(subdir, 0755); err != nil {
						t.Fatal(err)
					}
				})
			},
			path:    "subdir",
			wantErr: os.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range test.setupFiles {
				fullPath := filepath.Join(dir, file)
				createFileAndDirectories(t, fullPath)
			}
			if test.setup != nil {
				test.setup(t, dir)
			}
			lib := &config.Library{
				Output: dir,
			}
			gotErr := deleteUnlessKept(lib, test.path)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("CleanLibrary error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func createFileAndDirectories(t *testing.T, path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
}
