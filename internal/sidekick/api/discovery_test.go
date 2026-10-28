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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestLroServices(t *testing.T) {
	d := &Discovery{
		Pollers: []*Poller{
			{Prefix: "p1", MethodID: "Service1.Method1"},
			{Prefix: "p2", MethodID: "Service1.Method2"},
			{Prefix: "p3", MethodID: "Service2.Method3"},
		},
	}
	want := map[string]bool{
		"Service1": true,
		"Service2": true,
	}
	got := d.LroServices()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestPathParameters(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *Poller
		want  []string
	}{
		{
			name:  "multiple parameters",
			input: &Poller{Prefix: "projects/{project}/zones/{zone}"},
			want:  []string{"project", "zone"},
		},
		{
			name:  "no parameters",
			input: &Poller{Prefix: "abc/def"},
			want:  nil,
		},
		{
			name:  "single parameter",
			input: &Poller{Prefix: "a/{b}"},
			want:  []string{"b"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.input.PathParameters()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddDiscoveryLroAnnotations(t *testing.T) {
	discoveryConfig := &Discovery{
		OperationID: "..Operation",
		Pollers: []*Poller{
			{Prefix: "compute/v1/projects/{project}/zones/{zone}", MethodID: "..zoneOperations.get"},
		},
	}
	operation := NewTestMessage("Operation").WithPackage("")
	service := NewTestService("instances").WithPackage("").WithMethods(
		NewTestMethod("insert").
			WithID("..instances.insert").
			WithOutput(operation).
			WithVerb("POST").
			WithPathTemplate(NewPathTemplateBuilder().
				WithLiteral("compute").
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("zones").
				WithVariableNamed("zone").
				WithLiteral("instances").
				Build()).
			WithQueryParameters(),
	)
	pollerService := NewTestService("zoneOperations").WithPackage("").WithMethods(
		NewTestMethod("get").
			WithID("..zoneOperations.get").
			WithOutput(operation).
			WithVerb("GET").
			WithPathTemplate(NewPathTemplateBuilder().
				WithLiteral("compute").
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("zones").
				WithVariableNamed("zone").
				WithLiteral("operations").
				WithVariableNamed("operation").
				Build()).
			WithQueryParameters(),
	)
	model := NewTestAPI([]*Message{operation}, nil, []*Service{service, pollerService})
	if err := AddDiscoveryLroAnnotations(model, discoveryConfig); err != nil {
		t.Fatal(err)
	}
	gotInsert := model.Method("..instances.insert")
	wantLro := &DiscoveryLro{PollingPathParameters: []string{"project", "zone"}}
	if diff := cmp.Diff(wantLro, gotInsert.DiscoveryLro); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	gotPoller := model.Method("..instances.getOperation")
	if gotPoller == nil {
		t.Fatal("missing injected ..instances.getOperation method")
	}
	if !gotPoller.IsLroPoller {
		t.Errorf("expected IsLroPoller to be true")
	}
	if diff := cmp.Diff("..zoneOperations", gotPoller.SourceServiceID); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(pollerService.Methods[0].PathInfo, gotPoller.PathInfo, cmp.AllowUnexported(PathBinding{}, PathTemplate{}, PathSegment{}, PathVariable{}), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAddDiscoveryLroAnnotationsError(t *testing.T) {
	discoveryConfig := &Discovery{
		OperationID: "..Operation",
		Pollers: []*Poller{
			{Prefix: "p/{project}/l/{zone}", MethodID: "..Operations.get_1"},
			{Prefix: "p/{project}/l/{region}", MethodID: "..Operations.get_2"},
		},
	}
	operation := NewTestMessage("Operation").WithPackage("")
	badService := NewTestService("Service").WithPackage("").WithMethods(
		NewTestMethod("create_foo").
			WithOutput(operation).
			WithVerb("GET").
			WithPathTemplate(NewPathTemplateBuilder().
				WithLiteral("p").
				WithVariableNamed("project").
				WithLiteral("l").
				WithVariableNamed("zone").
				WithLiteral("foo").
				WithVariableNamed("id").
				Build()).
			WithQueryParameters(),
		NewTestMethod("create_bar").
			WithOutput(operation).
			WithVerb("GET").
			WithPathTemplate(NewPathTemplateBuilder().
				WithLiteral("p").
				WithVariableNamed("project").
				WithLiteral("l").
				WithVariableNamed("region").
				WithLiteral("foo").
				WithVariableNamed("id").
				Build()).
			WithQueryParameters(),
	)
	pollerService := NewTestService("Operations").WithPackage("").WithMethods(
		NewTestMethod("get_1").
			WithOutput(operation).
			WithVerb("GET").
			WithPathTemplate(NewPathTemplateBuilder().
				WithLiteral("p").
				WithVariableNamed("project").
				WithLiteral("l").
				WithVariableNamed("zone").
				WithLiteral("operations").
				WithVariableNamed("operation").
				Build()).
			WithQueryParameters(),
		NewTestMethod("get_2").
			WithOutput(operation).
			WithVerb("GET").
			WithPathTemplate(NewPathTemplateBuilder().
				WithLiteral("p").
				WithVariableNamed("project").
				WithLiteral("l").
				WithVariableNamed("region").
				WithLiteral("operations").
				WithVariableNamed("operation").
				Build()).
			WithQueryParameters(),
	)
	model := NewTestAPI([]*Message{operation}, nil, []*Service{badService, pollerService})
	if err := AddDiscoveryLroAnnotations(model, discoveryConfig); err == nil {
		t.Errorf("expected error for mismatched LRO mixin, got nil")
	}
}
