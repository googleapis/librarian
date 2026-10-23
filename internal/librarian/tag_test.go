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

package librarian

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestTagCommand(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name             string
		setup            func(t *testing.T, cfg *config.Config) []string
		wantTags         []string
		wantCommitLogIdx int
	}{
		{
			name: "latest release commit by default",
			setup: func(t *testing.T, cfg *config.Config) []string {
				cfg.Libraries[0].Version = sample.NextVersion
				writeConfigAndCommit(t, cfg)
				writeReadmeAndCommit(t, "chore after release")
				return []string{"librarian", "tag"}
			},
			wantTags:         []string{"lib1-v1.1.0"},
			wantCommitLogIdx: 1,
		},
		{
			name: "explicit release commit flag",
			setup: func(t *testing.T, cfg *config.Config) []string {
				cfg.Libraries[0].Version = sample.NextVersion
				writeConfigAndCommit(t, cfg)
				cfg.Libraries[1].Version = sample.NextVersion
				writeConfigAndCommit(t, cfg)
				return []string{"librarian", "tag", "--release-commit=HEAD~"}
			},
			wantTags:         []string{"lib1-v1.1.0"},
			wantCommitLogIdx: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language: config.LanguageFake,
				Default: &config.Default{
					TagFormat: "{name}-v{version}",
				},
				Libraries: []*config.Library{
					{Name: "lib1", Version: sample.InitialVersion},
					{Name: "lib2", Version: sample.InitialVersion},
				},
			}
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			args := test.setup(t, cfg)

			if err := Run(t.Context(), args...); err != nil {
				t.Fatal(err)
			}

			gotTags := listGitTags(t)
			if diff := cmp.Diff(test.wantTags, gotTags); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			commits, err := git.FindCommitsForPath(t.Context(), "git", ".")
			if err != nil {
				t.Fatal(err)
			}
			for _, tagName := range test.wantTags {
				gotCommit, err := git.GetCommitHash(t.Context(), "git", tagName)
				if err != nil {
					t.Fatal(err)
				}
				if gotCommit != commits[test.wantCommitLogIdx] {
					t.Errorf("tag %s points to %s, want %s", tagName, gotCommit, commits[test.wantCommitLogIdx])
				}
			}
		})
	}
}

func TestTag(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name             string
		cfg              *config.Config
		setup            func(t *testing.T, cfg *config.Config)
		wantTags         []string
		wantCommitLogIdx int
	}{
		{
			name: "single library bump ignores unchanged and unversioned libraries",
			cfg: &config.Config{
				Language: config.LanguageFake,
				Default:  &config.Default{TagFormat: "{name}-v{version}"},
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
					{Name: "lib2", Version: "1.2.0"},
				},
			},
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Libraries = append(cfg.Libraries, &config.Library{Name: "unversioned", Version: ""})
				writeConfigAndCommit(t, cfg)
			},
			wantTags:         []string{"lib1-v1.1.0"},
			wantCommitLogIdx: 0,
		},
		{
			name: "multi library bump and newly added versioned library",
			cfg: &config.Config{
				Language: config.LanguageFake,
				Default:  &config.Default{TagFormat: "{name}-v{version}"},
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
					{Name: "lib2", Version: "1.2.0"},
				},
			},
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Libraries[1].Version = "1.3.0"
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "lib3",
					Version: "0.1.0",
				})
				writeConfigAndCommit(t, cfg)
			},
			wantTags:         []string{"lib1-v1.1.0", "lib2-v1.3.0", "lib3-v0.1.0"},
			wantCommitLogIdx: 0,
		},
		{
			name: "dart changelog-only initial release commit followed by chore commit",
			cfg: &config.Config{
				Language: config.LanguageDart,
				Default: &config.Default{
					Output:    "generated/",
					TagFormat: "{name}-v{version}",
				},
				Libraries: []*config.Library{
					{Name: "google_cloud_compute_v1", Version: "0.1.0"},
					{Name: "google_cloud_tasks_v2", Version: "0.1.0"},
				},
			},
			setup: func(t *testing.T, cfg *config.Config) {
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.0\n\n- initial release\n")
				writeConfigAndCommit(t, cfg)

				// librarian bump --all writes only CHANGELOG.md for the unpublished 0.1.0 library.
				writeNestedFileAndCommit(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", "# Changelog\n\n## 0.1.0\n\n- initial release\n", "chore: bump tasks_v2")
				writeReadmeAndCommit(t, "docs chore after release")
			},
			wantTags:         []string{"google_cloud_tasks_v2-v0.1.0"},
			wantCommitLogIdx: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testhelper.Setup(t, testhelper.SetupOptions{Config: test.cfg})
			test.setup(t, test.cfg)

			if err := tag(t.Context(), ""); err != nil {
				t.Fatal(err)
			}

			gotTags := listGitTags(t)
			if diff := cmp.Diff(test.wantTags, gotTags); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			commits, err := git.FindCommitsForPath(t.Context(), "git", ".")
			if err != nil {
				t.Fatal(err)
			}
			for _, tagName := range test.wantTags {
				gotCommit, err := git.GetCommitHash(t.Context(), "git", tagName)
				if err != nil {
					t.Fatal(err)
				}
				if gotCommit != commits[test.wantCommitLogIdx] {
					t.Errorf("tag %s points to %s, want %s", tagName, gotCommit, commits[test.wantCommitLogIdx])
				}
			}
		})
	}
}

func TestTag_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name          string
		cfg           *config.Config
		dirty         bool
		releaseCommit string
		setup         func(t *testing.T, cfg *config.Config)
		wantErr       error
	}{
		{
			name: "dirty working tree",
			cfg: &config.Config{
				Language: config.LanguageFake,
				Default:  &config.Default{TagFormat: "{name}-v{version}"},
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
				},
			},
			dirty:   true,
			wantErr: git.ErrGitStatusUnclean,
		},
		{
			name: "no release commit in non-dart repo",
			cfg: &config.Config{
				Language: config.LanguageFake,
				Default:  &config.Default{TagFormat: "{name}-v{version}"},
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
				},
			},
			wantErr: errReleaseCommitNotFound,
		},
		{
			name: "no release commit in dart repo when new library lacks CHANGELOG.md",
			cfg: &config.Config{
				Language: config.LanguageDart,
				Default: &config.Default{
					Output:    "generated/",
					TagFormat: "{name}-v{version}",
				},
				Libraries: []*config.Library{
					{Name: "google_cloud_compute_v1", Version: "0.1.0"},
				},
			},
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/pubspec.yaml", "name: google_cloud_tasks_v2\nversion: 0.1.0\n")
				writeConfigAndCommit(t, cfg)
			},
			wantErr: errReleaseCommitNotFound,
		},
		{
			name: "explicit non-release commit in non-dart repo",
			cfg: &config.Config{
				Language: config.LanguageFake,
				Default:  &config.Default{TagFormat: "{name}-v{version}"},
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
				},
			},
			releaseCommit: "HEAD",
			setup: func(t *testing.T, cfg *config.Config) {
				writeReadmeAndCommit(t, "not a release")
			},
			wantErr: errNoLibrariesAtReleaseCommit,
		},
		{
			name: "explicit non-release commit in dart repo",
			cfg: &config.Config{
				Language: config.LanguageDart,
				Default: &config.Default{
					Output:    "generated/",
					TagFormat: "{name}-v{version}",
				},
				Libraries: []*config.Library{
					{Name: "google_cloud_compute_v1", Version: "0.1.0"},
				},
			},
			releaseCommit: "HEAD",
			setup: func(t *testing.T, cfg *config.Config) {
				writeReadmeAndCommit(t, "not a release")
			},
			wantErr: errNoLibrariesAtReleaseCommit,
		},
		{
			name: "missing tag format",
			cfg: &config.Config{
				Language: config.LanguageFake,
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
				},
			},
			releaseCommit: "HEAD",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			wantErr: errMissingTagFormat,
		},
		{
			name: "invalid release commit",
			cfg: &config.Config{
				Language: config.LanguageFake,
				Default:  &config.Default{TagFormat: "{name}-v{version}"},
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
				},
			},
			releaseCommit: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
			wantErr:       git.ErrGitShow,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testhelper.Setup(t, testhelper.SetupOptions{
				Clone:  test.dirty,
				Config: test.cfg,
				Dirty:  test.dirty,
			})
			if test.setup != nil {
				test.setup(t, test.cfg)
			}

			gotErr := tag(t.Context(), test.releaseCommit)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("tag(%q) error = %v, wantErr %v", test.releaseCommit, gotErr, test.wantErr)
			}
		})
	}
}

func listGitTags(t *testing.T) []string {
	t.Helper()
	out, err := command.Output(t.Context(), command.Git, "tag", "--list")
	if err != nil {
		t.Fatal(err)
	}
	tags := strings.Fields(out)
	slices.Sort(tags)
	return tags
}

func writeNestedFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeNestedFileAndCommit(t *testing.T, path, content, message string) {
	t.Helper()
	writeNestedFile(t, path, content)
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", message)
}
