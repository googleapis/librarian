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

func TestSimplifyRestNumericEnums(t *testing.T) {
	for _, test := range []struct {
		name             string
		restNumericEnums map[string]bool
		want             map[string]bool
	}{
		{
			name: "all true",
			restNumericEnums: map[string]bool{
				"csharp": true,
				"go":     true,
				"java":   true,
				"nodejs": true,
				"php":    true,
				"python": true,
				"ruby":   true,
			},
			want: map[string]bool{"all": true},
		},
		{
			name: "all false",
			restNumericEnums: map[string]bool{
				"csharp": false,
				"go":     false,
				"java":   false,
				"nodejs": false,
				"php":    false,
				"python": false,
				"ruby":   false,
			},
			want: map[string]bool{},
		},
		{
			name: "all present, different values",
			restNumericEnums: map[string]bool{
				"csharp": false,
				"go":     true,
				"java":   false,
				"nodejs": false,
				"php":    false,
				"python": true,
				"ruby":   false,
			},
			want: map[string]bool{
				"go":     true,
				"python": true,
			},
		},
		{
			name: "some present, different values",
			restNumericEnums: map[string]bool{
				"csharp": false,
				"java":   true,
				"python": true,
				"ruby":   false,
			},
			want: map[string]bool{
				"java":   true,
				"python": true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := simplifyRESTNumericEnums(test.restNumericEnums)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunUpdateRestNumericEnums(t *testing.T) {
	for _, test := range []struct {
		name       string
		apiGo      string
		buildBazel string
		want       string
	}{
		{
			name: "add new no rest numeric enums",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
			buildBazel: `
go_gapic_library(
    name = "google-cloud-foo-v1-go",
    rest_numeric_enums = False,
)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangGo: true}},
}
`,
		},
		{
			name: "update existing no rest numeric enums",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangAll: true}},
}
`,
			buildBazel: `
php_gapic_library(
    name = "google-cloud-foo-v1-php",
    rest_numeric_enums = False,
)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangPhp: true}},
}
`,
		},
		{
			name: "simplify all languages same (False)",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
			buildBazel: `
csharp_gapic_library(name = "foo-csharp", rest_numeric_enums = False)
go_gapic_library(name = "foo-go", rest_numeric_enums = False)
java_gapic_library(name = "foo-java", rest_numeric_enums = False)
nodejs_gapic_library(name = "foo-nodejs", rest_numeric_enums = False)
php_gapic_library(name = "foo-php", rest_numeric_enums = False)
py_gapic_library(name = "foo-python", rest_numeric_enums = False)
ruby_cloud_gapic_library(name = "foo-ruby", rest_numeric_enums = False)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangAll: true}},
}
`,
		},
		{
			name: "remove if all languages have default",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangAll: true}},
}
`,
			buildBazel: `
csharp_gapic_library(name = "foo-csharp", rest_numeric_enums = True)
go_gapic_library(name = "foo-go", rest_numeric_enums = True)
java_gapic_library(name = "foo-java", rest_numeric_enums = True)
nodejs_gapic_library(name = "foo-nodejs", rest_numeric_enums = True)
php_gapic_library(name = "foo-php", rest_numeric_enums = True)
py_gapic_library(name = "foo-python", rest_numeric_enums = True)
ruby_cloud_gapic_library(name = "foo-ruby", rest_numeric_enums = True)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
		},
		{
			name: "remove if BUILD.bazel is missing",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangAll: true}},
}
`,
			buildBazel: "", // No file will be created
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
		},
		{
			name: "remains same if all languages have default",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
			buildBazel: `
csharp_gapic_library(name = "foo-csharp", rest_numeric_enums = True)
go_gapic_library(name = "foo-go", rest_numeric_enums = True)
java_gapic_library(name = "foo-java")
nodejs_gapic_library(name = "foo-nodejs", rest_numeric_enums = True)
php_gapic_library(name = "foo-php", rest_numeric_enums = True)
py_gapic_library(name = "foo-python")
ruby_cloud_gapic_library(name = "foo-ruby", rest_numeric_enums = True)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
		},
		{
			name: "all present, different values",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1"},
}
`,
			buildBazel: `
csharp_gapic_library(name = "foo-csharp", rest_numeric_enums = True)
go_gapic_library(name = "foo-go", rest_numeric_enums = False)
java_gapic_library(name = "foo-java")
nodejs_gapic_library(name = "foo-nodejs", rest_numeric_enums = True)
php_gapic_library(name = "foo-php", rest_numeric_enums = False)
py_gapic_library(name = "foo-python")
ruby_cloud_gapic_library(name = "foo-ruby", rest_numeric_enums = True)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangGo: true, LangPhp: true}},
}
`,
		},
		{
			name: "merge",
			apiGo: `package serviceconfig
var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangCsharp: true}},
}
`,
			buildBazel: `
csharp_gapic_library(name = "foo-csharp", rest_numeric_enums = False)
go_gapic_library(name = "foo-go", rest_numeric_enums = False)
`,
			want: `package serviceconfig

var APIs = []API{
	{Path: "google/cloud/foo/v1", NoRESTNumericEnums: map[string]bool{LangCsharp: true, LangGo: true}},
}
`},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			apiGoPath := filepath.Join(tmpDir, "api.go")
			if err := os.WriteFile(apiGoPath, []byte(test.apiGo), 0644); err != nil {
				t.Fatal(err)
			}

			googleapisDir := filepath.Join(tmpDir, "googleapis")
			apiPath := "google/cloud/foo/v1"
			if test.buildBazel != "" {
				buildBazelDir := filepath.Join(googleapisDir, apiPath)
				if err := os.MkdirAll(buildBazelDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(buildBazelDir, "BUILD.bazel"), []byte(test.buildBazel), 0644); err != nil {
					t.Fatal(err)
				}
			}

			if err := runUpdateRestNumericEnums(apiGoPath, googleapisDir); err != nil {
				t.Fatalf("runUpdateRestNumericEnums() error = %v", err)
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

func TestRunUpdateRestNumericEnums_Error(t *testing.T) {
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
			if err := runUpdateRestNumericEnums(test.apiGoPath, googleapisDir); err == nil {
				t.Error("runUpdateRestNumericEnums() error = nil, want error")
			}
		})
	}
}

func TestReadRestNumericEnums_Error(t *testing.T) {
	tmpDir := t.TempDir()
	got := readRestNumericEnums(tmpDir, "missing")
	if got != nil {
		t.Errorf("readRestNumericEnums() = %v, want nil", got)
	}
}

func TestUpdateRestNumericEnumsCommand(t *testing.T) {
	cmd := updateRestNumericEnumsCommand()
	ctx := t.Context()
	// Just test that the command can be initialized and run with --help.
	if err := cmd.Run(ctx, []string{"update-rest-numeric-enums", "--help"}); err != nil {
		t.Fatalf("cmd.Run() error = %v", err)
	}
}
