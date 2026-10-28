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

package parser

import (
	"testing"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/types/known/apipb"
)

func TestProtobuf_LocationMixin(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "test.googleapis.com",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
		},
		Apis: []*apipb.Api{
			{
				Name: "google.cloud.location.Locations",
			},
			{
				Name: "test.googleapis.com.TestService",
			},
		},
		Http: &annotations.Http{
			Rules: []*httpRule{
				{
					Selector: "google.cloud.location.Locations.GetLocation",
					Pattern: &httpRuleGet{
						Get: "/v1/{name=projects/*/locations/*}",
					},
				},
			},
		},
	}
	test, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "test_service.proto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, service := range test.Services {
		if service.ID == ".google.cloud.location.Locations" {
			t.Fatalf("Mixin %s should not be in list of services to generate", service.ID)
		}
	}
	service := test.Service(".test.TestService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.TestService")
	}
	if test.Method(".test.TestService.GetLocation") == nil {
		t.Fatal("Cannot find .test.TestService.GetLocation")
	}

	want := api.NewTestMethod("GetLocation").
		WithID(".test.TestService.GetLocation").
		WithDocumentation("Provides the [Locations][google.cloud.location.Locations] service functionality in this service.").
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("projects").
				WithMatch().
				WithLiteral("locations").
				WithMatch())).
		WithQueryParameters(map[string]bool{})
	want.SourceServiceID = ".google.cloud.location.Locations"
	want.InputTypeID = ".google.cloud.location.GetLocationRequest"
	want.OutputTypeID = ".google.cloud.location.Location"
	apitest.CheckMethod(t, service, "GetLocation", want)
}

func TestProtobuf_IAMMixin(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "test.googleapis.com",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
		},
		Apis: []*apipb.Api{
			{
				Name: "google.iam.v1.IAMPolicy",
			},
			{
				Name: "test.googleapis.com.TestService",
			},
		},
		Http: &annotations.Http{
			Rules: []*httpRule{
				{
					Selector: "google.iam.v1.IAMPolicy.GetIamPolicy",
					Pattern: &httpRulePost{
						Post: "/v1/{resource=services/*}:getIamPolicy",
					},
					Body: "*",
				},
			},
		},
	}
	test, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "test_service.proto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, service := range test.Services {
		if service.ID == ".google.iam.v1.IAMPolicy" {
			t.Fatalf("Mixin %s should not be in list of services to generate", service.ID)
		}
	}

	service := test.Service(".test.TestService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.TestService")
	}
	if test.Method(".test.TestService.GetIamPolicy") == nil {
		t.Fatal("Cannot find .test.TestService.GetIamPolicy")
	}
	want := api.NewTestMethod("GetIamPolicy").
		WithID(".test.TestService.GetIamPolicy").
		WithDocumentation("Provides the [IAMPolicy][google.iam.v1.IAMPolicy] service functionality in this service.").
		WithVerb("POST").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithVariable(api.NewPathVariable("resource").
				WithLiteral("services").
				WithMatch()).
			WithVerb("getIamPolicy")).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("*")
	want.SourceServiceID = ".google.iam.v1.IAMPolicy"
	want.InputTypeID = ".google.iam.v1.GetIamPolicyRequest"
	want.OutputTypeID = ".google.iam.v1.Policy"
	apitest.CheckMethod(t, service, "GetIamPolicy", want)
}

func TestProtobuf_OperationMixin(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "test.googleapis.com",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
			Rules: []*serviceconfig.DocumentationRule{
				{
					Selector:    "google.longrunning.Operations.GetOperation",
					Description: "Custom docs.",
				},
			},
		},
		Apis: []*apipb.Api{
			{
				Name: "google.longrunning.Operations",
			},
			{
				Name: "test.googleapis.com.TestService",
			},
		},
		Http: &annotations.Http{
			Rules: []*httpRule{
				{
					Selector: "google.longrunning.Operations.GetOperation",
					Pattern: &httpRuleGet{
						Get: "/v2/{name=operations/*}",
					},
					Body: "*",
				},
			},
		},
	}
	test, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "test_service.proto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, service := range test.Services {
		if service.ID == ".google.longrunning.Operations" {
			t.Fatalf("Mixin %s should not be in list of services to generate", service.ID)
		}
	}
	service := test.Service(".test.TestService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.TestService")
	}
	if test.Method(".test.TestService.GetOperation") == nil {
		t.Fatal("Cannot find .test.TestService.GetOperation")
	}

	want := api.NewTestMethod("GetOperation").
		WithID(".test.TestService.GetOperation").
		WithDocumentation("Custom docs.").
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("operations").
				WithMatch())).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("*")
	want.SourceServiceID = ".google.longrunning.Operations"
	want.InputTypeID = ".google.longrunning.GetOperationRequest"
	want.OutputTypeID = ".google.longrunning.Operation"
	want.Signatures = []*api.MethodSignature{{Names: []string{"name"}}}
	apitest.CheckMethod(t, service, "GetOperation", want)
}

func TestProtobuf_OperationMixinNoEmpty(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "test.googleapis.com",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
			Rules: []*serviceconfig.DocumentationRule{
				{
					Selector:    "google.longrunning.Operations.GetOperation",
					Description: "Custom docs.",
				},
				{
					Selector:    "google.longrunning.Operations.CancelOperation",
					Description: "Custom docs.",
				},
			},
		},
		Apis: []*apipb.Api{
			{
				Name: "google.longrunning.Operations",
			},
			{
				Name: "test.googleapis.com.TestService",
			},
		},
		Http: &annotations.Http{
			Rules: []*httpRule{
				{
					Selector: "google.longrunning.Operations.GetOperation",
					Pattern: &httpRuleGet{
						Get: "/v2/{name=operations/*}",
					},
					Body: "*",
				},
				{
					Selector: "google.longrunning.Operations.CancelOperation",
					Pattern: &httpRuleDelete{
						Delete: "/v2/{name=operations/*}",
					},
					Body: "*",
				},
			},
		},
	}
	test, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "test_noempty_mixin.proto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, service := range test.Services {
		if service.ID == ".google.longrunning.Operations" {
			t.Fatalf("Mixin %s should not be in list of services to generate", service.ID)
		}
	}
	service := test.Service(".test.TestService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.TestService")
	}
	if test.Method(".test.TestService.GetOperation") == nil {
		t.Fatal("Cannot find .test.TestService.GetOperation")
	}

	want := api.NewTestMethod("CancelOperation").
		WithID(".test.TestService.CancelOperation").
		WithDocumentation("Custom docs.").
		WithVerb("DELETE").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v2").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("operations").
				WithMatch())).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("*").
		ReturnEmpty()
	want.SourceServiceID = ".google.longrunning.Operations"
	want.InputTypeID = ".google.longrunning.CancelOperationRequest"
	want.OutputTypeID = ".google.protobuf.Empty"
	want.Signatures = []*api.MethodSignature{{Names: []string{"name"}}}
	apitest.CheckMethod(t, service, "CancelOperation", want)
	got := test.Message(".google.protobuf.Empty")
	if got == nil {
		t.Fatal("Cannot find .google.protobuf.Empty")
	}
	apitest.CheckMessage(t, got, api.NewTestMessage("Empty").
		WithPackage("google.protobuf"))
}

func TestProtobuf_DuplicateMixin(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "test.googleapis.com",
		Title: "Test API",
		Documentation: &serviceconfig.Documentation{
			Summary:  "Used for testing generation.",
			Overview: "Test Overview",
			Rules: []*serviceconfig.DocumentationRule{
				{
					Selector:    "google.longrunning.Operations.GetOperation",
					Description: "Custom docs.",
				},
			},
		},
		Apis: []*apipb.Api{
			{
				Name: "google.longrunning.Operations",
			},
			{
				Name: "test.googleapis.com.LroService",
			},
		},
		Http: &annotations.Http{
			Rules: []*httpRule{
				{
					Selector: "google.longrunning.Operations.GetOperation",
					Pattern: &httpRuleGet{
						Get: "/v2/{name=operations/*}",
					},
					Body: "*",
				},
			},
		},
	}
	test, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "test_duplicate_mixin.proto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, service := range test.Services {
		if service.ID == ".google.longrunning.Operations" {
			t.Fatalf("Mixin %s should not be in list of services to generate", service.ID)
		}
	}
	service := test.Service(".test.LroService")
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", ".test.LroService")
	}
	if test.Method(".test.LroService.GetOperation") == nil {
		t.Fatal("Cannot find .test.LroService.GetOperation")
	}

	want := api.NewTestMethod("GetOperation").
		WithID(".test.LroService.GetOperation").
		WithDocumentation("Source file docs.").
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("v1").
			WithVariable(api.NewPathVariable("name").
				WithLiteral("operations").
				WithMatch())).
		WithQueryParameters(map[string]bool{})
	want.SourceServiceID = ".test.LroService"
	want.InputTypeID = ".google.longrunning.GetOperationRequest"
	want.OutputTypeID = ".google.longrunning.Operation"
	apitest.CheckMethod(t, service, "GetOperation", want)
}
