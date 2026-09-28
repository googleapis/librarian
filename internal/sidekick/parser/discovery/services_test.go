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

package discovery

import (
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestService(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := "..zones"
	got := model.Service(id)
	if got == nil {
		t.Fatalf("expected service %s in the API model", id)
	}
	getMethod := api.NewTestMethod("get").
		WithID("..zones.get").
		WithDocumentation("Returns the specified Zone resource.").
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("compute").
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("zones").
			WithVariableNamed("zone")).
		WithQueryParameters(map[string]bool{}).
		WithBodyFieldPath("")
	getMethod.InputTypeID = "..zones.getRequest"
	getMethod.OutputTypeID = "..Zone"
	getMethod.Signatures = []*api.MethodSignature{{Names: []string{"project", "zone"}}}

	listMethod := api.NewTestMethod("list").
		WithID("..zones.list").
		WithDocumentation("Retrieves the list of Zone resources available to the specified project.").
		WithVerb("GET").
		WithPathTemplate((&api.PathTemplate{}).
			WithLiteral("compute").
			WithLiteral("v1").
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("zones")).
		WithQueryParameters(map[string]bool{
			"filter":               true,
			"maxResults":           true,
			"orderBy":              true,
			"pageToken":            true,
			"returnPartialSuccess": true,
		}).
		WithBodyFieldPath("")
	listMethod.InputTypeID = "..zones.listRequest"
	listMethod.OutputTypeID = "..ZoneList"
	listMethod.Signatures = []*api.MethodSignature{{Names: []string{"project"}}}

	want := newDiscoveryTestService("zones").
		WithDocumentation("Service for the `zones` resource.").
		WithDefaultHost("compute.googleapis.com")
	want.Methods = []*api.Method{getMethod, listMethod}
	apitest.CheckService(t, got, want)
}

func TestServiceDeprecated(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := document{}
	input := resource{
		Name:       "TestDeprecated",
		Deprecated: true,
	}
	if err := addService(model, &doc, &input); err != nil {
		t.Fatal(err)
	}
	want := newDiscoveryTestService("TestDeprecated").
		WithDocumentation("Service for the `TestDeprecated` resource.").
		WithDeprecated(true)
	got := model.Service(want.ID)
	if got == nil {
		t.Fatalf("missing service %s", want.ID)
	}
	apitest.CheckService(t, got, want)
}

func TestServiceMessages(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}

	getMessage := model.Message("..zones.getRequest")
	if getMessage == nil {
		t.Fatalf("expected message %s in the API model", "..zones.getRequest")
	}
	listMessage := model.Message("..zones.listRequest")
	if listMessage == nil {
		t.Fatalf("expected message %s in the API model", "..zones.listRequest")
	}

	want := newDiscoveryTestMessage("zones").
		WithID("..zones").
		WithDocumentation("Synthetic messages for the [zones][.zones] service").
		WithMessages(getMessage, listMessage)
	want.ServicePlaceholder = true
	got := model.Message(want.ID)
	if got == nil {
		t.Fatalf("expected service %s in the API model", want.ID)
	}
	apitest.CheckMessage(t, got, want)
}

// newDiscoveryTestMessage creates a Message with an empty package, matching
// Discovery parser output for APIs without a service config.
func newDiscoveryTestMessage(name string) *api.Message {
	return api.NewTestMessage(name).WithPackage("")
}

// newDiscoveryTestService creates a Service with an empty package, matching
// Discovery parser output for APIs without a service config.
func newDiscoveryTestService(name string) *api.Service {
	return api.NewTestService(name).WithPackage("")
}

func TestServiceTopLevelMethodErrors(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := document{}
	input := resource{
		Methods: []*method{
			{MediaUpload: &mediaUpload{}},
		},
	}
	if err := addServiceRecursive(model, &doc, &input); err == nil {
		t.Errorf("expected error in addServiceRecursive invalid top-level method, got=%v", model.Services)
	}
}

func TestServiceChildMethodErrors(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := document{}
	input := resource{
		Resources: []*resource{
			{
				Methods: []*method{
					{MediaUpload: &mediaUpload{}},
				},
			},
		},
	}
	if err := addServiceRecursive(model, &doc, &input); err == nil {
		t.Errorf("expected error in addServiceRecursive invalid child method, got=%v", model.Services)
	}
}
