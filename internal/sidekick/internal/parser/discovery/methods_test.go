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
	"github.com/googleapis/librarian/internal/sidekick/internal/api/apitest"
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
		InputTypeID:   "..zones.getRequest",
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
			BodyFieldPath: "",
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestMethodEmptyBody(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := &api.Message{
		Name:             "getRequest",
		ID:               "..zones.getRequest",
		Documentation:    "Synthetic request message for the [get()][.zones.get] method.",
		SyntheticRequest: true,
		Fields: []*api.Field{
			{
				Name:          "project",
				JSONName:      "project",
				ID:            "..zones.getRequest.project",
				Documentation: "Project ID for this request.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "zone",
				JSONName:      "zone",
				ID:            "..zones.getRequest.zone",
				Documentation: "Name of the zone resource to return.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
		},
	}
	got, ok := model.State.MessageByID[want.ID]
	if !ok {
		t.Fatalf("expected message %s in the API model", want.ID)
	}
	apitest.CheckMessage(t, got, want)

	wantParent, ok := model.State.MessageByID["..zones"]
	if !ok {
		t.Fatalf("expected message %s in the API model", "..zones")
	}
	if wantParent != got.Parent {
		t.Errorf("mismatched parent for synthetic request message")
	}
}

func TestMethodWithQueryParameters(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := &api.Message{
		Name:             "listRequest",
		ID:               "..zones.listRequest",
		Documentation:    "Synthetic request message for the [list()][.zones.list] method.",
		SyntheticRequest: true,
		Fields: []*api.Field{
			{
				Name:          "filter",
				JSONName:      "filter",
				ID:            "..zones.listRequest.filter",
				Documentation: "A filter expression that filters resources listed in the response. Most Compute resources support two types of filter expressions: expressions that support regular expressions and expressions that follow API improvement proposal AIP-160. These two types of filter expressions cannot be mixed in one request. If you want to use AIP-160, your expression must specify the field name, an operator, and the value that you want to use for filtering. The value must be a string, a number, or a boolean. The operator must be either `=`, `!=`, `>`, `<`, `<=`, `>=` or `:`. For example, if you are filtering Compute Engine instances, you can exclude instances named `example-instance` by specifying `name != example-instance`. The `:*` comparison can be used to test whether a key has been defined. For example, to find all objects with `owner` label use: ``` labels.owner:* ``` You can also filter nested fields. For example, you could specify `scheduling.automaticRestart = false` to include instances only if they are not scheduled for automatic restarts. You can use filtering on nested fields to filter based on resource labels. To filter on multiple expressions, provide each separate expression within parentheses. For example: ``` (scheduling.automaticRestart = true) (cpuPlatform = \"Intel Skylake\") ``` By default, each expression is an `AND` expression. However, you can include `AND` and `OR` expressions explicitly. For example: ``` (cpuPlatform = \"Intel Skylake\") OR (cpuPlatform = \"Intel Broadwell\") AND (scheduling.automaticRestart = true) ``` If you want to use a regular expression, use the `eq` (equal) or `ne` (not equal) operator against a single un-parenthesized expression with or without quotes or against multiple parenthesized expressions. Examples: `fieldname eq unquoted literal` `fieldname eq 'single quoted literal'` `fieldname eq \"double quoted literal\"` `(fieldname1 eq literal) (fieldname2 ne \"literal\")` The literal value is interpreted as a regular expression using Google RE2 library syntax. The literal value must match the entire field. For example, to filter for instances that do not end with name \"instance\", you would use `name ne .*instance`. You cannot combine constraints on multiple fields using regular expressions.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
				Optional:      true,
			},
			{
				Name:          "maxResults",
				JSONName:      "maxResults",
				ID:            "..zones.listRequest.maxResults",
				Documentation: "The maximum number of results per page that should be returned. If the number of available results is larger than `maxResults`, Compute Engine returns a `nextPageToken` that can be used to get the next page of results in subsequent list requests. Acceptable values are `0` to `500`, inclusive. (Default: `500`)",
				Typez:         api.UINT32_TYPE,
				TypezID:       "uint32",
				Optional:      true,
			},
			{
				Name:          "orderBy",
				JSONName:      "orderBy",
				ID:            "..zones.listRequest.orderBy",
				Documentation: "Sorts list results by a certain order. By default, results are returned in alphanumerical order based on the resource name. You can also sort results in descending order based on the creation timestamp using `orderBy=\"creationTimestamp desc\"`. This sorts results based on the `creationTimestamp` field in reverse chronological order (newest result first). Use this to sort resources like operations so that the newest operation is returned first. Currently, only sorting by `name` or `creationTimestamp desc` is supported.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
				Optional:      true,
			},
			{
				Name:          "pageToken",
				JSONName:      "pageToken",
				ID:            "..zones.listRequest.pageToken",
				Documentation: "Specifies a page token to use. Set `pageToken` to the `nextPageToken` returned by a previous list request to get the next page of results.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
				Optional:      true,
			},
			{
				Name:          "project",
				JSONName:      "project",
				ID:            "..zones.listRequest.project",
				Documentation: "Project ID for this request.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "returnPartialSuccess",
				JSONName:      "returnPartialSuccess",
				ID:            "..zones.listRequest.returnPartialSuccess",
				Documentation: "Opt-in for partial success behavior which provides partial results in case of failure. The default value is false. For example, when partial success behavior is enabled, aggregatedList for a single zone scope either returns all resources in the zone or no resources, with an error code.",
				Typez:         api.BOOL_TYPE,
				TypezID:       "bool",
				Optional:      true,
			},
		},
	}
	gotListRequest, ok := model.State.MessageByID[want.ID]
	if !ok {
		t.Fatalf("expected message %s in the API model", want.ID)
	}
	apitest.CheckMessage(t, gotListRequest, want)
}

func TestMethodWithBody(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantInsertRequest := &api.Message{
		Name:             "insertRequest",
		ID:               "..addresses.insertRequest",
		Documentation:    "Synthetic request message for the [insert()][.addresses.insert] method.",
		SyntheticRequest: true,
		Fields: []*api.Field{
			{
				Name:          "body",
				JSONName:      "body",
				ID:            "..addresses.insertRequest.body",
				Documentation: "Synthetic request body field for the [insert()][.addresses.insert] method.",
				Typez:         api.MESSAGE_TYPE,
				TypezID:       "..Address",
				Optional:      true,
			},
			{
				Name:          "project",
				JSONName:      "project",
				ID:            "..addresses.insertRequest.project",
				Documentation: "Project ID for this request.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "region",
				JSONName:      "region",
				ID:            "..addresses.insertRequest.region",
				Documentation: "Name of the region for this request.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "requestId",
				JSONName:      "requestId",
				ID:            "..addresses.insertRequest.requestId",
				Documentation: "An optional request ID to identify requests. Specify a unique request ID so that if you must retry your request, the server will know to ignore the request if it has already been completed. For example, consider a situation where you make an initial request and the request times out. If you make the request again with the same request ID, the server can check if original operation with the same request ID was received, and if so, will ignore the second request. This prevents clients from accidentally creating duplicate commitments. The request ID must be a valid UUID with the exception that zero UUID is not supported ( 00000000-0000-0000-0000-000000000000).",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
				Optional:      true,
			},
		},
	}
	gotGetRequest, ok := model.State.MessageByID[wantInsertRequest.ID]
	if !ok {
		t.Fatalf("expected message %s in the API model", wantInsertRequest.ID)
	}
	apitest.CheckMessage(t, gotGetRequest, wantInsertRequest)
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
	service := &api.Service{
		Name: "Service",
		ID:   ".test.Service",
	}
	if err := makeServiceMethods(model, service, &doc, input); err == nil {
		t.Errorf("expected error on method with media upload, service=%v", service)
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
		{"badParameter", method{Path: "a/b", Parameters: []*parameter{
			{Name: "badParameter", schema: schema{Type: "string", Format: "--invalid--"}},
		}}},
		{"badParameterName", method{
			Path:    "a/b",
			Request: &schema{Ref: "Zone"},
			Parameters: []*parameter{
				{Name: "body", schema: schema{Type: "string"}},
			}}},
	} {
		doc := document{}
		parent := &api.Message{
			Name: "Service",
			ID:   ".test.Service",
		}
		if got, err := makeMethod(model, parent, &doc, &test.Input); err == nil {
			t.Errorf("expected error on method[%s], got=%v", test.Name, got)
		}
	}

}

func TestBodyFieldName(t *testing.T) {
	for _, test := range []struct {
		Input []string
		Want  string
	}{
		{[]string{"a", "b", "c"}, "body"},
		{[]string{"a", "body_", "c"}, "body"},
		{[]string{"a", "body_", "body__"}, "body"},
	} {
		fieldNames := map[string]bool{}
		for _, n := range test.Input {
			fieldNames[n] = true
		}
		got, err := bodyFieldName(fieldNames)
		if err != nil {
			t.Errorf("expected successful body field name, err=%v", err)
		}
		if test.Want != got {
			t.Errorf("mismatch want=%s, got=%s", test.Want, got)
		}
	}
}

func TestBodyFieldError(t *testing.T) {
	fieldNames := map[string]bool{}
	for _, n := range []string{"a", "b", "body"} {
		fieldNames[n] = true
	}
	if got, err := bodyFieldName(fieldNames); err == nil {
		t.Errorf("expected an error  got=%s", got)
	}
}
