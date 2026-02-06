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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRunUpdateTransports(t *testing.T) {
	for _, test := range []struct {
		name       string
		apiGo      string
		buildBazel string
		want       string
	}{
		{
			name: "update existing transports",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1", Transports: map[string]Transport{langAll: Rest}},
}
`,
			buildBazel: `
php_gapic_library(
    name = "google-cloud-foo-v1-php",
    transport = "grpc+rest",
)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", Transports: map[string]Transport{langPhp: GRPCRest}},
}
`,
		},
		{
			name: "add new transports",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
			buildBazel: `
go_gapic_library(
    name = "google-cloud-foo-v1-go",
    transport = "grpc",
)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", Transports: map[string]Transport{langGo: GRPC}},
}
`,
		},
		{
			name: "simplify all languages same transport",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
			buildBazel: `
csharp_gapic_library(name = "foo-csharp", transport = "rest")
go_gapic_library(name = "foo-go", transport = "rest")
java_gapic_library(name = "foo-java", transport = "rest")
nodejs_gapic_library(name = "foo-nodejs", transport = "rest")
php_gapic_library(name = "foo-php", transport = "rest")
py_gapic_library(name = "foo-python", transport = "rest")
ruby_cloud_gapic_library(name = "foo-ruby", transport = "rest")
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", Transports: map[string]Transport{langAll: Rest}},
}
`,
		},
		{
			name: "omit default GRPCRest for all",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1", Transports: map[string]Transport{langAll: Rest}},
}
`,
			buildBazel: `
csharp_gapic_library(name = "foo-csharp", transport = "grpc+rest")
go_gapic_library(name = "foo-go", transport = "grpc+rest")
java_gapic_library(name = "foo-java", transport = "grpc+rest")
nodejs_gapic_library(name = "foo-nodejs", transport = "grpc+rest")
php_gapic_library(name = "foo-php", transport = "grpc+rest")
py_gapic_library(name = "foo-python", transport = "grpc+rest")
ruby_cloud_gapic_library(name = "foo-ruby", transport = "grpc+rest")
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			apiGoPath := filepath.Join(tmpDir, "api.go")
			if err := os.WriteFile(apiGoPath, []byte(test.apiGo), 0644); err != nil {
				t.Fatal(err)
			}

			googleapisDir := filepath.Join(tmpDir, "googleapis")
			apiPath := "google/cloud/foo/v1"
			buildBazelDir := filepath.Join(googleapisDir, apiPath)
			if err := os.MkdirAll(buildBazelDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(buildBazelDir, "BUILD.bazel"), []byte(test.buildBazel), 0644); err != nil {
				t.Fatal(err)
			}
			if err := runUpdateTransports(apiGoPath, googleapisDir); err != nil {
				t.Fatalf("runUpdateTransports() error = %v", err)
			}
			got, err := os.ReadFile(apiGoPath)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(strings.TrimSpace(test.want), strings.TrimSpace(string(got))); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunUpdateTransports_Error(t *testing.T) {
	tmpDir := t.TempDir()
	googleapisDir := filepath.Join(tmpDir, "googleapis")

	for _, test := range []struct {
		name      string
		apiGo     string
		apiGoPath string
	}{
		{
			name:      "invalid go file",
			apiGo:     "invalid go code",
			apiGoPath: filepath.Join(tmpDir, "invalid.go"),
		},
		{
			name:      "missing APIs variable",
			apiGo:     "package foo\nvar Other = 1",
			apiGoPath: filepath.Join(tmpDir, "missing_apis.go"),
		},
		{
			name:      "non-existent apiGoPath",
			apiGoPath: filepath.Join(tmpDir, "non_existent.go"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.apiGo != "" {
				if err := os.WriteFile(test.apiGoPath, []byte(test.apiGo), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := runUpdateTransports(test.apiGoPath, googleapisDir); err == nil {
				t.Error("runUpdateTransports() error = nil, want error")
			}
		})
	}
}

func TestSimplifyTransports(t *testing.T) {
	for _, test := range []struct {
		name       string
		transports map[string]string
		want       map[string]string
	}{
		{
			name: "all same",
			transports: map[string]string{
				"csharp": "rest",
				"go":     "rest",
				"java":   "rest",
				"nodejs": "rest",
				"php":    "rest",
				"python": "rest",
				"ruby":   "rest",
			},
			want: map[string]string{"all": "rest"},
		},
		{
			name: "one different",
			transports: map[string]string{
				"csharp": "rest",
				"go":     "grpc",
				"java":   "rest",
				"nodejs": "rest",
				"php":    "rest",
				"python": "rest",
				"ruby":   "rest",
			},
			want: map[string]string{
				"csharp": "rest",
				"go":     "grpc",
				"java":   "rest",
				"nodejs": "rest",
				"php":    "rest",
				"python": "rest",
				"ruby":   "rest",
			},
		},
		{
			name: "incomplete",
			transports: map[string]string{
				"go": "grpc",
			},
			want: map[string]string{
				"go": "grpc",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := simplifyTransports(test.transports)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLangConstantExists(t *testing.T) {
	for _, test := range []struct {
		lang string
		want bool
	}{
		{"all", true},
		{"go", true},
		{"rust", true},
		{"unknown", false},
	} {
		t.Run(test.lang, func(t *testing.T) {
			got := langConstantExists(test.lang)
			if got != test.want {
				t.Errorf("langConstantExists(%q) = %v, want %v", test.lang, got, test.want)
			}
		})
	}
}

func TestReadTransports_Error(t *testing.T) {
	tmpDir := t.TempDir()
	got := readTransports(tmpDir, "missing")
	if got != nil {
		t.Errorf("readTransports() = %v, want nil", got)
	}
}

func TestUpdateTransportsCommand(t *testing.T) {
	cmd := updateTransportsCommand()
	ctx := t.Context()
	// Just test that the command can be initialized and run with --help.
	if err := cmd.Run(ctx, []string{"update-transports", "--help"}); err != nil {
		t.Fatalf("cmd.Run() error = %v", err)
	}
}
