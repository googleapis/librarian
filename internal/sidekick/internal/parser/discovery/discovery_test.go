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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/api/apitest"
	"github.com/googleapis/librarian/internal/sidekick/internal/sample"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestInfo(t *testing.T) {
	got, err := PublicCaDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantName := "publicca"
	wantTitle := "Public Certificate Authority API"
	wantDescription := "The Public Certificate Authority API may be used to create and manage ACME external account binding keys associated with Google Trust Services' publicly trusted certificate authority. "
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

func TestComputeParses(t *testing.T) {
	contents, err := os.ReadFile("../../../testdata/disco/compute.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewAPI(nil, contents)
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

	got, err := PublicCaDisco(t, sc)
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
	model, err := PublicCaDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := "..ExternalAccountKey"
	got, ok := model.State.MessageByID[id]
	if !ok {
		t.Fatalf("expected message %s in the API model", id)
	}
	want := &api.Message{
		Name:          "ExternalAccountKey",
		ID:            id,
		Package:       "",
		Documentation: "A representation of an ExternalAccountKey used for [external account binding](https://tools.ietf.org/html/rfc8555#section-7.3.4) within ACME.",
		Fields: []*api.Field{
			{
				Name:          "b64MacKey",
				JSONName:      "b64MacKey",
				Documentation: "Output only. Base64-URL-encoded HS256 key. It is generated by the PublicCertificateAuthorityService when the ExternalAccountKey is created",
				Typez:         api.BYTES_TYPE,
				TypezID:       "bytes",
			},
			{
				Name:          "keyId",
				JSONName:      "keyId",
				Documentation: "Output only. Key ID. It is generated by the PublicCertificateAuthorityService when the ExternalAccountKey is created",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
			},
			{
				Name:          "name",
				JSONName:      "name",
				Documentation: "Output only. Resource name. projects/{project}/locations/{location}/externalAccountKeys/{key_id}",
				Typez:         api.STRING_TYPE,
				TypezID:       "string",
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

func PublicCaDisco(t *testing.T, sc *serviceconfig.Service) (*api.API, error) {
	t.Helper()
	contents, err := os.ReadFile("../../../testdata/disco/publicca.v1.json")
	if err != nil {
		return nil, err
	}
	return NewAPI(sc, contents)
}
