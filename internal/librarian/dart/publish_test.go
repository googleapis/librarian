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
		mockNeededVersion map[string]string
		mockApitoolError  map[string]string
		skipSemverChecks  bool
		wantInvocations   []string
		wantErr           string
	}{
		{
			name: "needs publish (not published yet, skips apitool)",
			publishedVersions: map[string]string{
				"a": "",
				"b": "",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			mockApitoolError: map[string]string{
				"a": "Package not available on pub.dev",
				"b": "Package not available on pub.dev",
			},
			wantInvocations: []string{
				"generated/a|pub publish --dry-run",
				"generated/b|pub publish --dry-run",
			},
		},
		{
			name: "needs publish (repo version is greater than published version, apitool check passes)",
			publishedVersions: map[string]string{
				"a": "0.9.0",
				"b": "0.9.0",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			mockNeededVersion: map[string]string{
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
		{
			name: "semver failure (repo version < apitool needed version)",
			publishedVersions: map[string]string{
				"a": "0.9.0",
				"b": "0.9.0",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			mockNeededVersion: map[string]string{
				"a": "2.0.0", // requires major bump but repo is only minor
				"b": "1.0.0",
			},
			wantErr: `package a version 1.0.0 is less than required version 2.0.0 recommended by dart-apitool`,
		},
		{
			name: "semver check skipped by flag",
			publishedVersions: map[string]string{
				"a": "0.9.0",
				"b": "0.9.0",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			mockNeededVersion: map[string]string{
				"a": "2.0.0", // normally fails, but flag skips
				"b": "1.0.0",
			},
			skipSemverChecks: true,
			wantInvocations: []string{
				"generated/a|pub publish --dry-run",
				"generated/b|pub publish --dry-run",
			},
		},
		{
			name: "apitool package not available (handled as first release)",
			publishedVersions: map[string]string{
				"a": "0.9.0",
				"b": "0.9.0",
			},
			repoVersions: map[string]string{
				"a": "1.0.0",
				"b": "1.0.0",
			},
			mockApitoolError: map[string]string{
				"a": "Package not available on pub.dev",
			},
			mockNeededVersion: map[string]string{
				"b": "1.0.0",
			},
			wantInvocations: []string{
				"generated/a|pub publish --dry-run",
				"generated/b|pub publish --dry-run",
			},
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

			// Propagate mock expectations to environment for scripts to read
			if tc.mockNeededVersion != nil {
				marshaled, _ := json.Marshal(tc.mockNeededVersion)
				t.Setenv("MOCK_NEEDED_VERSIONS", string(marshaled))
			} else {
				t.Setenv("MOCK_NEEDED_VERSIONS", "")
			}
			if tc.mockApitoolError != nil {
				marshaled, _ := json.Marshal(tc.mockApitoolError)
				t.Setenv("MOCK_APITOOL_ERRORS", string(marshaled))
			} else {
				t.Setenv("MOCK_APITOOL_ERRORS", "")
			}

			// Set up fake dart script
			setupFakeScript(t, "dart", `#!/bin/bash
if [ "$1" == "pub" ] && [ "$2" == "deps" ] && [ "$3" == "--json" ]; then
	echo '{"packages":[{"name":"a","version":"`+tc.repoVersions["a"]+`","dependencies":[]},{"name":"b","version":"`+tc.repoVersions["b"]+`","dependencies":["a"]}]}'
else
	# We also need to log the pub publish command, but we shouldn't log other commands
	echo "$(pwd)|$*" >> "$TEST_LOG_FILE"
fi
`)

			// Set up fake apitool script
			setupFakeScript(t, "dart-apitool", `#!/bin/bash
# Find package name from pub:// argument
pkg_name=""
report_file=""
while [ $# -gt 0 ]; do
  if [[ "$1" == pub://* ]]; then
    pkg_name="${1#pub://}"
  elif [ "$1" == "--report-file-path" ]; then
    report_file="$2"
  fi
  shift
done

# Check if there is a mocked error for this package
if [ -n "$MOCK_APITOOL_ERRORS" ]; then
  err_msg=$(echo "$MOCK_APITOOL_ERRORS" | jq -r ".${pkg_name} // \"\"")
  if [ -n "$err_msg" ] && [ "$err_msg" != "null" ]; then
    echo "$err_msg" >&2
    exit 1
  fi
fi

# Write report with needed version
if [ -n "$report_file" ] && [ -n "$MOCK_NEEDED_VERSIONS" ]; then
  needed_version=$(echo "$MOCK_NEEDED_VERSIONS" | jq -r ".${pkg_name} // \"\"")
  if [ -n "$needed_version" ] && [ "$needed_version" != "null" ]; then
    echo "{\"version\":{\"needed\":\"$needed_version\"}}" > "$report_file"
  fi
fi
`)

			// Set up common log file path in environment for fake dart script to write to
			logFile := filepath.Join(t.TempDir(), "invocations.log")
			t.Setenv("TEST_LOG_FILE", logFile)

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
				Config:           cfg,
				DryRun:           true,
				SkipSemverChecks: tc.skipSemverChecks,
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

func setupFakeScript(t *testing.T, name, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows, bash script set up does not work")
	}
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, name)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
