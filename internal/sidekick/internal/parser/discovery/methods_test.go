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

import "testing"

func TestMakeServiceMethodsError(t *testing.T) {
	model, err := PublicCaDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	input := &resource{
		Name: "testResource",
		Methods: []*method{
			{
				Name:        "upload",
				MediaUpload: &mediaUpload{},
			},
		},
	}
	if methods, err := makeServiceMethods(model, "..testResource", input); err == nil {
		t.Errorf("expected error on method with media upload, got=%v", methods)
	}
}

func TestMakeMethodError(t *testing.T) {
	model, err := PublicCaDisco(t, nil)
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
	} {
		if method, err := makeMethod(model, "..Test", &test.Input); err == nil {
			t.Errorf("expected error on method[%s], got=%v", test.Name, method)
		}
	}

}
