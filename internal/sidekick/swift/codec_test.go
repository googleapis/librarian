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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestParseOptions(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
	for _, test := range []struct {
		name string
		cfg  *parser.ModelConfig
		want *codec
	}{
		{
			name: "baseline",
			cfg: &parser.ModelConfig{
				Codec: map[string]string{
					"copyright-year":        "2038",
					"package-name-override": "GoogleCloudBigtable",
					"root-name":             "test-root",
				},
			},
			want: &codec{
				GenerationYear:     "2038",
				PackageName:        "GoogleCloudBigtable",
				MonorepoRoot:       ".",
				RootName:           "test-root",
				Model:              model,
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
			},
		},
		{
			name: "discovery",
			cfg: &parser.ModelConfig{
				Codec: map[string]string{
					"copyright-year":        "2038",
					"package-name-override": "GoogleCloudComputeV1",
					"root-name":             "test-root",
				},
				SpecificationFormat: config.SpecDiscovery,
			},
			want: &codec{
				GenerationYear:     "2038",
				PackageName:        "GoogleCloudComputeV1",
				MonorepoRoot:       ".",
				RootName:           "test-root",
				Model:              model,
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				UrlSafeForBytes:    true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := newCodec(model, test.cfg, nil, ".")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreUnexported(api.API{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	wantApiPackages := map[string]*Dependency{
		"google.cloud.location": {SwiftDependency: swiftCfg.Dependencies[1]},
	}
	if diff := cmp.Diff(wantApiPackages, got.ApiPackages); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
				{Name: wellKnownSwiftPackage, ApiPackage: wellKnownProtobufPackage},
				{Name: paginationSwiftPackage, RequiredByServices: true},
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
		c.DependenciesByName[d.Name] = dep
		c.Dependencies = append(c.Dependencies, dep)
	}
}

func makeGatedTestModel() *api.API {
	makeEnum := func(name string) *api.Enum {
		e := &api.Enum{
			Name: name, ID: ".google.cloud.test.v1." + name, Package: "google.cloud.test.v1",
			Values: []*api.EnumValue{{Name: "UNSPECIFIED", Number: 0}},
		}
		e.UniqueNumberValues = e.Values
		return e
	}
	sharedEnum := makeEnum("SharedEnum")
	s1Enum := makeEnum("Service1Enum")
	s2Enum := makeEnum("Service2Enum")
	unusedEnum := makeEnum("UnusedEnum")

	sharedMessage := &api.Message{
		Name: "SharedMessage", ID: ".google.cloud.test.v1.SharedMessage", Package: "google.cloud.test.v1",
		Fields: []*api.Field{{Name: "e", Typez: api.TypezEnum, TypezID: sharedEnum.ID}},
	}
	s1Message := &api.Message{
		Name: "Service1Message", ID: ".google.cloud.test.v1.Service1Message", Package: "google.cloud.test.v1",
		Fields: []*api.Field{{Name: "e", Typez: api.TypezEnum, TypezID: s1Enum.ID}},
	}
	s2Message := &api.Message{
		Name: "Service2Message", ID: ".google.cloud.test.v1.Service2Message", Package: "google.cloud.test.v1",
		Fields: []*api.Field{{Name: "e", Typez: api.TypezEnum, TypezID: s2Enum.ID}},
	}
	unusedMessage := &api.Message{
		Name: "UnusedMessage", ID: ".google.cloud.test.v1.UnusedMessage", Package: "google.cloud.test.v1",
		Fields: []*api.Field{{Name: "e", Typez: api.TypezEnum, TypezID: unusedEnum.ID}},
	}

	s1 := &api.Service{
		Name: "Service1", ID: ".google.cloud.test.v1.Service1", Package: "google.cloud.test.v1",
		Methods: []*api.Method{
			{Name: "M1", ID: ".google.cloud.test.v1.Service1.M1", InputTypeID: sharedMessage.ID, OutputTypeID: s1Message.ID},
		},
	}
	s2 := &api.Service{
		Name: "Service2", ID: ".google.cloud.test.v1.Service2", Package: "google.cloud.test.v1",
		Methods: []*api.Method{
			{Name: "M2", ID: ".google.cloud.test.v1.Service2.M2", InputTypeID: sharedMessage.ID, OutputTypeID: s2Message.ID},
		},
	}

	model := api.NewTestAPI(
		[]*api.Message{sharedMessage, s1Message, s2Message, unusedMessage},
		[]*api.Enum{sharedEnum, s1Enum, s2Enum, unusedEnum},
		[]*api.Service{s1, s2},
	)
	model.PackageName = "google.cloud.test.v1"
	api.CrossReference(model)
	return model
}
