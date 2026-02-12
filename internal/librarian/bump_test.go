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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
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

func TestBumpCommand(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	lib1Change := filepath.Join(sample.Lib1Output, "src", "lib.rs")
	lib2Change := filepath.Join(sample.Lib2Output, "src", "lib.rs")

	for _, test := range []struct {
		name         string
		args         []string
		withChanges  []string
		wantVersions map[string]string
	}{
		{
			name:         "library name",
			args:         []string{"librarian", "bump", sample.Lib1Name},
			withChanges:  []string{lib1Change},
			wantVersions: map[string]string{sample.Lib1Name: sample.NextVersion},
		},
		{
			name:         "library name and explicit version",
			args:         []string{"librarian", "bump", sample.Lib1Name, "--version=1.2.3"},
			withChanges:  []string{lib1Change},
			wantVersions: map[string]string{sample.Lib1Name: "1.2.3"},
		},
		{
			name:        "all flag all have changes",
			args:        []string{"librarian", "bump", "--all"},
			withChanges: []string{lib1Change, lib2Change},
			wantVersions: map[string]string{
				sample.Lib1Name: sample.NextVersion,
				sample.Lib2Name: sample.NextVersion,
			},
		},
		{
			name: "all flag no changes",
			args: []string{"librarian", "bump", "--all"},
			wantVersions: map[string]string{
				sample.Lib1Name: sample.InitialVersion,
				sample.Lib2Name: sample.InitialVersion,
			},
		},
		{
			name:         "all flag 1 has changes",
			args:         []string{"librarian", "bump", "--all"},
			withChanges:  []string{lib1Change},
			wantVersions: map[string]string{sample.Lib1Name: sample.NextVersion},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			opts := testhelper.SetupOptions{
				Clone:       cfg.Release.Branch,
				Config:      cfg,
				Tags:        []string{sample.InitialLib1Tag, sample.InitialLib2Tag},
				WithChanges: test.withChanges,
			}
			testhelper.Setup(t, opts)

			if err := Run(t.Context(), test.args...); err != nil {
				t.Fatal(err)
			}

			got, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, lib := range got.Libraries {
				if want, ok := test.wantVersions[lib.Name]; ok {
					if lib.Version != want {
						t.Errorf("library %s: got version %q, want %q", lib.Name, lib.Version, want)
					}
				}
			}
		})
	}
}

func TestBumpCommandDeriveOutput(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	cfg := sample.Config()
	cfg.Default.Output = sample.Lib1Output
	cfg.Libraries[0].Output = ""

	testhelper.Setup(t, testhelper.SetupOptions{
		Clone:       cfg.Release.Branch,
		Config:      cfg,
		Tags:        []string{sample.InitialLib1Tag},
		WithChanges: []string{filepath.Join(sample.Lib1Output, "src", "lib.rs")},
	})

	if err := Run(t.Context(), "librarian", "bump", sample.Lib1Name); err != nil {
		t.Fatal(err)
	}

	got, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, lib := range got.Libraries {
		if lib.Name == sample.Lib1Name && lib.Version != sample.NextVersion {
			t.Errorf("got version %q, want %q", lib.Version, sample.NextVersion)
		}
	}
}

func TestBumpCommand_Error(t *testing.T) {
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
			args:    []string{"librarian", "bump"},
			wantErr: errMissingLibraryOrAllFlag,
		},
		{
			name:    "library name and all flag",
			args:    []string{"librarian", "bump", "foo", "--all"},
			wantErr: errBothLibraryAndAllFlag,
		},
		{
			name:    "version flag and all flag",
			args:    []string{"librarian", "bump", "--version=1.2.3", "--all"},
			wantErr: errBothVersionAndAllFlag,
		},
		{
			name:    "missing librarian yaml file",
			args:    []string{"librarian", "bump", "--all"},
			wantErr: errConfigNotFound,
		},
		{
			name:    "local repo is dirty",
			args:    []string{"librarian", "bump", "--all"},
			cfg:     sample.Config(),
			dirty:   true,
			wantErr: git.ErrGitStatusUnclean,
		},
		{
			name: "release config empty",
			args: []string{"librarian", "bump", "--all"},
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

func TestFindLibrary(t *testing.T) {
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
			wantErr:     ErrLibraryNotFound,
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
			wantErr: ErrLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findLibrary(test.cfg, test.libraryName)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("findLibrary(%q): %v", test.libraryName, err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunBump_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "git")

	tests := []struct {
		name            string
		libraryName     string
		versionOverride string
		wantErr         error
	}{
		{
			name:            "invalid version override",
			libraryName:     sample.Lib1Name,
			versionOverride: "0.9.0",
			wantErr:         semver.ErrInvalidNextVersion,
		},
		{
			name:        "library not found",
			libraryName: "not-found",
			wantErr:     ErrLibraryNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			opts := testhelper.SetupOptions{
				Clone:  cfg.Release.Branch,
				Config: cfg,
			}
			testhelper.Setup(t, opts)

			gotErr := runBump(t.Context(), cfg, false, test.libraryName, test.versionOverride)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("bumpLibrary() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestBumpLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	tests := []struct {
		name            string
		cfg             *config.Config
		versionOverride string
		wantVersion     string
	}{
		{
			name:        "library released",
			cfg:         sample.Config(),
			wantVersion: sample.NextVersion,
		},
		{
			name: "version override",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].Version = "1.3.0"
				return c
			}(),
			versionOverride: "2.0.0",
			wantVersion:     "2.0.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := testhelper.SetupOptions{
				Clone:  test.cfg.Release.Branch,
				Config: test.cfg,
			}
			testhelper.Setup(t, opts)

			targetLibCfg := test.cfg.Libraries[0]
			err := bumpLibrary(t.Context(), test.cfg, targetLibCfg, "git", test.versionOverride)
			if err != nil {
				t.Fatalf("bumpLibrary() error = %v", err)
			}
			if targetLibCfg.Version != test.wantVersion {
				t.Errorf("library %q version mismatch: want %q, got %q", targetLibCfg.Name, test.wantVersion, targetLibCfg.Version)
			}
		})
	}
}

func TestBumpLibrary_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	tests := []struct {
		name            string
		cfg             *config.Config
		versionOverride string
		wantErr         error
	}{
		{
			name:            "invalid version override",
			cfg:             sample.Config(),
			versionOverride: "0.9.0",
			wantErr:         semver.ErrInvalidNextVersion,
		},
		{
			name: "unsupported language",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Language = languageRust
				return c
			}(),
			versionOverride: "2.0.0",
			// There's no specific error we can specify; just test for non-nil.
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := testhelper.SetupOptions{
				Clone:  test.cfg.Release.Branch,
				Config: test.cfg,
			}
			testhelper.Setup(t, opts)

			targetLibCfg := test.cfg.Libraries[0]
			gotErr := bumpLibrary(t.Context(), test.cfg, targetLibCfg, "git", test.versionOverride)
			if gotErr == nil {
				t.Fatal("expected error; got nil")
			}
			if test.wantErr != nil && !errors.Is(gotErr, test.wantErr) {
				t.Errorf("bumpLibrary() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestFindLibrariesToBump(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	lib1Change := filepath.Join(sample.Lib1Output, "src", "lib.rs")
	lib2Change := filepath.Join(sample.Lib2Output, "src", "lib.rs")
	for _, test := range []struct {
		name        string
		all         bool
		libraryName string
		// withChanges is a list of files to modify and then commit; this is
		// used when that's all that's required.
		withChanges []string
		// setup is a function executed after setting up the repo (including
		// after applying withChanges) so that we can make more custom changes
		// such as "more tags after making changes".
		setup     func(*testing.T, *config.Config)
		wantNames []string
	}{
		{
			name:        "library specified directly",
			libraryName: sample.Lib2Name,
			wantNames:   []string{sample.Lib2Name},
		},
		{
			name:        "library specified directly, ignored skip",
			libraryName: sample.Lib2Name,
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[1].SkipPublish = true
				writeConfigAndCommit(t, cfg)
			},
			wantNames: []string{sample.Lib2Name},
		},
		{
			name:        "library specified directly, ignored empty version",
			libraryName: sample.Lib2Name,
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[1].Version = ""
				writeConfigAndCommit(t, cfg)
			},
			wantNames: []string{sample.Lib2Name},
		},
		{
			name:        "one library has changes",
			all:         true,
			withChanges: []string{lib1Change},
			wantNames:   []string{sample.Lib1Name},
		},
		{
			name:        "one library has changes, but it's skipped",
			all:         true,
			withChanges: []string{lib1Change},
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].SkipPublish = true
				writeConfigAndCommit(t, cfg)
			},
			wantNames: []string{},
		},
		{
			name:        "one library has changes, but it's unreleased",
			all:         true,
			withChanges: []string{lib1Change},
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = ""
				writeConfigAndCommit(t, cfg)
			},
			wantNames: []string{},
		},
		{
			name: "no libraries have changes",
			all:  true,
			setup: func(t *testing.T, cfg *config.Config) {
				writeFileAndCommit(t, "unrelated-README.txt", []byte("test"), "non-library-related-commit")
			},
			wantNames: []string{},
		},
		{
			name:        "multiple libraries have changes",
			all:         true,
			withChanges: []string{lib1Change, lib2Change},
			wantNames:   []string{sample.Lib1Name, sample.Lib2Name},
		},
		{
			name:        "multiple libraries have changes, one is skipped",
			all:         true,
			withChanges: []string{lib1Change, lib2Change},
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].SkipPublish = true
				writeConfigAndCommit(t, cfg)
			},
			wantNames: []string{sample.Lib2Name},
		},
		{
			name:        "two libraries have been changed but one has already been released",
			all:         true,
			withChanges: []string{lib1Change, lib2Change},
			wantNames:   []string{sample.Lib1Name},
			setup: func(t *testing.T, cfg *config.Config) {
				// Simulate the release of sample.Lib2: bump the version,
				// commit the config, tag it.
				cfg.Libraries[1].Version = sample.NextVersion
				writeConfigAndCommit(t, cfg)
				tagName := formatTagName(cfg.Default.TagFormat, cfg.Libraries[1])
				git.Tag(t.Context(), "git", tagName, "HEAD")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			opts := testhelper.SetupOptions{
				Config:      cfg,
				Tags:        []string{sample.InitialLib1Tag, sample.InitialLib2Tag},
				WithChanges: test.withChanges,
			}
			testhelper.Setup(t, opts)
			if test.setup != nil {
				test.setup(t, cfg)
			}

			gotLibraries, err := findLibrariesToBump(t.Context(), cfg, "git", test.all, test.libraryName)
			if err != nil {
				t.Fatal(err)
			}
			gotNames := []string{}
			for _, gotLibrary := range gotLibraries {
				gotNames = append(gotNames, gotLibrary.Name)
			}
			if diff := cmp.Diff(test.wantNames, gotNames); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindLibrariesToBump_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	for _, test := range []struct {
		name        string
		all         bool
		libraryName string
		setup       func(*testing.T, *config.Config)
		wantErr     error
	}{
		{
			name:        "specified library does not exist",
			libraryName: "non-existent",
			wantErr:     ErrLibraryNotFound,
		},
		{
			name: "library has no tag for last release",
			all:  true,
			setup: func(t *testing.T, cfg *config.Config) {
				// Simulate half a release of sample.Lib2: bump the version,
				// commit the config, but fail to tag.
				cfg.Libraries[1].Version = sample.NextVersion
				writeConfigAndCommit(t, cfg)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			opts := testhelper.SetupOptions{
				Config: cfg,
				Tags:   []string{sample.InitialLib1Tag, sample.InitialLib2Tag},
			}
			testhelper.Setup(t, opts)
			if test.setup != nil {
				test.setup(t, cfg)
			}

			_, gotErr := findLibrariesToBump(t.Context(), cfg, "git", test.all, test.libraryName)
			if gotErr == nil {
				t.Fatal("expected error; got nil")
			}
			if test.wantErr != nil && !errors.Is(gotErr, test.wantErr) {
				t.Errorf("findLibrariesToBump() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestPostBump(t *testing.T) {
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

			err := postBump(t.Context(), test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("postBump() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestDeriveNextVersion(t *testing.T) {
	for _, test := range []struct {
		name            string
		cfg             *config.Config
		previewCfg      *config.Config
		versionOpts     semver.DeriveNextOptions
		versionOverride string
		wantVersion     string
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
		{
			name: "version override, unreleased library",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].Version = ""
				return c
			}(),
			versionOverride: "1.0.0-override.1",
			wantVersion:     "1.0.0-override.1",
		},
		{
			name: "version override, already released library, later version",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].Version = "1.2.2"
				return c
			}(),
			versionOverride: "1.2.3",
			wantVersion:     "1.2.3",
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
	} {
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

			got, err := deriveNextVersion(t.Context(), "git", targetCfg, targetCfg.Libraries[0], test.versionOpts, test.versionOverride)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.wantVersion {
				t.Errorf("got version %s, want %s", got, test.wantVersion)
			}
		})
	}
}

func TestDeriveNextVersion_Error(t *testing.T) {
	for _, test := range []struct {
		name            string
		cfg             *config.Config
		versionOpts     semver.DeriveNextOptions
		versionOverride string
	}{
		{
			name: "version override, already released library, existing version",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].Version = "1.2.2"
				return c
			}(),
			versionOverride: "1.2.2",
		},
		{
			name: "version override, already released library, earlier version",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].Version = "1.2.2"
				return c
			}(),
			versionOverride: "1.2.1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := deriveNextVersion(t.Context(), "git", test.cfg, test.cfg.Libraries[0], test.versionOpts, test.versionOverride)
			if err == nil {
				t.Errorf("DeriveNextVersion() expected error; returned no error and version %s", got)
			}
		})
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

func TestFindReleasedLibraries(t *testing.T) {
	cfgBefore := &config.Config{
		Libraries: []*config.Library{
			{Name: "Unchanged", Version: "1.2.3"},
			{Name: "PatchBump", Version: "1.2.3"},
			{Name: "MinorBump", Version: "1.2.3"},
			{Name: "MajorBump", Version: "1.2.3"},
			{Name: "PreviewBump", Version: "1.0.0-beta.1"},
			{Name: "StaysUnversioned"},
			{Name: "Deleted", Version: "1.2.3"},
		},
	}
	cfgAfter := &config.Config{
		Libraries: []*config.Library{
			{Name: "Unchanged", Version: "1.2.3"},
			{Name: "PatchBump", Version: "1.2.4"},
			{Name: "MinorBump", Version: "1.3.0"},
			{Name: "MajorBump", Version: "2.0"},
			{Name: "PreviewBump", Version: "1.0.0-beta.2"},
			{Name: "StaysUnversioned"},
			{Name: "AddedUnversioned", Version: ""},
			{Name: "AddedWithVersion", Version: "1.0.0"},
		},
	}
	got, err := findReleasedLibraries(cfgBefore, cfgAfter)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"PatchBump", "MinorBump", "MajorBump", "PreviewBump", "AddedWithVersion"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindReleasedLibraries_Error(t *testing.T) {
	for _, test := range []struct {
		name      string
		cfgBefore *config.Config
		cfgAfter  *config.Config
	}{
		{
			name: "regression (version decreases)",
			cfgBefore: &config.Config{
				Libraries: []*config.Library{
					{Name: "MinorBump", Version: "1.3.0"},
					{Name: "Regression", Version: "1.3.0"},
				},
			},
			cfgAfter: &config.Config{
				Libraries: []*config.Library{
					{Name: "MinorBump", Version: "1.4.0"},
					{Name: "Regression", Version: "1.2.0"},
				},
			},
		},
		{
			name: "regression (version removed)",
			cfgBefore: &config.Config{
				Libraries: []*config.Library{
					{Name: "MinorBump", Version: "1.3.0"},
					{Name: "Regression", Version: "1.3.0"},
				},
			},
			cfgAfter: &config.Config{
				Libraries: []*config.Library{
					{Name: "MinorBump", Version: "1.4.0"},
					{Name: "Regression", Version: ""},
				},
			},
		},
		{
			name: "new library with invalid version",
			cfgBefore: &config.Config{
				Libraries: []*config.Library{
					{Name: "MinorBump", Version: "1.3.0"},
				},
			},
			cfgAfter: &config.Config{
				Libraries: []*config.Library{
					{Name: "MinorBump", Version: "1.4.0"},
					{Name: "NewLibraryInvalidVersion", Version: "invalid"},
				},
			},
		},
		{
			name: "existing library with invalid version",
			cfgBefore: &config.Config{
				Libraries: []*config.Library{
					{Name: "BecomesInvalid", Version: "1.3.0"},
				},
			},
			cfgAfter: &config.Config{
				Libraries: []*config.Library{
					{Name: "BecomesInvalid", Version: "invalid"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := findReleasedLibraries(test.cfgBefore, test.cfgAfter)
			if err == nil {
				t.Errorf("findReleasedLibraries() expected error; returned no error")
			}
		})
	}
}

func TestFindLatestReleaseCommitHash(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	for _, test := range []struct {
		name            string
		setup           func(cfg *config.Config)
		libraryName     string
		wantCommitCount int
		wantCommitIndex int // Commit index in the log: HEAD=0, HEAD~=1 etc
	}{
		{
			name: "HEAD commit releases, match any release",
			setup: func(cfg *config.Config) {
				// 2 commits in addition to the two in Setup:
				// - Chore commit with a modified readme
				// - Release commit with the first library version bumped
				writeReadmeAndCommit(t, "modified readme")
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			wantCommitCount: 4,
			wantCommitIndex: 0,
		},
		{
			name: "HEAD~ commit, match any release",
			setup: func(cfg *config.Config) {
				// 3 commits in addition to the two in Setup:
				// - Chore commit with a modified readme
				// - Release commit with the first library version bumped
				// - Chore commit with another modified readme
				writeReadmeAndCommit(t, "modified readme")
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				writeReadmeAndCommit(t, "modified readme again")
			},
			wantCommitCount: 5,
			wantCommitIndex: 1,
		},
		{
			name: "match specific library",
			setup: func(cfg *config.Config) {
				// 4 commits in addition to the two in Setup:
				// - Chore commit with a modified readme
				// - Release commit with the first library version bumped
				// - Chore commit with another modified readme
				// - Release commit with the second library version bumped
				// (We're looking for the first library, so effectively HEAD~2)
				writeReadmeAndCommit(t, "modified readme")
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				writeReadmeAndCommit(t, "modified readme again")
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			libraryName:     sample.Lib1Name,
			wantCommitCount: 6,
			wantCommitIndex: 2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Libraries: []*config.Library{
					{Name: sample.Lib1Name, Version: "1.0.0"},
					{Name: sample.Lib2Name, Version: "1.2.0"},
				},
			}
			opts := testhelper.SetupOptions{
				Config: cfg,
			}
			testhelper.Setup(t, opts)
			test.setup(cfg)
			commits, err := git.FindCommitsForPath(t.Context(), "git", ".")
			if err != nil {
				t.Fatal(err)
			}
			// This is effectively validating that the setup has worked as expected.
			if test.wantCommitCount != len(commits) {
				t.Fatalf("expected setup to create %d commits; got %d", test.wantCommitCount, len(commits))
			}
			got, err := findLatestReleaseCommitHash(t.Context(), "git", test.libraryName)
			if err != nil {
				t.Fatal(err)
			}
			if commits[test.wantCommitIndex] != got {
				// Deliberately not using diff as the hashes are basically opaque
				t.Errorf("findLatestReleaseCommitHash: got = %s; want = %s; all commits = %s", got, commits[test.wantCommitIndex], strings.Join(commits, ", "))
			}
		})
	}
}

func TestFindLatestReleaseCommitHash_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	for _, test := range []struct {
		name                      string
		setup                     func(cfg *config.Config)
		libraryName               string
		wantReleaseCommitNotFound bool
	}{
		{
			name: "no releases",
			setup: func(cfg *config.Config) {
				// We're modifying the description, but that isn't a release.
				cfg.Libraries[0].DescriptionOverride = "modified description"
				writeConfigAndCommit(t, cfg)
			},
			wantReleaseCommitNotFound: true,
		},
		{
			name: "no library with given name",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			libraryName:               "nonexistent",
			wantReleaseCommitNotFound: true,
		},
		{
			name: "release, but not for the specified library",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			libraryName:               sample.Lib2Name,
			wantReleaseCommitNotFound: true,
		},
		{
			name: "invalid release",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "invalid"
				writeConfigAndCommit(t, cfg)
			},
		},
		{
			name: "invalid config file",
			setup: func(cfg *config.Config) {
				writeFileAndCommit(t, librarianConfigPath, []byte("not a config file"), "broke config file")
			},
		},
		{
			name: "deleted config file",
			setup: func(cfg *config.Config) {
				if err := os.Remove(librarianConfigPath); err != nil {
					t.Fatal(err)
				}
				if err := command.Run(t.Context(), "git", "commit", "-m", "deleted config file", "."); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "provoke git failure looking for commits",
			setup: func(cfg *config.Config) {
				if err := os.Rename(".git", "notgit"); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Libraries: []*config.Library{
					{Name: sample.Lib1Name, Version: "1.0.0"},
					{Name: sample.Lib2Name, Version: "1.2.0"},
				},
			}
			opts := testhelper.SetupOptions{
				Config: cfg,
			}
			testhelper.Setup(t, opts)
			test.setup(cfg)
			got, err := findLatestReleaseCommitHash(t.Context(), "git", test.libraryName)
			if err == nil {
				t.Errorf("expected error; succeeded with hash %s", got)
			}
			if errors.Is(err, errReleaseCommitNotFound) != test.wantReleaseCommitNotFound {
				t.Errorf("findLatestReleaseCommitHash() error = %v, wantReleaseCommitNotFound = %v", err, test.wantReleaseCommitNotFound)
			}
		})
	}
}

func TestLegacyRustBumpLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	tests := []struct {
		name            string
		cfg             *config.Config
		previewCfg      *config.Config
		versionOverride string
		wantVersion     string
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
		{
			name: "version override",
			cfg: func() *config.Config {
				c := sample.Config()
				c.Libraries[0].Version = "1.3.0"
				return c
			}(),
			versionOverride: "2.0.0",
			wantVersion:     "2.0.0",
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
			err := legacyRustBumpLibrary(t.Context(), targetCfg, targetLibCfg, testUnusedStringParam, "git", test.versionOverride)
			if err != nil {
				t.Fatalf("legacyRustBumpLibrary() error = %v", err)
			}
			if targetLibCfg.Version != test.wantVersion {
				t.Errorf("library %q version mismatch: want %q, got %q", targetLibCfg.Name, test.wantVersion, targetLibCfg.Version)
			}
		})

	}
}

func TestLegacyRustBump(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	lib1Change := filepath.Join(sample.Lib1Output, "src", "lib.rs")
	lib2Change := filepath.Join(sample.Lib2Output, "src", "lib.rs")

	for _, test := range []struct {
		name            string
		libraryName     string
		versionOverride string
		all             bool
		withChanges     []string
		wantVersions    map[string]string
	}{
		{
			name:         "library name",
			libraryName:  sample.Lib1Name,
			withChanges:  []string{lib1Change},
			wantVersions: map[string]string{sample.Lib1Name: sample.NextVersion},
		},
		{
			name:            "library name and explicit version",
			libraryName:     sample.Lib1Name,
			versionOverride: "1.2.3",
			withChanges:     []string{lib1Change},
			wantVersions:    map[string]string{sample.Lib1Name: "1.2.3"},
		},
		{
			name:        "all flag all have changes",
			all:         true,
			withChanges: []string{lib1Change, lib2Change},
			wantVersions: map[string]string{
				sample.Lib1Name: sample.NextVersion,
				sample.Lib2Name: sample.NextVersion,
			},
		},
		{
			name:         "all flag 1 has changes",
			all:          true,
			withChanges:  []string{lib1Change},
			wantVersions: map[string]string{sample.Lib1Name: sample.NextVersion},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			opts := testhelper.SetupOptions{
				Clone:       cfg.Release.Branch,
				Config:      cfg,
				Tags:        []string{sample.InitialLegacyRustTag},
				WithChanges: test.withChanges,
			}
			testhelper.Setup(t, opts)

			if err := legacyRustBump(t.Context(), cfg, test.all, test.libraryName, test.versionOverride, "git"); err != nil {
				t.Fatal(err)
			}

			got, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, lib := range got.Libraries {
				if want, ok := test.wantVersions[lib.Name]; ok {
					if lib.Version != want {
						t.Errorf("library %s: got version %q, want %q", lib.Name, lib.Version, want)
					}
				}
			}
		})
	}
}

func TestLegacyRustBumpAll(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name        string
		cfg         *config.Config
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
	} {
		t.Run(test.name, func(t *testing.T) {
			targetCfg := test.cfg
			sinceTag := sample.InitialLegacyRustTag
			opts := testhelper.SetupOptions{
				Clone:       test.cfg.Release.Branch,
				Config:      test.cfg,
				Tags:        []string{sample.InitialLegacyRustTag},
				WithChanges: test.withChanges,
			}
			testhelper.Setup(t, opts)

			err := legacyRustBumpAll(t.Context(), targetCfg, sinceTag, "git")
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

func writeReadmeAndCommit(t *testing.T, newContent string) {
	writeFileAndCommit(t, testhelper.ReadmeFile, []byte(newContent), "Modified readme")
}

func writeConfigAndCommit(t *testing.T, cfg *config.Config) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeFileAndCommit(t, librarianConfigPath, data, "Modified config")
}

func writeFileAndCommit(t *testing.T, path string, content []byte, message string) {
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), "git", "commit", "-m", message); err != nil {
		t.Fatal(err)
	}
}
