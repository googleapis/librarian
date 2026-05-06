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

	"github.com/google/go-cmp/cmp"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestConstructSurfaceModel(t *testing.T) {
	for _, test := range []struct {
		name  string
		model *api.API
		want  SurfaceModel
	}{
		{
			name: "single service with one command",
			model: &api.API{
				Name:        "parallelstore",
				Title:       "Parallelstore",
				PackageName: "google.cloud.parallelstore.v1",
				Services: []*api.Service{{
					Name: "InstanceService",
					Methods: []*api.Method{{
						Name:      "GetInstance",
						InputType: &api.Message{Name: "GetInstanceRequest"},
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{{
								Verb: "GET",
								PathTemplate: (&api.PathTemplate{}).
									WithLiteral("v1").
									WithLiteral("instances").
									WithVariable(api.NewPathVariable("instance")),
							}},
						},
					}},
				}},
			},
			want: SurfaceModel{
				PackageName: "parallelstore",
				Imports:     nil,
				Group: Group{
					Name:  "parallelstore",
					Usage: "manage Parallelstore resources",
					Subgroups: []Subgroup{{
						Name:  "instances",
						Usage: "manage instances resources",
						Commands: []Command{{
							Name:       "describe",
							Usage:      "describe instances",
							ClientCall: nil,
						}},
					}},
				},
			},
		},
		{
			name: "subgroups sorted alphabetically",
			model: &api.API{
				Name:        "parallelstore",
				Title:       "Parallelstore",
				PackageName: "google.cloud.parallelstore.v1",
				Services: []*api.Service{{
					Name: "InstanceService",
					Methods: []*api.Method{
						{
							Name: "ListInstances",
							PathInfo: &api.PathInfo{
								Bindings: []*api.PathBinding{{
									Verb: "GET",
									PathTemplate: (&api.PathTemplate{}).
										WithLiteral("instances"),
								}},
							},
						},
						{
							Name: "ListBackups",
							PathInfo: &api.PathInfo{
								Bindings: []*api.PathBinding{{
									Verb: "GET",
									PathTemplate: (&api.PathTemplate{}).
										WithLiteral("backups"),
								}},
							},
						},
					},
				}},
			},
			want: SurfaceModel{
				PackageName: "parallelstore",
				Imports:     nil,
				Group: Group{
					Name:  "parallelstore",
					Usage: "manage Parallelstore resources",
					Subgroups: []Subgroup{
						{
							Name:     "backups",
							Usage:    "manage backups resources",
							Commands: []Command{{Name: "list", Usage: "list backups"}},
						},
						{
							Name:     "instances",
							Usage:    "manage instances resources",
							Commands: []Command{{Name: "list", Usage: "list instances"}},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := constructSurfaceModel(test.model, "")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
