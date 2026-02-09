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
	"os"
	"testing"

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
		name           string
		setup          func(cfg *config.Config)
		library        string
		taggedRevision string
		wantTags       []string
	}{
		{
			name: "tag Lib1Name and Lib2Name",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			taggedRevision: "HEAD",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0", sample.Lib2Name + "/v1.3.0"},
		},
		{
			name: "tag Lib1Name (Lib2Name not released)",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			taggedRevision: "HEAD",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0"},
		},
		{
			name: "tag Lib1Name in earlier commit",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				writeReadmeAndCommit(t, "modified readme")
			},
			taggedRevision: "HEAD~",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0"},
		},
		{
			name: "tag Lib1Name, specified in flags, with a later release of Lib2Name ignored",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			library:        sample.Lib1Name,
			taggedRevision: "HEAD~",
			wantTags:       []string{sample.Lib1Name + "/v1.1.0"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			cfg.Default.TagFormat = "{name}/v{version}"
			cfg.Libraries[1].Version = "1.2.0"
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			test.setup(cfg)

			if err := tag(t.Context(), cfg, test.library); err != nil {
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
		name    string
		setup   func(cfg *config.Config)
		library string
	}{
		{
			name: "custom tool specified for git and doesn't exist",
			setup: func(cfg *config.Config) {
				// Add a release commit to distinguish this case from "no releases"
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Release = &config.Release{
					Preinstalled: map[string]string{
						"git": "/usr/bin/does-not-exist",
					},
				}
				writeConfigAndCommit(t, cfg)
			},
		},
		{
			name: "repo is dirty",
			setup: func(cfg *config.Config) {
				// Add a release commit to distinguish this case from "no releases"
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				if err := os.WriteFile(testhelper.ReadmeFile, []byte("uncommitted change"), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "language isn't supported",
			setup: func(cfg *config.Config) {
				// Add a release commit to distinguish this case from "no releases"
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Language = "unsupported-for-publish"
				writeConfigAndCommit(t, cfg)
			},
		},
		{
			name: "no release commit",
			setup: func(cfg *config.Config) {
			},
		},
		{
			name: "no release commit for specified library",
			setup: func(cfg *config.Config) {
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			library: sample.Lib1Name,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			cfg.Libraries[1].Version = "1.2.0"
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			test.setup(cfg)
			err := tag(t.Context(), cfg, test.library)
			if err == nil {
				t.Errorf("publish(): expected error, got none")
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

	if err := Run(t.Context(), "librarian", "tag", "--library", sample.Lib1Name); err != nil {
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
