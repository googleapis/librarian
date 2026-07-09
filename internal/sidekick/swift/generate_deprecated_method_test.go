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

func TestGenerateService_DeprecatedMethods(t *testing.T) {
	// Common messages
	requestType := api.NewTestMessage("Request").
		WithFields(api.NewTestField("name").WithType(api.TypezString))

	responseType := api.NewTestMessage("Response")

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

	operationType := api.NewTestMessage("Operation").WithPackage("google.longrunning")

	lroResultType := api.NewTestMessage("LROResult")
	lroMetadataType := api.NewTestMessage("LROMetadata")
	getOperationInputType := api.NewTestMessage("GetOperationRequest").WithPackage("google.longrunning")

	for _, test := range []struct {
		name  string
		setup func() *api.Method
		want  []struct {
			start string
			end   string
			want  string
		}
	}{
		{
			name: "Simple_Deprecated",
			setup: func() *api.Method {
				m := api.NewTestMethod("SimpleMethod").
					WithInput(requestType).
					WithOutput(responseType).
					WithVerb("POST").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("simple"))
				m.Deprecated = true
				m.Documentation = "-- simple marker --"
				return m
			},
			want: []struct {
				start string
				end   string
				want  string
			}{
				{
					start: "    /// See `TestServiceClient.simpleMethod`.",
					end:   "-> GoogleTest.Response",
					want:  "    /// See `TestServiceClient.simpleMethod`.\n    @available(*, deprecated)\n    func simpleMethod(request: Request) async throws -> GoogleTest.Response",
				},
				{
					start: "  /// -- simple marker --",
					end:   "async throws -> GoogleTest.Response",
					want:  "  /// -- simple marker --\n  ///\n  /// @Snippet(path: \"TestService_SimpleMethod\")\n  @available(*, deprecated)\n  public func simpleMethod(\n    request: Request, options: GoogleCloudGax.RequestOptions\n) async throws -> GoogleTest.Response",
				},
			},
		},
		{
			name: "Pagination_Deprecated",
			setup: func() *api.Method {
				m := api.NewTestMethod("PaginationMethod").
					WithInput(requestType).
					WithOutput(paginationResponseType).
					WithVerb("GET").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("pagination"))
				m.Deprecated = true
				m.Pagination = requestType.Fields[0]
				m.Documentation = "-- pagination marker --"
				return m
			},
			want: []struct {
				start string
				end   string
				want  string
			}{
				{
					start: "    /// See `TestServiceClient.paginationMethod`.",
					end:   "-> any AsyncSequence<Item, Swift.Error>",
					want:  "    /// See `TestServiceClient.paginationMethod`.\n    @available(*, deprecated)\n    func paginationMethod(request: Request) async throws -> GoogleTest.PaginationResponse\n\n    /// See `TestServiceClient.paginationMethod`.\n    @available(*, deprecated)\n    func paginationMethod(\n  byItem: Request\n) throws -> any AsyncSequence<Item, Swift.Error>",
				},
				{
					start: "  /// -- pagination marker --",
					end:   "-> any AsyncSequence<Item, Swift.Error>",
					want:  "  /// -- pagination marker --\n  ///\n  /// @Snippet(path: \"TestService_PaginationMethod\")\n  @available(*, deprecated)\n  public func paginationMethod(\n    request: Request, options: GoogleCloudGax.RequestOptions\n) async throws -> GoogleTest.PaginationResponse\n {\n      try await self.inner.paginationMethod(request: request, options: options)\n  }\n\n  /// -- pagination marker --\n  ///\n  /// @Snippet(path: \"TestService_PaginationMethod\")\n  @available(*, deprecated)\n  public func paginationMethod(\n    byItem: Request, options: GoogleCloudGax.RequestOptions\n) throws -> any AsyncSequence<Item, Swift.Error>",
				},
			},
		},
		{
			name: "LRO_Deprecated",
			setup: func() *api.Method {
				m := api.NewTestMethod("LROMethod").
					WithInput(requestType).
					WithOutput(operationType).
					WithVerb("POST").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("lro"))
				m.Deprecated = true
				m.IsLRO = true
				m.OperationInfo = &api.OperationInfo{
					ResponseTypeID: lroResultType.ID,
					MetadataTypeID: lroMetadataType.ID,
				}
				m.Documentation = "-- lro marker --"
				return m
			},
			want: []struct {
				start string
				end   string
				want  string
			}{
				{
					start: "    /// See `TestServiceClient.lromethod`.",
					end:   "-> any GoogleCloudGax.PollableOperation<LROResult>",
					want:  "    /// See `TestServiceClient.lromethod`.\n    @available(*, deprecated)\n    func lromethod(request: Request) async throws -> GoogleCloudLongrunningV1.Operation\n\n    /// See `TestServiceClient.lromethod`.\n    @available(*, deprecated)\n    func lromethod(withPolling: Request) async throws -> any GoogleCloudGax.PollableOperation<LROResult>",
				},
				{
					start: "  /// -- lro marker --",
					end:   "-> any GoogleCloudGax.PollableOperation<LROResult>",
					want:  "  /// -- lro marker --\n  ///\n  /// @Snippet(path: \"TestService_LROMethod\")\n  @available(*, deprecated)\n  public func lromethod(\n    request: Request, options: GoogleCloudGax.RequestOptions\n) async throws -> GoogleCloudLongrunningV1.Operation\n {\n      try await self.inner.lromethod(request: request, options: options)\n  }\n\n  /// -- lro marker --\n  ///\n  /// @Snippet(path: \"TestService_LROMethod\")\n  @available(*, deprecated)\n  public func lromethod(\n    withPolling: Request, options: GoogleCloudGax.RequestOptions\n) async throws -> any GoogleCloudGax.PollableOperation<LROResult>",
				},
			},
		},
		{
			name: "Simple_NotDeprecated",
			setup: func() *api.Method {
				m := api.NewTestMethod("NotDeprecatedMethod").
					WithInput(requestType).
					WithOutput(responseType).
					WithVerb("POST").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("notDeprecated"))
				m.Documentation = "-- not deprecated marker --"
				return m
			},
			want: []struct {
				start string
				end   string
				want  string
			}{
				{
					start: "    /// See `TestServiceClient.notDeprecatedMethod`.",
					end:   "-> GoogleTest.Response",
					want:  "    /// See `TestServiceClient.notDeprecatedMethod`.\n    func notDeprecatedMethod(request: Request) async throws -> GoogleTest.Response",
				},
				{
					start: "  /// -- not deprecated marker --",
					end:   "async throws -> GoogleTest.Response",
					want:  "  /// -- not deprecated marker --\n  ///\n  /// @Snippet(path: \"TestService_NotDeprecatedMethod\")\n  public func notDeprecatedMethod(\n    request: Request, options: GoogleCloudGax.RequestOptions\n) async throws -> GoogleTest.Response",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			method := test.setup()

			// We need a fresh service for each test case
			service := api.NewTestService("TestService").WithMethods(
				method,
				api.NewTestMethod("GetOperation").
					WithInput(getOperationInputType).
					WithOutput(operationType).
					WithVerb("GET").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations")),
			)

			model := api.NewTestAPI([]*api.Message{
				requestType, responseType, itemType, paginationResponseType,
				operationType, lroResultType, lroMetadataType, getOperationInputType,
			}, nil, []*api.Service{service})
			model.PackageName = "test"

			cfg := &parser.ModelConfig{
				Codec: map[string]string{
					"copyright-year": "2038",
				},
			}

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
			contentStr := string(content)

			for _, want := range test.want {
				got := extractBlock(t, contentStr, want.start, want.end)
				if diff := cmp.Diff(want.want, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
