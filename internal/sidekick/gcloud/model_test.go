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

func TestBuildCommands(t *testing.T) {
	for _, test := range []struct {
		name  string
		model *api.API
		want  []commandWithSubgroup
	}{
		{
			name: "single service with one command",
			model: &api.API{
				Services: []*api.Service{{
					Name: "InstanceService",
					Methods: []*api.Method{{
						Name: "GetInstance",
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
			want: []commandWithSubgroup{{
				Command: Command{
					Name:  "describe",
					Usage: "describe instances",
				},
				Subgroup: "instances",
			}},
		},
		{
			name: "method skipped: no primary binding",
			model: &api.API{
				Services: []*api.Service{{
					Name: "Skip",
					Methods: []*api.Method{{
						Name: "GetThing",
					}},
				}},
			},
			want: nil,
		},
		{
			name: "method skipped: binding has no literal segments",
			model: &api.API{
				Services: []*api.Service{{
					Name: "Skip",
					Methods: []*api.Method{{
						Name: "GetThing",
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{{
								Verb: "GET",
								PathTemplate: (&api.PathTemplate{}).
									WithVariable(api.NewPathVariable("name")),
							}},
						},
					}},
				}},
			},
			want: nil,
		},
		{
			name: "two services, mixed methods",
			model: &api.API{
				Services: []*api.Service{
					{
						Name: "InstanceService",
						Methods: []*api.Method{
							{
								Name: "GetInstance",
								PathInfo: &api.PathInfo{
									Bindings: []*api.PathBinding{{
										Verb: "GET",
										PathTemplate: (&api.PathTemplate{}).
											WithLiteral("instances").
											WithVariable(api.NewPathVariable("instance")),
									}},
								},
							},
							{
								Name: "SkipNoBinding",
							},
						},
					},
					{
						Name: "BackupService",
						Methods: []*api.Method{{
							Name: "ListBackups",
							PathInfo: &api.PathInfo{
								Bindings: []*api.PathBinding{{
									Verb: "GET",
									PathTemplate: (&api.PathTemplate{}).
										WithLiteral("backups"),
								}},
							},
						}},
					},
				},
			},
			want: []commandWithSubgroup{
				{
					Command: Command{
						Name:  "describe",
						Usage: "describe instances",
					},
					Subgroup: "instances",
				},
				{
					Command: Command{
						Name:  "list",
						Usage: "list backups",
					},
					Subgroup: "backups",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildCommands(test.model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupBySubgroup(t *testing.T) {
	for _, test := range []struct {
		name string
		cmds []commandWithSubgroup
		want []Subgroup
	}{
		{
			name: "single subgroup with one command",
			cmds: []commandWithSubgroup{{
				Command:  Command{Name: "describe"},
				Subgroup: "instances",
			}},
			want: []Subgroup{{
				Name:     "instances",
				Usage:    "Manage instances resources",
				Commands: []Command{{Name: "describe"}},
			}},
		},
		{
			name: "single subgroup with multiple commands",
			cmds: []commandWithSubgroup{
				{Command: Command{Name: "describe"}, Subgroup: "instances"},
				{Command: Command{Name: "list"}, Subgroup: "instances"},
			},
			want: []Subgroup{{
				Name:  "instances",
				Usage: "Manage instances resources",
				Commands: []Command{
					{Name: "describe"},
					{Name: "list"},
				},
			}},
		},
		{
			name: "subgroups sorted alphabetically",
			cmds: []commandWithSubgroup{
				{Command: Command{Name: "list"}, Subgroup: "instances"},
				{Command: Command{Name: "describe"}, Subgroup: "backups"},
				{Command: Command{Name: "list"}, Subgroup: "addresses"},
			},
			want: []Subgroup{
				{
					Name:     "addresses",
					Usage:    "Manage addresses resources",
					Commands: []Command{{Name: "list"}},
				},
				{
					Name:     "backups",
					Usage:    "Manage backups resources",
					Commands: []Command{{Name: "describe"}},
				},
				{
					Name:     "instances",
					Usage:    "Manage instances resources",
					Commands: []Command{{Name: "list"}},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := groupBySubgroup(test.cmds)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClientImports(t *testing.T) {
	withCall := commandWithSubgroup{
		Command: Command{
			Name:       "describe",
			ClientCall: &ClientCall{Method: "GetInstance"},
		},
		Subgroup: "instances",
	}
	withoutCall := commandWithSubgroup{
		Command:  Command{Name: "list"},
		Subgroup: "instances",
	}
	for _, test := range []struct {
		name string
		pkg  string
		cmds []commandWithSubgroup
		want []Import
	}{
		{
			name: "no commands",
			pkg:  "google.cloud.parallelstore.v1",
			cmds: nil,
			want: nil,
		},
		{
			name: "no client calls",
			pkg:  "google.cloud.parallelstore.v1",
			cmds: []commandWithSubgroup{withoutCall},
			want: nil,
		},
		{
			name: "client call with valid package",
			pkg:  "google.cloud.parallelstore.v1",
			cmds: []commandWithSubgroup{withoutCall, withCall},
			want: []Import{
				{
					Alias: "parallelstore",
					Path:  "cloud.google.com/go/parallelstore/apiv1",
				},
				{
					Path: "cloud.google.com/go/parallelstore/apiv1/parallelstorepb",
				},
			},
		},
		{
			name: "client call with unsupported package",
			pkg:  "google.cloud.parallelstore.v1beta1",
			cmds: []commandWithSubgroup{withCall},
			want: nil,
		},
		{
			name: "client call with non-google-cloud package",
			pkg:  "google.api.X.v1",
			cmds: []commandWithSubgroup{withCall},
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := clientImports(test.pkg, test.cmds)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
