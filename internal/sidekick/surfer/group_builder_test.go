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

package surfer

import (
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
)

func TestGroupBuilder_BuildRoot(t *testing.T) {
	tests := []struct {
		name    string
		config  *provider.Config
		wantHid bool
	}{
		{
			name:    "default not hidden",
			config:  &provider.Config{},
			wantHid: false,
		},
		{
			name: "explicitly hidden",
			config: &provider.Config{
				APIs: []provider.API{
					{
						Name:         "ParallelstoreService",
						RootIsHidden: true,
					},
				},
			},
			wantHid: true,
		},
		{
			name: "explicitly not hidden",
			config: &provider.Config{
				APIs: []provider.API{
					{
						Name:         "ParallelstoreService",
						RootIsHidden: false,
					},
				},
			},
			wantHid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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

			group := newRootGroup(&groupParams{
				model:   model,
				service: model.Services[0],
				config:  test.config,
			})

			if group.ClassName != "parallelstore" {
				t.Errorf("group.ClassName = %q, want %q", group.ClassName, "parallelstore")
			}

			if group.FileName != "parallelstore" {
				t.Errorf("group.FileName = %q, want %q", group.FileName, "parallelstore")
			}

			wantHelp := "Manage Parallelstore resources."
			if group.HelpText != wantHelp {
				t.Errorf("group.HelpText = %q, want %q", group.HelpText, wantHelp)
			}

			if group.Hidden != test.wantHid {
				t.Errorf("group.Hidden = %v, want %v", group.Hidden, test.wantHid)
			}
		})
	}
}

func TestGroupBuilder_BuildGroup(t *testing.T) {
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

	group := newGroup(&groupParams{
		model:   model,
		service: model.Services[0],
		config:  &provider.Config{},
	}, []string{"instances"})

	if group.ClassName != "instances" {
		t.Errorf("group.ClassName = %q, want %q", group.ClassName, "instances")
	}

	if group.FileName != "instances" {
		t.Errorf("group.FileName = %q, want %q", group.FileName, "instances")
	}

	wantHelp := "Manage Instances."
	if group.HelpText != wantHelp {
		t.Errorf("group.HelpText = %q, want %q", group.HelpText, wantHelp)
	}
}
