// Copyright 2026 Google LLC
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
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

func TestCommandGroupBuilder_BuildRoot(t *testing.T) {
	model := &api.API{
		Name:  "parallelstore",
		Title: "Parallelstore API",
		Services: []*api.Service{
			{
				Name:        "ParallelstoreService",
				DefaultHost: "parallelstore.googleapis.com",
			},
		},
	}

	builder, err := newCommandGroupBuilder(model, model.Services[0], &provider.Config{})
	if err != nil {
		t.Fatalf("newCommandGroupBuilder() failed: %v", err)
	}
	group := builder.buildRoot()

	if group.Name != "parallelstore" {
		t.Errorf("group.Name = %q, want %q", group.Name, "parallelstore")
	}

	wantPath := []string{"parallelstore"}
	if len(group.Path) != len(wantPath) || group.Path[0] != wantPath[0] {
		t.Errorf("group.Path = %v, want %v", group.Path, wantPath)
	}

	wantHelp := "Manage Parallelstore resources."
	if group.HelpText != wantHelp {
		t.Errorf("group.HelpText = %q, want %q", group.HelpText, wantHelp)
	}
}

func TestCommandGroupBuilder_BuildGroup(t *testing.T) {
	model := &api.API{
		Name:        "parallelstore",
		PackageName: "google.cloud.parallelstore.v1beta1",
		Title:       "Parallelstore API",
		Services: []*api.Service{
			{
				Name:        "ParallelstoreService",
				DefaultHost: "parallelstore.googleapis.com",
			},
		},
	}

	builder, err := newCommandGroupBuilder(model, model.Services[0], &provider.Config{})
	if err != nil {
		t.Fatalf("newCommandGroupBuilder() failed: %v", err)
	}
	group := builder.build([]string{"instances"}, 0, []string{"parallelstore"})

	if group.Name != "instances" {
		t.Errorf("group.Name = %q, want %q", group.Name, "instances")
	}

	wantPath := []string{"parallelstore", "instances"}
	if len(group.Path) != len(wantPath) || group.Path[0] != wantPath[0] || group.Path[1] != wantPath[1] {
		t.Errorf("group.Path = %v, want %v", group.Path, wantPath)
	}

	wantHelp := "Manage Instances resources." // Fallback singular is the segment name if no resource found
	if group.HelpText != wantHelp {
		t.Errorf("group.HelpText = %q, want %q", group.HelpText, wantHelp)
	}
}

func TestNewCommandGroupBuilder_EmptyDefaultHost(t *testing.T) {
	model := &api.API{}
	service := &api.Service{Name: "ParallelstoreService", DefaultHost: ""}

	_, err := newCommandGroupBuilder(model, service, &provider.Config{})
	if err == nil {
		t.Fatal("newCommandGroupBuilder() passed when DefaultHost was empty")
	}
}
