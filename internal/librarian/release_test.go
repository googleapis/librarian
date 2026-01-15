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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

// testUnusedStringParam is used to fill the spot of a string parameter that
// won't be provided in the test, because the test does not exercise the
// functionality related to said parameter. It is an intentional signal
// rather than an ambiguous empty string.
const testUnusedStringParam = ""

func TestReleaseCommand(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name        string
		args        []string
		cfg         *config.Config
		previewCfg  *config.Config
		withChanges []string
		wantCfg     *config.Config
	}{
		{
			name:        "library name",
			args:        []string{"librarian", "release", sample.Lib1Name},
			cfg:         sample.Config(),
			withChanges: []string{filepath.Join(sample.Lib1Output, "src", "lib.rs")},
			wantCfg: func() *config.Config {
				c := sample.Config()

				c.Libraries[0].Version = sample.NextVersion
				return c
			}(),
		},
		{
			name: "all flag all have changes",
			args: []string{"librarian", "release", "--all"},
			cfg:  sample.Config(),
			withChanges: []string{
				filepath.Join(sample.Lib1Output, "src", "lib.rs"),
				filepath.Join(sample.Lib2Output, "src", "lib.rs"),
			},
			wantCfg: func() *config.Config {
				c := sample.Config()

				c.Libraries[0].Version = sample.NextVersion
				c.Libraries[1].Version = sample.NextVersion
				return c
			}(),
		},
		{
			name:        "all flag 1 has changes",
			args:        []string{"librarian", "release", "--all"},
			cfg:         sample.Config(),
			withChanges: []string{filepath.Join(sample.Lib1Output, "src", "lib.rs")},
			wantCfg: func() *config.Config {
				c := sample.Config()

				c.Libraries[0].Version = sample.NextVersion
				return c
			}(),
		},
		{
			name:        "preview library released",
			args:        []string{"librarian", "release", sample.Lib1Name},
			withChanges: []string{filepath.Join(sample.Lib1Output, "src", "lib.rs")},
			cfg:         sample.Config(),
			previewCfg:  sample.PreviewConfig(),
			wantCfg: func() *config.Config {
				c := sample.PreviewConfig()

				c.Libraries[0].Version = sample.NextPreviewPrereleaseVersion
				return c
			}(),
		},
		{
			name: "all preview libraries released",
			args: []string{"librarian", "release", "--all"},
			withChanges: []string{
				filepath.Join(sample.Lib1Output, "src", "lib.rs"),
				filepath.Join(sample.Lib2Output, "src", "lib.rs"),
			},
			cfg:        sample.Config(),
			previewCfg: sample.PreviewConfig(),
			wantCfg: func() *config.Config {
				c := sample.PreviewConfig()

				c.Libraries[0].Version = sample.NextPreviewPrereleaseVersion
				c.Libraries[1].Version = sample.NextPreviewPrereleaseVersion
				return c
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := testhelper.SetupOptions{
				Clone:       test.cfg.Release.Branch,
				Config:      test.cfg,
				Tag:         sample.InitialTag,
				WithChanges: test.withChanges,
			}
			// Test should target the preview branch instead of default main.
			if test.previewCfg != nil {
				opts.Clone = test.previewCfg.Release.Branch
				opts.PreviewOptions = &testhelper.SetupOptions{
					Config:      test.previewCfg,
					WithChanges: test.withChanges,
					Tag:         sample.InitialPreviewTag,
				}
			}
			testhelper.Setup(t, opts)

			err := Run(t.Context(), test.args...)
			if err != nil {
				t.Fatal(err)
			}

			// The CLI command writes the updated config to disc, so we need to
			// read it to check the result of the command.
			updatedConfig, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			// Libs need to be sorted because [yaml.Read] and [yaml.Write]
			// do not guarantee order.
			byLibraryName := func(a, b *config.Library) int {
				return strings.Compare(a.Name, b.Name)
			}
			slices.SortFunc(test.wantCfg.Libraries, byLibraryName)
			slices.SortFunc(updatedConfig.Libraries, byLibraryName)

			if diff := cmp.Diff(test.wantCfg, updatedConfig); diff != "" {
				t.Errorf("mismatch in config (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReleaseCommand_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name    string
		args    []string
		cfg     *config.Config
		dirty   bool
		wantErr error
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "release"},
			wantErr: errMissingLibraryOrAllFlag,
		},
		{
			name:    "library name and all flag",
			args:    []string{"librarian", "release", "foo", "--all"},
			wantErr: errBothLibraryAndAllFlag,
		},
		{
			name:    "missing librarian yaml file",
			args:    []string{"librarian", "release", "--all"},
			wantErr: errNoYaml,
		},
		{
			name:    "local repo is dirty",
			args:    []string{"librarian", "release", "--all"},
			cfg:     sample.Config(),
			dirty:   true,
			wantErr: git.ErrGitStatusUnclean,
		},
		{
			name: "release config empty",
			args: []string{"librarian", "release", "--all"},
			cfg: func() *config.Config {
				c := sample.Config()

				c.Release = nil
				return c
			}(),
			wantErr: errReleaseConfigEmpty,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testhelper.Setup(t, testhelper.SetupOptions{
				Clone:  "main",
				Config: test.cfg,
				Dirty:  test.dirty,
			})

			err := Run(t.Context(), test.args...)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Run() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestLibraryByName(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		cfg         *config.Config
		want        *config.Library
		wantErr     error
	}{
		{
			name:        "find_a_library",
			libraryName: "example-library",
			cfg: &config.Config{
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
			cfg:         &config.Config{},
			wantErr:     errLibraryNotFound,
		},
		{
			name:        "does_not_find_a_library",
			libraryName: "non-existent-library",
			cfg: &config.Config{
				Libraries: []*config.Library{
					{Name: "example-library"},
					{Name: "another-library"},
				},
			},
			wantErr: errLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := libraryByName(test.cfg, test.libraryName)
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

func TestReleaseLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	tests := []struct {
		name        string
		cfg         *config.Config
		previewCfg  *config.Config
		wantVersion string
	}{
		{
			name:        "library released",
			cfg:         sample.Config(),
			wantVersion: sample.NextVersion,
		},
		{
			name:        "preview library released",
			cfg:         sample.Config(),
			previewCfg:  sample.PreviewConfig(),
			wantVersion: sample.NextPreviewPrereleaseVersion,
		},
		{
			name: "preview library catches up to main",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].Version = sample.NextVersion
				return c
			}(),
			previewCfg:  sample.PreviewConfig(),
			wantVersion: sample.NextPreviewCoreVersion,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			targetCfg := test.cfg
			opts := testhelper.SetupOptions{
				Clone:  test.cfg.Release.Branch,
				Config: test.cfg,
			}
			// Test should target the preview branch instead of default main.
			if test.previewCfg != nil {
				targetCfg = test.previewCfg
				opts.Clone = test.previewCfg.Release.Branch
				opts.PreviewOptions = &testhelper.SetupOptions{
					Config: test.previewCfg,
				}
			}
			testhelper.Setup(t, opts)

			targetLibCfg := targetCfg.Libraries[0]
			// Unused string param: lastTag.
			err := releaseLibrary(t.Context(), targetCfg, targetLibCfg, testUnusedStringParam, "git")
			if err != nil {
				t.Fatalf("releaseLibrary() error = %v", err)
			}
			if targetLibCfg.Version != test.wantVersion {
				t.Errorf("library %q version mismatch: want %q, got %q", targetLibCfg.Name, test.wantVersion, targetLibCfg.Version)
			}
		})

	}
}

func TestReleaseAll(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name        string
		cfg         *config.Config
		previewCfg  *config.Config
		withChanges []string
		skipPublish bool
		wantVersion string
	}{
		{
			name:        "library has changes",
			cfg:         sample.Config(),
			withChanges: []string{filepath.Join(sample.Lib1Output, "src", "lib.rs")},
			wantVersion: sample.NextVersion,
		},
		{
			name:        "library does not have any changes",
			cfg:         sample.Config(),
			wantVersion: sample.InitialVersion,
		},
		{
			name: "library has changes but skipPublish is true",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].SkipPublish = true
				return c
			}(),
			withChanges: []string{filepath.Join(sample.Lib1Output, "src", "lib.rs")},
			wantVersion: sample.InitialVersion,
		},
		{
			name:        "preview library does not have any changes",
			cfg:         sample.Config(),
			previewCfg:  sample.PreviewConfig(),
			wantVersion: sample.InitialPreviewVersion,
		},
		{
			name:        "preview library has changes",
			withChanges: []string{filepath.Join(sample.Lib1Output, "src", "lib.rs")},
			cfg:         sample.Config(),
			previewCfg:  sample.PreviewConfig(),
			wantVersion: sample.NextPreviewPrereleaseVersion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			targetCfg := test.cfg
			sinceTag := sample.InitialTag
			opts := testhelper.SetupOptions{
				Clone:       test.cfg.Release.Branch,
				Config:      test.cfg,
				Tag:         sample.InitialTag,
				WithChanges: test.withChanges,
			}
			// Test should target the preview branch instead of default main.
			if test.previewCfg != nil {
				targetCfg = test.previewCfg
				sinceTag = sample.InitialPreviewTag
				opts.Clone = test.previewCfg.Release.Branch
				opts.PreviewOptions = &testhelper.SetupOptions{
					Config:      test.previewCfg,
					WithChanges: test.withChanges,
					Tag:         sample.InitialPreviewTag,
				}
			}
			testhelper.Setup(t, opts)

			err := releaseAll(t.Context(), targetCfg, sinceTag, "git")
			if err != nil {
				t.Fatal(err)
			}
			// releaseAll directly modifies the config provided, so we use it as
			// our "got".
			gotVersion := targetCfg.Libraries[0].Version
			if gotVersion != test.wantVersion {
				t.Errorf("got version %s, want %s", gotVersion, test.wantVersion)
			}
		})
	}
}

func TestPostRelease(t *testing.T) {
	fakeCargo := filepath.Join(t.TempDir(), "fake-cargo")
	for _, test := range []struct {
		name    string
		setup   func()
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "rust language runs cargo update",
			setup: func() {
				script := "#!/bin/sh\nexit 0"
				if err := os.WriteFile(fakeCargo, []byte(script), 0755); err != nil {
					t.Fatal(err)
				}
			},
			cfg: &config.Config{
				Language: languageRust,
				Release: &config.Release{
					Preinstalled: map[string]string{
						"cargo": fakeCargo,
					},
				},
			},
		},
		{
			name: "rust language runs cargo update fails",
			setup: func() {
				script := "#!/bin/sh\nexit 1"
				if err := os.WriteFile(fakeCargo, []byte(script), 0755); err != nil {
					t.Fatal(err)
				}
			},
			cfg: &config.Config{
				Language: languageRust,
				Release: &config.Release{
					Preinstalled: map[string]string{
						"cargo": fakeCargo,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "non-rust language does nothing",
			cfg: &config.Config{
				Language: languageFake,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup()
			}

			err := postRelease(t.Context(), test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("postRelease() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestDeriveNextVersion(t *testing.T) {
	for _, test := range []struct {
		name        string
		cfg         *config.Config
		versionOpts semver.DeriveNextOptions
		wantVersion string
	}{
		{
			name: "rust library next non-GA version",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Language = languageRust
				c.Libraries[0].Version = sample.RustNonGAVersion
				return c
			}(),
			versionOpts: languageVersioningOptions[languageRust],
			wantVersion: sample.RustNextNonGAVersion,
		},
		{
			name: "rust library next GA version",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Language = languageRust
				return c
			}(),
			versionOpts: languageVersioningOptions[languageRust],
			wantVersion: sample.NextVersion,
		},
		{
			name:        "default semver options next GA version",
			cfg:         sample.Config(),
			wantVersion: sample.NextVersion,
		},
	} {
		got, err := deriveNextVersion(t.Context(), "git", test.cfg, test.cfg.Libraries[0], test.versionOpts)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.wantVersion {
			t.Errorf("got version %s, want %s", got, test.wantVersion)
		}
	}
}

func TestLoadBranchLibraryVersion(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	want := sample.InitialVersion
	testhelper.Setup(t, testhelper.SetupOptions{
		Clone: "main",
		Config: &config.Config{
			Libraries: []*config.Library{{Name: sample.Lib1Name, Version: want}},
		},
	})

	got, err := loadBranchLibraryVersion(t.Context(), "git", "origin", "main", sample.Lib1Name)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got version %s, want %s", got, want)
	}
}
