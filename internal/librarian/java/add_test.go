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

package java

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestAdd(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "standard cloud API",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
			},
		},
		{
			name: "shopping API",
			lib: &config.Library{
				Name: "shopping-css",
				APIs: []*config.API{
					{Path: "google/shopping/css/v1"},
				},
			},
			want: &config.Library{
				Name: "shopping-css",
				APIs: []*config.API{
					{Path: "google/shopping/css/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID: "google-shopping-css",
					GroupID:    "com.google.shopping",
				},
			},
		},
		{
			name: "maps API",
			lib: &config.Library{
				Name: "maps-routing",
				APIs: []*config.API{
					{Path: "google/maps/routing/v1"},
				},
			},
			want: &config.Library{
				Name: "maps-routing",
				APIs: []*config.API{
					{Path: "google/maps/routing/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID: "google-maps-routing",
					GroupID:    "com.google.maps",
				},
			},
		},
		{
			name: "unrecognized non-cloud API",
			lib: &config.Library{
				Name: "foo-bar",
				APIs: []*config.API{
					{Path: "google/foo/bar/v1"},
				},
			},
			want: &config.Library{
				Name: "foo-bar",
				APIs: []*config.API{
					{Path: "google/foo/bar/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID: "google-foo-bar",
					GroupID:    "please-configure-java-group-id",
				},
			},
		},
		{
			name: "ads API",
			lib: &config.Library{
				Name: "ads-admanager",
				APIs: []*config.API{
					{Path: "google/ads/admanager/v1"},
				},
			},
			want: &config.Library{
				Name: "ads-admanager",
				APIs: []*config.API{
					{Path: "google/ads/admanager/v1"},
				},
				Version:       defaultVersion,
				CopyrightYear: "",
				Java: &config.JavaModule{
					ArtifactID: "google-ads-admanager",
					GroupID:    "com.google.api-ads",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(versionsFileName, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := Add(test.lib, nil)
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdd_VersionsTxt(t *testing.T) {
	for _, test := range []struct {
		name         string
		lib          *config.Library
		wantVersions []string
	}{
		{
			name: "standard cloud API",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantVersions: []string{
				"google-cloud-secretmanager-parent:0.0.0:0.1.0-SNAPSHOT",
				"google-cloud-secretmanager-bom:0.0.0:0.1.0-SNAPSHOT",
				"proto-google-cloud-secretmanager-v1:0.0.0:0.1.0-SNAPSHOT",
				"grpc-google-cloud-secretmanager-v1:0.0.0:0.1.0-SNAPSHOT",
				"google-cloud-secretmanager:0.0.0:0.1.0-SNAPSHOT",
			},
		},
		{
			name: "shopping API",
			lib: &config.Library{
				Name: "shopping-css",
				APIs: []*config.API{
					{Path: "google/shopping/css/v1"},
				},
			},
			wantVersions: []string{
				"google-shopping-css-parent:0.0.0:0.1.0-SNAPSHOT",
				"google-shopping-css-bom:0.0.0:0.1.0-SNAPSHOT",
				"proto-google-shopping-css-v1:0.0.0:0.1.0-SNAPSHOT",
				"grpc-google-shopping-css-v1:0.0.0:0.1.0-SNAPSHOT",
				"google-shopping-css:0.0.0:0.1.0-SNAPSHOT",
			},
		},
		{
			name: "maps API",
			lib: &config.Library{
				Name: "maps-routing",
				APIs: []*config.API{
					{Path: "google/maps/routing/v1"},
				},
			},
			wantVersions: []string{
				"google-maps-routing-parent:0.0.0:0.1.0-SNAPSHOT",
				"google-maps-routing-bom:0.0.0:0.1.0-SNAPSHOT",
				"proto-google-maps-routing-v1:0.0.0:0.1.0-SNAPSHOT",
				"grpc-google-maps-routing-v1:0.0.0:0.1.0-SNAPSHOT",
				"google-maps-routing:0.0.0:0.1.0-SNAPSHOT",
			},
		},
		{
			name: "unrecognized non-cloud API",
			lib: &config.Library{
				Name: "foo-bar",
				APIs: []*config.API{
					{Path: "google/foo/bar/v1"},
				},
			},
			wantVersions: []string{
				"google-foo-bar-parent:0.0.0:0.1.0-SNAPSHOT",
				"google-foo-bar-bom:0.0.0:0.1.0-SNAPSHOT",
				"proto-google-foo-bar-v1:0.0.0:0.1.0-SNAPSHOT",
				"grpc-google-foo-bar-v1:0.0.0:0.1.0-SNAPSHOT",
				"google-foo-bar:0.0.0:0.1.0-SNAPSHOT",
			},
		},
		{
			name: "ads API",
			lib: &config.Library{
				Name: "ads-admanager",
				APIs: []*config.API{
					{Path: "google/ads/admanager/v1"},
				},
			},
			wantVersions: []string{
				"google-ads-admanager-parent:0.0.0:0.1.0-SNAPSHOT",
				"google-ads-admanager-bom:0.0.0:0.1.0-SNAPSHOT",
				"proto-google-ads-admanager-v1:0.0.0:0.1.0-SNAPSHOT",
				"google-ads-admanager:0.0.0:0.1.0-SNAPSHOT",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(versionsFileName, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Add(test.lib, nil)
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
			content, err := os.ReadFile(versionsFileName)
			if err != nil {
				t.Fatal(err)
			}
			var gotVersions []string
			for line := range strings.SplitSeq(string(content), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					gotVersions = append(gotVersions, line)
				}
			}
			if diff := cmp.Diff(test.wantVersions, gotVersions); diff != "" {
				t.Errorf("versions mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdd_ExistingLibrary(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	// Write initial versions
	initial := []string{
		"google-cloud-secretmanager-parent:1.0.0:1.1.0-SNAPSHOT",
		"google-cloud-secretmanager-bom:1.0.0:1.1.0-SNAPSHOT",
		"proto-google-cloud-secretmanager-v1:1.0.0:1.1.0-SNAPSHOT",
		"grpc-google-cloud-secretmanager-v1:1.0.0:1.1.0-SNAPSHOT",
		"google-cloud-secretmanager:1.0.0:1.1.0-SNAPSHOT",
	}
	if err := os.WriteFile(versionsFileName, []byte(strings.Join(initial, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name:    "secretmanager",
		Version: "1.1.0-SNAPSHOT",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
			{Path: "google/cloud/secretmanager/v1beta1"},
		},
	}
	addedAPI := &config.API{Path: "google/cloud/secretmanager/v1beta1"}

	got, err := Add(lib, addedAPI)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got.Version != "1.1.0-SNAPSHOT" {
		t.Errorf("version = %q, want %q", got.Version, "1.1.0-SNAPSHOT")
	}

	content, err := os.ReadFile(versionsFileName)
	if err != nil {
		t.Fatal(err)
	}
	var gotVersions []string
	for line := range strings.SplitSeq(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			gotVersions = append(gotVersions, line)
		}
	}

	wantVersions := append(initial, "proto-google-cloud-secretmanager-v1beta1:1.0.0:1.1.0-SNAPSHOT", "grpc-google-cloud-secretmanager-v1beta1:1.0.0:1.1.0-SNAPSHOT")
	if diff := cmp.Diff(wantVersions, gotVersions); diff != "" {
		t.Errorf("versions mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		api  string
		want string
	}{
		{"google/cloud/secretmanager/v1", "secretmanager"},
		{"google/api/serviceusage/v1", "serviceusage"},
		{"google/devtools/cloudbuild/v1", "cloudbuild"},
		{"google/pubsub/v1", "pubsub"},
		{"other/api/v1", "other-api"},
		{"google/cloud/datacatalog/lineage/v1", "datacatalog-lineage"},
		{"google/cloud/aiplatform/v1beta1", "aiplatform"},
		{"google/shopping/merchant/datasources/v1", "shopping-merchant-datasources"},
	} {
		t.Run(test.api, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAppendVersions(t *testing.T) {
	for _, test := range []struct {
		name    string
		initial string
		lines   []string
		want    string
	}{
		{
			name:    "empty file",
			initial: "",
			lines:   []string{"a:1.0.0"},
			want:    "a:1.0.0\n",
		},
		{
			name:    "already has newline",
			initial: "a:1.0.0\n",
			lines:   []string{"b:2.0.0"},
			want:    "a:1.0.0\nb:2.0.0\n",
		},
		{
			name:    "missing newline",
			initial: "a:1.0.0",
			lines:   []string{"b:2.0.0"},
			want:    "a:1.0.0\nb:2.0.0\n",
		},
		{
			name:    "multiple lines missing newline",
			initial: "a:1.0.0",
			lines:   []string{"b:2.0.0", "c:3.0.0"},
			want:    "a:1.0.0\nb:2.0.0\nc:3.0.0\n",
		},
		{
			name:    "no lines does nothing",
			initial: "a:1.0.0\n",
			lines:   nil,
			want:    "a:1.0.0\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Chdir(tmpDir)
			if err := os.WriteFile(versionsFileName, []byte(test.initial), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := appendVersions(test.lines); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(versionsFileName)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAppendVersions_Error(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	if err := os.Mkdir(versionsFileName, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := appendVersions([]string{"line"}); err == nil {
		t.Error("appendVersions() expected error, got nil")
	}
}
