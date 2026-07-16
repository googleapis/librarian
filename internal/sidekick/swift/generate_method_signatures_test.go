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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateService_MethodSignatures(t *testing.T) {
	for _, test := range []struct {
		name string
		want []expectedBlock
	}{
		{
			name: "Simple",
			want: []expectedBlock{
				{
					start: "public func simpleMethod(\n  name: Swift.String,\n  optionalField: Swift.String?,",
					end:   "    }\n",
					want: `public func simpleMethod(
  name: Swift.String,
  optionalField: Swift.String?,
) async throws -> GoogleTest.Response
 {
    let request = Request().with {
      $0.name = name
      $0.group = optionalField.map { .optionalField($0) }
    }
`,
				},
				{
					start: "public func simpleMethod(\n  name: Swift.String,\n  normalField: Swift.String,",
					end:   "    }\n",
					want: `public func simpleMethod(
  name: Swift.String,
  normalField: Swift.String,
) async throws -> GoogleTest.Response
 {
    let request = Request().with {
      $0.name = name
      $0.group = .normalField(normalField)
    }
`,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			model := newModelWithSignatures(t)
			cfg := &parser.ModelConfig{}
			swiftCfg := swiftConfig(t, []config.SwiftDependency{
				{Name: "GoogleCloudGax", RequiredByServices: true},
				{Name: "GoogleCloudAuth", RequiredByServices: true},
				{ApiPackage: "google.longrunning", Name: "GoogleCloudLongrunningV1"},
				{ApiPackage: "google.rpc", Name: "GoogleRpc"},
			})

			if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleTest", "TestService.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			// We only want to search within the default implementations.
			contentStr := extractBlock(t, string(content), "// Default implementations\nextension Clients.TestServiceProtocol {", "\n}")
			for i, want := range test.want {
				t.Run(fmt.Sprintf("block %d", i), func(t *testing.T) {
					got := extractBlock(t, contentStr, want.start, want.end)
					if diff := cmp.Diff(want.want, got); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

// Creates a model with a normal method, a paginated method, and a LRO method.
//
// All of them have a method signatures with a normal field, a oneof field, and an optional oneof field.
func newModelWithSignatures(t *testing.T) *api.API {
	t.Helper()
	oneof := api.NewTestOneOf("group").WithFields(
		api.NewTestField("optional_field").WithType(api.TypezString).WithOptional(),
		api.NewTestField("normal_field").WithType(api.TypezString),
	)
	requestType := api.NewTestMessage("Request").WithPackage("test").
		WithFields(api.NewTestField("name").WithType(api.TypezString)).
		WithOneOfs(oneof)
	responseType := api.NewTestMessage("Response")

	simpleMethod := api.NewTestMethod("SimpleMethod").
		WithInput(requestType).
		WithOutput(responseType).
		WithVerb("POST").
		WithSignatures(
			&api.MethodSignature{Names: []string{"name", "optional_field"}},
			&api.MethodSignature{Names: []string{"name", "normal_field"}},
		).
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("simple"))

	itemType := api.NewTestMessage("Item")
	paginationResponseType := api.NewTestMessage("PaginationResponse").
		WithFields(
			api.NewTestField("items").WithMessageType(itemType).WithRepeated(),
			api.NewTestField("next_page_token").WithType(api.TypezString),
		)
	paginationResponseType.Pagination = &api.PaginationInfo{
		PageableItem:  paginationResponseType.Fields[0],
		NextPageToken: paginationResponseType.Fields[1],
	}
	paginationMethod := api.NewTestMethod("PaginationMethod").
		WithInput(requestType).
		WithOutput(paginationResponseType).
		WithVerb("GET").
		WithSignatures(
			&api.MethodSignature{Names: []string{"name", "optional_field"}},
			&api.MethodSignature{Names: []string{"name", "normal_field"}},
		).
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("page"))

	operationType := api.NewTestMessage("Operation").WithPackage("google.longrunning")
	getOperationInputType := api.NewTestMessage("GetOperationRequest").WithPackage("google.longrunning")

	lroResultType := api.NewTestMessage("LROResult")
	lroMetadataType := api.NewTestMessage("LROMetadata")
	lroMethod := api.NewTestMethod("LroMethod").
		WithInput(requestType).
		WithOutput(getOperationInputType).
		WithVerb("POST").
		WithSignatures(
			&api.MethodSignature{Names: []string{"name", "optional_field"}},
			&api.MethodSignature{Names: []string{"name", "normal_field"}},
		).
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("lro"))
	lroMethod.IsLRO = true
	lroMethod.OperationInfo = &api.OperationInfo{
		ResponseTypeID: lroResultType.ID,
		MetadataTypeID: lroMetadataType.ID,
	}

	discoveryLroMethod := api.NewTestMethod("LroDiscoveryMethod").
		WithInput(requestType).
		WithOutput(getOperationInputType).
		WithVerb("POST").
		WithSignatures(
			&api.MethodSignature{Names: []string{"name", "optional_field"}},
			&api.MethodSignature{Names: []string{"name", "normal_field"}},
		).
		WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("discoveryLro"))
	discoveryLroMethod.DiscoveryLro = &api.DiscoveryLro{
		PollingPathParameters: []string{"test_only"},
	}

	service := api.NewTestService("TestService").
		WithPackage("test").
		WithMethods(simpleMethod, paginationMethod, lroMethod, discoveryLroMethod)
	model := api.NewTestAPI([]*api.Message{
		requestType, responseType, itemType, paginationResponseType,
		operationType, lroResultType, lroMetadataType,
	}, nil, []*api.Service{service})
	model.PackageName = "test"
	model.AddMessage(getOperationInputType)
	model.AddMessage(operationType)
	return model
}
