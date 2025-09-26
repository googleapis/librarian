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
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/api/apitest"
	"github.com/googleapis/librarian/internal/sidekick/internal/sample"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestSorted(t *testing.T) {
	got, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.IsSortedFunc(got.Messages, compareMessages) {
		t.Fatalf("unsorted messages after parsing")
	}
	if !slices.IsSortedFunc(got.Services, compareServices) {
		t.Fatalf("unsorted messages after parsing")
	}
}

func TestInfo(t *testing.T) {
	got, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantName := "compute"
	wantTitle := "Compute Engine API"
	wantDescription := "Creates and runs virtual machines on Google Cloud Platform. "
	if got.Name != wantName {
		t.Errorf("want = %q; got = %q", wantName, got.Name)
	}
	if got.Title != wantTitle {
		t.Errorf("want = %q; got = %q", wantTitle, got.Title)
	}
	if diff := cmp.Diff(wantDescription, got.Description); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got.PackageName != "" {
		t.Errorf("expected empty package name")
	}
}

func TestServiceConfigOverridesInfo(t *testing.T) {
	sc := sample.ServiceConfig()
	sc.Title = "Change the title for testing"
	sc.Documentation.Summary = "Change the description for testing"
	sc.Name = "not-secretmanager"

	got, err := ComputeDisco(t, sc)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != sc.Name {
		t.Errorf("want = %q; got = %q", sc.Title, got.Title)
	}
	if got.Title != sc.Title {
		t.Errorf("want = %q; got = %q", sc.Title, got.Title)
	}
	if diff := cmp.Diff(sc.Documentation.Summary, got.Description); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if len(sc.Apis) != 2 {
		t.Fatalf("expected 2 APIs in service config")
	}
	if got.PackageName == "" {
		t.Errorf("got empty package name")
	}
	if !strings.HasPrefix(sc.Apis[1].Name, got.PackageName) {
		t.Errorf("mismatched package name want = %q, got = %q", sc.Apis[1].Name, got.PackageName)
	}
}

func TestBadParse(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Contents string
	}{
		{"empty", ""},
		{"auth parse", `{"auth": {"oauth2": {"scopes": "should-be-object"}}}`},
		{"unknown schema", `{"schemas": {"Bad": {"type": "unknown"}}}`},
		{"schema must be object", `{"schemas": {"mustBeObject": {"type": "string"}}}`},
		{"schema is ref", `{"schemas": {"cannotBeRef": {"$ref": "AnotherSchema"}}}`},
		{"property parse", `{"schemas": {"badProperty": {"properties": {"typeShouldbeString": {"type": 123}}}}}`},
		{"property with unknown schema", `{"schemas": {"badProperty": {"type": "object", "properties": {"bad": {"type": "unknown"}}}}}`},
		{"property with bad array", `{"schemas": {"badProperty": {"type": "object", "properties": {"badArray": {"type": "array"}}}}}`},
		{"property with bad array", `{"schemas": {"badProperty": {"type": "object", "properties": {"badItem": {"type": "array", "items": {"$ref": "notFound"}}}}}}`},
		{"property with bad array", `{"schemas": {"badProperty": {"type": "object", "properties": {"itemInNonArray": {"type": "string", "items": {"type": "string"}}}}}}`},
		{"property with bad array", `{"schemas": {"badProperty": {"type": "object", "properties": {"badAdditional": {"type": "object", "additionalProperties": {"$ref": "notFound"} }}}}}`},
		{"method cannot parse", `{"methods": {"idShouldBeString": {"id": 123}}}`},
		{"method parameter cannot parse", `{"methods": {"badParameter": {"parameters": {"locationShouldBeString": {"location": 123}}}}}`},
		{"method with bad request", `{"methods": {"badRequest": {"request": {"$ref": "notThere"}}}}`},
		{"method with bad response", `{"methods": {"badResponse": {"response": {"$ref": "notThere"}}}}`},
		{"resource cannot parse", `{"resources": {"childShouldBeMap": {"resources": 123}}}`},
		{"resource with bad method", `{"resources": {"badResource": {"methods": {"badResponse": {"response": {"$ref": "notThere"}}}}}}`},
		{"resource with bad child", `{"resources": {"badResource": {"resources": {"badChild": {"methods": {"badResponse": {"response": {"$ref": "notThere"}}}}}}}}`},
	} {
		contents := []byte(test.Contents)
		if _, err := NewAPI(nil, contents); err == nil {
			t.Fatalf("expected error for %s input", test.Name)
		}
	}
}

func TestMessage(t *testing.T) {
	model, err := ComputeDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := "..WeightedBackendService"
	got, ok := model.State.MessageByID[id]
	if !ok {
		t.Fatalf("expected message %s in the API model", id)
	}
	want := &api.Message{
		Name:          "WeightedBackendService",
		ID:            id,
		Package:       "",
		Documentation: "In contrast to a single BackendService in HttpRouteAction to which all matching traffic is directed to, WeightedBackendService allows traffic to be split across multiple backend services. The volume of traffic for each backend service is proportional to the weight specified in each WeightedBackendService",
		Fields: []*api.Field{
			{
				Name:          "backendService",
				JSONName:      "backendService",
				ID:            "..WeightedBackendService.backendService",
				Documentation: "The full or partial URL to the default BackendService resource. Before forwarding the request to backendService, the load balancer applies any relevant headerActions specified as part of this backendServiceWeight.",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "headerAction",
				JSONName:      "headerAction",
				ID:            "..WeightedBackendService.headerAction",
				Documentation: "Specifies changes to request and response headers that need to take effect for the selected backendService. headerAction specified here take effect before headerAction in the enclosing HttpRouteRule, PathMatcher and UrlMap. headerAction is not supported for load balancers that have their loadBalancingScheme set to EXTERNAL. Not supported when the URL map is bound to a target gRPC proxy that has validateForProxyless field set to true.",
				Typez:         api.MESSAGE_TYPE,
				TypezID:       "..HttpHeaderAction",
				Optional:      true,
			},
			{
				Name:          "weight",
				JSONName:      "weight",
				ID:            "..WeightedBackendService.weight",
				Documentation: "Specifies the fraction of traffic sent to a backend service, computed as weight / (sum of all weightedBackendService weights in routeAction) . The selection of a backend service is determined only for new traffic. Once a user's request has been directed to a backend service, subsequent requests are sent to the same backend service as determined by the backend service's session affinity policy. Don't configure session affinity if you're using weighted traffic splitting. If you do, the weighted traffic splitting configuration takes precedence. The value must be from 0 to 1000.",
				Typez:         api.UINT32_TYPE,
				TypezID:       "uint32",
			},
		},
	}
	apitest.CheckMessage(t, got, want)
}

func TestMessageErrors(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Contents string
	}{
		{"bad message field", `{"schemas": {"withBadField": {"type": "object", "properties": {"badFormat": {"type": "string", "format": "--bad--"}}}}}`},
	} {
		contents := []byte(test.Contents)
		if _, err := NewAPI(nil, contents); err == nil {
			t.Fatalf("expected error for %s input", test.Name)
		}
	}
}

func TestServiceErrors(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Contents string
	}{
		{"bad method", `{"resources": {"withBadMethod": {"methods": {"uploadNotSupported": { "mediaUpload": {} }}}}}`},
	} {
		contents := []byte(test.Contents)
		if got, err := NewAPI(nil, contents); err == nil {
			t.Fatalf("expected error for %s input, got=%v", test.Name, got)
		}
	}
}

func ComputeDisco(t *testing.T, sc *serviceconfig.Service) (*api.API, error) {
	t.Helper()
	contents, err := os.ReadFile("../../../testdata/disco/compute.v1.json")
	if err != nil {
		return nil, err
	}
	return NewAPI(sc, contents)
}
