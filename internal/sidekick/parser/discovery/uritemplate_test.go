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

package discovery

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestParseUriTemplateSuccess(t *testing.T) {
	for _, test := range []struct {
		input string
		want  *api.PathTemplate
	}{
		{"locations/global/firewallPolicies", api.NewPathTemplate().
			WithLiteral("locations").
			WithLiteral("global").
			WithLiteral("firewallPolicies")},
		{"locations/global/operations/{operation}", api.NewPathTemplate().
			WithLiteral("locations").
			WithLiteral("global").
			WithLiteral("operations").
			WithVariableNamed("operation")},
		{"projects/{project}/zones/{zone}/{parentName}/reservationSubBlocks", api.NewPathTemplate().
			WithLiteral("projects").
			WithVariableNamed("project").
			WithLiteral("zones").
			WithVariableNamed("zone").
			WithVariableNamed("parentName").
			WithLiteral("reservationSubBlocks")},
		{"v1/{+parent}/externalAccountKeys", api.NewPathTemplate().
			WithLiteral("v1").
			WithVariable(api.NewPathVariable("parent").WithAllowReserved().WithMatch()).
			WithLiteral("externalAccountKeys")},
	} {
		got, err := ParseUriTemplate(test.input)
		if err != nil {
			t.Errorf("expected a successful parse with input=%s, err=%v", test.input, err)
			continue
		}
		if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("mismatch [%s] (-want, +got):\n%s", test.input, diff)
		}
	}
}

func TestParseUriTemplateError(t *testing.T) {
	for _, test := range []struct {
		input string
	}{
		{"a/b/c/"},
		{"a/b/c|"},
		{"a/b/{c}|"},
		{"a/b/{c}}/d"},
		{"a/b/{c}}"},
		{"a/b/{c}/"},
		{"{foo}}bar"},
	} {
		if got, err := ParseUriTemplate(test.input); err == nil {
			t.Errorf("expected a parsing error with input=%s, got=%v", test.input, got)
		}
	}
}

func TestParseExpression(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{"{abc}", "abc"},
		{"{Abc}", "Abc"},
		{"{abc012}", "abc012"},
		{"{abc_012}", "abc_012"},
		{"{abc_012}/foo/{bar}", "abc_012"},
	} {
		gotSegment, gotWidth, err := parseExpression(test.input)
		if err != nil {
			t.Errorf("expected a successful parse with input=%s, err=%v", test.input, err)
			continue
		}
		if diff := cmp.Diff(&api.PathSegment{Variable: api.NewPathVariable(test.want).WithMatch()}, gotSegment); diff != "" {
			t.Errorf("mismatch [%s] (-want, +got):\n%s", test.input, diff)
		}
		if len(test.want)+2 != gotWidth {
			t.Errorf("mismatch want=%d, got=%d", len(test.want), gotWidth)
		}
	}
}

func TestParseExpressionError(t *testing.T) {
	for _, input := range []string{
		"",
		"(a)",
		"{#a}",
		"{.a}", "{/a}", "{?a}", "{&a}",
		"{=a}", "{,a}", "{!a}", "{@a}", "{|a}",
		"{a,b}", "{_abc}", "{0abc}", "{ab", "{abc/}",
		"{+abc", "{+abc/}"} {
		if gotSegment, gotWidth, err := parseExpression(input); err == nil {
			t.Errorf("expected a parsing error with input=%s, gotSegment=%v, gotWidth=%v", input, gotSegment, gotWidth)
		}
	}
}

func TestParseLiteral(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{"abc/def", "abc"},
		{"abcde/f", "abcde"},
		{"abcdef", "abcdef"},
	} {
		gotSegment, gotWidth, err := parseLiteral(test.input)
		if err != nil {
			t.Errorf("expected a successful parse with input=%s, err=%v", test.input, err)
			continue
		}
		if diff := cmp.Diff(&api.PathSegment{Literal: &test.want}, gotSegment); diff != "" {
			t.Errorf("mismatch [%s] (-want, +got):\n%s", test.input, diff)
		}
		if len(test.want) != gotWidth {
			t.Errorf("mismatch want=%d, got=%d", len(test.want), gotWidth)
		}
	}
}

func TestParseLiteralError(t *testing.T) {
	for _, input := range []string{"", "^", "'", "/", "abc^"} {
		if gotSegment, gotWidth, err := parseLiteral(input); err == nil {
			t.Errorf("expected a parsing error with input=%s, gotSegment=%v, gotWidth=%v", input, gotSegment, gotWidth)
		}
	}
}
