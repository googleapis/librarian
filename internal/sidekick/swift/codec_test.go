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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestParseOptions(t *testing.T) {
	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year":        "2038",
			"package-name-override": "GoogleCloudBigtable",
			"root-name":             "test-root",
		},
	}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	got, err := newCodec(model, cfg, nil, ".")
	if err != nil {
		t.Fatal(err)
	}
	want := &codec{
		GenerationYear: "2038",
		PackageName:    "GoogleCloudBigtable",
		MonorepoRoot:   ".",
		RootName:       "test-root",
		Model:          model,
		ApiPackages:    map[string]*Dependency{},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in codec (-want, +got)\n:%s", diff)
	}
}

func TestNewCodec_WithSwiftCfg(t *testing.T) {
	swiftCfg := &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: []config.SwiftDependency{
				{Name: "gax", Path: "packages/gax"},
				{Name: "google-cloud-location", Path: "generated/google-cloud-location", ApiPackage: "google.cloud.location"},
			},
		},
	}
	cfg := &parser.ModelConfig{}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	got, err := newCodec(model, cfg, swiftCfg, ".")
	if err != nil {
		t.Fatal(err)
	}

	wantDeps := []*Dependency{
		{SwiftDependency: swiftCfg.Dependencies[0]},
		{SwiftDependency: swiftCfg.Dependencies[1]},
	}
	if diff := cmp.Diff(wantDeps, got.Dependencies); diff != "" {
		t.Errorf("mismatch in Dependencies (-want +got):\n%s", diff)
	}

	wantApiPackages := map[string]*Dependency{
		"google.cloud.location": {SwiftDependency: swiftCfg.Dependencies[1]},
	}
	if diff := cmp.Diff(wantApiPackages, got.ApiPackages); diff != "" {
		t.Errorf("mismatch in ApiPackages (-want +got):\n%s", diff)
	}
}

// newTestCodec creates a simple codec for the tests.
func newTestCodec(t *testing.T, model *api.API, options map[string]string) *codec {
	t.Helper()
	cfg := &parser.ModelConfig{
		Codec: options,
	}
	// Configure the package for well-known types by default.
	swiftCfg := &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: []config.SwiftDependency{
				{Name: wellKnownSwiftPackage, ApiPackage: wellKnownPackage},
			},
		},
	}
	codec, err := newCodec(model, cfg, swiftCfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	return codec
}

func (c *codec) withExtraDependencies(t *testing.T, deps []config.SwiftDependency) {
	t.Helper()
	for _, d := range deps {
		dep := &Dependency{SwiftDependency: d}
		if d.ApiPackage != "" {
			if _, ok := c.ApiPackages[d.ApiPackage]; ok {
				t.Fatalf("conflicting definition for %s", d.ApiPackage)
			}
			c.ApiPackages[d.ApiPackage] = dep
		}
		c.Dependencies = append(c.Dependencies, dep)
	}
}
