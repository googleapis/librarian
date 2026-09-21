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
)

type expectedBlock struct {
	start string
	end   string
	want  string
}

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
		want  []expectedBlock
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
			want: []expectedBlock{
				{
					start: "    /// See `TestServiceClient.simpleMethod`.",
					end:   "-> Test.Response",
					want:  "    /// See `TestServiceClient.simpleMethod`.\n    @available(*, deprecated)\n    func simpleMethod(request: Request) async throws -> Test.Response",
				},
				{
					start: "  /// -- simple marker --",
					end:   "async throws -> Test.Response",
					want:  "  /// -- simple marker --\n  ///\n  /// @Snippet(path: \"TestService_SimpleMethod\")\n  @available(*, deprecated)\n  public func simpleMethod(\n    request: Request, options: GoogleGax.RequestOptions\n) async throws -> Test.Response",
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
			want: []expectedBlock{
				{
					start: "    /// See `TestServiceClient.paginationMethod`.",
					end:   "-> any AsyncSequence<Item, Swift.Error>",
					want:  "    /// See `TestServiceClient.paginationMethod`.\n    @available(*, deprecated)\n    func paginationMethod(request: Request) async throws -> Test.PaginationResponse\n\n    /// See `TestServiceClient.paginationMethod`.\n    @available(*, deprecated)\n    func paginationMethod(\n  byItem: Request\n) throws -> any AsyncSequence<Item, Swift.Error>",
				},
				{
					start: "  /// -- pagination marker --",
					end:   "-> any AsyncSequence<Item, Swift.Error>",
					want:  "  /// -- pagination marker --\n  ///\n  /// @Snippet(path: \"TestService_PaginationMethod\")\n  @available(*, deprecated)\n  public func paginationMethod(\n    request: Request, options: GoogleGax.RequestOptions\n) async throws -> Test.PaginationResponse\n {\n      try await self.inner.paginationMethod(request: request, options: options)\n  }\n\n  /// -- pagination marker --\n  ///\n  /// @Snippet(path: \"TestService_PaginationMethod\")\n  @available(*, deprecated)\n  public func paginationMethod(\n    byItem: Request, options: GoogleGax.RequestOptions\n) throws -> any AsyncSequence<Item, Swift.Error>",
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
			want: []expectedBlock{
				{
					start: "    /// See `TestServiceClient.lromethod`.",
					end:   "-> any GoogleGax.PollableOperation<LROResult>",
					want:  "    /// See `TestServiceClient.lromethod`.\n    @available(*, deprecated)\n    func lromethod(request: Request) async throws -> GoogleCloudLongrunningV1.Operation\n\n    /// See `TestServiceClient.lromethod`.\n    @available(*, deprecated)\n    func lromethod(withPolling: Request) async throws -> any GoogleGax.PollableOperation<LROResult>",
				},
				{
					start: "  /// -- lro marker --",
					end:   "-> any GoogleGax.PollableOperation<LROResult>",
					want:  "  /// -- lro marker --\n  ///\n  /// @Snippet(path: \"TestService_LROMethod\")\n  @available(*, deprecated)\n  public func lromethod(\n    request: Request, options: GoogleGax.RequestOptions\n) async throws -> GoogleCloudLongrunningV1.Operation\n {\n      try await self.inner.lromethod(request: request, options: options)\n  }\n\n  /// -- lro marker --\n  ///\n  /// @Snippet(path: \"TestService_LROMethod\")\n  @available(*, deprecated)\n  public func lromethod(\n    withPolling: Request, options: GoogleGax.RequestOptions\n) async throws -> any GoogleGax.PollableOperation<LROResult>",
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
			want: []expectedBlock{
				{
					start: "    /// See `TestServiceClient.notDeprecatedMethod`.",
					end:   "-> Test.Response",
					want:  "    /// See `TestServiceClient.notDeprecatedMethod`.\n    func notDeprecatedMethod(request: Request) async throws -> Test.Response",
				},
				{
					start: "  /// -- not deprecated marker --",
					end:   "async throws -> Test.Response",
					want:  "  /// -- not deprecated marker --\n  ///\n  /// @Snippet(path: \"TestService_NotDeprecatedMethod\")\n  public func notDeprecatedMethod(\n    request: Request, options: GoogleGax.RequestOptions\n) async throws -> Test.Response",
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

			swiftCfg := swiftConfig(t, []config.SwiftDependency{
				{Name: "GoogleGax", RequiredByServices: true},
				{Name: "GoogleAuth", RequiredByServices: true},
				{ApiPackage: "google.longrunning", Name: "GoogleCloudLongrunningV1"},
				{ApiPackage: "google.rpc", Name: "GoogleRpc"},
			})

			library := &config.Library{
				Swift: swiftCfg,
			}

			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "Test", "TestService.swift")
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

// TestGenerateService_DiagnoseMethodTypes covers the declarations that name a
// deprecated request or response type.
//
// The public client and protocol declarations carry `@available(*, deprecated)`
// when the method or its service is deprecated, which already suppresses the
// warning. The internal stub, retry, logging and transport declarations never
// do, so they always need the attribute.
func TestGenerateService_DiagnoseMethodTypes(t *testing.T) {
	for _, test := range []struct {
		name              string
		methodDeprecated  bool
		serviceDeprecated bool
		requestDeprecated bool
		wantPublic        bool
		wantStub          bool
	}{
		{
			name:              "deprecated-request-type",
			requestDeprecated: true,
			wantPublic:        true,
			wantStub:          true,
		},
		{
			name:              "deprecated-method",
			methodDeprecated:  true,
			requestDeprecated: true,
			wantPublic:        false,
			wantStub:          true,
		},
		{
			name:              "deprecated-service",
			serviceDeprecated: true,
			requestDeprecated: true,
			wantPublic:        false,
			wantStub:          true,
		},
		{
			name:       "not-deprecated",
			wantPublic: false,
			wantStub:   false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			requestType := api.NewTestMessage("Request").
				WithDeprecated(test.requestDeprecated).
				WithFields(api.NewTestField("name").WithType(api.TypezString))
			responseType := api.NewTestMessage("Response")
			operationType := api.NewTestMessage("Operation").WithPackage("google.longrunning")
			getOperationInputType := api.NewTestMessage("GetOperationRequest").
				WithPackage("google.longrunning")

			service := api.NewTestService("TestService").
				WithDeprecated(test.serviceDeprecated).
				WithMethods(
					api.NewTestMethod("SimpleMethod").
						WithInput(requestType).
						WithOutput(responseType).
						WithVerb("POST").
						WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("simple")).
						WithSignatures(&api.MethodSignature{Fields: []*api.Field{requestType.Fields[0]}}).
						WithDeprecated(test.methodDeprecated),
					api.NewTestMethod("GetOperation").
						WithInput(getOperationInputType).
						WithOutput(operationType).
						WithVerb("GET").
						WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations")),
				)

			model := api.NewTestAPI([]*api.Message{
				requestType, responseType, operationType, getOperationInputType,
			}, nil, []*api.Service{service})
			model.PackageName = "test"

			library := &config.Library{
				Swift: swiftConfig(t, []config.SwiftDependency{
					{Name: "GoogleGax", RequiredByServices: true},
					{Name: "GoogleAuth", RequiredByServices: true},
					{ApiPackage: "google.longrunning", Name: "GoogleCloudLongrunningV1"},
					{ApiPackage: "google.rpc", Name: "GoogleRpc"},
				}),
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			read := func(basename string) string {
				t.Helper()
				content, err := os.ReadFile(filepath.Join(outDir, "Sources", "Test", basename))
				if err != nil {
					t.Fatal(err)
				}
				return string(content)
			}

			// The client class and the protocol default implementations.
			serviceFile := read("TestService.swift")
			checkDiagnose(t, serviceFile, "  ", "public func simpleMethod(", test.wantPublic)
			// The protocol requirements.
			checkDiagnose(t, serviceFile, "    ", "func simpleMethod(", test.wantPublic)

			checkDiagnose(t, read("TestService+Stub.swift"), "    ",
				"func simpleMethod(", test.wantStub)
			for _, basename := range []string{
				"TestService+Retry.swift",
				"TestService+Logging.swift",
				"TestService+Transport.swift",
			} {
				checkDiagnose(t, read(basename), "    ", "public func simpleMethod(", test.wantStub)
			}
		})
	}
}

// TestGenerateService_DiagnoseRequestFields covers the declarations that name
// individual request fields.
//
// An overload assigns each selected request field by name, and a transport
// reads them to build the path and the query string, so both warn when a field
// is deprecated even though every type in their signatures is live. The
// declarations that only pass the request through must stay clean.
func TestGenerateService_DiagnoseRequestFields(t *testing.T) {
	for _, test := range []struct {
		name            string
		fieldDeprecated bool
	}{
		{name: "deprecated-request-field", fieldDeprecated: true},
		{name: "not-deprecated", fieldDeprecated: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			requestType := api.NewTestMessage("Request").
				WithFields(api.NewTestField("name").
					WithType(api.TypezString).
					WithDeprecated(test.fieldDeprecated))
			responseType := api.NewTestMessage("Response")
			operationType := api.NewTestMessage("Operation").WithPackage("google.longrunning")
			getOperationInputType := api.NewTestMessage("GetOperationRequest").
				WithPackage("google.longrunning")

			service := api.NewTestService("TestService").WithMethods(
				api.NewTestMethod("SimpleMethod").
					WithInput(requestType).
					WithOutput(responseType).
					WithVerb("POST").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("simple")).
					WithSignatures(&api.MethodSignature{Fields: []*api.Field{requestType.Fields[0]}}),
				api.NewTestMethod("GetOperation").
					WithInput(getOperationInputType).
					WithOutput(operationType).
					WithVerb("GET").
					WithPathTemplate((&api.PathTemplate{}).WithLiteral("v1").WithLiteral("operations")),
			)

			model := api.NewTestAPI([]*api.Message{
				requestType, responseType, operationType, getOperationInputType,
			}, nil, []*api.Service{service})
			model.PackageName = "test"

			library := &config.Library{
				Swift: swiftConfig(t, []config.SwiftDependency{
					{Name: "GoogleGax", RequiredByServices: true},
					{Name: "GoogleAuth", RequiredByServices: true},
					{ApiPackage: "google.longrunning", Name: "GoogleCloudLongrunningV1"},
					{ApiPackage: "google.rpc", Name: "GoogleRpc"},
				}),
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			read := func(basename string) string {
				t.Helper()
				content, err := os.ReadFile(filepath.Join(outDir, "Sources", "Test", basename))
				if err != nil {
					t.Fatal(err)
				}
				return string(content)
			}
			contentStr := read("TestService.swift")

			// The overload requirement and its default implementation.
			checkDiagnose(t, contentStr, "    ",
				"func simpleMethod(\n  name: Swift.String,", test.fieldDeprecated)
			checkDiagnose(t, contentStr, "  ",
				"public func simpleMethod(\n  name: Swift.String,", test.fieldDeprecated)

			// The declarations that only pass the request through.
			checkDiagnose(t, contentStr, "    ",
				"func simpleMethod(request: Request)", false)
			checkDiagnose(t, contentStr, "  ",
				"public func simpleMethod(\n    request: Request, options:", false)

			// The transport reads the request fields to build the path and the
			// query string.
			checkDiagnose(t, read("TestService+Transport.swift"), "    ",
				"public func simpleMethod(", test.fieldDeprecated)

			// The rest of the internal layer only names the request and
			// response types.
			for _, basename := range []string{
				"TestService+Retry.swift",
				"TestService+Logging.swift",
			} {
				checkDiagnose(t, read(basename), "    ", "public func simpleMethod(", false)
			}
			checkDiagnose(t, read("TestService+Stub.swift"), "    ", "func simpleMethod(", false)
		})
	}
}
