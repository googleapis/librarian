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

package surfer

import (
	"bytes"
	"fmt"
	"log/slog"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser/httprule"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
	"github.com/iancoleman/strcase"
)

func TestSurfaceBuilder_Build_Structure(t *testing.T) {
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

	root, err := buildSurface(model, config)
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.Root)
	want := []string{
		"parallelstore/instances/create",
		"parallelstore/instances/list",
		"parallelstore/operations/describe",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got):\n%s", diff)
	}
}

func TestSurfaceBuilder_Build_Operations_Disabled(t *testing.T) {
	service := mockService("parallelstore.googleapis.com", mockMethod("GetOperation", "v1/{name=projects/*/locations/*/operations/*}"))

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{service},
	}

	root, err := buildSurface(model, &provider.Config{GenerateOperations: boolPtr(false)})
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.Root)
	if len(got) != 0 {
		t.Errorf("flattenTree() = %v, want empty when GenerateOperations is false", got)
	}
}

func TestSurfaceBuilder_Build_Operations_Enabled(t *testing.T) {
	service := mockService("parallelstore.googleapis.com", mockMethod("GetOperation", "v1/{name=projects/*/locations/*/operations/*}"))

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{service},
	}

	root, err := buildSurface(model, &provider.Config{GenerateOperations: boolPtr(true)})
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.Root)
	want := []string{
		"parallelstore/operations/describe",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got) when GenerateOperations is true:\n%s", diff)
	}
}

func TestSurfaceBuilder_Build_MultipleServices(t *testing.T) {
	serviceOne := mockService("ParallelstoreService", mockMethod("CreateInstance", "v1/{parent=projects/*/locations/*}/instances"))
	serviceTwo := mockService("OtherParallelstoreService", mockMethod("CreateOtherInstance", "v1/{parent=projects/*/locations/*}/otherInstances"))

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{serviceOne, serviceTwo},
	}

	root, err := buildSurface(model, &provider.Config{GenerateOperations: boolPtr(true)})
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.Root)
	want := []string{
		"parallelstore/instances/create",
		"parallelstore/other_instances/create",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got):\n%s", diff)
	}
}

func TestSurfaceBuilder_Build_HelpTextOverride(t *testing.T) {
	service := mockService("ParallelstoreService",
		mockMethod("CreateInstance", "v1/{parent=projects/*/locations/*}/instances"),
	)
	model := &api.API{
		Name:        "parallelstore",
		PackageName: "google.cloud.parallelstore.v1",
		Title:       "Parallelstore API",
		Services:    []*api.Service{service},
	}
	service.Methods[0].ID = "google.cloud.parallelstore.v1.Parallelstore.CreateInstance"

	config := &provider.Config{
		APIs: []provider.API{
			{
				Name: "parallelstore",
				HelpText: &provider.HelpTextRules{
					MethodRules: []*provider.HelpTextRule{
						{
							Selector: "google.cloud.parallelstore.v1.Parallelstore.CreateInstance",
							HelpText: &provider.HelpTextElement{
								Brief: "Override Brief",
							},
						},
					},
				},
			},
		},
	}

	root, err := buildSurface(model, config)
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	instancesGroup, ok := root.Root.Groups["instances"]
	if !ok {
		t.Fatal("instances group not found")
	}
	createCmd, ok := instancesGroup.Commands["create"]
	if !ok {
		t.Fatal("create command not found")
	}

	if createCmd.HelpText.Brief != "Override Brief" {
		t.Errorf("expected brief to be 'Override Brief', got %q", createCmd.HelpText.Brief)
	}
}

func flattenTree(g *CommandGroup) []string {
	var paths []string
	var walk func(prefix string, current *CommandGroup)
	walk = func(prefix string, current *CommandGroup) {
		for name := range current.Commands {
			paths = append(paths, path.Join(prefix, strcase.ToSnake(current.FileName), name))
		}
		for _, sub := range current.Groups {
			walk(path.Join(prefix, strcase.ToSnake(current.FileName)), sub)
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

func TestSurfaceBuilder_Build_SynthesizeWaitCommand(t *testing.T) {
	opMethod := mockMethod("GetOperation", "v1/{name=projects/*/locations/*/operations/*}")
	opMethod.SourceServiceID = ".google.longrunning.Operations"
	opMethod.InputType.Fields = append(opMethod.InputType.Fields, &api.Field{
		Name: "name",
	})
	service := mockService("parallelstore.googleapis.com", opMethod)

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{service},
	}

	root, err := buildSurface(model, &provider.Config{GenerateOperations: boolPtr(true)})
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.Root)
	want := []string{
		"parallelstore/operations/describe",
		"parallelstore/operations/wait",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got) expecting both describe and wait:\n%s", diff)
	}
}

func TestSurfaceBuilder_Build_SynthesizeWaitCommand_Warning(t *testing.T) {
	opMethod := mockMethod("GetOperation", "v1/{name=projects/*/locations/*/operations/*}")
	opMethod.SourceServiceID = ".google.longrunning.Operations"
	// Do not add the "name" field to opMethod.InputType.Fields so buildWaitCommand fails.
	service := mockService("parallelstore.googleapis.com", opMethod)

	model := &api.API{
		Name:     "parallelstore",
		Title:    "Parallelstore API",
		Services: []*api.Service{service},
	}

	// Set up custom slog logger to capture warnings.
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, nil)
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(oldLogger)

	root, err := buildSurface(model, &provider.Config{GenerateOperations: boolPtr(true)})
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	got := flattenTree(root.Root)
	want := []string{
		"parallelstore/operations/describe",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("flattenTree() mismatch (-want +got) expecting wait to be skipped:\n%s", diff)
	}

	logMsg := buf.String()
	wantWarning := "failed to build wait command for operations"
	if !strings.Contains(logMsg, wantWarning) {
		t.Errorf("expected log to contain warning %q, got log:\n%s", wantWarning, logMsg)
	}
}
