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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateClientMethod_AsyncReturnTypes(t *testing.T) {
	req := api.NewTestMessage("Req").WithPackage("google.cloud.test.v1")
	resp := api.NewTestMessage("Resp").WithPackage("google.cloud.test.v1")
	item := api.NewTestMessage("Item").WithPackage("google.cloud.test.v1")

	nextPageToken := api.NewTestField("next_page_token").WithType(api.TypezString)
	itemsField := api.NewTestField("items").WithMessageType(item).WithRepeated()
	pagedResp := api.NewTestMessage("PagedResp").
		WithPackage("google.cloud.test.v1").
		WithFields(itemsField, nextPageToken).
		WithPagination(nextPageToken, itemsField)

	pageToken := api.NewTestField("page_token").WithType(api.TypezString)
	pagedReq := api.NewTestMessage("PagedReq").
		WithPackage("google.cloud.test.v1").
		WithFields(pageToken)

	unaryMethod := api.NewTestMethod("GetThing").WithInput(req).WithOutput(resp)
	pagedMethod := api.NewTestMethod("ListThings").WithInput(pagedReq).WithOutput(pagedResp).WithPagination(pageToken)
	lroMethod := api.NewTestMethod("CreateThing").
		WithInput(req).
		WithOutput(resp).
		WithOperationInfo(&api.OperationInfo{
			ResponseTypeID: ".google.cloud.test.v1.Resp",
			MetadataTypeID: ".google.cloud.test.v1.Resp",
		})

	svc := api.NewTestService("TestService").
		WithPackage("google.cloud.test.v1").
		WithMethods(unaryMethod, pagedMethod, lroMethod)

	model := api.NewTestAPI([]*api.Message{req, resp, item, pagedReq, pagedResp}, nil, []*api.Service{svc}).
		WithPackageName("google.cloud.test.v1")

	ann := &clientAnnotations{
		VersionPackage: "google.cloud.test_v1",
		DirectoryName:  "test_service",
	}

	c := newTestCodec(t, model, nil)

	for _, test := range []struct {
		name           string
		method         *api.Method
		wantReturnType string
		wantSphinxType string
		wantAsyncPager string
	}{
		{
			name:           "unary method",
			method:         unaryMethod,
			wantReturnType: "test.Resp",
			wantSphinxType: "google.cloud.test_v1.types.Resp",
			wantAsyncPager: "",
		},
		{
			name:           "paged method",
			method:         pagedMethod,
			wantReturnType: "pagers.ListThingsAsyncPager",
			wantSphinxType: "google.cloud.test_v1.services.test_service.pagers.ListThingsAsyncPager",
			wantAsyncPager: "ListThingsAsyncPager",
		},
		{
			name:           "lro method",
			method:         lroMethod,
			wantReturnType: "operation_async.AsyncOperation",
			wantSphinxType: "google.api_core.operation_async.AsyncOperation",
			wantAsyncPager: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := c.annotateClientMethod(test.method, svc, ann)
			type asyncMethodInfo struct {
				returnType string
				sphinxType string
				asyncPager string
			}
			gotInfo := asyncMethodInfo{
				returnType: got.AsyncReturnType,
				sphinxType: got.AsyncReturnSphinxType,
				asyncPager: got.AsyncPagerClassName,
			}
			wantInfo := asyncMethodInfo{
				returnType: test.wantReturnType,
				sphinxType: test.wantSphinxType,
				asyncPager: test.wantAsyncPager,
			}
			if diff := cmp.Diff(wantInfo, gotInfo, cmp.AllowUnexported(asyncMethodInfo{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAnnotateClientMethod_FlattenedParams(t *testing.T) {
	nameField := api.NewTestField("name").WithType(api.TypezString)
	delegatesField := api.NewTestField("delegates").WithType(api.TypezString).WithRepeated()
	req := api.NewTestMessage("SignJwtRequest").
		WithPackage("google.iam.credentials.v1").
		WithFields(nameField, delegatesField)
	resp := api.NewTestMessage("SignJwtResponse").WithPackage("google.iam.credentials.v1")

	method := api.NewTestMethod("SignJwt").
		WithInput(req).
		WithOutput(resp).
		WithSignatures(&api.MethodSignature{Names: []string{"name", "delegates"}})

	svc := api.NewTestService("IamCredentials").
		WithPackage("google.iam.credentials.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{req, resp}, nil, []*api.Service{svc}).
		WithPackageName("google.iam.credentials.v1")

	c := newTestCodec(t, model, nil)
	ann := &clientAnnotations{
		VersionPackage: "google.iam.credentials_v1",
		DirectoryName:  "iam_credentials",
	}

	got := c.annotateClientMethod(method, svc, ann)

	type paramInfo struct {
		name       string
		isRepeated bool
	}
	var gotParams []paramInfo
	for _, p := range got.FlattenedParams {
		gotParams = append(gotParams, paramInfo{
			name:       p.Name,
			isRepeated: p.IsRepeated,
		})
	}
	wantParams := []paramInfo{
		{name: "name", isRepeated: false},
		{name: "delegates", isRepeated: true},
	}
	if diff := cmp.Diff(wantParams, gotParams, cmp.AllowUnexported(paramInfo{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
