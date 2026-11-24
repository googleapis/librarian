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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateStub_GRPCClient_RoutingFieldTypes(t *testing.T) {
	outDir := t.TempDir()
	model := makeRoutingFieldTypesModel(t)

	module := &config.SwiftModule{
		Output:     outDir,
		ModuleType: "grpc-client",
		ModulePath: "RoutingTestProtos",
	}
	swiftPkg := swiftConfig(t, []config.SwiftDependency{
		{Name: "GoogleType", ApiPackage: "google.type"},
	})
	swiftPkg.PackageNameOverride = "RoutingTestLib"
	swiftPkg.LibraryNameOverride = "RoutingTestLib"
	library := &config.Library{
		Name:  "routing-test-lib",
		Swift: swiftPkg,
	}

	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	transportContent, err := os.ReadFile(filepath.Join(outDir, "RoutingTestService+Transport.swift"))
	if err != nil {
		t.Fatal(err)
	}
	transportStr := string(transportContent)

	for _, wantImport := range []string{"import GoogleType"} {
		if !strings.Contains(transportStr, wantImport) {
			t.Errorf("transport missing %q:\n%s", wantImport, transportStr)
		}
	}

	for _, test := range []struct {
		method string
		want   string
	}{
		{
			method: "explicitScalarsAndEnums",
			want: `    public func explicitScalarsAndEnums(
    request: RouteRequest, options: GoogleGax.RequestOptions
) async throws -> RouteResponse
 {
      let routingParams = GoogleGaxGRPC._RoutingMatcher.format([
        ("page_size",
          GoogleGaxGRPC._RoutingMatcher.value(
            Swift.String(request.pageSize),
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("opt_count",
          GoogleGaxGRPC._RoutingMatcher.value(
            request.optCount.map { Swift.String($0) },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("active",
          GoogleGaxGRPC._RoutingMatcher.value(
            Swift.String(request.active),
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("state",
          GoogleGaxGRPC._RoutingMatcher.value(
            request.state.stringValue ?? request.state.intValue.map { Swift.String($0) },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("opt_state",
          GoogleGaxGRPC._RoutingMatcher.value(
            request.optState.flatMap { $0.stringValue ?? $0.intValue.map { Swift.String($0) } },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("info_count",
          GoogleGaxGRPC._RoutingMatcher.value(
            (request.info?.count).map { Swift.String($0) },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("info_opt_state",
          GoogleGaxGRPC._RoutingMatcher.value(
            (request.info?.optState).flatMap { $0.stringValue ?? $0.intValue.map { Swift.String($0) } },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("ext_code",
          GoogleGaxGRPC._RoutingMatcher.value(
            (request.extDetail?.code).map { Swift.String($0) },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
      ])

      let protoRequest = try request.toProto()
      let protoResponse: RoutingTestProtos.Google_Cloud_Test_V1_RouteResponse = try await self.inner.execute(
        path: "/google.cloud.test.v1.RoutingTestService/ExplicitScalarsAndEnums",
        request: protoRequest,
        options: options,
        routingParams: routingParams
      )
      return try RouteResponse(proto: protoResponse)
    }`,
		},
		{
			method: "implicitHttpBindings",
			want: `    public func implicitHttpBindings(
    request: RouteRequest, options: GoogleGax.RequestOptions
) async throws -> RouteResponse
 {
      let routingParams = GoogleGaxGRPC._RoutingMatcher.format([
        ("name",
          GoogleGaxGRPC._RoutingMatcher.value(
            request.name,
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("page_size",
          GoogleGaxGRPC._RoutingMatcher.value(
            Swift.String(request.pageSize),
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("state",
          GoogleGaxGRPC._RoutingMatcher.value(
            request.state.stringValue ?? request.state.intValue.map { Swift.String($0) },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
      ])

      let protoRequest = try request.toProto()
      let protoResponse: RoutingTestProtos.Google_Cloud_Test_V1_RouteResponse = try await self.inner.execute(
        path: "/google.cloud.test.v1.RoutingTestService/ImplicitHttpBindings",
        request: protoRequest,
        options: options,
        routingParams: routingParams
      )
      return try RouteResponse(proto: protoResponse)
    }`,
		},
	} {
		t.Run(test.method, func(t *testing.T) {
			got := extractBlock(t, transportStr, "    public func "+test.method+"(", "\n    }")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateStub_GRPCClient_StringEnumsRouting(t *testing.T) {
	outDir := t.TempDir()
	modeEnum := api.NewTestEnum("Mode").
		WithPackage("google.cloud.test.v1").
		WithValues(api.NewTestEnumValue("MODE_FAST", 0))
	requestMsg := api.NewTestMessage("StringEnumRequest").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("mode").WithEnumType(modeEnum),
			api.NewTestField("opt_mode").WithEnumType(modeEnum).WithOptional(),
		)
	responseMsg := api.NewTestMessage("StringEnumResponse").WithPackage("google.cloud.test.v1")
	method := api.NewTestMethod("CheckMode").
		WithInput(requestMsg).
		WithOutput(responseMsg).
		WithRouting(
			newWildcardRoutingInfo("mode", "mode"),
			newWildcardRoutingInfo("opt_mode", "opt_mode"),
		)
	service := api.NewTestService("StringEnumService").
		WithPackage("google.cloud.test.v1").
		WithDefaultHost("test.googleapis.com").
		WithMethods(method)
	model := api.NewTestAPI([]*api.Message{requestMsg, responseMsg}, []*api.Enum{modeEnum}, []*api.Service{service}).
		WithPackageName("google.cloud.test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:                "routing-string-enum",
		SpecificationFormat: config.SpecDiscovery,
		Swift:               swiftConfig(t, nil),
	}
	module := &config.SwiftModule{
		Output:     outDir,
		ModuleType: "grpc-client",
		ModulePath: "StringEnumProtos",
	}
	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}
	transportContent, err := os.ReadFile(filepath.Join(outDir, "StringEnumService+Transport.swift"))
	if err != nil {
		t.Fatal(err)
	}
	got := extractBlock(t, string(transportContent), "    public func checkMode(", "\n    }")
	want := `    public func checkMode(
    request: StringEnumRequest, options: GoogleGax.RequestOptions
) async throws -> StringEnumResponse
 {
      let routingParams = GoogleGaxGRPC._RoutingMatcher.format([
        ("mode",
          GoogleGaxGRPC._RoutingMatcher.value(
            request.mode.stringValue,
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
        ("opt_mode",
          GoogleGaxGRPC._RoutingMatcher.value(
            request.optMode.map { $0.stringValue },
            prefix: [],
            matching: [.multiWildcard, ],
            suffix: []
          )
        ),
      ])

      let protoRequest = try request.toProto()
      let protoResponse: StringEnumProtos.Google_Cloud_Test_V1_StringEnumResponse = try await self.inner.execute(
        path: "/google.cloud.test.v1.StringEnumService/CheckMode",
        request: protoRequest,
        options: options,
        routingParams: routingParams
      )
      return try StringEnumResponse(proto: protoResponse)
    }`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func newWildcardRoutingInfo(key string, fieldPath ...string) *api.RoutingInfo {
	return api.NewTestRoutingInfo(
		key,
		api.NewRoutingInfoVariantBuilder().
			WithFieldPath(fieldPath...).
			WithMatchingSegments(api.MultiSegmentWildcard).
			Build(),
	)
}

func makeRoutingFieldTypesModel(t *testing.T) *api.API {
	t.Helper()

	stateEnum := api.NewTestEnum("State").
		WithPackage("google.cloud.test.v1").
		WithValues(
			api.NewTestEnumValue("STATE_UNSPECIFIED", 0),
			api.NewTestEnumValue("STATE_ACTIVE", 1),
		)

	extDetailMsg := api.NewTestMessage("ExtDetail").
		WithPackage("google.type").
		WithFields(
			api.NewTestField("code").WithType(api.TypezInt32),
		)

	innerMsg := api.NewTestMessage("Inner").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("count").WithType(api.TypezInt32),
			api.NewTestField("opt_count").WithType(api.TypezInt32).WithOptional(),
			api.NewTestField("state").WithEnumType(stateEnum),
			api.NewTestField("opt_state").WithEnumType(stateEnum).WithOptional(),
		)

	requestMsg := api.NewTestMessage("RouteRequest").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("page_size").WithType(api.TypezInt32),
			api.NewTestField("opt_count").WithType(api.TypezInt32).WithOptional(),
			api.NewTestField("active").WithType(api.TypezBool),
			api.NewTestField("state").WithEnumType(stateEnum),
			api.NewTestField("opt_state").WithEnumType(stateEnum).WithOptional(),
			api.NewTestField("info").WithMessageType(innerMsg).WithOptional(),
			api.NewTestField("ext_detail").WithMessageType(extDetailMsg).WithOptional(),
		)

	responseMsg := api.NewTestMessage("RouteResponse").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("result").WithType(api.TypezString),
		)

	explicitScalarsAndEnums := api.NewTestMethod("ExplicitScalarsAndEnums").
		WithInput(requestMsg).
		WithOutput(responseMsg).
		WithRouting(
			newWildcardRoutingInfo("page_size", "page_size"),
			newWildcardRoutingInfo("opt_count", "opt_count"),
			newWildcardRoutingInfo("active", "active"),
			newWildcardRoutingInfo("state", "state"),
			newWildcardRoutingInfo("opt_state", "opt_state"),
			newWildcardRoutingInfo("info_count", "info", "count"),
			newWildcardRoutingInfo("info_opt_state", "info", "opt_state"),
			newWildcardRoutingInfo("ext_code", "ext_detail", "code"),
		)

	implicitHTTPBindings := api.NewTestMethod("ImplicitHttpBindings").
		WithInput(requestMsg).
		WithOutput(responseMsg).
		WithBindings(
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("name").Build()),
			newTestPathBinding("GET", api.NewPathTemplateBuilder().WithLiteral("v1").WithVariableNamed("page_size").WithVariableNamed("state").Build()),
		)

	service := api.NewTestService("RoutingTestService").
		WithPackage("google.cloud.test.v1").
		WithDefaultHost("test.googleapis.com").
		WithMethods(explicitScalarsAndEnums, implicitHTTPBindings)

	model := api.NewTestAPI(
		[]*api.Message{innerMsg, requestMsg, responseMsg},
		[]*api.Enum{stateEnum},
		[]*api.Service{service},
	).WithPackageName("google.cloud.test.v1")
	model.AddMessage(extDetailMsg)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}
	return model
}
