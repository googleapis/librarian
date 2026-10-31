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

func TestPathBinding(t *testing.T) {
	pt := NewPathTemplateBuilder().WithLiteral("v1").Build()
	tr := &TargetResource{
		FieldPaths: [][]string{{"parent"}},
		Template:   []PathSegment{*NewPathSegmentBuilder().WithLiteral("items").Build()},
	}
	got := NewPathBindingBuilder().
		WithQueryParameter("initial").
		WithQueryParameterMap(map[string]bool{"overwritten": true}).
		WithVerb("POST").
		WithPathTemplate(pt).
		WithQueryParameters("filter").
		WithQueryParameter("extra").
		WithTargetResource(tr).
		Build()
	got.SetAnnotations("my-codec")

	want := &PathBinding{
		verb:            "POST",
		pathTemplate:    pt,
		queryParameters: map[string]bool{"filter": true, "extra": true},
		targetResource:  tr,
		codec:           "my-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got.Verb() != want.verb {
		t.Errorf("Verb() = %q, want %q", got.Verb(), want.verb)
	}
	if diff := cmp.Diff(want.pathTemplate, got.PathTemplate(), allowUnexported()); diff != "" {
		t.Errorf("PathTemplate() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.queryParameters, got.QueryParameters()); diff != "" {
		t.Errorf("QueryParameters() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.targetResource, got.TargetResource(), allowUnexported()); diff != "" {
		t.Errorf("TargetResource() mismatch (-want +got):\n%s", diff)
	}
	if got.Codec() != want.codec {
		t.Errorf("Codec() = %v, want %v", got.Codec(), want.codec)
	}
}
