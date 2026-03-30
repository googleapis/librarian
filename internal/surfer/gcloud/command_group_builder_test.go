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
	"reflect"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

func TestCommandGroupBuilder_BuildRoot(t *testing.T) {
	model := &api.API{
		Name:        "parallelstore",
		PackageName: "google.cloud.parallelstore.v1",
		Title:       "Parallelstore API",
		Services: []*api.Service{
			{
				Name:        "ParallelstoreService",
				DefaultHost: "parallelstore.googleapis.com",
			},
		},
	}

	builder := newCommandGroupBuilder(model, &provider.Config{})
	root := builder.buildRoot()

	if root.Name != "parallelstore" {
		t.Errorf("root.Name = %q, want %q", root.Name, "parallelstore")
	}

	wantHelp := "Manage Parallelstore resources."
	if root.HelpText != wantHelp {
		t.Errorf("root.HelpText = %q, want %q", root.HelpText, wantHelp)
	}

	wantTracks := []string{"GA"}
	if !reflect.DeepEqual(root.Tracks, wantTracks) {
		t.Errorf("root.Tracks = %v, want %v", root.Tracks, wantTracks)
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

	builder := newCommandGroupBuilder(model, &provider.Config{})
	group := builder.build([]string{"instances"}, 0)

	if group.Name != "instances" {
		t.Errorf("group.Name = %q, want %q", group.Name, "instances")
	}

	wantHelp := "Manage Parallelstore instances resources." // Fallback singular is the segment name if no resource found
	if group.HelpText != wantHelp {
		t.Errorf("group.HelpText = %q, want %q", group.HelpText, wantHelp)
	}

	wantTracks := []string{"BETA"}
	if !reflect.DeepEqual(group.Tracks, wantTracks) {
		t.Errorf("group.Tracks = %v, want %v", group.Tracks, wantTracks)
	}
}
