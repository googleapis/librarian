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

package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestIsOperationsServiceMethod(t *testing.T) {
	tests := []struct {
		name   string
		method *api.Method
		want   bool
	}{
		{
			name: "Is Operations Method",
			method: &api.Method{
				SourceServiceID: ".google.longrunning.Operations",
			},
			want: true,
		},
		{
			name: "Is Regular Method",
			method: &api.Method{
				SourceServiceID: "google.cloud.test.v1.TestService",
			},
			want: false,
		},
		{
			name:   "Nil Service ID",
			method: &api.Method{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsOperationsServiceMethod(tt.method); got != tt.want {
				t.Errorf("IsOperationsServiceMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationMethodDocumentation(t *testing.T) {
	tests := []struct {
		name string
		op   string
		want string
	}{
		{
			name: "GetOperation",
			op:   GetOperation,
			want: "The name of the operation resource.",
		},
		{
			name: "CancelOperation",
			op:   CancelOperation,
			want: "The name of the operation resource to be cancelled.",
		},
		{
			name: "DeleteOperation",
			op:   DeleteOperation,
			want: "The name of the operation resource to be deleted.",
		},
		{
			name: "ListOperations",
			op:   ListOperations,
			want: "The name of the operation's parent resource.",
		},
		{
			name: "Unknown",
			op:   "UnknownOp",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := OperationMethodDocumentation(tt.op); got != tt.want {
				t.Errorf("OperationMethodDocumentation(%q) = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestInferOperationResource(t *testing.T) {
	mockModel := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "example.googleapis.com/Instance",
				Patterns: []api.ResourcePattern{
					parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
				},
			},
		},
	}

	mockMultitypeModel := &api.API{
		ResourceDefinitions: []*api.Resource{
			{
				Type: "example.googleapis.com/Instance",
				Patterns: []api.ResourcePattern{
					parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
					parseResourcePattern("organizations/{organization}/locations/{location}/instances/{instance}"),
				},
			},
		},
	}

	tests := []struct {
		name    string
		method  *api.Method
		want    *api.Resource
		wantErr bool
	}{
		{
			name: "Standard Path",
			method: &api.Method{
				Model: mockModel,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									*(&api.PathSegment{}).WithLiteral("v1"),
									*(&api.PathSegment{}).WithVariable(
										api.NewPathVariable("name").
											WithLiteral("projects").WithMatch().
											WithLiteral("locations").WithMatch().
											WithLiteral("operations").WithMatch(),
									),
								},
							},
						},
					},
				},
			},
			want: &api.Resource{
				Type:     "longrunning.googleapis.com/Operation",
				Singular: "operation",
				Plural:   "operations",
				Patterns: []api.ResourcePattern{
					parseResourcePattern("projects/{project}/locations/{location}/operations/{operation}"),
				},
			},
		},
		{
			name: "No Bindings",
			method: &api.Method{
				Model: mockModel,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{},
				},
			},
			want: nil,
		},
		{
			name: "Multiple Bindings (Multitype operations)",
			method: &api.Method{
				Model: mockMultitypeModel,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									*(&api.PathSegment{}).WithLiteral("v1"),
									*(&api.PathSegment{}).WithVariable(
										api.NewPathVariable("name").
											WithLiteral("projects").WithMatch().
											WithLiteral("locations").WithMatch().
											WithLiteral("operations").WithMatch(),
									),
								},
							},
						},
						{
							PathTemplate: &api.PathTemplate{
								Segments: []api.PathSegment{
									*(&api.PathSegment{}).WithLiteral("v1"),
									*(&api.PathSegment{}).WithVariable(
										api.NewPathVariable("name").
											WithLiteral("organizations").WithMatch().
											WithLiteral("locations").WithMatch().
											WithLiteral("operations").WithMatch(),
									),
								},
							},
						},
					},
				},
			},
			want: &api.Resource{
				Type:     "longrunning.googleapis.com/Operation",
				Singular: "operation",
				Plural:   "operations",
				Patterns: []api.ResourcePattern{
					parseResourcePattern("projects/{project}/locations/{location}/operations/{operation}"),
					parseResourcePattern("organizations/{organization}/locations/{location}/operations/{operation}"),
				},
			},
		},

		{
			name: "Path Info Nil",
			method: &api.Method{
				PathInfo: nil,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := inferOperationResource(tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("inferOperationResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("inferOperationResource() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
