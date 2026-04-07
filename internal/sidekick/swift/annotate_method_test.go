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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateMethod(t *testing.T) {
	method := &api.Method{
		Name:          "CreateSecret",
		Documentation: "Creates a secret.",
	}
	service := &api.Service{
		Name:    "SecretManagerService",
		Methods: []*api.Method{method},
	}
	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	codec := newTestCodec(t, model, nil)

	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	want := &methodAnnotations{
		Name:     "createSecret",
		DocLines: []string{"Creates a secret."},
	}

	if diff := cmp.Diff(want, method.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateMethod_EscapedName(t *testing.T) {
	for _, test := range []struct {
		name       string
		methodName string
		wantName   string
	}{
		{"escaped func", "Func", "`func`"},
		{"escaped self", "Self", "self_"},
		{"escaped default", "Default", "`default`"},
	} {
		t.Run(test.name, func(t *testing.T) {
			method := &api.Method{
				Name:          test.methodName,
				Documentation: "Test documentation.",
			}
			service := &api.Service{
				Name:    "TestService",
				Methods: []*api.Method{method},
			}
			model := api.NewTestAPI(nil, nil, []*api.Service{service})
			codec := newTestCodec(t, model, nil)

			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			want := &methodAnnotations{
				Name:     test.wantName,
				DocLines: []string{"Test documentation."},
			}

			if diff := cmp.Diff(want, method.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
