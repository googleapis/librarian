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
	keyField := &api.Field{Name: "key", ID: ".test.Request.key", Typez: api.STRING_TYPE}
	inputType := &api.Message{
		Name:   "Request",
		ID:     ".test.Request",
		Fields: []*api.Field{keyField},
	}
	outputType := &api.Message{
		Name: "Response",
		ID:   ".test.Response",
		Fields: []*api.Field{
			{Name: "value", ID: ".test.Request.value", Typez: api.STRING_TYPE},
		},
	}
	for _, test := range []struct {
		name   string
		method *api.Method
		want   *methodAnnotations
	}{
		{
			name: "GET request",
			method: &api.Method{
				Name:          "GetOperation",
				Documentation: "Gets a thing.\n\nTest multiple comment lines.\n",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "GET",
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithLiteral("operations"),
						},
					},
				},
			},
			want: &methodAnnotations{
				Name:       "getOperation",
				Path:       "/v1/operations",
				DocLines:   []string{"Gets a thing.", "", "Test multiple comment lines.", ""},
				HTTPMethod: "GET",
				HasBody:    false,
			},
		},
		{
			name: "POST request with body field",
			method: &api.Method{
				Name: "CreateKey",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithLiteral("keys"),
						},
					},
					BodyFieldPath: "key",
				},
			},
			want: &methodAnnotations{
				Name:           "createKey",
				Path:           "/v1/keys",
				HTTPMethod:     "POST",
				HasBody:        true,
				IsBodyWildcard: false,
				BodyField:      "key",
			},
		},
		{
			name: "POST request with wildcard body",
			method: &api.Method{
				Name: "UploadData",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:         "POST",
							PathTemplate: api.NewPathTemplate().WithLiteral("v1").WithLiteral("data"),
						},
					},
					BodyFieldPath: "*",
				},
			},
			want: &methodAnnotations{
				Name:           "uploadData",
				Path:           "/v1/data",
				HTTPMethod:     "POST",
				HasBody:        true,
				IsBodyWildcard: true,
			},
		},
		{
			name: "List request",
			method: &api.Method{
				Name:          "ListThings",
				Documentation: "Lists things.",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb:            "GET",
							PathTemplate:    api.NewPathTemplate().WithLiteral("v1").WithLiteral("things"),
							QueryParameters: map[string]bool{"key": true},
						},
					},
				},
			},
			want: &methodAnnotations{
				Name:        "listThings",
				Path:        "/v1/things",
				DocLines:    []string{"Lists things."},
				HTTPMethod:  "GET",
				HasBody:     false,
				QueryParams: []*api.Field{keyField},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.method.InputType = inputType
			test.method.InputTypeID = inputType.ID
			test.method.OutputType = outputType
			test.method.OutputTypeID = outputType.ID
			service := &api.Service{
				Name:    "TestService",
				ID:      ".test.TestService",
				Package: "test",
				Methods: []*api.Method{test.method},
			}
			model := api.NewTestAPI(nil, nil, []*api.Service{service})
			codec := newTestCodec(t, model, nil)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			got := test.method.Codec.(*methodAnnotations)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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
			inputType := &api.Message{
				Name: "Request",
				ID:   ".test.Request",
				Fields: []*api.Field{
					{Name: "key", ID: ".test.Request.key", Typez: api.STRING_TYPE},
				},
			}
			outputType := &api.Message{
				Name: "Response",
				ID:   ".test.Response",
				Fields: []*api.Field{
					{Name: "value", ID: ".test.Request.value", Typez: api.STRING_TYPE},
				},
			}
			method := &api.Method{
				Name:          test.methodName,
				Documentation: "Test documentation.",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
				InputTypeID:  inputType.ID,
				InputType:    inputType,
				OutputTypeID: outputType.ID,
				OutputType:   outputType,
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
				Name:       test.wantName,
				DocLines:   []string{"Test documentation."},
				Path:       "/",
				HTTPMethod: "GET",
			}

			if diff := cmp.Diff(want, method.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
