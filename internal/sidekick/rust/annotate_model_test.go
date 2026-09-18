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
	"slices"
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
		model := newTestAnnotateModelAPI(t)
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
		model := newTestAnnotateModelAPI(t)
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
		model := newTestAnnotateModelAPI(t)
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
		model := newTestAnnotateModelAPI(t)
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

func TestHandwrittenSurfaceAnnotation(t *testing.T) {
	for _, test := range []struct {
		name                 string
		options              map[string]string
		wantService0Veneer   bool
		wantService0Internal bool
		wantService0Samples  bool
		wantService1Veneer   bool
		wantService1Internal bool
		wantService1Samples  bool
	}{
		{
			name: "handwritten surface for service0 only",
			options: map[string]string{
				"handwritten-surface":  "..Service0",
				"generate-rpc-samples": "true",
			},
			wantService0Veneer:   true,
			wantService0Internal: true,
			wantService0Samples:  false,
			wantService1Veneer:   false,
			wantService1Internal: false,
			wantService1Samples:  true,
		},
		{
			name: "handwritten surface true for all",
			options: map[string]string{
				"handwritten-surface":  "true",
				"generate-rpc-samples": "true",
			},
			wantService0Veneer:   true,
			wantService0Internal: true,
			wantService0Samples:  false,
			wantService1Veneer:   true,
			wantService1Internal: true,
			wantService1Samples:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := newTestAnnotateModelAPI(t)
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			_, err := annotateModel(model, codec)
			if err != nil {
				t.Fatal(err)
			}
			svc0Ann := model.Services[0].Codec.(*serviceAnnotations)
			if svc0Ann.HasVeneer != test.wantService0Veneer {
				t.Errorf("mismatch in Service0 HasVeneer, want=%v, got=%v", test.wantService0Veneer, svc0Ann.HasVeneer)
			}
			if svc0Ann.InternalBuilders != test.wantService0Internal {
				t.Errorf("mismatch in Service0 InternalBuilders, want=%v, got=%v", test.wantService0Internal, svc0Ann.InternalBuilders)
			}
			if svc0Ann.GenerateRpcSamples != test.wantService0Samples {
				t.Errorf("mismatch in Service0 GenerateRpcSamples, want=%v, got=%v", test.wantService0Samples, svc0Ann.GenerateRpcSamples)
			}

			svc1Ann := model.Services[1].Codec.(*serviceAnnotations)
			if svc1Ann.HasVeneer != test.wantService1Veneer {
				t.Errorf("mismatch in Service1 HasVeneer, want=%v, got=%v", test.wantService1Veneer, svc1Ann.HasVeneer)
			}
			if svc1Ann.InternalBuilders != test.wantService1Internal {
				t.Errorf("mismatch in Service1 InternalBuilders, want=%v, got=%v", test.wantService1Internal, svc1Ann.InternalBuilders)
			}
			if svc1Ann.GenerateRpcSamples != test.wantService1Samples {
				t.Errorf("mismatch in Service1 GenerateRpcSamples, want=%v, got=%v", test.wantService1Samples, svc1Ann.GenerateRpcSamples)
			}
		})
	}
}

func TestGrpcClientAnnotation(t *testing.T) {
	for _, test := range []struct {
		Options map[string]string
		Want    string
	}{
		{
			Options: map[string]string{},
			Want:    "gaxi::grpc::Client",
		},
		{
			Options: map[string]string{
				"grpc-client": "crate::storage::bidi::GrpcClient",
			},
			Want: "crate::storage::bidi::GrpcClient",
		},
	} {
		model := newTestAnnotateModelAPI(t)
		codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.Options)
		got, err := annotateModel(model, codec)
		if err != nil {
			t.Fatal(err)
		}
		if got.GrpcClient != test.Want {
			t.Errorf("mismatch in GrpcClient, want=%v, got=%v", test.Want, got.GrpcClient)
		}
		svcAnn := model.Services[0].Codec.(*serviceAnnotations)
		if svcAnn.GrpcClient != test.Want {
			t.Errorf("mismatch in service GrpcClient, want=%v, got=%v", test.Want, svcAnn.GrpcClient)
		}
	}
}

func TestQuickstartServiceAnnotation(t *testing.T) {
	t.Run("survives filtering", func(t *testing.T) {
		model := newTestAnnotateModelAPI(t)
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
		model := newTestAnnotateModelAPI(t)

		// Create a service that has no methods with bindings, so it will be filtered out.
		empty := model.Message(api.WktEmptyID)
		noBindingsMethod := api.NewTestMethod("noBindings").
			WithInput(empty).
			WithOutput(empty).
			WithBindings()
		filteredService := api.NewTestService("FilteredService").
			WithPackage("").
			WithMethods(noBindingsMethod)
		model.AddService(filteredService)
		model.Services = append(model.Services, filteredService)
		if err := api.CrossReference(model); err != nil {
			t.Fatal(err)
		}

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
		model := newTestAnnotateModelAPI(t)
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
		model := newTestAnnotateModelAPI(t)

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

func newTestAnnotateModelAPI(t *testing.T) *api.API {
	t.Helper()
	empty := api.NewTestMessage("Empty").WithPackage("google.protobuf")
	method0 := api.NewTestMethod("get").
		WithInput(empty).
		WithOutput(empty).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("resource"))

	service0 := api.NewTestService("Service0").
		WithPackage("").
		WithMethods(method0)

	method1 := api.NewTestMethod("get").
		WithInput(empty).
		WithOutput(empty).
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("resource"))

	service1 := api.NewTestService("Service1").
		WithPackage("").
		WithMethods(method1)

	model := api.NewTestAPI(
		nil,
		nil,
		[]*api.Service{service0, service1})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	return model
}

func TestPackageNames(t *testing.T) {
	model := api.NewTestAPI(
		nil, nil,
		[]*api.Service{api.NewTestService("Workflows").WithPackage("google.cloud.workflows.v1")})
	if err := api.CrossReference(model); err != nil {
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
		PackageModuleName:         "google::cloud::workflows::v1",
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
		GrpcClient:                "gaxi::grpc::Client",
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

func TestAnnotateModelWithLroStubOptions(t *testing.T) {
	for _, test := range []struct {
		name    string
		options map[string]string
		want    bool
	}{
		{
			name:    "LroStubOptionsTrue",
			options: map[string]string{"lro-stub-options": "true"},
			want:    true,
		},
		{
			name:    "LroStubOptionsFalse",
			options: map[string]string{"lro-stub-options": "false"},
			want:    false,
		},
		{
			name:    "LroStubOptionsMissing",
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
			if got.LroStubOptions != test.want {
				t.Errorf("annotateModel() LroStubOptions = %v, want %v", got.LroStubOptions, test.want)
			}
		})
	}
}

func TestRoutingRequired(t *testing.T) {
	message := api.NewTestMessage("Message")
	method := api.NewTestMethod("DoFoo").
		WithInput(message).
		WithOutput(message).
		WithBindings()
	service := api.NewTestService("FooService").
		WithMethods(method)
	model := api.NewTestAPI([]*api.Message{message},
		nil,
		[]*api.Service{service})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"include-grpc-only-methods": "true",
		"routing-required":          "true",
	})
	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}

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

func TestModelAnnotationsHasStreaming(t *testing.T) {
	msg := api.NewTestMessage("Request").WithPackage("test.v1")

	bidiChat := api.NewTestMethod("Chat").
		WithInput(msg).
		WithOutput(msg).
		WithBidiStreaming().
		WithBindings()
	bidiService := api.NewTestService("BidiService").
		WithPackage("test.v1").
		WithMethods(bidiChat)

	unaryGet := api.NewTestMethod("Get").
		WithInput(msg).
		WithOutput(msg).
		WithBindings()
	unaryService := api.NewTestService("UnaryService").
		WithPackage("test.v1").
		WithMethods(unaryGet)

	serverExpand := api.NewTestMethod("Expand").
		WithInput(msg).
		WithOutput(msg).
		WithServerSideStreaming().
		WithBindings()
	serverStreamingService := api.NewTestService("ServerStreamingService").
		WithPackage("test.v1").
		WithMethods(serverExpand)

	mixedChat := api.NewTestMethod("Chat").
		WithInput(msg).
		WithOutput(msg).
		WithBidiStreaming().
		WithBindings()
	mixedExpand := api.NewTestMethod("Expand").
		WithInput(msg).
		WithOutput(msg).
		WithServerSideStreaming().
		WithBindings()
	mixedService := api.NewTestService("MixedService").
		WithPackage("test.v1").
		WithMethods(mixedChat, mixedExpand)

	for _, test := range []struct {
		name                 string
		service              *api.Service
		options              map[string]string
		wantBidi             bool
		wantServer           bool
		wantStreaming        bool
		wantGaxiFeatures     []string
		wantRequiredPackages []string
	}{
		{
			name:    "bidi streaming enabled with bidi method",
			service: bidiService,
			options: map[string]string{
				"package:gaxi":  "package=google-cloud-gax,used-if=services",
				"package:prost": "package=prost,used-if=streaming",
			},
			wantBidi:             true,
			wantServer:           false,
			wantStreaming:        true,
			wantGaxiFeatures:     []string{"_internal-grpc-client"},
			wantRequiredPackages: []string{`gaxi                 = { workspace = true, features = ["_internal-grpc-client"] }`, "prost.workspace      = true"},
		},
		{
			name:    "server streaming enabled with server streaming method",
			service: serverStreamingService,
			options: map[string]string{
				"package:gaxi":  "package=google-cloud-gax,used-if=services",
				"package:prost": "package=prost,used-if=streaming",
			},
			wantBidi:             false,
			wantServer:           true,
			wantStreaming:        true,
			wantGaxiFeatures:     []string{"_internal-grpc-client"},
			wantRequiredPackages: []string{`gaxi                 = { workspace = true, features = ["_internal-grpc-client"] }`, "prost.workspace      = true"},
		},
		{
			name:    "both streaming enabled with mixed service",
			service: mixedService,
			options: map[string]string{
				"package:gaxi":  "package=google-cloud-gax,used-if=services",
				"package:prost": "package=prost,used-if=streaming",
			},
			wantBidi:             true,
			wantServer:           true,
			wantStreaming:        true,
			wantGaxiFeatures:     []string{"_internal-grpc-client"},
			wantRequiredPackages: []string{`gaxi                 = { workspace = true, features = ["_internal-grpc-client"] }`, "prost.workspace      = true"},
		},
		{
			name:    "unary method",
			service: unaryService,
			options: map[string]string{
				"package:gaxi":  "package=google-cloud-gax,used-if=services",
				"package:prost": "package=prost,used-if=streaming",
			},
			wantBidi:             false,
			wantServer:           false,
			wantStreaming:        false,
			wantRequiredPackages: []string{"gaxi.workspace       = true"},
		},
		{
			name:    "template override with bidi method",
			service: bidiService,
			options: map[string]string{
				"template-override": "templates/tonic",
				"package:gaxi":      "package=google-cloud-gax,used-if=services",
				"package:prost":     "package=prost,used-if=streaming",
			},
			wantBidi:             false,
			wantServer:           false,
			wantStreaming:        false,
			wantRequiredPackages: []string{"gaxi.workspace       = true"},
		},
		{
			name:    "template override with server streaming method",
			service: serverStreamingService,
			options: map[string]string{
				"template-override": "templates/tonic",
				"package:gaxi":      "package=google-cloud-gax,used-if=services",
				"package:prost":     "package=prost,used-if=streaming",
			},
			wantBidi:             false,
			wantServer:           false,
			wantStreaming:        false,
			wantRequiredPackages: []string{"gaxi.workspace       = true"},
		},
		{
			name:    "grpc-client template override with server streaming method",
			service: serverStreamingService,
			options: map[string]string{
				"template-override": "templates/grpc-client",
				"package:gaxi":      "package=google-cloud-gax,used-if=services",
				"package:prost":     "package=prost,used-if=streaming",
			},
			wantBidi:             false,
			wantServer:           true,
			wantStreaming:        true,
			wantGaxiFeatures:     []string{"_internal-grpc-client"},
			wantRequiredPackages: []string{`gaxi                 = { workspace = true, features = ["_internal-grpc-client"] }`, "prost.workspace      = true"},
		},
		{
			name:    "grpc-client template override with bidi streaming method",
			service: bidiService,
			options: map[string]string{
				"template-override": "templates/grpc-client",
				"package:gaxi":      "package=google-cloud-gax,used-if=services",
				"package:prost":     "package=prost,used-if=streaming",
			},
			wantBidi:             true,
			wantServer:           false,
			wantStreaming:        true,
			wantGaxiFeatures:     []string{"_internal-grpc-client"},
			wantRequiredPackages: []string{`gaxi                 = { workspace = true, features = ["_internal-grpc-client"] }`, "prost.workspace      = true"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{test.service})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			got, err := annotateModel(model, codec)
			if err != nil {
				t.Fatal(err)
			}
			sAnn := test.service.Codec.(*serviceAnnotations)
			if sAnn.HasBidiStreaming() != test.wantBidi {
				t.Errorf("HasBidiStreaming = %v, want %v", sAnn.HasBidiStreaming(), test.wantBidi)
			}
			if sAnn.HasStreaming() != test.wantStreaming {
				t.Errorf("HasStreaming = %v, want %v", sAnn.HasStreaming(), test.wantStreaming)
			}
			if len(test.wantGaxiFeatures) > 0 {
				idx := slices.IndexFunc(codec.extraPackages, func(pkg *packagez) bool {
					return pkg.name == gaxiPackageName
				})
				if idx == -1 {
					t.Fatalf("gaxi package not found in extraPackages")
				}
				if diff := cmp.Diff(test.wantGaxiFeatures, codec.extraPackages[idx].features); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
			less := func(a, b string) bool { return a < b }
			if diff := cmp.Diff(test.wantRequiredPackages, got.RequiredPackages, cmpopts.SortSlices(less), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestModelAnnotationsGrpcServices(t *testing.T) {
	msg := api.NewTestMessage("Request").WithPackage("test.v1")

	bidiMethod := api.NewTestMethod("Chat").WithInput(msg).WithOutput(msg).WithBidiStreaming().WithPathTemplate(&api.PathTemplate{})
	bidiService := api.NewTestService("BidiService").WithPackage("test.v1").WithMethods(bidiMethod)

	serverMethod := api.NewTestMethod("Expand").WithInput(msg).WithOutput(msg).WithServerSideStreaming().WithPathTemplate(&api.PathTemplate{})
	serverService := api.NewTestService("ServerService").WithPackage("test.v1").WithMethods(serverMethod)

	unaryMethod := api.NewTestMethod("Get").WithInput(msg).WithOutput(msg).WithVerb("GET").WithPathTemplate(&api.PathTemplate{})
	unaryService := api.NewTestService("UnaryService").WithPackage("test.v1").WithMethods(unaryMethod)

	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{bidiService, serverService, unaryService})
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"per-service-features": "true",
	})
	got, err := annotateModel(model, codec)
	if err != nil {
		t.Fatal(err)
	}

	grpcServices := got.GrpcServices()
	if len(grpcServices) != 2 {
		t.Fatalf("expected 2 GrpcServices, got %d", len(grpcServices))
	}
	if grpcServices[0].Name != "BidiService" {
		t.Errorf("expected GrpcServices[0].Name == %q, got %q", "BidiService", grpcServices[0].Name)
	}
	if grpcServices[1].Name != "ServerService" {
		t.Errorf("expected GrpcServices[1].Name == %q, got %q", "ServerService", grpcServices[1].Name)
	}
	if !got.HasGrpc() {
		t.Errorf("expected HasGrpc() == true")
	}
	if !got.HasHttp() {
		t.Errorf("expected HasHttp() == true")
	}
}

func TestExternalTypesAnnotations(t *testing.T) {
	extMsg := api.NewTestMessage("LatLng").WithPackage("google.type")
	extEnum := api.NewTestEnum("DayOfWeek").WithPackage("google.type")

	model := api.NewTestAPI(nil, nil, nil)
	model.ExternalMessages = []*api.Message{extMsg}
	model.ExternalEnums = []*api.Enum{extEnum}

	codec := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
		"template-override": "templates/convert-prost",
	})
	codec.packageMapping["google.type"] = &packagez{name: "google_cloud_type", packageName: "google.type"}

	if _, err := annotateModel(model, codec); err != nil {
		t.Fatal(err)
	}

	msgAnn, ok := extMsg.Codec.(*messageAnnotation)
	if !ok {
		t.Fatalf("expected messageAnnotation on extMsg")
	}
	if want := "google_cloud_type::model::LatLng"; msgAnn.RelativeName != want {
		t.Errorf("msgAnn.RelativeName = %q, want %q", msgAnn.RelativeName, want)
	}
	if want := "crate::prost::google::r#type::LatLng"; msgAnn.ProstRelativeName != want {
		t.Errorf("msgAnn.ProstRelativeName = %q, want %q", msgAnn.ProstRelativeName, want)
	}

	enumAnn, ok := extEnum.Codec.(*enumAnnotation)
	if !ok {
		t.Fatalf("expected enumAnnotation on extEnum")
	}
	if want := "google_cloud_type::model::DayOfWeek"; enumAnn.RelativeName != want {
		t.Errorf("enumAnn.RelativeName = %q, want %q", enumAnn.RelativeName, want)
	}
	t.Run("with prost-path option", func(t *testing.T) {
		extMsg2 := api.NewTestMessage("LatLng").WithPackage("google.type")
		extEnum2 := api.NewTestEnum("DayOfWeek").WithPackage("google.type")
		model2 := api.NewTestAPI(nil, nil, nil)
		model2.ExternalMessages = []*api.Message{extMsg2}
		model2.ExternalEnums = []*api.Enum{extEnum2}

		codec2 := newTestCodec(t, libconfig.SpecProtobuf, "", map[string]string{
			"template-override": "templates/convert-prost",
			"prost-path":        "super::prost",
		})
		codec2.packageMapping["google.type"] = &packagez{name: "google_cloud_type", packageName: "google.type"}

		if _, err := annotateModel(model2, codec2); err != nil {
			t.Fatal(err)
		}

		msgAnn2 := extMsg2.Codec.(*messageAnnotation)
		if want := "super::prost::google::r#type::LatLng"; msgAnn2.ProstRelativeName != want {
			t.Errorf("msgAnn2.ProstRelativeName = %q, want %q", msgAnn2.ProstRelativeName, want)
		}

		enumAnn2 := extEnum2.Codec.(*enumAnnotation)
		if want := "super::prost::google::r#type::DayOfWeek"; enumAnn2.ProstRelativeName != want {
			t.Errorf("enumAnn2.ProstRelativeName = %q, want %q", enumAnn2.ProstRelativeName, want)
		}
	})
}

func TestGrpcRootTypeIDs(t *testing.T) {
	req := api.NewTestMessage("Req").WithPackage("google.cloud.test.v1")
	resp := api.NewTestMessage("Resp").WithPackage("google.cloud.test.v1")

	unaryMethod := api.NewTestMethod("Unary").
		WithInput(req).
		WithOutput(resp).
		WithVerb("GET").
		WithPathTemplate(&api.PathTemplate{})
	streamMethod := api.NewTestMethod("Stream").WithInput(req).WithOutput(resp).WithBidiStreaming()

	for _, test := range []struct {
		name    string
		methods []*api.Method
		options map[string]string
		want    []string
	}{
		{
			name:    "unary method with default http returns no grpc root types",
			methods: []*api.Method{unaryMethod},
			options: map[string]string{},
			want:    nil,
		},
		{
			name:    "unary method with default_transport grpc returns root types",
			methods: []*api.Method{unaryMethod},
			options: map[string]string{
				"default-transport": "grpc",
			},
			want: []string{req.ID, resp.ID},
		},
		{
			name:    "streaming method returns root types",
			methods: []*api.Method{streamMethod},
			options: map[string]string{},
			want:    []string{req.ID, resp.ID},
		},
		{
			name:    "mixed methods with default http returns only streaming root types",
			methods: []*api.Method{unaryMethod, streamMethod},
			options: map[string]string{},
			want:    []string{req.ID, resp.ID},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService("Service").WithPackage("google.cloud.test.v1").WithMethods(test.methods...)
			model := api.NewTestAPI([]*api.Message{req, resp}, []*api.Enum{}, []*api.Service{svc})
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			codec := newTestCodec(t, libconfig.SpecProtobuf, "", test.options)
			if _, err := annotateModel(model, codec); err != nil {
				t.Fatal(err)
			}
			got := GrpcRootTypeIDs(model)
			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
