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
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
)

func TestMakeServiceMethods(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := "..zones.get"
	got, ok := model.State.MethodByID[id]
	if !ok {
		t.Fatalf("expected method %s in the API model", id)
	}
	want := &api.Method{
		ID:            "..zones.get",
		Name:          "get",
		Documentation: "Returns the specified Zone resource.",
		InputTypeID:   ".google.protobuf.Empty",
		OutputTypeID:  "..Zone",
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb: "GET",
					PathTemplate: api.NewPathTemplate().
						WithLiteral("compute").
						WithLiteral("v1").
						WithLiteral("projects").
						WithVariableNamed("project").
						WithLiteral("zones").
						WithVariableNamed("zone"),
					QueryParameters: map[string]bool{},
				},
			},
			BodyFieldPath: "*",
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestMakeServiceMethodsError(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := document{}
	input := &resource{
		Name: "testResource",
		Methods: []*method{
			{
				Name:        "upload",
				MediaUpload: &mediaUpload{},
			},
		},
	}
	if methods, err := makeServiceMethods(model, "..testResource", &doc, input); err == nil {
		t.Errorf("expected error on method with media upload, got=%v", methods)
	}
}

func TestMakeMethodError(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		Name  string
		Input method
	}{
		{"mediaUploadMustBeNil", method{MediaUpload: &mediaUpload{}}},
		{"requestMustHaveRef", method{Request: &schema{}}},
		{"responseMustHaveRef", method{Response: &schema{}}},
		{"badPath", method{Path: "{+var"}},
	} {
		doc := document{}
		if method, err := makeMethod(model, "..Test", &doc, &test.Input); err == nil {
			t.Errorf("expected error on method[%s], got=%v", test.Name, method)
		}
	}

}
