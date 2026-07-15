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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPublish(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	// Start a local test server to mock pub.dev API responses
	var mockPublishedVersions map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pkgName := filepath.Base(r.URL.Path)
		version, ok := mockPublishedVersions[pkgName]
		if !ok || version == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return JSON matching pub.dev structure
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": pkgName,
			"latest": map[string]string{
				"version": version,
			},
		})
	}))
	defer ts.Close()

	// Redirect pub.dev requests to local test server
	oldPubdevAPIURL := pubdevAPIURL
	pubdevAPIURL = ts.URL + "/"
	defer func() { pubdevAPIURL = oldPubdevAPIURL }()

	for _, tc := range []struct {
		name              string
		publishedVersions map[string]string
		repoVersions      map[string]string
		wantInvocations   []string
		wantErr           string
	}{
		{
			name: "needs publish (not published yet)",
			publishedVersions: map[string]string{
				"a": "",
				"b": "",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			wantInvocations: []string{
				"generated/a|pub publish --dry-run",
				"generated/b|pub publish --dry-run",
			},
		},
		{
			name: "needs publish (repo version is greater than published version)",
			publishedVersions: map[string]string{
				"a": "0.9.0",
				"b": "0.9.0",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			wantInvocations: []string{
				"generated/a|pub publish --dry-run",
				"generated/b|pub publish --dry-run",
			},
		},
		{
			name: "no-op (already published with same version)",
			publishedVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			wantInvocations: nil,
		},
		{
			name: "error (published version is greater than repo version)",
			publishedVersions: map[string]string{
				"a": "2.0.0",
				"b": "1.0.0",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			wantErr: `published version "2.0.0" is greater than repo version "1.0.0" for package a`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockPublishedVersions = tc.publishedVersions

			// Set up a fresh clone directory for this test case
			remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
			if err := command.Run(t.Context(), command.Git, "-C", remoteDir, "config", "receive.denyCurrentBranch", "ignore"); err != nil {
				t.Fatal(err)
			}
			testhelper.CloneRepository(t, remoteDir)

			if err := os.MkdirAll("generated/a", 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll("generated/b", 0755); err != nil {
				t.Fatal(err)
			}

			pubspecA := "name: a\nversion: " + tc.repoVersions["a"] + "\ndependencies:\n  sdk: \">=3.0.0 <4.0.0\"\n"
			pubspecB := "name: b\nversion: " + tc.repoVersions["b"] + "\ndependencies:\n  sdk: \">=3.0.0 <4.0.0\"\n  a: ^1.0.0\n"

			if err := os.WriteFile("generated/a/pubspec.yaml", []byte(pubspecA), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("generated/b/pubspec.yaml", []byte(pubspecB), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("generated/a/lib.dart", []byte("// library a"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("generated/b/lib.dart", []byte("// library b"), 0644); err != nil {
				t.Fatal(err)
			}

			testhelper.RunGit(t, "add", ".")
			testhelper.RunGit(t, "commit", "-m", "feat: added pubspec files", ".")
			testhelper.RunGit(t, "push", config.RemoteUpstream, config.BranchMain)

			// Set up fake dart script to log invocations and deps
			logFile := filepath.Join(t.TempDir(), "invocations.log")
			setupFakeDartScript(t, `#!/bin/bash
if [ "$1" == "pub" ] && [ "$2" == "deps" ] && [ "$3" == "--json" ]; then
	# Return JSON deps outputting the specified repoVersions
	echo '{"packages":[{"name":"a","version":"`+tc.repoVersions["a"]+`","dependencies":[]},{"name":"b","version":"`+tc.repoVersions["b"]+`","dependencies":["a"]}]}'
else
	echo "$(pwd)|$*" >> "`+logFile+`"
fi
`)

			cfg := &config.Config{
				Default: &config.Default{
					Output: "generated",
				},
				Libraries: []*config.Library{
					{Name: "a", Version: tc.repoVersions["a"]},
					{Name: "b", Version: tc.repoVersions["b"]},
				},
			}

			err := Publish(t.Context(), PublishParams{
				Config: cfg,
				DryRun: true,
			})

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Publish failed: %v", err)
			}

			var gotInvocations []string
			if _, err := os.Stat(logFile); err == nil {
				logContent, err := os.ReadFile(logFile)
				if err != nil {
					t.Fatal(err)
				}
				gotInvocations = strings.Split(strings.TrimSpace(string(logContent)), "\n")
				gotInvocations = slices.DeleteFunc(gotInvocations, func(s string) bool { return s == "" })
			}

			workingDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}

			var wantInvocations []string
			for _, inv := range tc.wantInvocations {
				parts := strings.SplitN(inv, "|", 2)
				wantInvocations = append(wantInvocations, filepath.Join(workingDir, parts[0])+"|"+parts[1])
			}

			if diff := cmp.Diff(wantInvocations, gotInvocations); diff != "" {
				t.Errorf("invocations mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func setupFakeDartScript(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	tmpDir := t.TempDir()
	dartScript := filepath.Join(tmpDir, "dart")
	if err := os.WriteFile(dartScript, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
