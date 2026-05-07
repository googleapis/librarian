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
		{
			name: "list method adds iterator import",
			model: &api.API{
				Name:        "parallelstore",
				Title:       "Parallelstore",
				PackageName: "google.cloud.parallelstore.v1",
				Services: []*api.Service{{
					Name: "InstanceService",
					Methods: []*api.Method{{
						Name:              "ListInstances",
						IsAIPStandardList: true,
						InputType: &api.Message{
							Name: "ListInstancesRequest",
							Fields: []*api.Field{
								{
									Name:              "parent",
									ResourceReference: &api.ResourceReference{ChildType: "parallelstore.googleapis.com/Instance"},
								},
							},
						},
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{{
								Verb: "GET",
								PathTemplate: (&api.PathTemplate{}).
									WithLiteral("v1").
									WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
									WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
									WithLiteral("instances"),
							}},
						},
					}},
				}},
				ResourceDefinitions: []*api.Resource{{
					Type: "parallelstore.googleapis.com/Instance",
					Patterns: []api.ResourcePattern{
						(&api.PathTemplate{}).
							WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
							WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
							WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
							Segments,
					},
				}},
			},
			want: SurfaceModel{
				PackageName: "parallelstore",
				Imports: []Import{
					{Alias: "parallelstore", Path: "cloud.google.com/go/parallelstore/apiv1"},
					{Path: "cloud.google.com/go/parallelstore/apiv1/parallelstorepb"},
					{Path: "google.golang.org/api/iterator"},
				},
				Group: Group{
					Name:  "parallelstore",
					Usage: "manage Parallelstore resources",
					Subgroups: []Subgroup{{
						Name:  "instances",
						Usage: "manage instances resources",
						Commands: []Command{{
							Name:       "list",
							Usage:      "list instances",
							PathFormat: "projects/%s/locations/%s",
							Args:       []string{"project", "location"},
							PathLabel:  "parent",
							Flags: []Flag{
								{Name: "limit", Kind: "Int", Required: false, Usage: "The limit."},
								{Name: "location", Kind: "String", Required: true, Usage: "The location."},
							},
							ClientCall: &ClientCall{
								Method:      "ListInstances",
								NameField:   "Parent",
								Package:     "parallelstore",
								RequestType: "parallelstorepb.ListInstancesRequest",
								Paged:       true,
							},
						}},
					}},
				},
			},
		},
		{
			name: "delete method",
			model: &api.API{
				Name:        "parallelstore",
				Title:       "Parallelstore",
				PackageName: "google.cloud.parallelstore.v1",
				Services: []*api.Service{{
					Name: "InstanceService",
					Methods: []*api.Method{{
						Name:                "DeleteInstance",
						IsAIPStandardDelete: true,
						IsLRO:               true,
						InputType: &api.Message{
							Name: "DeleteInstanceRequest",
							Fields: []*api.Field{
								{
									Name:              "name",
									ResourceReference: &api.ResourceReference{Type: "parallelstore.googleapis.com/Instance"},
								},
							},
						},
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{{
								Verb: "DELETE",
								PathTemplate: (&api.PathTemplate{}).
									WithLiteral("v1").
									WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
									WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
									WithLiteral("instances").WithVariable(api.NewPathVariable("instance")),
							}},
						},
					}},
				}},
				ResourceDefinitions: []*api.Resource{{
					Type: "parallelstore.googleapis.com/Instance",
					Patterns: []api.ResourcePattern{
						(&api.PathTemplate{}).
							WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
							WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
							WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
							Segments,
					},
				}},
			},
			want: SurfaceModel{
				PackageName: "parallelstore",
				Imports: []Import{
					{Alias: "parallelstore", Path: "cloud.google.com/go/parallelstore/apiv1"},
					{Path: "cloud.google.com/go/parallelstore/apiv1/parallelstorepb"},
				},
				Group: Group{
					Name:  "parallelstore",
					Usage: "manage Parallelstore resources",
					Subgroups: []Subgroup{{
						Name:  "instances",
						Usage: "manage instances resources",
						Commands: []Command{{
							Name:       "delete",
							Usage:      "delete instances",
							PathFormat: "projects/%s/locations/%s/instances/%s",
							Args:       []string{"project", "location", "instance"},
							PathLabel:  "name",
							Flags: []Flag{
								{Name: "instance", Kind: "String", Required: true, Usage: "The instance."},
								{Name: "location", Kind: "String", Required: true, Usage: "The location."},
							},
							ClientCall: &ClientCall{
								Method:      "DeleteInstance",
								NameField:   "Name",
								Package:     "parallelstore",
								RequestType: "parallelstorepb.DeleteInstanceRequest",
								IsDelete:    true,
								IsLRO:       true,
							},
						}},
					}},
				},
			},
		},
		{
			name:  "create method",
			model: createModel(),
			want: SurfaceModel{
				PackageName: "parallelstore",
				Imports: []Import{
					{Alias: "parallelstore", Path: "cloud.google.com/go/parallelstore/apiv1"},
					{Path: "cloud.google.com/go/parallelstore/apiv1/parallelstorepb"},
				},
				Group: Group{
					Name:  "parallelstore",
					Usage: "manage Parallelstore resources",
					Subgroups: []Subgroup{{
						Name:  "instances",
						Usage: "manage instances resources",
						Commands: []Command{{
							Name:       "create",
							Usage:      "create instances",
							PathFormat: "projects/%s/locations/%s",
							Args:       []string{"project", "location"},
							PathLabel:  "parent",
							Flags: []Flag{
								{Name: "capacity-gib", Kind: "Int64", Required: true, Usage: "The capacity gib."},
								{Name: "description", Kind: "String", Usage: "The description."},
								{Name: "instance-id", Kind: "String", Required: true, Usage: "The instance id."},
								{Name: "location", Kind: "String", Required: true, Usage: "The location."},
							},
							ClientCall: &ClientCall{
								IsCreate:    true,
								IsLRO:       true,
								Method:      "CreateInstance",
								NameField:   "Parent",
								Package:     "parallelstore",
								RequestType: "parallelstorepb.CreateInstanceRequest",
								IDField:     "InstanceId",
								IDFlag:      "instance-id",
								BodyField:   "Instance",
								BodyType:    "parallelstorepb.Instance",
								BodyAssignments: []BodyAssignment{
									{Name: "Description", Flag: "description", Kind: "String"},
									{Name: "CapacityGib", Flag: "capacity-gib", Kind: "Int64"},
								},
							},
						}},
					}},
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

// createModel returns a minimal API model that exercises the AIP-133
// Create code path: an Instance resource with a couple of scalar body
// fields plus an identifier field that should be skipped, and a
// CreateInstance method that takes the Instance as its body.
func createModel() *api.API {
	instance := &api.Message{
		Name: "Instance",
		ID:   ".google.cloud.parallelstore.v1.Instance",
		Fields: []*api.Field{
			{Name: "name", Typez: api.TypezString, Behavior: []api.FieldBehavior{api.FieldBehaviorIdentifier}},
			{Name: "description", Typez: api.TypezString},
			{Name: "capacity_gib", Typez: api.TypezInt64, Behavior: []api.FieldBehavior{api.FieldBehaviorRequired}},
		},
	}
	instance.Resource = &api.Resource{
		Type:     "parallelstore.googleapis.com/Instance",
		Singular: "instance",
		Self:     instance,
		Patterns: []api.ResourcePattern{
			(&api.PathTemplate{}).
				WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
				WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
				WithLiteral("instances").WithVariable(api.NewPathVariable("instance")).
				Segments,
		},
	}
	return &api.API{
		Name:        "parallelstore",
		Title:       "Parallelstore",
		PackageName: "google.cloud.parallelstore.v1",
		Messages:    []*api.Message{instance},
		Services: []*api.Service{{
			Name: "InstanceService",
			Methods: []*api.Method{{
				Name:                "CreateInstance",
				IsAIPStandardCreate: true,
				IsLRO:               true,
				InputType: &api.Message{
					Name: "CreateInstanceRequest",
					Fields: []*api.Field{
						{
							Name:              "parent",
							Typez:             api.TypezString,
							ResourceReference: &api.ResourceReference{ChildType: "parallelstore.googleapis.com/Instance"},
						},
						{Name: "instance_id", Typez: api.TypezString, Behavior: []api.FieldBehavior{api.FieldBehaviorRequired}},
						{Name: "instance", Typez: api.TypezMessage, TypezID: instance.ID, MessageType: instance, Behavior: []api.FieldBehavior{api.FieldBehaviorRequired}},
					},
				},
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb: "POST",
						PathTemplate: (&api.PathTemplate{}).
							WithLiteral("v1").
							WithLiteral("projects").WithVariable(api.NewPathVariable("project")).
							WithLiteral("locations").WithVariable(api.NewPathVariable("location")).
							WithLiteral("instances"),
					}},
				},
			}},
		}},
		ResourceDefinitions: []*api.Resource{instance.Resource},
	}
}
