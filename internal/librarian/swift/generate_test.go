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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

const (
	googleapisRoot = "../../../internal/testdata/googleapis"
	discoveryRoot  = "fake/path/to/testdata/discovery"
)

func absPath(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func defaultSwiftConfig(t *testing.T) *config.SwiftPackage {
	// A package providing the `google.protobuf` API is required, as that package provides the
	// well-known types and the support for `Any`.
	t.Helper()
	return &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: []config.SwiftDependency{
				{Name: "FakeWkt", ApiPackage: "google.protobuf"},
			},
		},
	}
}

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
			Name:                "GoogleType",
			APIs:                []*config.API{{Path: "google/type"}},
			CopyrightYear:       "2038",
			SpecificationFormat: config.SpecProtobuf,
			Swift:               defaultSwiftConfig(t),
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

func TestLibraryToModelConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		api     *config.API
		want    *parser.ModelConfig
	}{
		{
			name: "minimal config",
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				CopyrightYear: "2038",
				Version:       "1.2.3",
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            config.LanguageSwift,
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Codec:               map[string]string{"copyright-year": "2038", "version": "1.2.3"},
				Source: &sources.SourceConfig{
					ActiveRoots: []string{"googleapis"},
				},
			},
		},
		{
			name: "explicit specification format",
			library: &config.Library{
				Name:                "google-cloud-secretmanager",
				CopyrightYear:       "2038",
				Version:             "1.2.3",
				SpecificationFormat: config.SpecProtobuf,
			},
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &parser.ModelConfig{
				Language:            config.LanguageSwift,
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "google/cloud/secretmanager/v1",
				ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				Codec:               map[string]string{"copyright-year": "2038", "version": "1.2.3"},
				Source: &sources.SourceConfig{
					ActiveRoots: []string{"googleapis"},
				},
			},
		},
		{
			name: "discovery config",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				CopyrightYear:       "2038",
				Version:             "1.2.3",
				Roots:               []string{"discovery", "googleapis"},
				SpecificationFormat: config.SpecDiscovery,
			},
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &parser.ModelConfig{
				Language:            config.LanguageSwift,
				SpecificationFormat: config.SpecDiscovery,
				SpecificationSource: "discoveries/compute.v1.json",
				ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				Codec:               map[string]string{"copyright-year": "2038", "version": "1.2.3"},
				Source: &sources.SourceConfig{
					ActiveRoots: []string{"discovery", "googleapis"},
				},
			},
		},
		{
			name: "discovery config with LRO",
			library: &config.Library{
				Name:                "google-cloud-compute-v1",
				CopyrightYear:       "2038",
				Version:             "1.2.3",
				Roots:               []string{"discovery", "googleapis"},
				SpecificationFormat: config.SpecDiscovery,
				Swift: &config.SwiftPackage{
					Discovery: &config.SwiftDiscovery{
						OperationID: ".google.cloud.compute.v1.",
						Pollers: []config.SwiftPoller{
							{
								Prefix:   "compute/v1/projects/{project}/zones/{zone}",
								MethodID: ".google.cloud.compute.v1.zoneOperations.get",
							},
						},
					},
				},
			},
			api: &config.API{
				Path: "discoveries/compute.v1.json",
			},
			want: &parser.ModelConfig{
				Language:            config.LanguageSwift,
				SpecificationFormat: config.SpecDiscovery,
				SpecificationSource: "discoveries/compute.v1.json",
				ServiceConfig:       "google/cloud/compute/v1/compute_v1.yaml",
				Codec:               map[string]string{"copyright-year": "2038", "version": "1.2.3"},
				Source: &sources.SourceConfig{
					ActiveRoots: []string{"discovery", "googleapis"},
				},
				Discovery: &api.Discovery{
					OperationID: ".google.cloud.compute.v1.",
					Pollers: []*api.Poller{
						{
							Prefix:   "compute/v1/projects/{project}/zones/{zone}",
							MethodID: ".google.cloud.compute.v1.zoneOperations.get",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			srcs := &sources.Sources{
				Discovery:  absPath(t, discoveryRoot),
				Googleapis: absPath(t, googleapisRoot),
			}
			if test.want.Source == nil {
				t.Fatal("bad test expectation: test.want.Sources should not be nil")
			}
			// Avoid typing this in every input
			test.want.Source.Sources = srcs

			got, err := libraryToModelConfig(test.library, test.api, srcs)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
