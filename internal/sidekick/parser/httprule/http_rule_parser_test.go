// Copyright 2024 Google LLC
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

package httprule

import (
	"testing"

	"cloud.google.com/go/sdk/internal/sidekick/api"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseSegments(t *testing.T) {
	for _, test := range []struct {
		path        string
		want        *api.PathTemplate
		explanation string
	}{
		{"/foo", api.NewPathTemplate().WithLiteral("foo"), ""},
		{"/foo/bar", api.NewPathTemplate().WithLiteral("foo").WithLiteral("bar"), ""},
		{"/v1/*/foo", nil, "matchers only exist within a variable segment"},
		{"/v1/**/foo", nil, "matchers only exist within a variable segment"},
		{"/foo:bar", api.NewPathTemplate().WithLiteral("foo").WithVerb("bar"), ""},
		{"/foo/{bar}", api.NewPathTemplate().WithLiteral("foo").WithVariableNamed("bar"), ""},
		{"/foo/{bar.baz}", api.NewPathTemplate().WithLiteral("foo").WithVariableNamed("bar", "baz"), ""},
		{"/foo/{bar=baz}", api.NewPathTemplate().WithLiteral("foo").WithVariable(
			api.NewPathVariable("bar").WithLiteral("baz")), ""},
		{"/foo/{bar=*}", api.NewPathTemplate().WithLiteral("foo").WithVariable(
			api.NewPathVariable("bar").WithMatch()), ""},
		{"/foo/{bar=*}/baz", api.NewPathTemplate().WithLiteral("foo").WithVariable(
			api.NewPathVariable("bar").WithMatch()).
			WithLiteral("baz"), ""},
		{"/foo/{bar=**}/baz:qux", api.NewPathTemplate().WithLiteral("foo").WithVariable(
			api.NewPathVariable("bar").WithMatchRecursive()).
			WithLiteral("baz").WithVerb("qux"), ""},
		{"/foo/{bar=baz/*/qux/*}", api.NewPathTemplate().WithLiteral("foo").WithVariable(
			api.NewPathVariable("bar").WithLiteral("baz").WithMatch().WithLiteral("qux").WithMatch()), ""},
		{"/foo/{bar}/{baz}/{qux}", api.NewPathTemplate().WithLiteral("foo").WithVariableNamed("bar").WithVariableNamed("baz").WithVariableNamed("qux"), ""},
		{"foo", nil, "path must start with slash"},
		{"/", nil, "path cannot end with slash"},
		{"/foo/", nil, "path cannot end with slash"},
		{"/foo/***/bar", nil, "wildcard literal cannot exceed two *, and * isn't allowed in a LITERAL"},
		{"/%0f", api.NewPathTemplate().WithLiteral("%0f"), ""},
		{"/%0z", nil, "bad percent encoding"},
		{"/foo//bar", nil, "segment is too short"},
		{"/foo/:", nil, "verb is too short"},
		{"/foo/{}/bar", nil, "var too short"},
		{"/foo/{a.}/bar", nil, "var identifier too short"},
		{"/foo/{.a}/bar", nil, "var identifier too short"},
		{"/foo/{a=}/bar", nil, "var value too short"},
		{"/foo/{9bar}", nil, "var identifier has bad first character"},
		{"/foo/{bar9}", api.NewPathTemplate().WithLiteral("foo").WithVariableNamed("bar9"), ""},
		{"/foo/{b&r}", nil, "var identifier has bad character"},
		{"/foo/:bar", nil, "verb cannot come after slash"},
		{"/foo:bar/baz", nil, "verb must be the last segment, and : isn't allowed in a LITERAL"},
		{":foo", nil, "verb cannot be the first segment"},
		{"/foo/{bar={baz}}", nil, "variables cannot be nested"},
	} {
		t.Run(test.path, func(t *testing.T) {
			got, err := ParseSegments(test.path)
			if test.want != nil {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if got == nil {
					t.Fatalf("expected path template for %s, got nil", test.path)
				}
				if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
					t.Fatalf("failed parsing path [%s] (-want, +got):\n%s", test.path, diff)
				}
			} else {
				if err == nil {
					t.Fatalf("ParseSegments(%s) succeeded, want error: %s", test.path, test.explanation)
				}
			}
		})
	}
}

func TestParsePathTemplateInvalidVerb(t *testing.T) {
	if got, err := parsePathTemplate("/foo/{bar}/baz:"); err == nil {
		t.Errorf("expected an error, got=%v", got)
	}
	if got, err := parsePathTemplate("/foo/{bar}/baz^verb"); err == nil {
		t.Errorf("expected an error, got=%v", got)
	}
}

func TestParsePathTemplateInvalidVarSegment(t *testing.T) {
	if got, err := parsePathTemplate("/foo/{}"); err == nil {
		t.Errorf("expected an error, got=%v", got)
	}
}

func TestParsePathTemplateInvalidVarSubsegment(t *testing.T) {
	if got, err := parsePathTemplate("/foo/{a="); err == nil {
		t.Errorf("expected an error, got=%v", got)
	}
	if got, err := parsePathTemplate("/foo/{a=/"); err == nil {
		t.Errorf("expected an error, got=%v", got)
	}
	if got, err := parsePathTemplate("/foo/{a=^"); err == nil {
		t.Errorf("expected an error, got=%v", got)
	}
}

func TestParseLiteral(t *testing.T) {
	if got, pos, err := parseLiteral("abc%2"); err == nil {
		t.Errorf("expected an error, got=%v, pos=%v", got, pos)
	}
}

func TestParseIdentifier(t *testing.T) {
	if got, pos, err := parseIdentifier(""); err == nil {
		t.Errorf("expected an error, got=%v, pos=%v", got, pos)
	}
	if got, pos, err := parseIdentifier("_"); err == nil {
		t.Errorf("expected an error, got=%v, pos=%v", got, pos)
	}
	got, pos, err := parseIdentifier("abc")
	if err != nil {
		t.Fatal(err)
	}
	if *got != "abc" || pos != 3 {
		t.Errorf("mismatch want=abc, got=%v, wantPos=3, gotPos=%d", got, pos)
	}

	got, pos, err = parseIdentifier("abc/def")
	if err != nil {
		t.Fatal(err)
	}
	if *got != "abc" || pos != 3 {
		t.Errorf("mismatch want=abc, got=%v, wantPos=3, gotPos=%d", got, pos)
	}
}

func TestParseResourcePattern(t *testing.T) {
	for _, test := range []struct {
		name        string
		pattern     string
		want        *api.PathTemplate
		expectErr   bool
		explanation string
	}{
		{
			"single wildcard",
			api.SingleSegmentWildcard,
			api.NewPathTemplate(),
			false,
			"should parse a single wildcard as a literal segment",
		},
		{
			"standard hierarchical pattern",
			"projects/{project}/serviceAccounts/{service_account}",
			api.NewPathTemplate().
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("serviceAccounts").
				WithVariableNamed("service_account"),
			false,
			"should parse a standard hierarchical resource pattern correctly",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseResourcePattern(test.pattern)
			if test.expectErr {
				if err == nil {
					t.Fatalf("ParseResourcePattern(%s) succeeded, want error: %s", test.pattern, test.explanation)
				}
			} else {
				if err != nil {
					t.Fatalf("ParseResourcePattern(%s) failed: %v", test.pattern, err)
				}
				if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
					t.Fatalf("failed parsing resource pattern [\"%s\"] (-want, +got):\n%s", test.pattern, diff)
				}
			}
		})
	}
}

func TestParseResourcePatternWithNonStandardSeparators(t *testing.T) {
	// TODO(https://github.com/googleapis/librarian/issues/3258): at this
	// moment, we don't care what the exact representation is for this
	// input. We just care that parsing does not error.
	testCases := []struct {
		name    string
		pattern string
	}{
		{
			name:    "tilde separator",
			pattern: "users/{user}/profile/blurbs/legacy/{legacy_user}~{blurb}",
		},
		{
			name:    "dot separator",
			pattern: "rooms/{room}/blurbs/legacy/{legacy_room}.{blurb}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseResourcePattern(tc.pattern)
			if err != nil {
				t.Fatalf("ParseResourcePattern(%q) failed; want no error, got %v", tc.pattern, err)
			}
		})
	}
}
