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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateService_StubStructure(t *testing.T) {
	outDir := t.TempDir()

	request := &api.Message{
		Name:    "Request",
		ID:      ".test.Request",
		Package: "test",
	}
	response := &api.Message{
		Name:    "Response",
		ID:      ".test.Response",
		Package: "test",
	}
	service := &api.Service{
		Name:    "Protocol",
		ID:      ".test.Prototocol",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "GetThing",
				ID:           ".test.IAM.CreateRole",
				InputTypeID:  ".test.Request",
				InputType:    request,
				OutputTypeID: ".test.Response",
				OutputType:   response,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "GET", PathTemplate: &api.PathTemplate{}}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			Name:       "SomeTestPackage",
			ApiPackage: "test",
		},
	})
	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Clients", "ProtocolStub.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	got := extractBlock(t, contentStr, `  protocol ProtocolStub {`, "\n"+`  }`)
	want := `  protocol ProtocolStub {
    func getThing(
    request: Request, options: GoogleCloudGax.RequestOptions
) async throws -> SomeTestPackage.Response

  }`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, contentStr, `  class ProtocolTransport: `, `HTTPClient`)
	want = `  class ProtocolTransport: ProtocolStub {
    let inner: GoogleCloudGax.HTTPClient`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, contentStr, `return try GoogleCloudWkt._ProtoJSONDecoder()`, ", from: data)\n    }")
	want = `return try GoogleCloudWkt._ProtoJSONDecoder().decode(
        SomeTestPackage.Response.self, from: data)
    }`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateService_QueryParameters(t *testing.T) {
	outDir := t.TempDir()

	oneof := &api.OneOf{Name: "expiration"}
	oneofField := &api.Field{
		Name:     "ttl_days",
		JSONName: "ttlDays",
		ID:       ".google.test.Request.ttl_days",
		Typez:    api.TypezString,
		IsOneOf:  true,
		Group:    oneof,
	}
	oneof.Fields = []*api.Field{oneofField}

	request := &api.Message{
		Name:    "Request",
		ID:      ".test.Request",
		Package: "test",
		Fields: []*api.Field{
			oneofField,
			{
				Name:     "project",
				JSONName: "project",
				ID:       ".google.test.Request.project",
				Typez:    api.TypezString,
			},
			{
				Name:     "enable",
				JSONName: "enable",
				ID:       ".google.test.Request.enable",
				Typez:    api.TypezBool,
			},
		},
		OneOfs: []*api.OneOf{oneof},
	}
	response := &api.Message{
		Name:    "Response",
		ID:      ".test.Response",
		Package: "test",
	}
	service := &api.Service{
		Name:    "Service",
		ID:      ".test.Service",
		Package: "test",
		Methods: []*api.Method{
			{
				Name:         "GetThing",
				ID:           ".test.Service.GetThing",
				InputTypeID:  ".test.Request",
				InputType:    request,
				OutputTypeID: ".test.Response",
				OutputType:   response,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{
						Verb:         "GET",
						PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("projects").WithVariableNamed("project"),
						QueryParameters: map[string]bool{
							"ttl_days": true,
							"enable":   true,
						},
					}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{request, response}, nil, []*api.Service{service})
	model.PackageName = "test"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	swiftCfg := swiftConfig(t, []config.SwiftDependency{})
	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleTest", "Clients", "ServiceStub.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	got := extractBlock(t, contentStr, `contentsOf: try encoder.encode(request.enable`, `)`)
	want := `contentsOf: try encoder.encode(request.enable, prefix: "enable")`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	got = extractBlock(t, contentStr, `request.expiration.map {`, `prefix: "ttlDays")`)
	want = `request.expiration.map { (oneof) -> Swift.String? in
            if case let .ttlDays(v) = oneof { v } else { nil }
          }, prefix: "ttlDays")`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
