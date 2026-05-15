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

func TestSubgroupName(t *testing.T) {
	for _, test := range []struct {
		name     string
		method   *api.Method
		wantName string
		wantOK   bool
	}{
		{
			name: "single literal segment",
			method: &api.Method{
				Name: "ListBackups",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb: "GET",
						PathTemplate: (&api.PathTemplate{}).
							WithLiteral("backups"),
					}},
				},
			},
			wantName: "backups",
			wantOK:   true,
		},
		{
			name: "multi-segment ending in literal",
			method: &api.Method{
				Name: "GetInstance",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb: "GET",
						PathTemplate: (&api.PathTemplate{}).
							WithLiteral("v1").
							WithVariable(api.NewPathVariable("project")).
							WithLiteral("instances"),
					}},
				},
			},
			wantName: "instances",
			wantOK:   true,
		},
		{
			name: "no literal segments",
			method: &api.Method{
				Name: "GetThing",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb: "GET",
						PathTemplate: (&api.PathTemplate{}).
							WithVariable(api.NewPathVariable("name")),
					}},
				},
			},
			wantName: "",
			wantOK:   false,
		},
		{
			name: "no primary binding",
			method: &api.Method{
				Name: "GetThing",
			},
			wantName: "",
			wantOK:   false,
		},
		{
			name: "camelCase literal kebab-cased",
			method: &api.Method{
				Name: "ListMyResources",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb: "GET",
						PathTemplate: (&api.PathTemplate{}).
							WithLiteral("v1").
							WithLiteral("myResources"),
					}},
				},
			},
			wantName: "my-resources",
			wantOK:   true,
		},
		{
			name: "snake_case literal kebab-cased",
			method: &api.Method{
				Name: "ListMyResources",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb: "GET",
						PathTemplate: (&api.PathTemplate{}).
							WithLiteral("v1").
							WithLiteral("my_resource"),
					}},
				},
			},
			wantName: "my-resource",
			wantOK:   true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotName, gotOK := subgroupName(test.method)
			if diff := cmp.Diff(test.wantName, gotName); diff != "" {
				t.Errorf("name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantOK, gotOK); diff != "" {
				t.Errorf("ok mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
