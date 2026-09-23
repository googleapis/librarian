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

	"github.com/cbroglie/mustache"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestFormatPathTemplateURI(t *testing.T) {
	for _, test := range []struct {
		name     string
		template *api.PathTemplate
		want     string
	}{
		{name: "nil template", template: nil, want: ""},
		{name: "empty template", template: &api.PathTemplate{}, want: ""},
		{
			name:     "literals only",
			template: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("messages"),
			want:     "/v1/messages",
		},
		{
			name: "variable with pattern wildcard",
			template: (&api.PathTemplate{}).WithLiteral("v1").
				WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("serviceAccounts").WithMatch()).
				WithVerb("generateAccessToken"),
			want: "/v1/{name=projects/*/serviceAccounts/*}:generateAccessToken",
		},
		{
			name: "variable with recursive wildcard",
			template: (&api.PathTemplate{}).WithLiteral("v1").
				WithVariable(api.NewPathVariable("name").WithLiteral("projects").WithMatch().WithLiteral("locations").WithMatch().WithLiteral("instances").WithMatchRecursive()),
			want: "/v1/{name=projects/*/locations/*/instances/**}",
		},
		{
			name: "variable without segments",
			template: (&api.PathTemplate{}).WithLiteral("v1").
				WithVariable(api.NewPathVariable("project")).
				WithLiteral("instances"),
			want: "/v1/{project}/instances",
		},
		{
			name: "nested field in variable",
			template: (&api.PathTemplate{}).WithLiteral("v1").
				WithVariable(api.NewPathVariable("analysis_query", "scope").WithMatch().WithMatch()).
				WithVerb("analyzeIamPolicy"),
			want: "/v1/{analysis_query.scope=*/*}:analyzeIamPolicy",
		},
		{
			name:     "custom verb with literals",
			template: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations").WithVerb("cancel"),
			want:     "/v1/operations:cancel",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatPathTemplateURI(test.template)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateRestMethod_HTTPOptions(t *testing.T) {
	namePt := func() *api.PathTemplate {
		return (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("name").WithMatch())
	}
	parentPt := (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithMatch()).WithLiteral("instances")
	nestedPt := (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("instance", "name").WithMatch())

	for _, test := range []struct {
		name   string
		method *api.Method
		want   *restMethodAnnotation
	}{
		{name: "nil method", method: nil, want: nil},
		{
			name:   "GET method with no body",
			method: makeTestRestMethod("GetInstance", "GET", "", namePt(), nil),
			want: &restMethodAnnotation{
				Name:          "get_instance",
				BaseClassName: "_BaseGetInstance",
				HTTPOptions: []*httpOptionAnnotation{
					{Method: "get", URI: "/v1/{name}", Body: "", HasBody: false, First: true},
				},
			},
		},
		{
			name: "POST method with star body",
			method: func() *api.Method {
				m := makeTestRestMethod("GenerateAccessToken", "POST", "*", namePt(), nil)
				m.PathInfo.Bindings[0].PathTemplate.Verb = "generateAccessToken"
				return m
			}(),
			want: &restMethodAnnotation{
				Name:          "generate_access_token",
				BaseClassName: "_BaseGenerateAccessToken",
				HTTPOptions: []*httpOptionAnnotation{
					{Method: "post", URI: "/v1/{name}:generateAccessToken", Body: "*", HasBody: true, First: true},
				},
			},
		},
		{
			name:   "POST method with field body",
			method: makeTestRestMethod("CreateInstance", "POST", "instance", parentPt, nil),
			want: &restMethodAnnotation{
				Name:          "create_instance",
				BaseClassName: "_BaseCreateInstance",
				HTTPOptions: []*httpOptionAnnotation{
					{Method: "post", URI: "/v1/{parent}/instances", Body: "instance", HasBody: true, First: true},
				},
			},
		},
		{
			name:   "DELETE method",
			method: makeTestRestMethod("DeleteInstance", "DELETE", "", namePt(), nil),
			want: &restMethodAnnotation{
				Name:          "delete_instance",
				BaseClassName: "_BaseDeleteInstance",
				HTTPOptions: []*httpOptionAnnotation{
					{Method: "delete", URI: "/v1/{name}", Body: "", HasBody: false, First: true},
				},
			},
		},
		{
			name:   "PATCH method with field body",
			method: makeTestRestMethod("UpdateInstance", "PATCH", "instance", nestedPt, nil),
			want: &restMethodAnnotation{
				Name:          "update_instance",
				BaseClassName: "_BaseUpdateInstance",
				HTTPOptions: []*httpOptionAnnotation{
					{Method: "patch", URI: "/v1/{instance.name}", Body: "instance", HasBody: true, First: true},
				},
			},
		},
		{
			name:   "PUT method with star body",
			method: makeTestRestMethod("ReplaceInstance", "PUT", "*", namePt(), nil),
			want: &restMethodAnnotation{
				Name:          "replace_instance",
				BaseClassName: "_BaseReplaceInstance",
				HTTPOptions: []*httpOptionAnnotation{
					{Method: "put", URI: "/v1/{name}", Body: "*", HasBody: true, First: true},
				},
			},
		},
		{
			name: "multiple bindings",
			method: func() *api.Method {
				m := makeTestRestMethod("CustomMethod", "POST", "*", namePt(), nil)
				m.PathInfo.Bindings[0].PathTemplate.Verb = "custom"
				m.PathInfo.Bindings = append(m.PathInfo.Bindings, &api.PathBinding{
					Verb:         "GET",
					PathTemplate: namePt(),
				})
				return m
			}(),
			want: &restMethodAnnotation{
				Name:          "custom_method",
				BaseClassName: "_BaseCustomMethod",
				HTTPOptions: []*httpOptionAnnotation{
					{Method: "post", URI: "/v1/{name}:custom", Body: "*", HasBody: true, First: true},
					{Method: "get", URI: "/v1/{name}", Body: "", HasBody: false, First: false},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := annotateRestMethod(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateRestMethod_RequiredFields(t *testing.T) {
	v1Pt := (&api.PathTemplate{}).WithLiteral("v1")

	for _, test := range []struct {
		name   string
		method *api.Method
		want   *restMethodAnnotation
	}{
		{
			name: "scalar required field included",
			method: func() *api.Method {
				f := api.NewTestField("instance_id").WithType(api.TypezString).WithBehavior(api.FieldBehaviorRequired)
				f.JSONName = "instanceId"
				req := api.NewTestMessage("CreateReq").WithFields(f)
				return makeTestRestMethod("CreateInstance", "POST", "", v1Pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:                        "create_instance",
				BaseClassName:               "_BaseCreateInstance",
				HTTPOptions:                 []*httpOptionAnnotation{{Method: "post", URI: "/v1", Body: "", HasBody: false, First: true}},
				HasRequiredFields:           true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{{Key: "instanceId", Value: `""`, Last: true}},
			},
		},
		{
			name: "message required field included",
			method: func() *api.Method {
				msg := api.NewTestMessage("Query")
				f := api.NewTestField("analysis_query").WithMessageType(msg).WithBehavior(api.FieldBehaviorRequired)
				f.JSONName = "analysisQuery"
				req := api.NewTestMessage("AnalyzeReq").WithFields(f)
				return makeTestRestMethod("Analyze", "GET", "", v1Pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:                        "analyze",
				BaseClassName:               "_BaseAnalyze",
				HTTPOptions:                 []*httpOptionAnnotation{{Method: "get", URI: "/v1", Body: "", HasBody: false, First: true}},
				HasRequiredFields:           true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{{Key: "analysisQuery", Value: "{}", Last: true}},
			},
		},
		{
			name: "enum required field included as empty dict",
			method: func() *api.Method {
				f := api.NewTestField("view").WithType(api.TypezEnum).WithBehavior(api.FieldBehaviorRequired)
				req := api.NewTestMessage("GetReq").WithFields(f)
				return makeTestRestMethod("GetItem", "GET", "", v1Pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:                        "get_item",
				BaseClassName:               "_BaseGetItem",
				HTTPOptions:                 []*httpOptionAnnotation{{Method: "get", URI: "/v1", Body: "", HasBody: false, First: true}},
				HasRequiredFields:           true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{{Key: "view", Value: "{}", Last: true}},
			},
		},
		{
			name: "path parameter excluded",
			method: func() *api.Method {
				f := api.NewTestField("parent").WithType(api.TypezString).WithBehavior(api.FieldBehaviorRequired)
				req := api.NewTestMessage("ListReq").WithFields(f)
				pt := (&api.PathTemplate{}).WithLiteral("v1").WithVariable(api.NewPathVariable("parent").WithMatch())
				return makeTestRestMethod("ListInstances", "GET", "", pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:              "list_instances",
				BaseClassName:     "_BaseListInstances",
				HTTPOptions:       []*httpOptionAnnotation{{Method: "get", URI: "/v1/{parent}", Body: "", HasBody: false, First: true}},
				HasRequiredFields: true,
			},
		},
		{
			name: "star body excludes all required fields",
			method: func() *api.Method {
				f := api.NewTestField("name").WithType(api.TypezString).WithBehavior(api.FieldBehaviorRequired)
				req := api.NewTestMessage("Req").WithFields(f)
				return makeTestRestMethod("DoSomething", "POST", "*", v1Pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:              "do_something",
				BaseClassName:     "_BaseDoSomething",
				HTTPOptions:       []*httpOptionAnnotation{{Method: "post", URI: "/v1", Body: "*", HasBody: true, First: true}},
				HasRequiredFields: true,
			},
		},
		{
			name: "body field excluded",
			method: func() *api.Method {
				inst := api.NewTestMessage("Instance")
				bf := api.NewTestField("instance").WithMessageType(inst).WithBehavior(api.FieldBehaviorRequired)
				rf := api.NewTestField("instance_id").WithType(api.TypezString).WithBehavior(api.FieldBehaviorRequired)
				rf.JSONName = "instanceId"
				req := api.NewTestMessage("CreateReq").WithFields(bf, rf)
				return makeTestRestMethod("CreateInstance", "POST", "instance", v1Pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:                        "create_instance",
				BaseClassName:               "_BaseCreateInstance",
				HTTPOptions:                 []*httpOptionAnnotation{{Method: "post", URI: "/v1", Body: "instance", HasBody: true, First: true}},
				HasRequiredFields:           true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{{Key: "instanceId", Value: `""`, Last: true}},
			},
		},
		{
			name: "nested path parameter does not exclude top level message field",
			method: func() *api.Method {
				q := api.NewTestMessage("AnalysisQuery")
				qf := api.NewTestField("analysis_query").WithMessageType(q).WithBehavior(api.FieldBehaviorRequired)
				qf.JSONName = "analysisQuery"
				req := api.NewTestMessage("AnalyzeReq").WithFields(qf)
				pt := (&api.PathTemplate{}).WithLiteral("v1").
					WithVariable(api.NewPathVariable("analysis_query", "scope").WithMatch()).
					WithVerb("analyzeIamPolicy")
				return makeTestRestMethod("AnalyzeIamPolicy", "GET", "", pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:                        "analyze_iam_policy",
				BaseClassName:               "_BaseAnalyzeIamPolicy",
				HTTPOptions:                 []*httpOptionAnnotation{{Method: "get", URI: "/v1/{analysis_query.scope}:analyzeIamPolicy", Body: "", HasBody: false, First: true}},
				HasRequiredFields:           true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{{Key: "analysisQuery", Value: "{}", Last: true}},
			},
		},
		{
			name: "multiple required fields formatted in order",
			method: func() *api.Method {
				f1 := api.NewTestField("first_field").WithType(api.TypezString).WithBehavior(api.FieldBehaviorRequired)
				f1.JSONName = "firstField"
				msg := api.NewTestMessage("Nested")
				f2 := api.NewTestField("second_field").WithMessageType(msg).WithBehavior(api.FieldBehaviorRequired)
				f2.JSONName = "secondField"
				req := api.NewTestMessage("MultiReq").WithFields(f1, f2)
				return makeTestRestMethod("Multi", "GET", "", v1Pt, req)
			}(),
			want: &restMethodAnnotation{
				Name:              "multi",
				BaseClassName:     "_BaseMulti",
				HTTPOptions:       []*httpOptionAnnotation{{Method: "get", URI: "/v1", Body: "", HasBody: false, First: true}},
				HasRequiredFields: true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{
					{Key: "firstField", Value: `""`, Last: false},
					{Key: "secondField", Value: "{}", Last: true},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := annotateRestMethod(test.method)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRestMethodBase_TemplateRendering(t *testing.T) {
	tmplBytes, err := templates.ReadFile("templates/partials/rest_method_base.mustache")
	if err != nil {
		t.Fatal(err)
	}
	tmplStr := string(tmplBytes)

	for _, test := range []struct {
		name string
		data *restMethodAnnotation
		want string
	}{
		{
			name: "method with required field and body",
			data: &restMethodAnnotation{
				BaseClassName: "_BaseCreateInstance",
				HTTPOptions: []*httpOptionAnnotation{
					{
						Method:  "post",
						URI:     "/v1/{parent=projects/*/locations/*}/instances",
						Body:    "instance",
						HasBody: true,
						First:   true,
					},
				},
				HasRequiredFields: true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{
					{Key: "instanceId", Value: `""`, Last: true},
				},
			},
			want: `    class _BaseCreateInstance:
        def __hash__(self):  # pragma: NO COVER
            return NotImplementedError("__hash__ must be implemented.")

        __REQUIRED_FIELDS_DEFAULT_VALUES: Dict[str, Any] =  {
            "instanceId" : "",        }

        @staticmethod
        def _get_http_options():
            http_options: List[Dict[str, str]] = [{
                'method': 'post',
                'uri': '/v1/{parent=projects/*/locations/*}/instances',
                'body': 'instance',
            },
            ]
            return http_options

`,
		},
		{
			name: "method with multiple required fields",
			data: &restMethodAnnotation{
				BaseClassName: "_BaseMultiRequired",
				HTTPOptions: []*httpOptionAnnotation{
					{
						Method: "get",
						URI:    "/v1/items",
						First:  true,
					},
				},
				HasRequiredFields: true,
				RequiredFieldsDefaultValues: []*requiredFieldDefaultAnnotation{
					{Key: "firstField", Value: `""`, Last: false},
					{Key: "secondField", Value: "{}", Last: true},
				},
			},
			want: `    class _BaseMultiRequired:
        def __hash__(self):  # pragma: NO COVER
            return NotImplementedError("__hash__ must be implemented.")

        __REQUIRED_FIELDS_DEFAULT_VALUES: Dict[str, Any] =  {
            "firstField" : "",
            "secondField" : {},        }

        @staticmethod
        def _get_http_options():
            http_options: List[Dict[str, str]] = [{
                'method': 'get',
                'uri': '/v1/items',
            },
            ]
            return http_options

`,
		},
		{
			name: "method without required fields and without body",
			data: &restMethodAnnotation{
				BaseClassName: "_BaseDeleteInstance",
				HTTPOptions: []*httpOptionAnnotation{
					{
						Method:  "delete",
						URI:     "/v1/{name=projects/*/locations/*/instances/*}",
						Body:    "",
						HasBody: false,
						First:   true,
					},
				},
				HasRequiredFields: false,
			},
			want: `    class _BaseDeleteInstance:
        def __hash__(self):  # pragma: NO COVER
            return NotImplementedError("__hash__ must be implemented.")

        @staticmethod
        def _get_http_options():
            http_options: List[Dict[str, str]] = [{
                'method': 'delete',
                'uri': '/v1/{name=projects/*/locations/*/instances/*}',
            },
            ]
            return http_options

`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := mustache.Render(tmplStr, test.data)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func makeTestRestMethod(name, verb, body string, pt *api.PathTemplate, req *api.Message) *api.Method {
	m := api.NewTestMethod(name).WithVerb(verb).WithInput(req)
	m.PathInfo.BodyFieldPath = body
	return m.WithPathTemplate(pt)
}
