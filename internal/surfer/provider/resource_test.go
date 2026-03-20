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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGetPluralFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     "instances",
		},
		{
			name:     "Short",
			segments: parseResourcePattern("shelves/{shelf}"),
			want:     "shelves",
		},
		{
			name: "No Variable End",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
			},
			want: "",
		},
		{
			name:     "Empty",
			segments: nil,
			want:     "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetPluralFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetParentFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     []api.PathSegment
	}{
		{
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     parseResourcePattern("projects/{project}/locations/{location}"),
		},
		{
			name:     "Root",
			segments: parseResourcePattern("projects/{project}"),
			want:     []api.PathSegment{},
		},
		{
			name: "Too Short",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
			},
			want: nil,
		},
		{
			name: "Invalid Pattern (Ends in Literal)",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
			},
			want: nil,
		},
		{
			name:     "Empty",
			segments: nil,
			want:     nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetParentFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetSingularFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     "instance",
		},
		{
			name:     "Short",
			segments: parseResourcePattern("shelves/{shelf}"),
			want:     "shelf",
		},
		{
			name: "No Variable End",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project").WithMatch()),
				*api.NewPathSegment().WithLiteral("locations"),
			},
			want: "",
		},
		{
			name:     "Empty",
			segments: nil,
			want:     "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetSingularFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCollectionPathFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name:     "Standard",
			segments: parseResourcePattern("projects/{project}/locations/{location}/instances/{instance}"),
			want:     "projects.locations.instances",
		},
		{
			name:     "Short",
			segments: parseResourcePattern("shelves/{shelf}"),
			want:     "shelves",
		},
		{
			name:     "Root",
			segments: parseResourcePattern("projects/{project}"),
			want:     "projects",
		},
		{
			name:     "Mixed",
			segments: parseResourcePattern("organizations/{organization}/locations/{location}/clusters/{cluster}"),
			want:     "organizations.locations.clusters",
		},
		{
			name:     "Global",
			segments: parseResourcePattern("projects/{project}/global/networks/{network}"),
			want:     "projects.networks",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := GetCollectionPathFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractPathFromSegments(t *testing.T) {
	for _, test := range []struct {
		name     string
		segments []api.PathSegment
		want     string
	}{
		{
			name:     "Standard Regional",
			segments: parseResourcePattern("v1/projects/{project}/locations/{location}/instances/{instance}"),
			want:     "projects.locations.instances",
		},
		{
			name: "Complex Variable",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("v1"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch().WithLiteral("instances").WithMatch()),
			},
			want: "projects.locations.instances",
		},
		{
			name: "Trailing Literal (List)",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("v1"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch()),
				*api.NewPathSegment().WithLiteral("instances"),
			},
			want: "projects.locations.instances",
		},
		{
			name: "No Version",
			segments: []api.PathSegment{
				*api.NewPathSegment().WithLiteral("projects"),
				*api.NewPathSegment().WithVariable(api.NewPathVariable("project")),
			},
			want: "projects",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractPathFromSegments(test.segments)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceNameFromType(t *testing.T) {
	for _, test := range []struct {
		name    string
		typeStr string
		want    string
	}{
		{"Standard", "example.googleapis.com/Instance", "Instance"},
		{"No Slash", "Instance", "Instance"},
		{"Empty", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := getResourceNameFromType(test.typeStr)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// parseResourcePattern converts a resource pattern string into a
// []api.PathSegment slice for testing. It handles AIP resource patterns
// (e.g., "projects/{project}/locations/{location}"). Variables
// automatically get a single-segment wildcard match.
func parseResourcePattern(pattern string) []api.PathSegment {
	var segments []api.PathSegment
	for part := range strings.SplitSeq(pattern, "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := part[1 : len(part)-1]
			segments = append(segments, *api.NewPathSegment().WithVariable(api.NewPathVariable(name).WithMatch()))
		} else {
			segments = append(segments, *api.NewPathSegment().WithLiteral(part))
		}
	}
	return segments
}
