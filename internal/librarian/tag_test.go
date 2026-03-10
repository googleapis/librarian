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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestTag(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	// Each test starts (before setup) with Lib1Name with a version of 1.0.0 and
	// Lib2Name with a version of 1.2.0.
	for _, test := range []struct {
		name             string
		setup            func(t *testing.T, cfg *config.Config)
		releaseCommit    string
		taggedRevision   string
		createReleaseTag bool
		wantTags         []string
	}{
		{
			name: "tag Lib1Name and Lib2Name",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			taggedRevision: "HEAD",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0", sample.Lib2Name + "/v1.3.0"},
		},
		{
			name: "tag Lib1Name (Lib2Name not released)",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			taggedRevision: "HEAD",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0"},
		},
		{
			name: "tag Lib1Name in earlier commit",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				writeReadmeAndCommit(t, "modified readme")
			},
			taggedRevision: "HEAD~",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0"},
		},
		{
			name: "tag Lib1Name, specified in flags, with a later release of Lib2Name ignored",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			releaseCommit:  "HEAD~",
			taggedRevision: "HEAD~",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0"},
		},
		{
			name: "release tag",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommitWithMessage(t, cfg, "chore: release a library (#12345)")
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			releaseCommit:    "HEAD~",
			taggedRevision:   "HEAD~",
			createReleaseTag: true,
			wantTags:         []string{sample.Lib1Name + "/v1.1.0", "release-12345"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			cfg.Default.TagFormat = "{name}/v{version}"
			cfg.Libraries[1].Version = "1.2.0"
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			test.setup(t, cfg)

			if err := tag(t.Context(), cfg, test.releaseCommit, test.createReleaseTag); err != nil {
				t.Fatal(err)
			}

			wantTaggedCommit, err := git.GetCommitHash(t.Context(), "git", test.taggedRevision)
			if err != nil {
				t.Fatal(err)
			}

			for _, tagName := range test.wantTags {
				gotTaggedCommit, err := git.GetCommitHash(t.Context(), "git", tagName)
				if err != nil {
					t.Fatal(err)
				}
				if gotTaggedCommit != wantTaggedCommit {
					// Deliberately not using diff as the hashes are basically opaque
					t.Errorf("incorrect tagged commit for revision %s: got = %s; want = %s", test.taggedRevision, gotTaggedCommit, wantTaggedCommit)
				}
			}
		})
	}
}

func TestTag_Error(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	for _, test := range []struct {
		name             string
		setup            func(t *testing.T, cfg *config.Config)
		releaseCommit    string
		createReleaseTag bool
		wantErr          error
	}{
		{
			name: "custom tool specified for git and doesn't exist",
			setup: func(t *testing.T, cfg *config.Config) {
				// Add a release commit to distinguish this case from "no releases"
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Release = &config.Release{
					Preinstalled: map[string]string{
						"git": "/usr/bin/does-not-exist",
					},
				}
				writeConfigAndCommit(t, cfg)
			},
			// Can't easily check this error
		},
		{
			name: "repo is dirty",
			setup: func(t *testing.T, cfg *config.Config) {
				// Add a release commit to distinguish this case from "no releases"
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				if err := os.WriteFile(testhelper.ReadmeFile, []byte("uncommitted change"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: git.ErrGitStatusUnclean,
		},
		{
			name: "no release commit",
			setup: func(t *testing.T, cfg *config.Config) {
			},
			wantErr: errReleaseCommitNotFound,
		},
		{
			name: "specified release commit is not a release commit",
			setup: func(t *testing.T, cfg *config.Config) {
				writeFileAndCommit(t, "README.txt", []byte("Just a readme"), "Modified config")
			},
			releaseCommit: "HEAD",
			wantErr:       errNoLibrariesAtReleaseCommit,
		},
		{
			name: "specified release commit is invalid",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			releaseCommit: "not-a-commit",
			// Can't easily check this error
		},
		{
			name: "createReleaseTag but bad commit subject",
			setup: func(t *testing.T, cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			createReleaseTag: true,
			wantErr:          errCannotDeriveReleaseTag,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			cfg.Default.TagFormat = "{name}/v{version}"
			cfg.Libraries[1].Version = "1.2.0"
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			test.setup(t, cfg)
			err := tag(t.Context(), cfg, test.releaseCommit, test.createReleaseTag)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

// TestTagCommand is just a single "does it look like it passes things
// through to the publish function" test. TestTag tests the bulk of the logic.
func TestTagCommand(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	cfg := sample.Config()
	cfg.Default.TagFormat = "{name}/v{version}"
	cfg.Libraries[1].Version = "1.2.0"
	testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
	cfg.Libraries[0].Version = "1.1.0"
	writeConfigAndCommit(t, cfg)
	cfg.Libraries[1].Version = "1.3.0"
	writeConfigAndCommit(t, cfg)

	if err := Run(t.Context(), "librarian", "tag", "--release-commit", "HEAD~"); err != nil {
		t.Fatal(err)
	}

	wantTaggedCommit, err := git.GetCommitHash(t.Context(), "git", "HEAD~")
	if err != nil {
		t.Fatal(err)
	}
	gotTaggedCommit, err := git.GetCommitHash(t.Context(), "git", cfg.Libraries[0].Name+"/v1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if gotTaggedCommit != wantTaggedCommit {
		// Deliberately not using diff as the hashes are basically opaque
		t.Errorf("incorrect commit for tag: got = %s; want = %s", gotTaggedCommit, wantTaggedCommit)
	}
}

func TestPullRequestCommitSubjectRegex(t *testing.T) {
	for _, test := range []struct {
		name      string
		text      string
		wantMatch bool
	}{
		{
			name:      "match",
			text:      "chore: release a library (#12345)",
			wantMatch: true,
		},
		{
			name: "no number at all",
			text: "chore: release a library",
		},
		{
			name: "no number",
			text: "chore: release a library (#)",
		},
		{
			name: "no hash",
			text: "chore: release a library (12345)",
		},
		{
			name: "no parens",
			text: "chore: release a library #12345",
		},
		{
			name: "match not at end",
			text: "chore: release (#12345) a library",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotMatch := pullRequestCommitSubjectRegex.MatchString(test.text)

			if diff := cmp.Diff(test.wantMatch, gotMatch); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
