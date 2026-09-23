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
)

func TestParseOptions(t *testing.T) {
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{}).
		WithPackageName("test")
	for _, test := range []struct {
		name    string
		library *config.Library
		module  *config.SwiftModule
		want    *codec
	}{
		{
			name: "baseline",
			library: &config.Library{
				CopyrightYear: "2038",
			},
			want: &codec{
				GenerationYear:     "2038",
				LibraryName:        "Test",
				TargetLibraryName:  "Test",
				PackageName:        "test",
				PackageRepoName:    "swift-test",
				PackageVersion:     "0.0.0",
				MonorepoRoot:       ".",
				Model:              model,
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				ResponseEncoding:   defaultResponseEncoding,
			},
		},
		{
			name: "package name override",
			library: &config.Library{
				CopyrightYear: "2038",
				Swift: &config.SwiftPackage{
					PackageNameOverride: "google-cloud-bigtable",
				},
			},
			want: &codec{
				GenerationYear:     "2038",
				LibraryName:        "Test",
				TargetLibraryName:  "Test",
				PackageName:        "google-cloud-bigtable",
				PackageRepoName:    "swift-google-cloud-bigtable",
				PackageVersion:     "0.0.0",
				MonorepoRoot:       ".",
				Model:              model,
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				ResponseEncoding:   defaultResponseEncoding,
			},
		},
		{
			name: "module",
			library: &config.Library{
				Name:          "google-cloud-wkt",
				CopyrightYear: "2038",
				Swift: &config.SwiftPackage{
					LibraryNameOverride: "GoogleWKT",
				},
			},
			module: &config.SwiftModule{
				ModulePath: "GoogleTestProtos",
			},
			want: &codec{
				Module:             true,
				GenerationYear:     "2038",
				TargetLibraryName:  "GoogleWKT",
				PackageName:        "test",
				PackageRepoName:    "swift-google-cloud-wkt",
				PackageVersion:     "0.0.0",
				MonorepoRoot:       ".",
				Model:              model,
				ModulePath:         "GoogleTestProtos",
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				ResponseEncoding:   defaultResponseEncoding,
			},
		},
		{
			name: "module with library name override",
			library: &config.Library{
				Name:          "google-cloud-bigquery",
				CopyrightYear: "2038",
				Swift: &config.SwiftPackage{
					PackageNameOverride: "GoogleCloudBigQuery",
					LibraryNameOverride: "GoogleCloudBigQuery",
				},
			},
			module: &config.SwiftModule{
				ModulePath: "GoogleTestProtos",
			},
			want: &codec{
				Module:             true,
				GenerationYear:     "2038",
				TargetLibraryName:  "GoogleCloudBigQuery",
				PackageName:        "GoogleCloudBigQuery",
				PackageRepoName:    "swift-google-cloud-bigquery",
				PackageVersion:     "0.0.0",
				MonorepoRoot:       ".",
				Model:              model,
				ModulePath:         "GoogleTestProtos",
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				ResponseEncoding:   defaultResponseEncoding,
			},
		},
		{
			name: "module with grpc transport",
			library: &config.Library{
				Name:          "google-cloud-storage",
				CopyrightYear: "2038",
				Swift: &config.SwiftPackage{
					PackageNameOverride: "GoogleCloudStorage",
					LibraryNameOverride: "GoogleCloudStorage",
				},
			},
			module: &config.SwiftModule{
				ModuleType: "grpc-client",
				ModulePath: "StorageControlProtos",
			},
			want: &codec{
				Module:             true,
				ModuleType:         "grpc-client",
				GenerationYear:     "2038",
				TargetLibraryName:  "GoogleCloudStorage",
				PackageName:        "GoogleCloudStorage",
				PackageRepoName:    "swift-google-cloud-storage",
				PackageVersion:     "0.0.0",
				MonorepoRoot:       ".",
				Model:              model,
				ModulePath:         "StorageControlProtos",
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				ResponseEncoding:   defaultResponseEncoding,
			},
		},
		{
			name: "discovery",
			library: &config.Library{
				CopyrightYear:       "2038",
				SpecificationFormat: config.SpecDiscovery,
			},
			want: &codec{
				GenerationYear:     "2038",
				LibraryName:        "Test",
				TargetLibraryName:  "Test",
				PackageName:        "test",
				PackageRepoName:    "swift-test",
				PackageVersion:     "0.0.0",
				MonorepoRoot:       ".",
				Model:              model,
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				UrlSafeForBytes:    true,
				ResponseEncoding:   discoveryResponseEncoding,
				UseStringEnums:     true,
			},
		},
		{
			name: "name_overrides",
			library: &config.Library{
				CopyrightYear: "2038",
				Swift: &config.SwiftPackage{
					NameOverrides: map[string]string{
						".google.storage.v2.Storage":                "StorageAdmin",
						".google.storage.control.v2.StorageControl": "StorageAdmin",
					},
				},
			},
			want: &codec{
				GenerationYear:     "2038",
				LibraryName:        "Test",
				TargetLibraryName:  "Test",
				PackageName:        "test",
				PackageRepoName:    "swift-test",
				PackageVersion:     "0.0.0",
				MonorepoRoot:       ".",
				Model:              model,
				ApiPackages:        map[string]*Dependency{},
				DependenciesByName: map[string]*Dependency{},
				ResponseEncoding:   defaultResponseEncoding,
				NameOverrides: map[string]string{
					".google.storage.v2.Storage":                "StorageAdmin",
					".google.storage.control.v2.StorageControl": "StorageAdmin",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := newCodec(model, test.library, test.module, ".")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreUnexported(api.API{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			wantGrpc := test.module != nil && test.module.ModuleType == "grpc-client"
			if got.isGrpc() != wantGrpc {
				t.Errorf("isGrpc() = %v, want %v", got.isGrpc(), wantGrpc)
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
	library := &config.Library{
		Swift: swiftCfg,
	}
	model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{}).WithPackageName("test")
	got, err := newCodec(model, library, nil, ".")
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
func newTestCodec(t *testing.T, model *api.API, library *config.Library) *codec {
	t.Helper()
	if library == nil {
		library = &config.Library{}
	}
	if library.Swift == nil {
		// Configure the package for well-known types by default.
		library.Swift = &config.SwiftPackage{
			SwiftDefault: config.SwiftDefault{
				Dependencies: []config.SwiftDependency{
					{Name: wellKnownSwiftPackage, ApiPackage: wellKnownProtobufPackage},
					{Name: paginationSwiftPackage, RequiredByServices: true},
				},
			},
		}
	}
	codec, err := newCodec(model, library, nil, ".")
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
		return api.NewTestEnum(name).
			WithPackage("google.cloud.test.v1").
			WithValues(api.NewTestEnumValue("UNSPECIFIED", 0))
	}
	sharedEnum := makeEnum("SharedEnum")
	s1Enum := makeEnum("Service1Enum")
	s2Enum := makeEnum("Service2Enum")
	unusedEnum := makeEnum("UnusedEnum")

	makeMsg := func(name string, e *api.Enum) *api.Message {
		return api.NewTestMessage(name).
			WithPackage("google.cloud.test.v1").
			WithFields(api.NewTestField("e").WithType(api.TypezEnum).WithTypezID(e.ID))
	}
	sharedMessage := makeMsg("SharedMessage", sharedEnum)
	s1Message := makeMsg("Service1Message", s1Enum)
	s2Message := makeMsg("Service2Message", s2Enum)
	unusedMessage := makeMsg("UnusedMessage", unusedEnum)

	s1 := api.NewTestService("Service1").
		WithPackage("google.cloud.test.v1").
		WithMethods(
			api.NewTestMethod("M1").
				WithInput(sharedMessage).
				WithOutput(s1Message).
				WithPathTemplate(&api.PathTemplate{}),
		)
	s2 := api.NewTestService("Service2").
		WithPackage("google.cloud.test.v1").
		WithMethods(
			api.NewTestMethod("M2").
				WithInput(sharedMessage).
				WithOutput(s2Message).
				WithPathTemplate(&api.PathTemplate{}),
		)

	model := api.NewTestAPI(
		[]*api.Message{sharedMessage, s1Message, s2Message, unusedMessage},
		[]*api.Enum{sharedEnum, s1Enum, s2Enum, unusedEnum},
		[]*api.Service{s1, s2},
	).WithPackageName("google.cloud.test.v1")
	api.CrossReference(model)
	return model
}

func makeRequiredServicesTestModel() *api.API {
	externalRequest := api.NewTestMessage("Request").WithPackage("external")
	externalResponse := api.NewTestMessage("Response").WithPackage("external")
	externalMethod := api.NewTestMethod("GetThing").
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("things")).
		WithInput(externalRequest).
		WithOutput(externalResponse)
	externalService := api.NewTestService("ExternalService").
		WithPackage("external").
		WithMethods(externalMethod)

	placeholder := api.NewTestMessage("zoneOperations")
	placeholder.ServicePlaceholder = true

	inputType := api.NewTestMessage("GetOperationRequest").
		WithID(".test.zoneOperations.GetOperationRequest")

	outputType := api.NewTestMessage("Operation")

	sourceMethod := api.NewTestMethod("GetOperation").
		WithInput(inputType).
		WithOutput(outputType).
		WithVerb("GET").
		WithPathTemplate(&api.PathTemplate{})
	sourceService := api.NewTestService("zoneOperations").
		WithPackage("test").
		WithMethods(sourceMethod)
	sourceMethod.Service = sourceService
	internalMixin := api.NewTestMethod(sourceMethod.Name).
		WithSourceMethod(sourceMethod)
	internalMixin.IsLroPoller = true
	externalMixin := api.NewTestMethod(externalMethod.Name).
		WithSourceMethod(externalMethod)
	externalMixin.IsLroPoller = true
	targetService := api.NewTestService("TestService").
		WithPackage("test").
		WithMethods(internalMixin, externalMixin)

	model := api.NewTestAPI([]*api.Message{placeholder, inputType, outputType}, nil, []*api.Service{sourceService, targetService})
	model.AddMessage(externalRequest)
	model.AddMessage(externalResponse)
	model.AddService(externalService)
	api.CrossReference(model)
	return model
}

func TestSkipDependency(t *testing.T) {
	for _, tc := range []struct {
		name              string
		targetLibraryName string
		libraryName       string
		packageName       string
		module            bool
		depName           string
		wantSkip          bool
	}{
		{
			name:              "multi-module self import skipped (e.g. wkt messages importing GoogleWKT)",
			targetLibraryName: "GoogleWKT",
			packageName:       "google-protobuf",
			module:            true,
			depName:           "GoogleWKT",
			wantSkip:          true,
		},
		{
			name:              "multi-module non-self import preserved (e.g. storage convert importing GoogleType)",
			targetLibraryName: "GoogleCloudStorage",
			packageName:       "GoogleType",
			module:            true,
			depName:           "GoogleType",
			wantSkip:          false,
		},
		{
			name:              "single-module self import skipped",
			targetLibraryName: "GoogleCloudSecretManagerV1",
			libraryName:       "GoogleCloudSecretManagerV1",
			packageName:       "google-cloud-secretmanager-v1",
			module:            false,
			depName:           "GoogleCloudSecretManagerV1",
			wantSkip:          true,
		},
		{
			name:              "single-module non-self import preserved",
			targetLibraryName: "GoogleCloudSecretManagerV1",
			libraryName:       "GoogleCloudSecretManagerV1",
			packageName:       "google-cloud-secretmanager-v1",
			module:            false,
			depName:           "GoogleGax",
			wantSkip:          false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &codec{
				TargetLibraryName: tc.targetLibraryName,
				LibraryName:       tc.libraryName,
				PackageName:       tc.packageName,
				Module:            tc.module,
			}
			dep := &Dependency{SwiftDependency: config.SwiftDependency{Name: tc.depName}}
			got := c.skipDependency(dep)
			if got != tc.wantSkip {
				t.Errorf("skipDependency(%q) = %v, want %v", tc.depName, got, tc.wantSkip)
			}
		})
	}
}
