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
	"testing"

	"github.com/google/go-cmp/cmp"
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

			gotTags, err := git.Tags(t.Context(), "git")
			if err != nil {
				t.Fatal(err)
			}
			slices.Sort(gotTags)
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
		setup            func(t *testing.T, cfg *config.Config)
		wantTags         []string
		wantCommitLogIdx int
	}{
		{
			name: "single library bump ignores unchanged and unversioned and skipped libraries",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Libraries = append(cfg.Libraries,
					&config.Library{Name: "unversioned", Version: ""},
					&config.Library{Name: "skipped_new", Version: "0.1.0", SkipRelease: true},
				)
				cfg.Libraries[2].Version = "2.0.0"
				writeConfigAndCommit(t, cfg)
			},
			wantTags:         []string{"lib1-v1.1.0"},
			wantCommitLogIdx: 0,
		},
		{
			name: "multi library bump and newly added versioned library",
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
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language: config.LanguageFake,
				Default: &config.Default{
					TagFormat: "{name}-v{version}",
				},
				Libraries: []*config.Library{
					{Name: "lib1", Version: "1.0.0"},
					{Name: "lib2", Version: "1.2.0"},
					{Name: "skipped_existing", Version: "1.0.0", SkipRelease: true},
				},
			}
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			test.setup(t, cfg)

			if err := tag(t.Context(), ""); err != nil {
				t.Fatal(err)
			}

			gotTags, err := git.Tags(t.Context(), "git")
			if err != nil {
				t.Fatal(err)
			}
			slices.Sort(gotTags)
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

func TestTag_Dart(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	const initialChangelog = "# Changelog\n\n## 0.1.0\n\n- initial release\n"

	for _, test := range []struct {
		name          string
		releaseCommit string
		setup         func(t *testing.T, cfg *config.Config) (wantTagCommits map[string]string)
	}{
		{
			name: "mixed release tags bumped and newly bumped initial library while ignoring skipped and unchanged",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				// Simulate librarian add + generate for tasks_v2 (version 0.1.0, no CHANGELOG.md yet).
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFileAndCommit(t, "generated/google_cloud_tasks_v2/pubspec.yaml", "name: google_cloud_tasks_v2\nversion: 0.1.0\n", "feat: generate tasks_v2")
				writeConfigAndCommit(t, cfg)

				// Simulate librarian bump --all: compute_v1 bumps 0.1.0 -> 0.1.1,
				// tasks_v2 stays at 0.1.0 and gets CHANGELOG.md.
				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg)
				releaseSHA := headSHA(t)

				return map[string]string{
					"google_cloud_compute_v1-v0.1.1": releaseSHA,
					"google_cloud_tasks_v2-v0.1.0":   releaseSHA,
				}
			},
		},
		{
			name: "mixed release followed by librarian version chore commit tags release commit",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeConfigAndCommit(t, cfg)

				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg)
				releaseSHA := headSHA(t)

				// Subsequent non-release update to librarian.yaml (e.g. librarian version bump).
				cfg.Version = "v0.47.0"
				writeConfigAndCommit(t, cfg)

				return map[string]string{
					"google_cloud_compute_v1-v0.1.1": releaseSHA,
					"google_cloud_tasks_v2-v0.1.0":   releaseSHA,
				}
			},
		},
		{
			name: "new library only release where bump commit only writes CHANGELOG.md",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				// Step 1: librarian add + generate commits librarian.yaml with version 0.1.0 (no CHANGELOG.md).
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/pubspec.yaml", "name: google_cloud_tasks_v2\nversion: 0.1.0\n")
				writeConfigAndCommit(t, cfg)

				// Step 2: librarian bump --all writes only CHANGELOG.md; librarian.yaml is unchanged.
				writeNestedFileAndCommit(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog, "chore: bump unpublished library")
				releaseSHA := headSHA(t)

				// Step 3: unrelated chore commit after the release commit.
				writeReadmeAndCommit(t, "docs update")

				return map[string]string{
					"google_cloud_tasks_v2-v0.1.0": releaseSHA,
				}
			},
		},
		{
			name: "new library added and bumped in a single commit",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg)
				releaseSHA := headSHA(t)

				return map[string]string{
					"google_cloud_tasks_v2-v0.1.0": releaseSHA,
				}
			},
		},
		{
			name: "subsequent release does not re-tag already tagged 0.1.0 library",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg)
				firstReleaseSHA := headSHA(t)
				if err := git.Tag(t.Context(), "git", "google_cloud_tasks_v2-v0.1.0", firstReleaseSHA); err != nil {
					t.Fatal(err)
				}

				// Next release bumps compute_v1 only; tasks_v2 stays at 0.1.0.
				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeConfigAndCommit(t, cfg)
				secondReleaseSHA := headSHA(t)

				return map[string]string{
					"google_cloud_compute_v1-v0.1.1": secondReleaseSHA,
					"google_cloud_tasks_v2-v0.1.0":   firstReleaseSHA,
				}
			},
		},
		{
			name: "unbumped generated library without CHANGELOG.md is not tagged during another bump",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_unreleased_v1",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_unreleased_v1/pubspec.yaml", "name: google_cloud_unreleased_v1\nversion: 0.1.0\n")
				writeConfigAndCommit(t, cfg)

				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeConfigAndCommit(t, cfg)
				releaseSHA := headSHA(t)

				return map[string]string{
					"google_cloud_compute_v1-v0.1.1": releaseSHA,
				}
			},
		},
		{
			name: "multi-commit pre-publish fixup tags all untagged bumped and initial libraries",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeConfigAndCommit(t, cfg)

				// First release PR bumps compute_v1 and prepares tasks_v2, not yet published/tagged.
				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg)

				// Pre-publish fixup PR bumps secretmanager_v1 before publishing all three at HEAD.
				cfg.Libraries[1].Version = "0.6.0"
				writeNestedFile(t, "generated/google_cloud_secretmanager_v1/CHANGELOG.md", "# Changelog\n\n## 0.6.0\n\n- feat!: breaking fix\n")
				writeConfigAndCommit(t, cfg)
				finalReleaseSHA := headSHA(t)

				return map[string]string{
					"google_cloud_compute_v1-v0.1.1":       finalReleaseSHA,
					"google_cloud_secretmanager_v1-v0.6.0": finalReleaseSHA,
					"google_cloud_tasks_v2-v0.1.0":         finalReleaseSHA,
				}
			},
		},
		{
			name: "flipping skip_release to false with CHANGELOG.md tags initial release",
			setup: func(t *testing.T, cfg *config.Config) map[string]string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:        "google_cloud_tasks_v2",
					Version:     "0.1.0",
					SkipRelease: true,
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg)

				for _, lib := range cfg.Libraries {
					if lib.Name == "google_cloud_tasks_v2" {
						lib.SkipRelease = false
					}
				}
				writeConfigAndCommit(t, cfg)
				releaseSHA := headSHA(t)

				return map[string]string{
					"google_cloud_tasks_v2-v0.1.0": releaseSHA,
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Language: config.LanguageDart,
				Default: &config.Default{
					Output:    "generated/",
					TagFormat: "{name}-v{version}",
				},
				Libraries: []*config.Library{
					{Name: "google_cloud_compute_v1", Version: "0.1.0"},
					{Name: "google_cloud_secretmanager_v1", Version: "0.5.5"},
					{Name: "google_cloud_showcase_v1beta1", Version: "0.5.3", SkipRelease: true},
				},
			}
			testhelper.ContinueInNewGitRepository(t, t.TempDir())
			writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", initialChangelog)
			writeNestedFile(t, "generated/google_cloud_secretmanager_v1/CHANGELOG.md", "# Changelog\n\n## 0.5.5\n\n- initial\n")
			writeConfigAndCommit(t, cfg)
			baseSHA := headSHA(t)
			if err := git.Tag(t.Context(), "git", "google_cloud_compute_v1-v0.1.0", baseSHA); err != nil {
				t.Fatal(err)
			}
			if err := git.Tag(t.Context(), "git", "google_cloud_secretmanager_v1-v0.5.5", baseSHA); err != nil {
				t.Fatal(err)
			}

			wantTagCommits := test.setup(t, cfg)
			wantTagCommits["google_cloud_compute_v1-v0.1.0"] = baseSHA
			wantTagCommits["google_cloud_secretmanager_v1-v0.5.5"] = baseSHA

			if err := tag(t.Context(), test.releaseCommit); err != nil {
				t.Fatal(err)
			}

			gotTags, err := git.Tags(t.Context(), "git")
			if err != nil {
				t.Fatal(err)
			}
			gotTagCommits := make(map[string]string, len(gotTags))
			for _, tagName := range gotTags {
				sha, err := git.GetCommitHash(t.Context(), "git", tagName)
				if err != nil {
					t.Fatal(err)
				}
				gotTagCommits[tagName] = sha
			}
			if diff := cmp.Diff(wantTagCommits, gotTagCommits); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
				if err := git.Tag(t.Context(), "git", "google_cloud_compute_v1-v0.1.0", "HEAD"); err != nil {
					t.Fatal(err)
				}
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
				if err := git.Tag(t.Context(), "git", "google_cloud_compute_v1-v0.1.0", "HEAD"); err != nil {
					t.Fatal(err)
				}
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

func headSHA(t *testing.T) string {
	t.Helper()
	sha, err := git.GetCommitHash(t.Context(), "git", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return sha
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
