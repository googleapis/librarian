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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRunRubyMigration(t *testing.T) {
	oldFetchSource := fetchSource
	t.Cleanup(func() {
		fetchSource = oldFetchSource
	})
	absGoogleapis, err := filepath.Abs("../../internal/testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	// Override fetchSource.
	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    absGoogleapis,
		}, nil
	}
	dir := t.TempDir()
	t.Chdir(dir)
	err = runRubyMigration(t.Context(), ".")
	if err != nil {
		t.Fatal(err)
	}
	// Verify librarian.yaml is written and contains the expected content.
	got, err := yaml.Read[config.Config](config.LibrarianYAML)
	if err != nil {
		t.Fatalf("reading generated librarian.yaml: %v", err)
	}
	want := &config.Config{
		Language: config.LanguageRuby,
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "abcd123",
				SHA256: "sha123",
			},
		},
		Tools: &config.Tools{
			Gem: []*config.GemTool{
				{
					Name:    "gapic-generator-cloud",
					Version: "0.49.0",
				},
				{
					Name:    "grpc",
					Version: "1.78.1",
				},
			},
			Protoc: &config.Protoc{
				Version: "33.2",
				SHA256:  "b24b53f87c151bfd48b112fe4c3a6e6574e5198874f38036aff41df3456b8caf",
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindRubyLibraries(t *testing.T) {
	for _, test := range []struct {
		name  string
		files []string
		want  []*config.Library
	}{
		{
			name: "single library with .OwlBot.yaml",
			files: []string{
				"google-cloud-secret_manager/.OwlBot.yaml",
			},
			want: []*config.Library{
				{Name: "google-cloud-secret_manager"},
			},
		},
		{
			name: "multiple libraries with non-library files and directories",
			files: []string{
				"google-cloud-secret_manager/.OwlBot.yaml",
				"google-cloud-storage/.OwlBot.yaml",
				"README.md",
				".OwlBot.yaml",
				"script/helper.rb",
			},
			want: []*config.Library{
				{Name: "google-cloud-secret_manager"},
				{Name: "google-cloud-storage"},
			},
		},
		{
			name:  "no libraries found",
			files: []string{"README.md", "script/helper.rb"},
			want:  nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range test.files {
				path := filepath.Join(dir, f)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := findRubyLibraries(dir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseAPIFromOwlBot(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "apigeeconnect v1 api",
			path: "testdata/ruby/parse_api_from_owlbot/apigeeconnect_v1.yaml",
			want: "google/cloud/apigeeconnect/v1",
		},
		{
			name: "marketingplatform admin v1alpha api",
			path: "testdata/ruby/parse_api_from_owlbot/marketing_v1alpha.yaml",
			want: "google/marketingplatform/admin/v1alpha",
		},
		{
			name: "video livestream v1 api",
			path: "testdata/ruby/parse_api_from_owlbot/video_v1.yaml",
			want: "google/cloud/video/livestream/v1",
		},
		{
			name: "wrapper library",
			path: "testdata/ruby/parse_api_from_owlbot/wrapper.yaml",
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseAPIFromOwlBot(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseWrapperOf(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		want      []*config.Library
	}{
		{
			name: "wrapper library with multiple versioned libraries",
			libraries: []*config.Library{
				{Name: "google-cloud-secret_manager-v1", APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}}},
				{Name: "google-cloud-secret_manager-v1beta1", APIs: []*config.API{{Path: "google/cloud/secretmanager/v1beta1"}}},
				{Name: "google-cloud-secret_manager"},
			},
			want: []*config.Library{
				{
					Name: "google-cloud-secret_manager",
					Ruby: &config.RubyPackage{
						WrapperOf: []string{
							"google-cloud-secret_manager-v1",
							"google-cloud-secret_manager-v1beta1",
						},
					},
				},
				{Name: "google-cloud-secret_manager-v1", APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}}},
				{Name: "google-cloud-secret_manager-v1beta1", APIs: []*config.API{{Path: "google/cloud/secretmanager/v1beta1"}}},
			},
		},
		{
			name: "library with APIs set is not treated as wrapper",
			libraries: []*config.Library{
				{Name: "google-cloud-storage-v2", APIs: []*config.API{{Path: "google/cloud/storage/v2"}}},
				{Name: "google-cloud-storage-v1", APIs: []*config.API{{Path: "google/cloud/storage/v1"}}},
			},
			want: []*config.Library{
				{Name: "google-cloud-storage-v1", APIs: []*config.API{{Path: "google/cloud/storage/v1"}}},
				{Name: "google-cloud-storage-v2", APIs: []*config.API{{Path: "google/cloud/storage/v2"}}},
			},
		},
		{
			name: "wrapper library with no matching versioned gems",
			libraries: []*config.Library{
				{Name: "google-cloud-storage"},
			},
			want: []*config.Library{
				{Name: "google-cloud-storage"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parseWrapperOf(test.libraries)
			if diff := cmp.Diff(test.want, test.libraries); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
