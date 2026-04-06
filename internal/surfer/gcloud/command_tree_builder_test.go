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

package gcloud

import (
	"fmt"
	"path"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser/httprule"
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

func TestCommandTreeBuilder_Build_Structure(t *testing.T) {
	service := mockService("parallelstore.googleapis.com",
		mockMethod("CreateInstance", "v1/{parent=projects/*/locations/*}/instances"),
		mockMethod("ListInstances", "v1/{parent=projects/*/locations/*}/instances"),
		mockMethod("GetOperation", "v1/{name=projects/*/locations/*/operations/*}"),
	)
	model := &api.API{
		Name:        "parallelstore",
		PackageName: "google.cloud.parallelstore.v1",
		Title:       "Parallelstore API",
		Services:    []*api.Service{service},
	}

	config := &provider.Config{
		GenerateOperations: boolPtr(true),
		APIs: []provider.API{
			{
				Name: "parallelstore",
			},
		},
	}

	root, err := newCommandTreeBuilder(model, config).build()
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.GA)
	want := []string{
		"parallelstore/instances/create",
		"parallelstore/instances/list",
		"parallelstore/operations/describe",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got):\n%s", diff)
	}
}

func TestCommandTreeBuilder_Build_Operations_Disabled(t *testing.T) {
	service := mockService("parallelstore.googleapis.com", mockMethod("GetOperation", "v1/{name=projects/*/locations/*/operations/*}"))

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{service},
	}

	root, err := newCommandTreeBuilder(model, &provider.Config{GenerateOperations: boolPtr(false)}).build()
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.GA)
	if len(got) != 0 {
		t.Errorf("flattenTree() = %v, want empty when GenerateOperations is false", got)
	}
}

func TestCommandTreeBuilder_Build_Operations_Enabled(t *testing.T) {
	service := mockService("parallelstore.googleapis.com", mockMethod("GetOperation", "v1/{name=projects/*/locations/*/operations/*}"))

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{service},
	}

	root, err := newCommandTreeBuilder(model, &provider.Config{GenerateOperations: boolPtr(true)}).build()
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.GA)
	want := []string{
		"parallelstore/operations/describe",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got) when GenerateOperations is true:\n%s", diff)
	}
}

func TestCommandTreeBuilder_Build_MultipleServices(t *testing.T) {
	serviceOne := mockService("ParallelstoreService", mockMethod("CreateInstance", "v1/{parent=projects/*/locations/*}/instances"))
	serviceTwo := mockService("OtherParallelstoreService", mockMethod("CreateOtherInstance", "v1/{parent=projects/*/locations/*}/otherInstances"))

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{serviceOne, serviceTwo},
	}

	root, err := newCommandTreeBuilder(model, &provider.Config{GenerateOperations: boolPtr(true)}).build()
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.GA)
	want := []string{
		"parallelstore/instances/create",
		"parallelstore/otherInstances/create",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got):\n%s", diff)
	}
}

func TestCommandTreeBuilder_Build_MultipleReleaseTracks(t *testing.T) {
	serviceOne := mockService("ParallelstoreService", mockMethod("CreateInstance", "v1/{parent=projects/*/locations/*}/instances"))
	serviceTwo := mockService("ParallelstoreService", mockMethod("CreateInstance", "v1alpha/{parent=projects/*/locations/*}/instances"))
	serviceTwo.Package = "google.cloud.parallelstore.v1alpha"

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{serviceOne, serviceTwo},
	}

	root, err := newCommandTreeBuilder(model, &provider.Config{GenerateOperations: boolPtr(true)}).build()
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	// GA release track
	got := flattenTree(root.GA)
	want := []string{
		"parallelstore/instances/create",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got):\n%s", diff)
	}

	// Alpha release track
	got = flattenTree(root.ALPHA)
	want = []string{
		"parallelstore/instances/create",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got):\n%s", diff)
	}
}

func flattenTree(g *CommandGroup) []string {
	var paths []string
	var walk func(prefix string, current *CommandGroup)
	walk = func(prefix string, current *CommandGroup) {
		for name := range current.Commands {
			paths = append(paths, path.Join(prefix, current.Name, name))
		}
		for _, sub := range current.Groups {
			walk(path.Join(prefix, current.Name), sub)
		}
	}
	walk("", g)
	slices.Sort(paths)
	return paths
}

func mockMethod(name, path string) *api.Method {
	pt, err := httprule.ParseResourcePattern(path)
	if err != nil {
		panic(fmt.Sprintf("failed to parse path %q: %v", path, err))
	}
	return &api.Method{
		Name: name,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					PathTemplate: pt,
				},
			},
		},
		InputType: &api.Message{
			Fields: []*api.Field{},
		},
	}
}

func mockService(name string, methods ...*api.Method) *api.Service {
	s := &api.Service{
		Name:        name,
		DefaultHost: "parallelstore.googleapis.com",
		Package:     "google.cloud.parallelstore.v1",
		Methods:     methods,
	}
	for _, m := range methods {
		m.Service = s
	}
	return s
}

func boolPtr(b bool) *bool {
	return &b
}
