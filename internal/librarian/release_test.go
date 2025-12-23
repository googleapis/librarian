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

package librarian

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelpers"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestReleaseCommand(t *testing.T) {
	const testlib = "test-lib"
	const testlib2 = "test-lib2"

	for _, test := range []struct {
		name             string
		args             []string
		srcPaths         map[string]string
		skipYamlCreation bool
		dirtyGitStatus   bool
		wantErr          error
		wantVersions     map[string]string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "release"},
			wantErr: errMissingLibraryOrAllFlag,
		},
		{
			name:    "library name and all flag",
			args:    []string{"librarian", "release", testlib, "--all"},
			wantErr: errBothLibraryAndAllFlag,
		},
		{
			name: "library name",
			args: []string{"librarian", "release", testlib},
			srcPaths: map[string]string{
				testlib: "src/storage",
			},
			wantVersions: map[string]string{
				testlib:  testReleaseVersion,
				testlib2: "0.1.0",
			},
		},
		{
			name: "all flag all have changes",
			args: []string{"librarian", "release", "--all"},
			srcPaths: map[string]string{
				testlib:  "src/storage",
				testlib2: "src/storage",
			},
			wantVersions: map[string]string{
				testlib:  testReleaseVersion,
				testlib2: testReleaseVersion,
			},
		},
		{
			name: "all flag 1 has changes",
			args: []string{"librarian", "release", "--all"},
			srcPaths: map[string]string{
				testlib:  "src/storage",
				testlib2: "src/gax-internal",
			},
			wantVersions: map[string]string{
				testlib:  testReleaseVersion,
				testlib2: "0.1.0",
			},
		},
		{
			name:    "no src path provided",
			args:    []string{"librarian", "release", "--all"},
			wantErr: errCouldNotDeriveSrcPath,
		},
		{
			name:             "missing librarian yaml file",
			args:             []string{"librarian", "release", "--all"},
			skipYamlCreation: true,
		},
		{
			name:           "local repo is dirty",
			args:           []string{"librarian", "release", "--all"},
			dirtyGitStatus: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testhelpers.RequireCommand(t, "git")
			remoteDir := testhelpers.SetupRepoWithChange(t, "v1.0.0")
			testhelpers.CloneRepository(t, remoteDir)

			configPath := filepath.Join("./", librarianConfigPath)
			cfg := &config.Config{
				Language: "testhelper",
				Release: &config.Release{
					Remote: "origin",
					Branch: "main",
				},
				Libraries: []*config.Library{
					{
						Name:    testlib,
						Version: "0.1.0",
						Output:  test.srcPaths[testlib],
					},
					{
						Name:    testlib2,
						Version: "0.1.0",
						Output:  test.srcPaths[testlib2],
					},
				},
			}
			if !test.skipYamlCreation {
				if err := yaml.Write(configPath, cfg); err != nil {
					t.Fatal(err)
				}
				if !test.dirtyGitStatus {
					if err := command.Run(t.Context(), "git", "add", "."); err != nil {
						t.Fatal(err)
					}

					if err := command.Run(t.Context(), "git", "commit", "-m", "chore: update lib yaml", "."); err != nil {
						t.Fatal(err)
					}
				}
			}
			err := Run(t.Context(), test.args...)
			if (test.skipYamlCreation || test.dirtyGitStatus) && err != nil {
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Run() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr != nil {
				return
			}

			if test.wantVersions != nil {
				cfg, err := yaml.Read[config.Config](configPath)
				if err != nil {
					t.Fatal(err)
				}
				gotVersions := make(map[string]string)
				for _, lib := range cfg.Libraries {
					gotVersions[lib.Name] = lib.Version
				}
				if diff := cmp.Diff(test.wantVersions, gotVersions); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLibraryByName(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		config      *config.Config
		want        *config.Library
		wantErr     error
	}{
		{
			name:        "find_a_library",
			libraryName: "example-library",
			config: &config.Config{
				Libraries: []*config.Library{
					{Name: "example-library"},
					{Name: "another-library"},
				},
			},
			want: &config.Library{Name: "example-library"},
		},
		{
			name:        "no_library_in_config",
			libraryName: "example-library",
			config:      &config.Config{},
			wantErr:     errLibraryNotFound,
		},
		{
			name:        "does_not_find_a_library",
			libraryName: "non-existent-library",
			config: &config.Config{
				Libraries: []*config.Library{
					{Name: "example-library"},
					{Name: "another-library"},
				},
			},
			wantErr: errLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := libraryByName(test.config, test.libraryName)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("libraryByName(%q): %v", test.libraryName, err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReleaseRust(t *testing.T) {
	origRustReleaseLibrary := rustReleaseLibrary
	origLibrarianGenerateLibrary := librarianGenerateLibrary

	defer func() {
		rustReleaseLibrary = origRustReleaseLibrary
		librarianGenerateLibrary = origLibrarianGenerateLibrary
	}()

	tests := []struct {
		name               string
		srcPath            string
		releaseError       error
		generateError      error
		wantReleaseCalled  bool
		wantGenerateCalled bool
		wantErr            bool
	}{
		{
			name:               "library released",
			srcPath:            "src/storage",
			wantReleaseCalled:  true,
			wantGenerateCalled: true,
		},
		{
			name:               "generate error",
			srcPath:            "src/storage",
			wantReleaseCalled:  true,
			wantGenerateCalled: true,
			generateError:      errors.New("generate error"),
			wantErr:            true,
		},
		{
			name:               "rust release error",
			srcPath:            "src/storage",
			wantReleaseCalled:  true,
			wantGenerateCalled: false,
			releaseError:       errors.New("release error"),
			wantErr:            true,
		},
	}
	testhelpers.RequireCommand(t, "git")
	remoteDir := testhelpers.SetupRepoWithChange(t, "v1.0.0")
	testhelpers.CloneRepository(t, remoteDir)
	cfg := &config.Config{
		Language: "rust",
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				generateCalled bool
				releaseCalled  bool
			)
			rustReleaseLibrary = func(library *config.Library, srcPath string) error {
				releaseCalled = true
				return test.releaseError
			}
			librarianGenerateLibrary = func(ctx context.Context, cfg *config.Config, libraryName string) (*config.Library, error) {
				generateCalled = true
				return nil, test.generateError
			}
			libConfg := &config.Library{}
			err := releaseLibrary(t.Context(), cfg, libConfg, test.srcPath)

			if (err != nil) != test.wantErr {
				t.Fatalf("releaseLibrary() error = %v, wantErr %v", err, test.wantErr)
			}
			if releaseCalled != test.wantReleaseCalled {
				t.Errorf("releaseCalled = %v, want %v", releaseCalled, test.wantReleaseCalled)
			}
			if generateCalled != test.wantGenerateCalled {
				t.Errorf("generateCalled = %v, want %v", generateCalled, test.wantGenerateCalled)
			}
			if test.releaseError != nil && test.releaseError != err {
				t.Errorf("releaseError= %v, want %v", err, test.releaseError)

			}
			if test.generateError != nil && test.generateError != err {
				t.Errorf("generateError= %v, want %v", err, test.generateError)

			}
		})

	}
}

func TestMissingReleaseConfig(t *testing.T) {
	cfg := &config.Config{}
	_, err := shouldReleaseLibrary(t.Context(), cfg, "")
	if !errors.Is(err, errReleaseConfigEmpty) {
		t.Fatalf("Run() error = %v, wantErr %v", err, errReleaseConfigEmpty)
	}

}
