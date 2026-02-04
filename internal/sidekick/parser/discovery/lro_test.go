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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

func TestLroAnnotations(t *testing.T) {
	cfg := &config.Config{
		Discovery: &config.Discovery{
			OperationID: "..Operation",
			Pollers: []*config.Poller{
				{Prefix: "compute/v1/projects/{project}/zones/{zone}", MethodID: "..zoneOperations.get"},
				{Prefix: "compute/v1/projects/{project}/regions/{region}", MethodID: "..regionOperations.get"},
			},
		},
	}
	model, err := ComputeDiscoWithLros(t, cfg)
	if err != nil {
		t.Fatal(err)
	}

	want := &api.Method{
		ID:           "..instances.insert",
		Name:         "insert",
		InputTypeID:  "..instances.insertRequest",
		OutputTypeID: "..Operation",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "POST",
					PathTemplate: api.NewPathTemplate().
						WithLiteral("compute").
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("zones").
						WithVariableNamed("zone").
						WithLiteral("instances"),
					QueryParameters: map[string]bool{
						"requestId":              true,
						"sourceInstanceTemplate": true,
						"sourceMachineImage":     true,
					},
				},
			},
			BodyFieldPath: "body",
		},
		DiscoveryLro: &api.DiscoveryLro{
			PollingPathParameters: []string{"project", "zone"},
		},
	}
	got, ok := model.State.MethodByID[want.ID]
	if !ok {
		t.Fatalf("missing method %s in model", want.ID)
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Method{}, "Documentation")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

	// The parser should have injected a mixin method.
	wantMixin := &api.Method{
		ID:           "..instances.getOperation",
		Name:         "getOperation",
		InputTypeID:  "..zoneOperations.getRequest",
		OutputTypeID: "..Operation",
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
						WithVariableNamed("zone").
						WithLiteral("operations").
						WithVariableNamed("operation"),
					QueryParameters: map[string]bool{},
				},
			},
			BodyFieldPath: "",
		},
	}
	gotMixin, ok := model.State.MethodByID[wantMixin.ID]
	if !ok {
		t.Fatalf("missing method %s in model", wantMixin.ID)
	}
	if diff := cmp.Diff(wantMixin, gotMixin, cmpopts.IgnoreFields(api.Method{}, "Documentation", "Service")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestLroAnnotationsError(t *testing.T) {
	cfg := &config.Config{
		Discovery: &config.Discovery{
			OperationID: "..Operation",
			Pollers: []*config.Poller{
				{Prefix: "p/{project}/l/{zone}", MethodID: "..Operations.get_1"},
				{Prefix: "p/{project}/l/{region}", MethodID: "..Operations.get_2"},
			},
		},
	}

	// None of the real services does this, but we want the parser to report an
	// error if it ever does.
	badService := &api.Service{
		ID:   "..Service",
		Name: "Service",
		Methods: []*api.Method{
			{
				Name:         "create_foo",
				ID:           "..Service.create_foo",
				OutputTypeID: "..Operation",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "GET",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("p").
								WithVariableNamed("project").
								WithLiteral("l").
								WithVariableNamed("zone").
								WithLiteral("foo").
								WithVariableNamed("id"),
							QueryParameters: map[string]bool{},
						},
					},
				},
			},
			{
				Name:         "create_bar",
				ID:           "..Service.create_bar",
				OutputTypeID: "..Operation",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "GET",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("p").
								WithVariableNamed("project").
								WithLiteral("l").
								WithVariableNamed("region").
								WithLiteral("foo").
								WithVariableNamed("id"),
							QueryParameters: map[string]bool{},
						},
					},
				},
			},
		},
	}
	pollerService := &api.Service{
		ID:   "..Operations",
		Name: "Operations",
		Methods: []*api.Method{
			{
				Name:         "get_1",
				ID:           "..Operations.get_1",
				OutputTypeID: "..Operation",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "GET",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("p").
								WithVariableNamed("project").
								WithLiteral("l").
								WithVariableNamed("zone").
								WithLiteral("operations").
								WithVariableNamed("operation"),
							QueryParameters: map[string]bool{},
						},
					},
				},
			},
			{
				Name:         "get_2",
				ID:           "..Operations.get_2",
				OutputTypeID: "..Operation",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "GET",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("p").
								WithVariableNamed("project").
								WithLiteral("l").
								WithVariableNamed("region").
								WithLiteral("operations").
								WithVariableNamed("operation"),
							QueryParameters: map[string]bool{},
						},
					},
				},
			},
		},
	}
	operation := &api.Message{
		ID:   "..Operation",
		Name: "Operation",
	}

	model := api.NewTestAPI([]*api.Message{operation}, []*api.Enum{}, []*api.Service{badService, pollerService})
	if lroAnnotations(model, cfg.Discovery) == nil {
		t.Errorf("expected an error, got %v", badService)
	}
}
