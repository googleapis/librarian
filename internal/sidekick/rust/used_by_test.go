// Copyright 2024 Google LLC
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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestUsedByServicesWithServices(t *testing.T) {
	service := api.NewTestService("TestService")
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:tracing":  "used-if=services,package=tracing",
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "tracing",
			packageName: "tracing",
			used:        true,
			usedIf:      []string{"services"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByServicesNoServices(t *testing.T) {
	model := api.NewTestAPI(nil, nil, nil)
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:tracing":  "used-if=services,package=tracing",
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "tracing",
			packageName: "tracing",
			used:        false,
			usedIf:      []string{"services"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByLROsWithLRO(t *testing.T) {
	method := api.NewTestMethod("CreateResource").WithOperationInfo(&api.OperationInfo{})
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
		"package:lro":      "used-if=lro,package=google-cloud-lro",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "lro",
			packageName: "google-cloud-lro",
			used:        true,
			usedIf:      []string{"lro"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByLROsWithoutLRO(t *testing.T) {
	method := api.NewTestMethod("CreateResource")
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
		"package:lro":      "used-if=lro,package=google-cloud-lro",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "lro",
			packageName: "google-cloud-lro",
			used:        false,
			usedIf:      []string{"lro"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByUuidWithAutoPopulation(t *testing.T) {
	requestID := api.NewTestField("request_id").WithType(api.TypezString).WithAutoPopulated()
	method := api.NewTestMethod("CreateResource").WithAutoPopulated(requestID)
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
		"package:uuid":     "used-if=autopopulated,package=uuid,feature=v4",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "uuid",
			packageName: "uuid",
			used:        true,
			usedIf:      []string{"autopopulated"},
			features:    []string{"v4"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByUuidWithoutAutoPopulation(t *testing.T) {
	method := api.NewTestMethod("CreateResource")
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
		"package:uuid":     "used-if=autopopulated,package=uuid,feature=v4",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "uuid",
			packageName: "uuid",
			used:        false,
			usedIf:      []string{"autopopulated"},
			features:    []string{"v4"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByStreamingWithStreaming(t *testing.T) {
	method := api.NewTestMethod("Chat").WithBidiStreaming()
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
		"package:prost":    "used-if=streaming,package=prost",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "prost",
			packageName: "prost",
			used:        true,
			usedIf:      []string{"streaming"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByStreamingWithoutStreaming(t *testing.T) {
	method := api.NewTestMethod("CreateResource")
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
		"package:prost":    "used-if=streaming,package=prost",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "prost",
			packageName: "prost",
			used:        false,
			usedIf:      []string{"streaming"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByStreamingServerStreamingOnly(t *testing.T) {
	method := api.NewTestMethod("ServerStream").WithServerSideStreaming()
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:location": "package=gcp-sdk-location,source=google.cloud.location",
		"package:prost":    "used-if=streaming,package=prost",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "prost",
			packageName: "prost",
			used:        false,
			usedIf:      []string{"streaming"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByStreamingBidiDisabled(t *testing.T) {
	method := api.NewTestMethod("Chat").WithBidiStreaming()
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"template-override": "templates/http-client",
		"package:location":  "package=gcp-sdk-location,source=google.cloud.location",
		"package:prost":     "used-if=streaming,package=prost",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasBidiStreaming(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "prost",
			packageName: "prost",
			used:        false,
			usedIf:      []string{"streaming"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRequiredPackages(t *testing.T) {
	options := map[string]string{
		"package:async-trait": "package=async-trait,force-used=true",
		"package:serde_with":  "package=serde_with,force-used=true,feature=base64,feature=macro,feature=std",
		"package:gtype":       "package=gcp-sdk-type,source=google.type,source=test-only",
		"package:gax":         "package=gcp-sdk-gax,force-used=true",
		"package:auth":        "ignore=true",
	}
	c, err := newCodec(libconfig.SpecProtobuf, options)
	if err != nil {
		t.Fatal(err)
	}
	got := requiredPackages(c.extraPackages)
	want := []string{
		"async-trait.workspace = true",
		"gax.workspace        = true",
		"serde_with           = { workspace = true, features = [\"base64\", \"macro\", \"std\"] }",
	}
	less := func(a, b string) bool { return a < b }
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRequiredPackagesLocal(t *testing.T) {
	// This is not a thing we expect to do in the Rust repository, but the
	// behavior is consistent.
	options := map[string]string{
		"package:gtype": "package=types,source=google.type,source=test-only,force-used=true",
	}
	c, err := newCodec(libconfig.SpecProtobuf, options)
	if err != nil {
		t.Fatal(err)
	}
	got := requiredPackages(c.extraPackages)
	want := []string{
		"gtype.workspace      = true",
	}
	less := func(a, b string) bool { return a < b }
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindUsedPackages(t *testing.T) {
	createResourceReq := api.NewTestMessage("CreateResourceRequest").WithPackage("test")
	resource := api.NewTestMessage("Resource").WithPackage("test")
	operation := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	operationMetadata := api.NewTestMessage("OperationMetadata").WithPackage("google.cloud.common")

	method := api.NewTestMethod("CreateResource").
		WithInput(createResourceReq).
		WithOutput(operation).
		WithOperationInfo(&api.OperationInfo{
			MetadataTypeID: operationMetadata.ID,
			ResponseTypeID: resource.ID,
		})
	service := api.NewTestService("LroService").
		WithPackage("test").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{resource, createResourceReq}, nil, []*api.Service{service})
	model.AddMessage(operation)
	model.AddMessage(operationMetadata)

	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:common":      "package=google-cloud-common,source=google.cloud.common",
		"package:longrunning": "package=google-longrunning,source=google.longrunning",
	})
	if err != nil {
		t.Fatal(err)
	}
	findUsedPackages(model, c)
	want := []*packagez{
		{
			name:        "common",
			packageName: "google-cloud-common",
			used:        true,
		},
		{
			name:        "longrunning",
			packageName: "google-longrunning",
			used:        true,
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindUsedPackages_MapFields(t *testing.T) {
	externalMessage := api.NewTestMessage("ExternalMessage").
		WithPackage("external")

	message := api.NewTestMessage("Fake").
		WithPackage("test")

	mapEntry := api.NewTestMapMessageWithFields(
		"FakeMapEntry",
		api.NewTestField("key").WithType(api.TypezString),
		api.NewTestField("value").WithMessageType(externalMessage),
	)
	message.WithMessages(mapEntry)

	mapField := api.NewTestField("map_field").
		WithMessageType(mapEntry).
		WithMap()

	message.WithFields(mapField)

	model := api.NewTestAPI([]*api.Message{message}, nil, nil)
	model.AddMessage(externalMessage)

	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"package:external": "package=external-package,source=external",
	})
	if err != nil {
		t.Fatal(err)
	}

	findUsedPackages(model, c)

	want := []*packagez{
		{
			name:        "external",
			packageName: "external-package",
			used:        true,
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByGrpcWithDefaultTransport(t *testing.T) {
	method := api.NewTestMethod("GetResource")
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"default-transport": "grpc",
		"package:location":  "package=gcp-sdk-location,source=google.cloud.location",
		"package:prost":     "used-if=streaming,package=prost",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasGrpc(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "prost",
			packageName: "prost",
			used:        true,
			usedIf:      []string{"streaming"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestUsedByGrpcNamedFeature(t *testing.T) {
	method := api.NewTestMethod("GetResource")
	service := api.NewTestService("TestService").WithMethods(method)
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	c, err := newCodec(libconfig.SpecProtobuf, map[string]string{
		"default-transport": "grpc",
		"package:location":  "package=gcp-sdk-location,source=google.cloud.location",
		"package:prost":     "used-if=grpc,package=prost",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolveUsedPackages(model, c.extraPackages, c.hasGrpc(model))
	want := []*packagez{
		{
			name:        "location",
			packageName: "gcp-sdk-location",
		},
		{
			name:        "prost",
			packageName: "prost",
			used:        true,
			usedIf:      []string{"grpc"},
		},
	}
	less := func(a, b *packagez) bool { return a.name < b.name }
	if diff := cmp.Diff(want, c.extraPackages, cmp.AllowUnexported(packagez{}), cmpopts.SortSlices(less)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
