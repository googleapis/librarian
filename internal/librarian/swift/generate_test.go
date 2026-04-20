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

package swift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		api  string
		want string
	}{
		{"google/cloud/secretmanager/v1", "GoogleCloudSecretmanagerV1"},
		{"google/maps/addressvalidation/v1", "GoogleMapsAddressvalidationV1"},
		{"google/api/v1", "GoogleApiV1"},
		{"grafeas/v1", "GoogleGrafeasV1"},
	} {
		t.Run(test.api, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if got != test.want {
				t.Errorf("DefaultLibraryName(%q) = %q, want %q", test.api, got, test.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	libraries := []*config.Library{
		{
			Name:          "GoogleType",
			APIs:          []*config.API{{Path: "google/type"}},
			CopyrightYear: "2038",
			Swift: &config.SwiftPackage{
				SwiftDefault: config.SwiftDefault{
					Dependencies: []config.SwiftDependency{
						{Name: "GoogleCloudWkt", ApiPackage: "google.protobuf"},
					},
				},
			},
		},
	}
	for _, library := range libraries {
		library.Output = filepath.Join(outDir, "generated", library.Name)
	}
	src := &sources.Sources{
		Googleapis: googleapisDir,
	}
	cfg := &config.Config{}

	for _, library := range libraries {
		if err := Generate(t.Context(), cfg, library, src); err != nil {
			t.Fatal(err)
		}
	}

	for _, library := range libraries {
		expectedFile := filepath.Join(library.Output, "README.md")
		if _, err := os.Stat(expectedFile); err != nil {
			t.Errorf("Stat(%s) returned error: %v", expectedFile, err)
		}
	}
}

func TestFormat(t *testing.T) {
	testhelper.RequireCommand(t, "swift-format")

	outDir := t.TempDir()
	sourcesDir := filepath.Join(outDir, "Sources")
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(sourcesDir, "test.swift")
	// Write a file that needs formatting.
	if err := os.WriteFile(filePath, []byte("func foo(){\n  print(\"hello\")\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Output: outDir,
	}

	if err := Format(t.Context(), library); err != nil {
		t.Fatal(err)
	}
}
