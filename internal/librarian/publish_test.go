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
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPublish(t *testing.T) {
	// Each test starts (before setup) with Lib1Name with a version of 1.0.0 and
	// Lib2Name with a version of 1.2.0.
	for _, test := range []struct {
		name          string
		setup         func(cfg *config.Config)
		releaseCommit string
		execute       bool
		want          string
	}{
		{
			name: "publish Lib1Name and Lib2Name",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			want: fmt.Sprintf("libraries=%s,%s; execute=false", sample.Lib1Name, sample.Lib2Name),
		},
		{
			name: "publish Lib1Name and Lib2Name, with execute",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			execute: true,
			want:    fmt.Sprintf("libraries=%s,%s; execute=true", sample.Lib1Name, sample.Lib2Name),
		},
		{
			name: "publish Lib1Name (Lib2Name not released)",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
			},
			want: fmt.Sprintf("libraries=%s; execute=false", sample.Lib1Name),
		},
		{
			name: "publish Lib1Name due to release commit specified in flags, with a later release of Lib2Name ignored",
			setup: func(cfg *config.Config) {
				cfg.Libraries[0].Version = "1.1.0"
				writeConfigAndCommit(t, cfg)
				cfg.Libraries[1].Version = "1.3.0"
				writeConfigAndCommit(t, cfg)
			},
			// Fortunately this doesn't have to be an actual commit, just a
			// committish, so we don't need to know the hash of the HEAD~
			// commit when creating the test data.
			releaseCommit: "HEAD~",
			want:          fmt.Sprintf("libraries=%s; execute=false", sample.Lib1Name),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			cfg.Libraries[1].Version = "1.2.0"
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			test.setup(cfg)
			if err := publish(t.Context(), cfg, test.releaseCommit, test.execute); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(fakePublishedFile)
			if err != nil {
				t.Fatalf("error reading file %s, error = %v", fakePublishedFile, err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch in output (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPublish_Error(t *testing.T) {
	for _, test := range []struct {
		name          string
		setup         func(cfg *config.Config)
		releaseCommit string
		wantErr       error
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
			// Can't easily check this error.
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
			wantErr: git.ErrGitStatusUnclean,
		},
		{
			name: "language isn't supported",
			setup: func(cfg *config.Config) {
				// Add a release commit to distinguish this case from "no releases"
				cfg.Libraries[0].Version = "1.1.0"
				cfg.Language = "unsupported-for-publish"
				writeConfigAndCommit(t, cfg)
			},
			// Can't easily check this error.
		},
		{
			name: "no release commit",
			setup: func(cfg *config.Config) {
			},
		},
		{
			name: "specified release commit is invalid",
			setup: func(cfg *config.Config) {
			},
			releaseCommit: "not a commit",
			// Can't easily check this error
		},
		{
			name: "release commit does not release anything",
			setup: func(cfg *config.Config) {
				writeFileAndCommit(t, "README.txt", []byte("Just a readme"), "Modified config")
			},
			releaseCommit: "HEAD",
			wantErr:       errNoLibrariesAtReleaseCommit,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := sample.Config()
			cfg.Libraries[1].Version = "1.2.0"
			testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
			test.setup(cfg)
			err := publish(t.Context(), cfg, test.releaseCommit, false)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, err)
			}

		})
	}
}

// TestPublishCommand is just a single "does it look like it passes things
// through to the publish function" test. TestPublish tests the bulk of the logic.
func TestPublishCommand(t *testing.T) {
	cfg := sample.Config()
	cfg.Libraries[1].Version = "1.2.0"
	testhelper.Setup(t, testhelper.SetupOptions{Config: cfg})
	cfg.Libraries[0].Version = "1.1.0"
	writeConfigAndCommit(t, cfg)
	cfg.Libraries[1].Version = "1.3.0"
	writeConfigAndCommit(t, cfg)

	if err := Run(t.Context(), "librarian", "publish", "--release-commit", "HEAD~", "--execute"); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("libraries=%s; execute=true", sample.Lib1Name)
	got, err := os.ReadFile(fakePublishedFile)
	if err != nil {
		t.Fatalf("error reading file %s, error = %v", fakePublishedFile, err)
	}
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("mismatch in output (-want +got):\n%s", diff)
	}
}
