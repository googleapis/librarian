// Copyright 2025 Google LLC
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

package parser

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestExamples(t *testing.T) {
	requireProtoc(t)
	for _, test := range []struct {
		methodID string
		want     []*api.RoutingInfo
	}{
		{
			".test.TestService.Example1",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("app_profile_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("app_profile_id").
							WithMatchingSegments("**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example2",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("routing_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("app_profile_id").
							WithMatchingSegments("**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example3a",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("table_name").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*", "instances", "*", "**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example3b",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("table_name").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("regions", "*", "zones", "*", "**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example3c",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("table_name").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*", "instances", "*", "**").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("regions", "*", "zones", "*", "**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example4",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("routing_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example5",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("routing_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*", "instances", "*").
							WithSuffixSegments("**").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example6a",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("instance_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithPrefixSegments("projects", "*").
							WithMatchingSegments("instances", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
				api.NewRoutingInfoBuilder().
					WithName("project_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*").
							WithSuffixSegments("instances", "*", "**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example6b",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("instance_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithPrefixSegments("projects", "*").
							WithMatchingSegments("instances", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
				api.NewRoutingInfoBuilder().
					WithName("project_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example7",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("project_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
				api.NewRoutingInfoBuilder().
					WithName("routing_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("app_profile_id").
							WithMatchingSegments("**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example8",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("routing_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("app_profile_id").
							WithMatchingSegments("**").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("regions", "*").
							WithSuffixSegments("**").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
			},
		},
		{
			".test.TestService.Example9",
			[]*api.RoutingInfo{
				api.NewRoutingInfoBuilder().
					WithName("routing_id").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("app_profile_id").
							WithPrefixSegments("profiles").
							WithMatchingSegments("*").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("app_profile_id").
							WithMatchingSegments("**").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("projects", "*").
							WithSuffixSegments("**").
							Build(),
					).
					Build(),
				api.NewRoutingInfoBuilder().
					WithName("table_location").
					WithVariants(
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithMatchingSegments("regions", "*", "zones", "*").
							WithSuffixSegments("tables", "*").
							Build(),
						api.NewRoutingInfoVariantBuilder().
							WithFieldPath("table_name").
							WithPrefixSegments("projects", "*").
							WithMatchingSegments("instances", "*").
							WithSuffixSegments("tables", "*").
							Build(),
					).
					Build(),
			},
		},
	} {
		api, err := makeAPIForProtobuf(nil, newTestCodeGeneratorRequest(t, "routing_info.proto"))
		if err != nil {
			t.Fatalf("Failed to make API for Protobuf %v", err)
		}
		t.Run(test.methodID, func(t *testing.T) {
			got := api.Method(test.methodID)
			if got == nil {
				t.Fatalf("Cannot find method %s in API State", test.methodID)
			}
			if diff := cmp.Diff(test.want, got.Routing(), apitest.AllowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParsePathTemplateSuccess(t *testing.T) {
	for _, test := range []struct {
		fieldPath string
		path      string
		want      *api.RoutingInfo
	}{
		{
			"default",
			"",
			api.NewRoutingInfoBuilder().
				WithName("default").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("default").
						WithMatchingSegments("**").
						Build(),
				).
				Build(),
		},
		{
			// AIP-4222: An empty google.api.routing annotation is acceptable.
			// It means that no routing headers should be generated for the RPC,
			// when they otherwise would be e.g. implicitly from the
			// google.api.http annotation.
			"",
			"",
			api.NewRoutingInfoBuilder().
				WithName("").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath([]string{}...).
						WithMatchingSegments([]string{}...).
						Build(),
				).
				Build(),
		},
		{
			// AIP-4222: It is acceptable to omit the pattern in the resource ID
			// segment, `{parent}` for example, is equivalent to `{parent=*}`.
			"parent",
			"projects/{parent}",
			api.NewRoutingInfoBuilder().
				WithName("parent").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("parent").
						WithPrefixSegments("projects").
						WithMatchingSegments("*").
						Build(),
				).
				Build(),
		},
		{
			// AIP-4222: It is acceptable to omit the path_template field
			// altogether. An omitted path_template is equivalent to a
			// path_template with the same resource ID name as the field and
			// the pattern `**`.
			"parent",
			"",
			api.NewRoutingInfoBuilder().
				WithName("parent").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("parent").
						WithMatchingSegments("**").
						Build(),
				).
				Build(),
		},
		{
			"field.child",
			"",
			api.NewRoutingInfoBuilder().
				WithName("field.child").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("field", "child").
						WithMatchingSegments("**").
						Build(),
				).
				Build(),
		},
		{
			"default",
			"{**}",
			api.NewRoutingInfoBuilder().
				WithName("default").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("default").
						WithMatchingSegments("**").
						Build(),
				).
				Build(),
		},
		{
			"default",
			"{routing=**}",
			api.NewRoutingInfoBuilder().
				WithName("routing").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("default").
						WithMatchingSegments("**").
						Build(),
				).
				Build(),
		},
		{
			"default",
			"{routing=a/*/b/**}",
			api.NewRoutingInfoBuilder().
				WithName("routing").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("default").
						WithMatchingSegments("a", "*", "b", "**").
						Build(),
				).
				Build(),
		},
		{
			"default",
			"p/*/q/*/{routing=a/*/b/**}",
			api.NewRoutingInfoBuilder().
				WithName("routing").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("default").
						WithMatchingSegments("a", "*", "b", "**").
						WithPrefixSegments("p", "*", "q", "*").
						Build(),
				).
				Build(),
		},
		{
			"default",
			"p/*/q/*/{routing=a/*/b/*}/s/*/u/*/v/**",
			api.NewRoutingInfoBuilder().
				WithName("routing").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("default").
						WithMatchingSegments("a", "*", "b", "*").
						WithPrefixSegments("p", "*", "q", "*").
						WithSuffixSegments("s", "*", "u", "*", "v", "**").
						Build(),
				).
				Build(),
		},
		{
			"default",
			"p/*/q/*/{routing=a/*/b/**}/s/*/u/*/v/*",
			api.NewRoutingInfoBuilder().
				WithName("routing").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("default").
						WithMatchingSegments("a", "*", "b", "**").
						WithPrefixSegments("p", "*", "q", "*").
						WithSuffixSegments("s", "*", "u", "*", "v", "*").
						Build(),
				).
				Build(),
		},
		{
			"field.sub_field.child",
			"p/*/q/*/{routing=a/*/b/*}/s/*/u/*/v/**",
			api.NewRoutingInfoBuilder().
				WithName("routing").
				WithVariants(
					api.NewRoutingInfoVariantBuilder().
						WithFieldPath("field", "sub_field", "child").
						WithMatchingSegments("a", "*", "b", "*").
						WithPrefixSegments("p", "*", "q", "*").
						WithSuffixSegments("s", "*", "u", "*", "v", "**").
						Build(),
				).
				Build(),
		},
	} {
		t.Run(fmt.Sprintf("%s:%s", test.fieldPath, test.path), func(t *testing.T) {
			got, err := parseRoutingPathTemplate(test.fieldPath, test.path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, apitest.AllowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParsePathTemplateFailures(t *testing.T) {
	tests := []string{
		"projects/*",
		"projects/*/{routing_id=**}/**",
		"projects/*}",
		"projects/*/}",
		"projects/*/{",
		"projects/*/{{",
		"projects/*/{a/b/c=**}",
		"projects/*/{routing_id=**}foo",
		// AIP-4222: A multi-segment wildcard must only appear as the final
		// segment or make up the entire path_template.
		"projects/**/{a}",
		"projects/**/b/{a}",
		"projects/*/{**/a}",
		"projects/*/{a/**/b}",
		"projects/*/{a/**/b/*}",
		"projects/*/{a}/**/b",
		"projects/*/{a}/*/b/**/c",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			got, err := parseRoutingPathTemplate("default", path)
			if err == nil {
				t.Errorf("expected error for %q, got=%v", path, got)
			}
		})
	}
}

func TestParseVariableSuccess(t *testing.T) {
	for _, test := range []struct {
		path        string
		wantName    string
		wantSpec    api.RoutingPathSpec
		wantTrailer string
	}{
		{"**", "default", api.NewRoutingPathSpec("**"), ""},
		{"routing=**", "routing", api.NewRoutingPathSpec("**"), ""},
		{"routing=a/*/b/**", "routing", api.NewRoutingPathSpec("a", "*", "b", "**"), ""},
		{"routing=a/*/b/**}", "routing", api.NewRoutingPathSpec("a", "*", "b", "**"), "}"},
		{"routing=a/*/b/**}/c/*", "routing", api.NewRoutingPathSpec("a", "*", "b", "**"), "}/c/*"},
	} {
		t.Run(test.path, func(t *testing.T) {
			gotName, gotSpec, width, err := parseRoutingVariable("default", test.path)
			if err != nil {
				t.Fatal(err)
			}
			if gotName != test.wantName {
				t.Errorf("mismatched variable names, want=%s, got=%s", test.wantName, gotName)
			}
			if diff := cmp.Diff(test.wantSpec, gotSpec, apitest.AllowUnexported()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if test.path[width:] != test.wantTrailer {
				t.Errorf("trailer segment mismatch, want=%s, got=%s", test.wantTrailer, test.path[width:])
			}
		})
	}
}

func TestParseRoutingVariableError(t *testing.T) {
	tests := []string{"=**", "a/b=**"}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			gotName, gotSpec, _, err := parseRoutingVariable("default", path)
			if err == nil {
				t.Errorf("expected error for %q, gotName=%s, gotSpec=%v", path, gotName, gotSpec)
			}
		})
	}
}

func TestParseRoutingPathSpecSuccess(t *testing.T) {
	for _, test := range []struct {
		path         string
		wantTrailer  string
		wantSegments []string
	}{
		{"**", "", []string{"**"}},
		{"a/b/c", "", []string{"a", "b", "c"}},
		{"a/*/b/*/c/**", "", []string{"a", "*", "b", "*", "c", "**"}},
		{"a/*/b/*}/c/**", "}/c/**", []string{"a", "*", "b", "*"}},
		{"a=b/*/c/*", "=b/*/c/*", []string{"a"}},
		{"a/b=b/*/c/*", "=b/*/c/*", []string{"a", "b"}},
	} {
		t.Run(test.path, func(t *testing.T) {
			got, width := parseRoutingPathSpec(test.path)
			if diff := cmp.Diff(test.wantSegments, got.Segments()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if test.path[width:] != test.wantTrailer {
				t.Errorf("trailer segment mismatch, want=%s, got=%s", test.wantTrailer, test.path[width:])
			}
		})
	}
}
