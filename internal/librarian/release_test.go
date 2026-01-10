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
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestReleaseCommand(t *testing.T) {
	const testlib = "test-lib"
	const testlib2 = "test-lib2"
	testhelper.RequireCommand(t, "git")

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
				testlib:  fakeReleaseVersion,
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
				testlib:  fakeReleaseVersion,
				testlib2: fakeReleaseVersion,
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
				testlib:  fakeReleaseVersion,
				testlib2: "0.1.0",
			},
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
		{
			name: "release config empty",
			args: []string{"librarian", "release", "--all"},
			srcPaths: map[string]string{
				testlib:  "src/storage",
				testlib2: "src/storage",
			},
			wantVersions: map[string]string{
				testlib:  fakeReleaseVersion,
				testlib2: fakeReleaseVersion,
			},
			wantErr: errReleaseConfigEmpty,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			remoteDir := testhelper.SetupRepoWithChange(t, "v1.0.0")
			testhelper.CloneRepository(t, remoteDir)

			configPath := filepath.Join("./", librarianConfigPath)
			cfg := &config.Config{
				Language: languageFake,
				Default:  &config.Default{},
				Release: &config.Release{
					Remote: "origin",
					Branch: "main",
				},
				Sources: &config.Sources{
					Googleapis: &config.Source{
						Commit: "9fcfbea0aa5b50fa22e190faceb073d74504172b",
						SHA256: "81e6057ffd85154af5268c2c3c8f2408745ca0f7fa03d43c68f4847f31eb5f98",
					},
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
			if test.wantErr == errReleaseConfigEmpty {
				cfg.Release = nil
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
				updatedConfig, err := yaml.Read[config.Config](configPath)
				if err != nil {
					t.Fatal(err)
				}
				gotVersions := make(map[string]string)
				for _, lib := range updatedConfig.Libraries {
					gotVersions[lib.Name] = lib.Version
				}
				if diff := cmp.Diff(test.wantVersions, gotVersions); diff != "" {
					t.Errorf("mismatch in versions (-want +got):\n%s", diff)
				}
				// Update original config versions to expected versions to compare entire config.
				for _, lib := range cfg.Libraries {
					if wantVersion, ok := test.wantVersions[lib.Name]; ok {
						lib.Version = wantVersion
					}
				}
				if diff := cmp.Diff(cfg, updatedConfig); diff != "" {
					t.Errorf("mismatch in config (-want +got):\n%s", diff)
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

func TestRelease(t *testing.T) {

	tests := []struct {
		name    string
		srcPath string
		version string
		lastTag string
	}{
		{
			name:    "library released",
			srcPath: "src/storage",
			version: "1.2.3",
		},
	}
	testhelper.RequireCommand(t, "git")
	remoteDir := testhelper.SetupRepoWithChange(t, "v1.2.2")
	testhelper.CloneRepository(t, remoteDir)
	cfg := &config.Config{
		Language: languageFake,
		Release: &config.Release{
			Remote: "origin",
			Branch: "main",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			libConfg := &config.Library{}
			err := releaseLibrary(t.Context(), cfg, libConfg, test.srcPath, test.lastTag, "git", "")
			if err != nil {
				t.Fatalf("releaseLibrary() error = %v", err)
			}
			if libConfg.Version != test.version {
				t.Errorf("library %q version mismatch: want %q, got %q", libConfg.Name, test.version, libConfg.Version)
			}
		})

	}
}

func TestReleaseAll(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name        string
		libName     string
		dir         string
		skipPublish bool
		wantVersion string
	}{
		{
			name:        "library has changes",
			libName:     "google-cloud-storage",
			dir:         "src/storage",
			wantVersion: "1.2.3",
			skipPublish: false,
		},
		{
			name:        "library does not have any changes",
			libName:     "gax-internal",
			dir:         "src/gax-internal",
			wantVersion: "1.2.2",
			skipPublish: false,
		},
		{
			name:        "library does not have any changes on shared directory prefix",
			libName:     "gax-internal",
			dir:         "src/stor",
			wantVersion: "1.2.2",
			skipPublish: false,
		},
		{
			name:        "library has changes but skipPublish is true",
			libName:     "google-cloud-storage",
			dir:         "src/storage",
			wantVersion: "1.2.2",
			skipPublish: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tag := "v1.2.3"
			config := &config.Config{
				Language: languageFake,
				Libraries: []*config.Library{
					{
						Name:        test.libName,
						Version:     "1.2.2",
						Output:      test.dir,
						SkipPublish: test.skipPublish,
					},
				},
				Release: &config.Release{
					Remote:         "origin",
					Branch:         "main",
					IgnoredChanges: []string{},
				},
			}
			remoteDir := testhelper.SetupRepoWithChange(t, tag)
			testhelper.CloneRepository(t, remoteDir)
			err := releaseAll(t.Context(), config, tag, "git", "")
			if err != nil {
				t.Fatal(err)
			}
			if config.Libraries[0].Version != test.wantVersion {
				t.Errorf("got version %s, want %s", config.Libraries[0].Version, test.wantVersion)
			}
		})
	}
}
