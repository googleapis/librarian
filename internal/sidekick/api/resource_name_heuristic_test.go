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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildHeuristicVocabulary(t *testing.T) {
	for _, test := range []struct {
		name     string
		services []*Service
		want     map[string]bool
	}{
		{
			name: "from standard method path",
			services: []*Service{
				{
					Methods: []*Method{
						{
							Name:          "GetWidget",
							IsAIPStandard: true,
							PathInfo: &PathInfo{
								Bindings: []*PathBinding{
									{
										PathTemplate: NewPathTemplate().
											WithLiteral("users").WithVariableNamed("user").
											WithLiteral("widgets").WithVariableNamed("widget"),
									},
								},
							},
						},
					},
				},
			},
			want: map[string]bool{
				"projects":        true,
				"locations":       true,
				"folders":         true,
				"organizations":   true,
				"billingAccounts": true,
				"users":           true,
				"widgets":         true,
			},
		},
		{
			name: "includes standard CRUD methods",
			services: []*Service{
				{
					Methods: []*Method{
						{
							Name:          "CreateWidget",
							IsAIPStandard: true,
							PathInfo: &PathInfo{
								Bindings: []*PathBinding{
									{
										PathTemplate: NewPathTemplate().
											WithLiteral("internal").WithVariableNamed("id"),
									},
								},
							},
						},
					},
				},
			},
			want: map[string]bool{
				"projects":        true,
				"locations":       true,
				"folders":         true,
				"organizations":   true,
				"billingAccounts": true,
				"internal":        true,
			},
		},
		{
			name: "ignores custom action methods",
			services: []*Service{
				{
					Methods: []*Method{
						{
							Name: "StartWidget",
							PathInfo: &PathInfo{
								Bindings: []*PathBinding{
									{
										PathTemplate: NewPathTemplate().
											WithLiteral("internal").WithVariableNamed("id"),
									},
								},
							},
						},
					},
				},
			},
			want: map[string]bool{
				"projects":        true,
				"locations":       true,
				"folders":         true,
				"organizations":   true,
				"billingAccounts": true,
			},
		},
		{
			name: "from nested variable template",
			services: []*Service{
				{
					Methods: []*Method{
						{
							Name:          "GetInstance",
							IsAIPStandard: true,
							PathInfo: &PathInfo{
								Bindings: []*PathBinding{
									{
										PathTemplate: NewPathTemplate().
											WithLiteral("v1").
											WithVariable(&PathVariable{
												FieldPath: []string{"name"},
												Segments:  []string{"projects", SingleSegmentWildcard, "instances", MultiSegmentWildcard},
											}),
									},
								},
							},
						},
					},
				},
			},
			want: map[string]bool{
				"projects":        true,
				"locations":       true,
				"folders":         true,
				"organizations":   true,
				"billingAccounts": true,
			},
		},
		{
			name: "empty model",
			want: map[string]bool{
				"projects":        true,
				"locations":       true,
				"folders":         true,
				"organizations":   true,
				"billingAccounts": true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := &API{
				Services: test.services,
			}
			got := BuildHeuristicVocabulary(model)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("BuildHeuristicVocabulary() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
