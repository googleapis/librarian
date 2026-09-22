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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateCustomResourcePaths(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func() *codec
		want  []*clientResourcePath
	}{
		{
			name: "custom resource definition with path template",
			setup: func() *codec {
				res := &api.Resource{
					Type:     "secretmanager.googleapis.com/Secret",
					Singular: "secret",
					Patterns: []api.ResourcePattern{
						{
							{Literal: "projects"},
							{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
							{Literal: "secrets"},
							{Variable: &api.PathVariable{FieldPath: []string{"secret"}}},
						},
					},
				}
				model := api.NewTestAPI(nil, nil, nil).
					WithPackageName("google.cloud.secretmanager.v1")
				model.ResourceDefinitions = []*api.Resource{res}
				return newTestCodec(t, model, nil)
			},
			want: []*clientResourcePath{
				{
					Name:              "secret",
					FunctionName:      "secret_path",
					ParseFunctionName: "parse_secret_path",
					Docstring:         "Returns a fully-qualified secret string.",
					ParseDocstring:    "Parses a secret path into its component segments.",
					Signature:         "project: str,secret: str,",
					HasSignature:      true,
					FormatPattern:     "projects/{project}/secrets/{secret}",
					FormatArgs:        "project=project, secret=secret, ",
					RegexPattern:      "^projects/(?P<project>.+?)/secrets/(?P<secret>.+?)$",
				},
			},
		},
		{
			name: "excludes common resources",
			setup: func() *codec {
				projectRes := &api.Resource{
					Type:     "cloudresourcemanager.googleapis.com/Project",
					Singular: "project",
					Patterns: []api.ResourcePattern{
						{
							{Literal: "projects"},
							{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
						},
					},
				}
				model := api.NewTestAPI(nil, nil, nil).
					WithPackageName("google.example.v1")
				model.ResourceDefinitions = []*api.Resource{projectRes}
				return newTestCodec(t, model, nil)
			},
			want: nil,
		},
		{
			name: "wildcard pattern without variables",
			setup: func() *codec {
				res := &api.Resource{
					Type:     "example.googleapis.com/WildcardResource",
					Singular: "wildcard_resource",
					Patterns: []api.ResourcePattern{
						{
							{Literal: "*"},
						},
					},
				}
				model := api.NewTestAPI(nil, nil, nil).
					WithPackageName("google.example.v1")
				model.ResourceDefinitions = []*api.Resource{res}
				return newTestCodec(t, model, nil)
			},
			want: []*clientResourcePath{
				{
					Name:              "wildcard_resource",
					FunctionName:      "wildcard_resource_path",
					ParseFunctionName: "parse_wildcard_resource_path",
					Docstring:         "Returns a fully-qualified wildcard_resource string.",
					ParseDocstring:    "Parses a wildcard_resource path into its component segments.",
					Signature:         "",
					HasSignature:      false,
					FormatPattern:     "*",
					FormatArgs:        "",
					RegexPattern:      "^.*$",
				},
			},
		},
		{
			name: "camelCase singular converts to snake_case function name",
			setup: func() *codec {
				res := api.NewTestResource("secretmanager.googleapis.com/SecretVersion").
					WithSingular("secretVersion").
					WithPatterns(api.ResourcePattern{
						{Literal: "projects"},
						{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
						{Literal: "secrets"},
						{Variable: &api.PathVariable{FieldPath: []string{"secret"}}},
						{Literal: "versions"},
						{Variable: &api.PathVariable{FieldPath: []string{"secret_version"}}},
					})
				model := api.NewTestAPI(nil, nil, nil).
					WithPackageName("google.cloud.secretmanager.v1")
				model.ResourceDefinitions = []*api.Resource{res}
				return newTestCodec(t, model, nil)
			},
			want: []*clientResourcePath{
				{
					Name:              "secret_version",
					FunctionName:      "secret_version_path",
					ParseFunctionName: "parse_secret_version_path",
					Docstring:         "Returns a fully-qualified secret_version string.",
					ParseDocstring:    "Parses a secret_version path into its component segments.",
					Signature:         "project: str,secret: str,secret_version: str,",
					HasSignature:      true,
					FormatPattern:     "projects/{project}/secrets/{secret}/versions/{secret_version}",
					FormatArgs:        "project=project, secret=secret, secret_version=secret_version, ",
					RegexPattern:      "^projects/(?P<project>.+?)/secrets/(?P<secret>.+?)/versions/(?P<secret_version>.+?)$",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := test.setup()
			got := c.annotateCustomResourcePaths(nil)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateCustomResourcePaths_ServiceScoped(t *testing.T) {
	res1 := &api.Resource{
		Type:     "example.googleapis.com/Included",
		Singular: "included",
		Patterns: []api.ResourcePattern{
			{
				{Literal: "projects"},
				{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
				{Literal: "includedItems"},
				{Variable: &api.PathVariable{FieldPath: []string{"included"}}},
			},
		},
	}
	res2 := &api.Resource{
		Type:     "example.googleapis.com/Excluded",
		Singular: "excluded",
		Patterns: []api.ResourcePattern{
			{
				{Literal: "projects"},
				{Variable: &api.PathVariable{FieldPath: []string{"project"}}},
				{Literal: "excludedItems"},
				{Variable: &api.PathVariable{FieldPath: []string{"excluded"}}},
			},
		},
	}
	msg1 := api.NewTestMessage("IncludedMsg").WithPackage("example.v1").WithResource(res1)
	res1.Self = msg1
	msg2 := api.NewTestMessage("ExcludedMsg").WithPackage("example.v1").WithResource(res2)
	res2.Self = msg2

	method := api.NewTestMethod("DoIncluded").WithInput(msg1).WithOutput(msg1)
	svc := api.NewTestService("IncludedService").WithPackage("example.v1").WithMethods(method)

	model := api.NewTestAPI([]*api.Message{msg1, msg2}, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.example.v1")
	c := newTestCodec(t, model, nil)

	got := c.annotateCustomResourcePaths(svc)
	if len(got) != 1 {
		t.Fatalf("expected 1 custom path, got %d", len(got))
	}
	if got[0].FunctionName != "included_path" {
		t.Errorf("expected included_path, got %s", got[0].FunctionName)
	}
}
