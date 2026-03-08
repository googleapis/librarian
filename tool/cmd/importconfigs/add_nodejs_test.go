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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestFindNodejsAPIPaths(t *testing.T) {
	nodeRepo := filepath.Join("testdata", "google-cloud-node")
	googleapisDir := filepath.Join("testdata", "googleapis")

	got, err := findNodejsAPIPaths(nodeRepo, googleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"google/ads/admanager/v1",
		"google/cloud/speech/v1",
		"google/cloud/translate/v3",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunAddNodejs_addsNodejsToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	sdkPath := filepath.Join(tmpDir, "sdk.yaml")

	// Write an initial sdk.yaml with an entry that lacks nodejs.
	initial := []serviceconfig.API{
		{
			Path:      "google/ads/admanager/v1",
			Languages: []string{"python"},
		},
	}
	if err := yaml.Write(sdkPath, initial); err != nil {
		t.Fatal(err)
	}

	nodeRepo := filepath.Join("testdata", "google-cloud-node")
	googleapisDir := filepath.Join("testdata", "googleapis")

	if err := runAddNodejs(sdkPath, nodeRepo, googleapisDir); err != nil {
		t.Fatal(err)
	}

	got, err := yaml.Read[[]serviceconfig.API](sdkPath)
	if err != nil {
		t.Fatal(err)
	}

	want := []serviceconfig.API{
		{
			Path:      "google/ads/admanager/v1",
			Languages: []string{"nodejs", "python"},
		},
	}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunAddNodejs_doesNotDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	sdkPath := filepath.Join(tmpDir, "sdk.yaml")

	// Write an initial sdk.yaml with nodejs already present.
	initial := []serviceconfig.API{
		{
			Path:      "google/ads/admanager/v1",
			Languages: []string{"nodejs", "python"},
		},
	}
	if err := yaml.Write(sdkPath, initial); err != nil {
		t.Fatal(err)
	}

	nodeRepo := filepath.Join("testdata", "google-cloud-node")
	googleapisDir := filepath.Join("testdata", "googleapis")

	if err := runAddNodejs(sdkPath, nodeRepo, googleapisDir); err != nil {
		t.Fatal(err)
	}

	got, err := yaml.Read[[]serviceconfig.API](sdkPath)
	if err != nil {
		t.Fatal(err)
	}

	want := []serviceconfig.API{
		{
			Path:      "google/ads/admanager/v1",
			Languages: []string{"nodejs", "python"},
		},
	}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRunAddNodejs_addsNewEntryForNonCloudPath(t *testing.T) {
	tmpDir := t.TempDir()
	sdkPath := filepath.Join(tmpDir, "sdk.yaml")

	// Write an empty sdk.yaml (no entries).
	if err := yaml.Write(sdkPath, []serviceconfig.API{}); err != nil {
		t.Fatal(err)
	}

	nodeRepo := filepath.Join("testdata", "google-cloud-node")
	googleapisDir := filepath.Join("testdata", "googleapis")

	if err := runAddNodejs(sdkPath, nodeRepo, googleapisDir); err != nil {
		t.Fatal(err)
	}

	got, err := yaml.Read[[]serviceconfig.API](sdkPath)
	if err != nil {
		t.Fatal(err)
	}

	// Only google/ads/admanager/v1 should be added since the google/cloud/
	// paths are implicitly allowed.
	want := []serviceconfig.API{
		{
			Path:      "google/ads/admanager/v1",
			Languages: []string{"nodejs"},
		},
	}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAPIPathsFromOwlBot(t *testing.T) {
	googleapisDir := filepath.Join("testdata", "googleapis")

	for _, test := range []struct {
		name       string
		owlBotYAML string
		wantPaths  []string
		wantErrSub string
	}{
		{
			name: "speech",
			owlBotYAML: filepath.Join("testdata", "google-cloud-node",
				"packages", "google-cloud-speech", ".OwlBot.yaml"),
			wantPaths: []string{"google/cloud/speech/v1"},
		},
		{
			name: "translate",
			owlBotYAML: filepath.Join("testdata", "google-cloud-node",
				"packages", "google-cloud-translate", ".OwlBot.yaml"),
			wantPaths: []string{"google/cloud/translate/v3"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := apiPathsFromOwlBot(test.owlBotYAML, googleapisDir)
			if test.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", test.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantPaths, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHasNodejsGapicLibrary(t *testing.T) {
	googleapisDir := filepath.Join("testdata", "googleapis")

	for _, test := range []struct {
		name    string
		apiPath string
		want    bool
	}{
		{
			name:    "exists",
			apiPath: "google/cloud/speech/v1",
			want:    true,
		},
		{
			name:    "no_build_file",
			apiPath: "google/cloud/nonexistent/v1",
			want:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := hasNodejsGapicLibrary(googleapisDir, test.apiPath)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("hasNodejsGapicLibrary(%q) = %v, want %v", test.apiPath, got, test.want)
			}
		})
	}
}

// TestRunAddNodejs_writesFile verifies the output file is valid YAML that
// can be re-read.
func TestRunAddNodejs_writesFile(t *testing.T) {
	tmpDir := t.TempDir()
	sdkPath := filepath.Join(tmpDir, "sdk.yaml")

	initial := []serviceconfig.API{
		{
			Path:      "google/ads/admanager/v1",
			Languages: []string{"python"},
		},
		{
			Path:      "google/cloud/speech/v1",
			Languages: []string{"go", "python"},
		},
	}
	if err := yaml.Write(sdkPath, initial); err != nil {
		t.Fatal(err)
	}

	nodeRepo := filepath.Join("testdata", "google-cloud-node")
	googleapisDir := filepath.Join("testdata", "googleapis")

	if err := runAddNodejs(sdkPath, nodeRepo, googleapisDir); err != nil {
		t.Fatal(err)
	}

	// Verify the file can be read back.
	data, err := os.ReadFile(sdkPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("output file is empty")
	}

	got, err := yaml.Read[[]serviceconfig.API](sdkPath)
	if err != nil {
		t.Fatal(err)
	}

	// Both entries should have nodejs added. speech is google/cloud/ but it
	// already has an entry in sdk.yaml, so it gets updated.
	want := []serviceconfig.API{
		{
			Path:      "google/ads/admanager/v1",
			Languages: []string{"nodejs", "python"},
		},
		{
			Path:      "google/cloud/speech/v1",
			Languages: []string{"go", "nodejs", "python"},
		},
	}
	if diff := cmp.Diff(want, *got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
