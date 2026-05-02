// Copyright 2025 Google LLC
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

package rust

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestDefaultFeatures(t *testing.T) {
	for _, test := range []struct {
		Options map[string]string
		Want    []string
	}{
		{
			Options: map[string]string{
				"per-service-features": "true",
			},
			Want: []string{"service-0", "service-1"},
		},
		{
			Options: map[string]string{
				"per-service-features": "false",
			},
			Want: nil,
		},
		{
			Options: map[string]string{
				"per-service-features": "true",
				"default-features":     "service-1",
			},
			Want: []string{"service-1"},
		},
		{
			Options: map[string]string{
				"per-service-features": "true",
				"default-features":     "",
			},
			Want: []string{},
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Options=%v", test.Options)
		if diff := cmp.Diff(test.Want, got.DefaultFeatures); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestRustdocWarnings(t *testing.T) {
	for _, test := range []struct {
		Options map[string]string
		Want    []string
	}{
		{
			Options: map[string]string{},
			Want:    nil,
		},
		{
			Options: map[string]string{
				"disabled-rustdoc-warnings": "",
			},
			Want: []string{},
		},
		{
			Options: map[string]string{
				"disabled-rustdoc-warnings": "a,b,c",
			},
			Want: []string{"a", "b", "c"},
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Options=%v", test.Options)
		if diff := cmp.Diff(test.Want, got.DisabledRustdocWarnings); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestClippyWarnings(t *testing.T) {
	for _, test := range []struct {
		Options map[string]string
		Want    []string
	}{
		{
			Options: map[string]string{},
			Want:    nil,
		},
		{
			Options: map[string]string{
				"disabled-clippy-warnings": "",
			},
			Want: []string{},
		},
		{
			Options: map[string]string{
				"disabled-clippy-warnings": "a,b,c",
			},
			Want: []string{"a", "b", "c"},
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Options=%v", test.Options)
		if diff := cmp.Diff(test.Want, got.DisabledClippyWarnings); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestInternalBuildersAnnotation(t *testing.T) {
	for _, test := range []struct {
		Options        map[string]string
		Want           bool
		WantVisibility string
	}{
		{
			Options:        map[string]string{},
			Want:           false,
			WantVisibility: "pub",
		},
		{
			Options: map[string]string{
				"internal-builders": "true",
			},
			Want:           true,
			WantVisibility: "pub(crate)",
		},
		{
			Options: map[string]string{
				"internal-builders": "false",
			},
			Want:           false,
			WantVisibility: "pub",
		},
	} {
		model := newTestAnnotateModelAPI()
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		if got.InternalBuilders != test.Want {
			t.Errorf("mismatch in InternalBuilders, want=%v, got=%v", test.Want, got.InternalBuilders)
		}
		svcAnn := model.Services[0].Codec.(*serviceAnnotations)
		if svcAnn.InternalBuilders != test.Want {
			t.Errorf("mismatch in service InternalBuilders, want=%v, got=%v", test.Want, svcAnn.InternalBuilders)
		}
		if got.BuilderVisibility() != test.WantVisibility {
			t.Errorf("mismatch in BuilderVisibility, want=%s, got=%s", test.WantVisibility, got.BuilderVisibility())
		}
		if svcAnn.BuilderVisibility() != test.WantVisibility {
			t.Errorf("mismatch in service BuilderVisibility, want=%s, got=%s", test.WantVisibility, svcAnn.BuilderVisibility())
		}
	}
}

func TestQuickstartServiceAnnotation(t *testing.T) {
	t.Run("survives filtering", func(t *testing.T) {
		model := newTestAnnotateModelAPI()
		// model.Services[0] is Service0, model.Services[1] is Service1
		model.QuickstartService = model.Services[1]

		codec := newTestCodec(t, libconfig.SpecProtobuf, "", nil)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}

		if got.QuickstartService == nil {
			t.Fatal("QuickstartService should not be nil")
		}
		if got.QuickstartService != model.Services[1] {
			t.Errorf("expected QuickstartService to be Service1, got %v", got.QuickstartService.Name)
		}
	})

	t.Run("filtered out fallback", func(t *testing.T) {
		model := newTestAnnotateModelAPI()

		// Create a service that has no methods with bindings, so it will be filtered out.
		filteredService := &api.Service{
			Name:    "FilteredService",
			ID:      "..FilteredService",
			Package: "test.v1",
			Methods: []*api.Method{
				{
					Name: "noBindings",
					ID:   "..FilteredService.noBindings",
				},
			},
		}
		model.Services = append(model.Services, filteredService)
		for _, s := range model.Services {
			s.Model = model
		}
		api.CrossReference(model)

		// Set the filtered service as the global quickstart.
		model.QuickstartService = filteredService

		codec := newTestCodec(t, libconfig.SpecProtobuf, "", nil)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}

		if got.QuickstartService != nil {
			t.Errorf("expected QuickstartService to be nil because it was filtered out and there is no override, got %v", got.QuickstartService.Name)
		}
	})

	t.Run("with override", func(t *testing.T) {
		model := newTestAnnotateModelAPI()
		model.QuickstartService = model.Services[0] // Set default to 0

		codec := newTestCodec(t, libconfig.SpecProtobuf, "", nil)
		// Set override to Service1
		codec.quickstartServiceOverride = "Service1"

		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}

		if got.QuickstartService == nil {
			t.Fatal("QuickstartService should not be nil")
		}
		if got.QuickstartService != model.Services[1] {
			t.Errorf("expected QuickstartService to be overridden to Service1, got %v", got.QuickstartService.Name)
		}
	})

	t.Run("with missing override", func(t *testing.T) {
		model := newTestAnnotateModelAPI()

		codec := newTestCodec(t, libconfig.SpecProtobuf, "", nil)
		codec.quickstartServiceOverride = "NonExistentService"

		_, err := annotateModel(model, codec)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errQuickstartServiceNotFound) {
			t.Errorf("expected error to be errQuickstartServiceNotFound, got %v", err)
		}
	})
}

func newTestAnnotateModelAPI() *api.API {
	service0 := &api.Service{
		Name: "Service0",
		ID:   "..Service0",
		Methods: []*api.Method{
			{
				Name:         "get",
				ID:           "..Service0.get",
				InputTypeID:  ".google.protobuf.Empty",
				OutputTypeID: ".google.protobuf.Empty",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "GET",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("resource"),
						},
					},
				},
			},
		},
	}
	service1 := &api.Service{
		Name: "Service1",
		ID:   "..Service1",
		Methods: []*api.Method{
			{
				Name:         "get",
				ID:           "..Service1.get",
				InputTypeID:  ".google.protobuf.Empty",
				OutputTypeID: ".google.protobuf.Empty",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "GET",
							PathTemplate: (&api.PathTemplate{}).WithLiteral("resource"),
						},
					},
				},
			},
		},
	}
	model := api.NewTestAPI(
		[]*api.Message{},
		[]*api.Enum{},
		[]*api.Service{service0, service1})
	api.CrossReference(model)
	return model
}

func TestPackageNames(t *testing.T) {
	model := api.NewTestAPI(
		[]*api.Message{}, []*api.Enum{},
		[]*api.Service{{Name: "Workflows", Package: "google.cloud.workflows.v1"}})
	err := api.CrossReference(model)
	if err != nil {
		t.Fatal(err)
	}
	// Override the default name for test APIs ("Test").
	model.Name = "workflows-v1"
	codec, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"version":                     "1.2.3",
		"release-level":               "stable",
		"copyright-year":              "2035",
		"per-service-features":        "true",
		"extra-modules":               "operation",
		"generate-setter-samples":     "true",
		"generate-rpc-samples":        "true",
		"detailed-tracing-attributes": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	codec.packageMapping = map[string]*packagez{
		"google.protobuf": {name: "wkt"},
	}
	got, err := annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}
	want := &modelAnnotations{
		PackageName:               "google-cloud-workflows-v1",
		PackageVersion:            "1.2.3",
		ReleaseLevel:              "stable",
		PackageNamespace:          "google_cloud_workflows_v1",
		RequiredPackages:          []string{},
		ExternPackages:            []string{},
		HasLROs:                   false,
		CopyrightYear:             "2035",
		Services:                  []*api.Service{},
		NameToLower:               "workflows-v1",
		PerServiceFeatures:        false, // no services
		ExtraModules:              []string{"operation"},
		GenerateSetterSamples:     true,
		GenerateRpcSamples:        true,
		DetailedTracingAttributes: true,
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate")); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateModelWithDetailedTracing(t *testing.T) {
	for _, test := range []struct {
		name    string
		options map[string]string
		want    bool
	}{
		{
			name:    "DetailedTracingTrue",
			options: map[string]string{"detailed-tracing-attributes": "true"},
			want:    true,
		},
		{
			name:    "DetailedTracingFalse",
			options: map[string]string{"detailed-tracing-attributes": "false"},
			want:    false,
		},
		{
			name:    "DetailedTracingMissing",
			options: map[string]string{},
			want:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			got, err := annotateModel(model, codec)
			if err != nil {
				t.Fatal(err)
			}
			if got.DetailedTracingAttributes != test.want {
				t.Errorf("annotateModel() DetailedTracingAttributes = %v, want %v", got.DetailedTracingAttributes, test.want)
			}
		})
	}
}

func TestRoutingRequired(t *testing.T) {
	message := &api.Message{
		Name:    "Message",
		ID:      ".test.Message",
		Package: "test",
	}
	method := &api.Method{
		Name:         "DoFoo",
		ID:           ".test.Service.DoFoo",
		InputTypeID:  ".test.Message",
		OutputTypeID: ".test.Message",
		PathInfo:     &api.PathInfo{},
	}
	service := &api.Service{
		Name:    "FooService",
		ID:      ".test.FooService",
		Package: "test",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI([]*api.Message{message},
		[]*api.Enum{},
		[]*api.Service{service})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
		"routing-required":          "true",
	})
	annotateModel(model, codec)

	if !method.Codec.(*methodAnnotation).RoutingRequired {
		t.Errorf("codec setting `routing-required` not respected")
	}
}

func TestGenerateRpcSamples(t *testing.T) {
	model := serviceAnnotationsModel()
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"generate-rpc-samples": "true",
	})
	annotateModel(model, codec)
	if !model.Codec.(*modelAnnotations).GenerateRpcSamples {
		t.Errorf("GenerateRpcSamples should be true")
	}
}

func TestGenerateSetterSamples(t *testing.T) {
	model := serviceAnnotationsModel()
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"generate-setter-samples": "true",
	})
	annotateModel(model, codec)
	if !model.Codec.(*modelAnnotations).GenerateSetterSamples {
		t.Errorf("GenerateSetterSamples should be true")
	}
}
