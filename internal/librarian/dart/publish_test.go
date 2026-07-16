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

func TestPublishSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	publishedVersions := map[string]string{
		"a": "0.9.0",
		"b": "0.9.0",
	}
	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}
	mockNeededVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}

	setupMockPubdevServer(t, publishedVersions)
	setupTestRepo(t, repoVersions)
	logFile := setupFakeDartAndApitool(t, repoVersions, mockNeededVersions, nil)

	cfg := &config.Config{
		Default: &config.Default{
			Output: "generated",
		},
		Libraries: []*config.Library{
			{Name: "a", Version: "1.0.0"},
			{Name: "b", Version: "1.0.0"},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	gotInvocations := readLogFile(t, logFile)
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	wantInvocations := []string{
		filepath.Join(workingDir, "generated/a") + "|pub publish --skip-validation --dry-run",
		filepath.Join(workingDir, "generated/b") + "|pub publish --skip-validation --dry-run",
	}

	if diff := cmp.Diff(wantInvocations, gotInvocations); diff != "" {
		t.Errorf("invocations mismatch (-want +got):\n%s", diff)
	}
}

func TestPublishSuccessNotPublished(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	publishedVersions := map[string]string{
		"a": "",
		"b": "",
	}
	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}

	setupMockPubdevServer(t, publishedVersions)
	setupTestRepo(t, repoVersions)
	logFile := setupFakeDartAndApitool(t, repoVersions, nil, nil)

	cfg := &config.Config{
		Default: &config.Default{
			Output: "generated",
		},
		Libraries: []*config.Library{
			{Name: "a", Version: "1.0.0"},
			{Name: "b", Version: "1.0.0"},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	gotInvocations := readLogFile(t, logFile)
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	wantInvocations := []string{
		filepath.Join(workingDir, "generated/a") + "|pub publish --skip-validation --dry-run",
		filepath.Join(workingDir, "generated/b") + "|pub publish --skip-validation --dry-run",
	}

	if diff := cmp.Diff(wantInvocations, gotInvocations); diff != "" {
		t.Errorf("invocations mismatch (-want +got):\n%s", diff)
	}
}

func TestPublishNoop(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	publishedVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}
	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}

	setupMockPubdevServer(t, publishedVersions)
	setupTestRepo(t, repoVersions)
	logFile := setupFakeDartAndApitool(t, repoVersions, nil, nil)

	cfg := &config.Config{
		Default: &config.Default{
			Output: "generated",
		},
		Libraries: []*config.Library{
			{Name: "a", Version: "1.0.0"},
			{Name: "b", Version: "1.0.0"},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	gotInvocations := readLogFile(t, logFile)
	if len(gotInvocations) > 0 {
		t.Errorf("expected no publish invocations, got: %v", gotInvocations)
	}
}

func TestPublishErrorGreaterPublishedVersion(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	publishedVersions := map[string]string{
		"a": "2.0.0",
		"b": "1.0.0",
	}
	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}

	setupMockPubdevServer(t, publishedVersions)
	setupTestRepo(t, repoVersions)
	_ = setupFakeDartAndApitool(t, repoVersions, nil, nil)

	cfg := &config.Config{
		Default: &config.Default{
			Output: "generated",
		},
		Libraries: []*config.Library{
			{Name: "a", Version: "1.0.0"},
			{Name: "b", Version: "1.0.0"},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	})
	wantErr := `published version "2.0.0" is greater than repo version "1.0.0" for package a`
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantErr)
	}
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("expected error containing %q, got %q", wantErr, err.Error())
	}
}

func TestPublishSemverFailure(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	publishedVersions := map[string]string{
		"a": "0.9.0",
		"b": "0.9.0",
	}
	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}
	mockApitoolErrors := map[string]string{
		"a": "package a version 1.0.0 is less than required version 2.0.0 recommended by dart-apitool",
	}

	setupMockPubdevServer(t, publishedVersions)
	setupTestRepo(t, repoVersions)
	_ = setupFakeDartAndApitool(t, repoVersions, nil, mockApitoolErrors)

	cfg := &config.Config{
		Default: &config.Default{
			Output: "generated",
		},
		Libraries: []*config.Library{
			{Name: "a", Version: "1.0.0"},
			{Name: "b", Version: "1.0.0"},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config: cfg,
		DryRun: true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want1 := "dart-apitool failed for a:"
	want2 := "package a version 1.0.0 is less than required version 2.0.0 recommended by dart-apitool"
	if !strings.Contains(err.Error(), want1) || !strings.Contains(err.Error(), want2) {
		t.Fatalf("expected error containing %q and %q, got %q", want1, want2, err.Error())
	}
}

func TestPublishSemverSkipByFlag(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	publishedVersions := map[string]string{
		"a": "0.9.0",
		"b": "0.9.0",
	}
	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}
	mockApitoolErrors := map[string]string{
		"a": "package a version 1.0.0 is less than required version 2.0.0 recommended by dart-apitool",
	}

	setupMockPubdevServer(t, publishedVersions)
	setupTestRepo(t, repoVersions)
	logFile := setupFakeDartAndApitool(t, repoVersions, nil, mockApitoolErrors)

	cfg := &config.Config{
		Default: &config.Default{
			Output: "generated",
		},
		Libraries: []*config.Library{
			{Name: "a", Version: "1.0.0"},
			{Name: "b", Version: "1.0.0"},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config:           cfg,
		DryRun:           true,
		SkipSemverChecks: true,
	})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	gotInvocations := readLogFile(t, logFile)
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	wantInvocations := []string{
		filepath.Join(workingDir, "generated/a") + "|pub publish --skip-validation --dry-run",
		filepath.Join(workingDir, "generated/b") + "|pub publish --skip-validation --dry-run",
	}

	if diff := cmp.Diff(wantInvocations, gotInvocations); diff != "" {
		t.Errorf("invocations mismatch (-want +got):\n%s", diff)
	}
}

// Helpers

func setupMockPubdevServer(t *testing.T, mockPublishedVersions map[string]string) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pkgName := filepath.Base(r.URL.Path)
		version, ok := mockPublishedVersions[pkgName]
		if !ok || version == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": pkgName,
			"latest": map[string]string{
				"version": version,
			},
		})
	}))
	t.Cleanup(ts.Close)

	oldPubdevAPIURL := pubdevAPIURL
	pubdevAPIURL = ts.URL + "/"
	t.Cleanup(func() {
		pubdevAPIURL = oldPubdevAPIURL
	})
}

func setupTestRepo(t *testing.T, repoVersions map[string]string) {
	t.Helper()
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

	pubspecA := "name: a\nversion: " + repoVersions["a"] + "\ndependencies:\n  sdk: \">=3.0.0 <4.0.0\"\n"
	pubspecB := "name: b\nversion: " + repoVersions["b"] + "\ndependencies:\n  sdk: \">=3.0.0 <4.0.0\"\n  a: ^1.0.0\n"

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
}

func setupFakeDartAndApitool(t *testing.T, repoVersions map[string]string, mockNeededVersions map[string]string, mockApitoolErrors map[string]string) string {
	t.Helper()
	// Propagate mock expectations to environment for scripts to read
	if mockNeededVersions != nil {
		marshaled, _ := json.Marshal(mockNeededVersions)
		t.Setenv("MOCK_NEEDED_VERSIONS", string(marshaled))
	} else {
		t.Setenv("MOCK_NEEDED_VERSIONS", "")
	}
	if mockApitoolErrors != nil {
		marshaled, _ := json.Marshal(mockApitoolErrors)
		t.Setenv("MOCK_APITOOL_ERRORS", string(marshaled))
	} else {
		t.Setenv("MOCK_APITOOL_ERRORS", "")
	}

	// Set up fake dart script
	setupFakeScript(t, "dart", `#!/bin/bash
if [ "$1" == "pub" ] && [ "$2" == "deps" ] && [ "$3" == "--json" ]; then
	echo '{"packages":[{"name":"a","version":"`+repoVersions["a"]+`","dependencies":[]},{"name":"b","version":"`+repoVersions["b"]+`","dependencies":["a"]}]}'
else
	echo "$(pwd)|$*" >> "$TEST_LOG_FILE"
fi
`)

	// Set up fake apitool script
	setupFakeScript(t, "dart-apitool", `#!/bin/bash
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

if [ -n "$MOCK_APITOOL_ERRORS" ]; then
  err_msg=$(echo "$MOCK_APITOOL_ERRORS" | jq -r ".${pkg_name} // \"\"")
  if [ -n "$err_msg" ] && [ "$err_msg" != "null" ]; then
    echo "$err_msg" >&2
    exit 1
  fi
fi

if [ -n "$report_file" ] && [ -n "$MOCK_NEEDED_VERSIONS" ]; then
  needed_version=$(echo "$MOCK_NEEDED_VERSIONS" | jq -r ".${pkg_name} // \"\"")
  if [ -n "$needed_version" ] && [ "$needed_version" != "null" ]; then
    echo "{\"version\":{\"needed\":\"$needed_version\"}}" > "$report_file"
  fi
fi
`)

	logFile := filepath.Join(t.TempDir(), "invocations.log")
	t.Setenv("TEST_LOG_FILE", logFile)
	return logFile
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

func readLogFile(t *testing.T, logFile string) []string {
	t.Helper()
	var gotInvocations []string
	if _, err := os.Stat(logFile); err == nil {
		logContent, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatal(err)
		}
		gotInvocations = strings.Split(strings.TrimSpace(string(logContent)), "\n")
		gotInvocations = slices.DeleteFunc(gotInvocations, func(s string) bool { return s == "" })
	}
	return gotInvocations
}

func TestSortByDeps(t *testing.T) {
	for _, tc := range []struct {
		name          string
		libraryNames  []string
		deps          map[string][]string
		want          []string
		wantErrSubstr string
	}{
		{
			name:         "empty",
			libraryNames: nil,
			deps:         map[string][]string{},
			want:         nil,
		},
		{
			name:         "single library",
			libraryNames: []string{"a"},
			deps:         map[string][]string{},
			want:         []string{"a"},
		},
		{
			name:         "independent libraries (stable sort)",
			libraryNames: []string{"b", "a", "c"},
			deps:         map[string][]string{},
			want:         []string{"a", "b", "c"},
		},
		{
			name:         "simple chain",
			libraryNames: []string{"a", "b"},
			deps: map[string][]string{
				"a": {"b"},
			},
			want: []string{"b", "a"},
		},
		{
			name:         "simple DAG",
			libraryNames: []string{"a", "b", "c"},
			deps: map[string][]string{
				"a": {"b", "c"},
				"b": {"c"},
			},
			want: []string{"c", "b", "a"},
		},
		{
			name:         "cycle detected",
			libraryNames: []string{"a", "b"},
			deps: map[string][]string{
				"a": {"b"},
				"b": {"a"},
			},
			wantErrSubstr: "cycle detected",
		},
		{
			name:         "self loop",
			libraryNames: []string{"a"},
			deps: map[string][]string{
				"a": {"a"},
			},
			wantErrSubstr: "cycle detected",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			libraryByName := make(map[string]*config.Library)
			for _, name := range tc.libraryNames {
				libraryByName[name] = &config.Library{Name: name}
			}

			got, err := sortByDeps(libraryByName, tc.deps)
			if tc.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErrSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("sortByDeps mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
