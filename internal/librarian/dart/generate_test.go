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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/source"
	"github.com/googleapis/librarian/internal/testhelper"
)

// TestGenerateLibraries performs simple testing that multiple libraries can
// be generated. Only the presence of a single expected file per library is
// performed; TestGenerate is responsible for more detailed testing of
// per-library generation.
func TestGenerateLibraries(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "dart")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	libraries := []*config.Library{
		{
			Name:          "google_cloud_rpc",
			APIs:          []*config.API{{Path: "google/rpc"}},
			CopyrightYear: "2025",
			Dart: &config.DartPackage{
				IssueTrackerURL: "https://placeholder",
				Packages: map[string]string{
					"package:google_cloud_protobuf": "^0.5.0",
				},
			},
		},
		{
			Name:          "google_type",
			APIs:          []*config.API{{Path: "google/type"}},
			CopyrightYear: "2025",
			Dart: &config.DartPackage{
				IssueTrackerURL: "https://placeholder",
				Packages: map[string]string{
					"package:google_cloud_protobuf": "^0.5.0",
				},
			},
		},
	}
	for _, library := range libraries {
		library.Output = filepath.Join(outDir, "generated", library.Name)
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}
	if err := GenerateLibraries(t.Context(), libraries, sources); err != nil {
		t.Fatal(err)
	}
	// Just check that a pubspec.yaml has been created for each library.
	for _, library := range libraries {
		expectedPubspec := filepath.Join(library.Output, "pubspec.yaml")
		_, err := os.Stat(expectedPubspec)
		if err != nil {
			t.Errorf("Stat(%s) returned error: %v", expectedPubspec, err)
		}
	}
}

func TestGenerateLibraries_Error(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "dart")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	libraries := []*config.Library{
		{
			Name:                "broken",
			SpecificationFormat: "invalid",
			Output:              filepath.Join(outDir, "broken"),
			APIs:                []*config.API{{Path: "broken"}},
		},
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}
	gotErr := GenerateLibraries(t.Context(), libraries, sources)
	wantErr := errInvalidSpecificationFormat
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("GenerateLibraries error = %v, wantErr %v", gotErr, wantErr)
	}
}

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "dart")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	library := &config.Library{
		Name:          "google-cloud-secretmanager-v1",
		Version:       "0.1.0",
		Output:        outDir,
		CopyrightYear: "2025",
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
			},
		},
		Dart: &config.DartPackage{
			APIKeysEnvironmentVariables: "GOOGLE_API_KEY",
			IssueTrackerURL:             "https://github.com/googleapis/google-cloud-dart/issues",
			Packages: map[string]string{
				"package:googleapis_auth":           "^2.0.0",
				"package:http":                      "^1.3.0",
				"package:google_cloud_api":          "^0.4.0",
				"package:google_cloud_iam_v1":       "^0.4.0",
				"package:google_cloud_protobuf":     "^0.4.0",
				"package:google_cloud_location":     "^0.4.0",
				"package:google_cloud_longrunning":  "^0.4.0",
				"package:google_cloud_logging_type": "^0.4.0",
				"package:google_cloud_rpc":          "^0.4.0",
				"package:google_cloud_type":         "^0.4.0",
			},
		},
	}
	sources := &source.Sources{
		Googleapis: googleapisDir,
	}
	if err := generate(t.Context(), library, sources); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path string
		want string
	}{
		{filepath.Join(outDir, "pubspec.yaml"), "name:"},
		{filepath.Join(outDir, "pubspec.yaml"), "google_cloud_secretmanager_v1"},
		{filepath.Join(outDir, "README.md"), "Secret Manager"},
		{filepath.Join(outDir, "lib", "secretmanager.dart"), "library"},
	} {
		t.Run(test.path, func(t *testing.T) {
			if _, err := os.Stat(test.path); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), test.want) {
				t.Errorf("%q missing expected string: %q", test.path, test.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "dart")
	outDir := t.TempDir()
	dartFile := filepath.Join(outDir, "test.dart")
	if err := os.WriteFile(dartFile, []byte("void main() { print('hello'); }"), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Output: outDir,
	}
	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
}

func TestDeriveAPIPath(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  string
		want string
	}{
		{
			name: "simple",
			lib:  "google_cloud_secretmanager_v1",
			want: "google/cloud/secretmanager/v1",
		},
		{
			name: "no underscore",
			lib:  "name",
			want: "name",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DeriveAPIPath(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name          string
		libName       string
		defaultOutput string
		want          string
	}{
		{
			name:          "simple case",
			libName:       "google-cloud-secretmanager-v1",
			defaultOutput: "packages/",
			want:          "packages/google-cloud-secretmanager-v1",
		},
		{
			name:          "empty default output",
			libName:       "my-lib",
			defaultOutput: "",
			want:          "my-lib",
		},
		{
			name:          "empty lib name",
			libName:       "",
			defaultOutput: "packages/",
			want:          "packages",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.libName, test.defaultOutput)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "google/ as prefix",
			path: "google/ai/generativelanguage/v1beta",
			want: "google_cloud_ai_generativelanguage_v1beta",
		},
		{
			name: "google/cloud/ as prefix",
			path: "google/cloud/example/nested/v1",
			want: "google_cloud_example_nested_v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultLibraryName(test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
