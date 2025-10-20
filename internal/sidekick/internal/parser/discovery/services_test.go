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

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/api/apitest"
)

func TestService(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := "..zones"
	got, ok := model.State.ServiceByID[id]
	if !ok {
		t.Fatalf("expected service %s in the API model", id)
	}
	want := &api.Service{
		Name:          "zones",
		ID:            id,
		Package:       "",
		Documentation: "Service for the `zones` resource.",
		DefaultHost:   "compute.googleapis.com",
		Methods: []*api.Method{
			{
				ID:            "..zones.get",
				Name:          "get",
				Documentation: "Returns the specified Zone resource.",
				InputTypeID:   "..zones.getRequest",
				OutputTypeID:  "..Zone",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "GET",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("compute").
								WithLiteral("v1").
								WithLiteral("projects").
								WithVariableNamed("project").
								WithLiteral("zones").
								WithVariableNamed("zone"),
							QueryParameters: map[string]bool{},
						},
					},
					BodyFieldPath: "",
				},
			},
			{
				ID:            "..zones.list",
				Name:          "list",
				Documentation: "Retrieves the list of Zone resources available to the specified project.",
				InputTypeID:   "..zones.listRequest",
				OutputTypeID:  "..ZoneList",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "GET",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("compute").
								WithLiteral("v1").
								WithLiteral("projects").
								WithVariableNamed("project").
								WithLiteral("zones"),
							QueryParameters: map[string]bool{
								"filter":               true,
								"maxResults":           true,
								"orderBy":              true,
								"pageToken":            true,
								"returnPartialSuccess": true,
							},
						},
					},
					BodyFieldPath: "",
				},
			},
		},
	}
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
	want := &api.Service{
		ID:            "..TestDeprecated",
		Name:          "TestDeprecated",
		Documentation: "Service for the `TestDeprecated` resource.",
		Deprecated:    true,
	}
	got, ok := model.State.ServiceByID[want.ID]
	if !ok {
		t.Fatalf("missing service %s", want.ID)
	}
	apitest.CheckService(t, got, want)
}

func TestServiceMessages(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}

	getMessage, ok := model.State.MessageByID["..zones.getRequest"]
	if !ok {
		t.Fatalf("expected message %s in the API model", "..zones.getRequest")
	}
	listMessage, ok := model.State.MessageByID["..zones.listRequest"]
	if !ok {
		t.Fatalf("expected message %s in the API model", "..zones.listRequest")
	}

	want := &api.Message{
		Name:               "zones",
		ID:                 "..zones",
		Package:            "",
		Documentation:      "Synthetic messages for the [zones][.zones] service",
		ServicePlaceholder: true,
		Messages:           []*api.Message{getMessage, listMessage},
	}

	got, ok := model.State.MessageByID[want.ID]
	if !ok {
		t.Fatalf("expected service %s in the API model", want.ID)
	}
	apitest.CheckMessage(t, got, want)
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
