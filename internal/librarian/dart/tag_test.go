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

package dart

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestFindReleaseCommitAndLibrariesToTag(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	const initialChangelog = "# Changelog\n\n## 0.1.0\n\n- initial release\n"

	for _, test := range []struct {
		name          string
		setup         func(t *testing.T, cfg *config.Config) (wantCommit string)
		wantLibraries []string
	}{
		{
			name: "mixed release bumps existing library and prepares initial CHANGELOG.md library",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/pubspec.yaml", "name: google_cloud_tasks_v2\nversion: 0.1.0\n")
				writeConfigAndCommit(t, cfg, "feat: generate tasks_v2")

				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg, "chore: release")
				return headSHA(t)
			},
			wantLibraries: []string{"google_cloud_compute_v1", "google_cloud_tasks_v2"},
		},
		{
			name: "add then generate then bump only releases on bump commit",
			setup: func(t *testing.T, cfg *config.Config) string {
				// Step 1: librarian add writes entry with empty version.
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name: "google_cloud_tasks_v2",
				})
				writeConfigAndCommit(t, cfg, "feat: add tasks_v2")

				// Step 2: librarian generate populates version 0.1.0 without CHANGELOG.md.
				cfg.Libraries[len(cfg.Libraries)-1].Version = "0.1.0"
				writeNestedFile(t, "generated/google_cloud_tasks_v2/pubspec.yaml", "name: google_cloud_tasks_v2\nversion: 0.1.0\n")
				writeConfigAndCommit(t, cfg, "feat: generate tasks_v2")

				// Step 3: librarian bump --all writes only CHANGELOG.md; librarian.yaml is unchanged.
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				commitAll(t, "chore: bump tasks_v2")
				releaseSHA := headSHA(t)

				// Step 4: non-release chore commit touching librarian.yaml and README.md after release.
				cfg.Version = "v0.47.0"
				writeNestedFile(t, "README.md", "# Updated\n")
				writeConfigAndCommit(t, cfg, "chore: update librarian version")

				return releaseSHA
			},
			wantLibraries: []string{"google_cloud_tasks_v2"},
		},
		{
			name: "new library added and bumped in a single commit",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg, "chore: add and bump tasks_v2")
				return headSHA(t)
			},
			wantLibraries: []string{"google_cloud_tasks_v2"},
		},
		{
			name: "subsequent release in tagless clone does not re-tag historical 0.1.0 library",
			setup: func(t *testing.T, cfg *config.Config) string {
				// First release prepares tasks_v2 at 0.1.0; intentionally create zero local git tags.
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg, "chore: initial tasks_v2 release")

				// Second release bumps compute_v1 only; tasks_v2 remains at 0.1.0 with its existing CHANGELOG.md.
				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeConfigAndCommit(t, cfg, "chore: bump compute_v1")
				return headSHA(t)
			},
			wantLibraries: []string{"google_cloud_compute_v1"},
		},
		{
			name: "unbumped generated library without CHANGELOG.md is ignored during another bump",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_unreleased_v1",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_unreleased_v1/pubspec.yaml", "name: google_cloud_unreleased_v1\nversion: 0.1.0\n")
				writeConfigAndCommit(t, cfg, "feat: generate unreleased_v1")

				cfg.Libraries[0].Version = "0.1.1"
				writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.1.1\n\n- feat: update\n")
				writeConfigAndCommit(t, cfg, "chore: bump compute_v1")
				return headSHA(t)
			},
			wantLibraries: []string{"google_cloud_compute_v1"},
		},
		{
			name: "flipping skip_release to false with CHANGELOG.md releases library",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:        "google_cloud_tasks_v2",
					Version:     "0.1.0",
					SkipRelease: true,
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/CHANGELOG.md", initialChangelog)
				writeConfigAndCommit(t, cfg, "chore: stage skipped tasks_v2")

				cfg.Libraries[len(cfg.Libraries)-1].SkipRelease = false
				writeConfigAndCommit(t, cfg, "chore: enable tasks_v2 release")
				return headSHA(t)
			},
			wantLibraries: []string{"google_cloud_tasks_v2"},
		},
		{
			name: "custom library output overrides default output directory",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_custom",
					Output:  "packages/google_cloud_custom",
					Version: "0.1.0",
				})
				writeConfigAndCommit(t, cfg, "feat: add custom package")

				writeNestedFile(t, "packages/google_cloud_custom/CHANGELOG.md", initialChangelog)
				commitAll(t, "chore: bump custom package")
				return headSHA(t)
			},
			wantLibraries: []string{"google_cloud_custom"},
		},
		{
			name: "no release commits after root init returns empty commit hash",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_tasks_v2",
					Version: "0.1.0",
				})
				writeNestedFile(t, "generated/google_cloud_tasks_v2/pubspec.yaml", "name: google_cloud_tasks_v2\nversion: 0.1.0\n")
				writeConfigAndCommit(t, cfg, "feat: generate tasks_v2 without changelog")
				return ""
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
			writeConfigAndCommit(t, cfg, "chore: initial repo setup")

			wantCommit := test.setup(t, cfg)
			headCfg, err := loadConfigAtRevision(t.Context(), "HEAD")
			if err != nil {
				t.Fatal(err)
			}

			gotCommit, err := FindReleaseCommit(t.Context(), headCfg)
			if err != nil {
				t.Fatal(err)
			}
			if gotCommit != wantCommit {
				t.Fatalf("FindReleaseCommit() = %q, want %q", gotCommit, wantCommit)
			}
			if wantCommit == "" {
				return
			}

			commitCfg, err := loadConfigAtRevision(t.Context(), gotCommit)
			if err != nil {
				t.Fatal(err)
			}
			gotLibraries, err := LibrariesToTag(t.Context(), gotCommit, commitCfg)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantLibraries, gotLibraries); diff != "" {
				t.Errorf("LibrariesToTag() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLibrariesToTag_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name    string
		setup   func(t *testing.T, cfg *config.Config) string
		wantErr error
	}{
		{
			name: "root commit without parent returns ErrGitShow",
			setup: func(t *testing.T, cfg *config.Config) string {
				return headSHA(t)
			},
			wantErr: git.ErrGitShow,
		},
		{
			name: "version regression returns error",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries[0].Version = "0.1.0"
				writeConfigAndCommit(t, cfg, "chore: regress version")
				return headSHA(t)
			},
		},
		{
			name: "cleared version returns error",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries[0].Version = ""
				writeConfigAndCommit(t, cfg, "chore: clear version")
				return headSHA(t)
			},
		},
		{
			name: "invalid initial semver returns error",
			setup: func(t *testing.T, cfg *config.Config) string {
				cfg.Libraries = append(cfg.Libraries, &config.Library{
					Name:    "google_cloud_bad",
					Version: "not-a-semver",
				})
				writeNestedFile(t, "generated/google_cloud_bad/CHANGELOG.md", "# Changelog\n")
				writeConfigAndCommit(t, cfg, "chore: invalid version")
				return headSHA(t)
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
					{Name: "google_cloud_compute_v1", Version: "0.2.0"},
				},
			}
			testhelper.ContinueInNewGitRepository(t, t.TempDir())
			writeNestedFile(t, "generated/google_cloud_compute_v1/CHANGELOG.md", "# Changelog\n\n## 0.2.0\n\n- init\n")
			writeConfigAndCommit(t, cfg, "chore: initial repo setup")

			commit := test.setup(t, cfg)
			commitCfg, err := loadConfigAtRevision(t.Context(), commit)
			if err != nil {
				t.Fatal(err)
			}
			_, gotErr := LibrariesToTag(t.Context(), commit, commitCfg)
			if gotErr == nil {
				t.Fatal("LibrariesToTag() succeeded, want error")
			}
			if test.wantErr != nil && !errors.Is(gotErr, test.wantErr) {
				t.Errorf("LibrariesToTag() error = %v, want %v", gotErr, test.wantErr)
			}
		})
	}
}

func headSHA(t *testing.T) string {
	t.Helper()
	sha, err := git.GetCommitHash(t.Context(), command.Git, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

func writeConfigAndCommit(t *testing.T, cfg *config.Config, message string) {
	t.Helper()
	if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
		t.Fatal(err)
	}
	commitAll(t, message)
}

func writeNestedFile(t *testing.T, filePath, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t *testing.T, message string) {
	t.Helper()
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", message)
}
