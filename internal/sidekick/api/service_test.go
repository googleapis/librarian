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

func TestService(t *testing.T) {
	m1 := NewMethodBuilder().WithName("M1").Build()
	m2 := NewMethodBuilder().WithName("M2").Build()
	model := NewAPIBuilder().WithName("test").BuildSyntaxTree()

	b := NewServiceBuilder().
		WithName("BookService").
		WithID(".test.BookService").
		WithPackage("test").
		WithDocumentation("Service docs.").
		WithDefaultHost("test.googleapis.com").
		WithDeprecated(true).
		WithMethods(m1).
		AddMethod(m2).
		WithModel(model).
		WithQuickstartMethod(m1)

	got := b.Build()
	if got != b.Build() {
		t.Errorf("ServiceBuilder.Build() should preserve pointer identity")
	}

	got.SetAnnotations("svc-codec")

	want := &Service{
		name:             "BookService",
		id:               ".test.BookService",
		packagez:         "test",
		documentation:    "Service docs.",
		defaultHost:      "test.googleapis.com",
		deprecated:       true,
		methods:          []*Method{m1, m2},
		model:            model,
		quickstartMethod: m1,
		codec:            "svc-codec",
	}
	if diff := cmp.Diff(want, got, allowUnexported()); diff != "" {
		t.Errorf("Service mismatch (-want +got):\n%s", diff)
	}
	if got.Name() != want.name || got.ID() != want.id || got.Package() != want.packagez || got.Documentation() != want.documentation || got.DefaultHost() != want.defaultHost || !got.Deprecated() {
		t.Errorf("unexpected Service scalars: %+v", got)
	}
	if got.Model() != model || got.QuickstartMethod() != m1 || got.Codec() != "svc-codec" {
		t.Errorf("unexpected Service pointers: %+v", got)
	}
	if len(got.Methods()) != 2 || got.Methods()[0] != m1 || got.Methods()[1] != m2 {
		t.Errorf("Service slice inputs were not cloned")
	}

	got.Methods()[0] = nil
	if got.Methods()[0] != m1 {
		t.Errorf("Service.Methods() getter should return a defensive copy")
	}
}
