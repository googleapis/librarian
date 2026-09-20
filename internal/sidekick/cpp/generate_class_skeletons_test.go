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

package cpp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

func TestGenerate_GRPCServiceFiles_ClassSkeletons(t *testing.T) {
	requireProtoc(t)

	for _, test := range []struct {
		name          string
		serviceName   string
		includeList   []string
		serviceConfig string
		year          string
		retryCodes    []string
		asyncRPCs     []string
		ns            string
		internalNs    string
		mocksNs       string
		hasLRO        bool
		hasStreaming  bool
		hasRequestId  bool
		isDeprecated  bool
	}{
		{
			name:          "request_id_service",
			serviceName:   "RequestIdService",
			includeList:   []string{"test_request_id.proto"},
			serviceConfig: "generator/integration_tests/test_request_id.yaml",
			year:          "2024",
			retryCodes:    []string{"kUnavailable"},
			asyncRPCs:     []string{"CreateFoo"},
			ns:            "golden_v1",
			internalNs:    "golden_v1_internal",
			mocksNs:       "golden_v1_mocks",
			hasLRO:        true,
			hasStreaming:  false,
			hasRequestId:  true,
			isDeprecated:  false,
		},
		{
			name:          "golden_thing_admin",
			serviceName:   "GoldenThingAdmin",
			includeList:   []string{"test.proto", "backup.proto"},
			serviceConfig: "generator/integration_tests/test.yaml",
			year:          "2022",
			retryCodes:    []string{"kDeadlineExceeded", "kUnavailable"},
			ns:            "golden_v1",
			internalNs:    "golden_v1_internal",
			mocksNs:       "golden_v1_mocks",
			hasLRO:        true,
			hasStreaming:  false,
			hasRequestId:  false,
			isDeprecated:  false,
		},
		{
			name:          "golden_kitchen_sink",
			serviceName:   "GoldenKitchenSink",
			includeList:   []string{"test.proto", "backup.proto"},
			serviceConfig: "generator/integration_tests/test.yaml",
			year:          "2022",
			retryCodes:    []string{"kInternal", "kUnavailable"},
			ns:            "golden_v1",
			internalNs:    "golden_v1_internal",
			mocksNs:       "golden_v1_mocks",
			hasLRO:        false,
			hasStreaming:  true,
			hasRequestId:  false,
			isDeprecated:  false,
		},
		{
			name:          "deprecated_service",
			serviceName:   "DeprecatedService",
			includeList:   []string{"test_deprecated.proto"},
			serviceConfig: "",
			year:          "2024",
			retryCodes:    []string{"kUnavailable"},
			ns:            "golden_v1",
			internalNs:    "golden_v1_internal",
			mocksNs:       "golden_v1_mocks",
			hasLRO:        false,
			hasStreaming:  false,
			hasRequestId:  false,
			isDeprecated:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			srcs := &sources.Sources{
				Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
				ProtobufSrc: filepath.Join("testdata", "protos"),
			}
			sourceConfig := &sources.SourceConfig{
				Sources:     srcs,
				ActiveRoots: []string{"protobuf-src", "googleapis"},
				IncludeList: test.includeList,
			}
			modelConfig := &parser.ModelConfig{
				Language:            config.LanguageCpp,
				SpecificationFormat: config.SpecProtobuf,
				SpecificationSource: "generator/integration_tests",
				ServiceConfig:       test.serviceConfig,
				Source:              sourceConfig,
			}

			model, err := parser.CreateModel(modelConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := api.Validate(model); err != nil {
				t.Fatal(err)
			}

			outdir := t.TempDir()
			libCfg := &config.CppLibrary{
				ProductPath:                   "generator/integration_tests/golden/v1",
				InitialCopyrightYear:          test.year,
				RetryableStatusCodes:          test.retryCodes,
				GenAsyncRPCs:                  test.asyncRPCs,
				OverrideServiceConfigYAMLName: test.serviceConfig,
			}

			if err := Generate(t.Context(), model, outdir, libCfg); err != nil {
				t.Fatal(err)
			}

			goldenDir := filepath.Join("testdata", "golden", "v1")
			prefix := libCfg.ProductPath + "/"

			readGenAndGolden := func(outputPath string) (string, string) {
				fullPath := filepath.Join(outdir, outputPath)
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Fatalf("missing expected generated file %q: %v", outputPath, err)
				}
				relPath := strings.TrimPrefix(outputPath, prefix)
				goldenPath := filepath.Join(goldenDir, relPath)
				goldenContentBytes, err := os.ReadFile(goldenPath)
				if err != nil {
					t.Fatalf("missing golden file %q: %v", goldenPath, err)
				}
				return string(content), string(goldenContentBytes)
			}

			assertBlockMatch := func(fileDesc, got, golden, startStr, endStr string) {
				t.Helper()
				gotBlock := extractBlock(t, got, startStr, endStr)
				wantBlock := extractBlock(t, golden, startStr, endStr)
				if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
					t.Logf("%s", fileDesc)
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}

			svc := test.serviceName
			clientClass := ClientClassName(svc)
			connClass := ConnectionClassName(svc)
			connImplClass := ConnectionImplClassName(svc)
			idempotencyClass := ConnectionIdempotencyPolicyClassName(svc)
			mockConnClass := MockConnectionClassName(svc)
			stubClass := StubClassName(svc)
			defaultStubClass := DefaultStubClassName(svc)
			authClass := AuthDecoratorClassName(svc)
			loggingClass := LoggingDecoratorClassName(svc)
			metadataClass := MetadataDecoratorClassName(svc)
			tracingConnClass := TracingConnectionClassName(svc)
			tracingStubClass := TracingStubClassName(svc)
			retryPolicy := RetryPolicyName(svc)
			limitedErrorCount := LimitedErrorCountRetryPolicyName(svc)
			limitedTime := LimitedTimeRetryPolicyName(svc)
			retryTraits := RetryTraitsName(svc)

			// 1. client.h
			{
				got, golden := readGenAndGolden(ClientHeaderPath(libCfg.ProductPath, svc))
				clientDeclStart := "class " + clientClass + " {\n public:\n  explicit " + clientClass + "(std::shared_ptr<" + connClass + "> connection, Options opts = {});\n  ~" + clientClass + "();"
				if test.isDeprecated {
					clientDeclStart = "class\n GOOGLE_CLOUD_CPP_DEPRECATED(\n      \"" + svc + " has been deprecated and will be turned down in the future.\"\n)\n" + clientClass + " {\n public:\n  explicit " + clientClass + "(std::shared_ptr<" + connClass + "> connection, Options opts = {});\n  ~" + clientClass + "();"
				}
				assertBlockMatch("client.h class decl", got, golden, clientDeclStart, "~"+clientClass+"();")
				assertBlockMatch("client.h copy move", got, golden, "  ///@{\n  /// @name Copy and move support\n  "+clientClass+"("+clientClass+" const&) = default;", "  ///@}\n")
				assertBlockMatch("client.h equality", got, golden, "  ///@{\n  /// @name Equality\n  friend bool operator==("+clientClass+" const& a, "+clientClass+" const& b)", "  ///@}\n")
				assertBlockMatch("client.h private members", got, golden, " private:\n  std::shared_ptr<"+connClass+"> connection_;\n  Options options_;\n};", "};")
			}

			// 2. client.cc
			{
				got, golden := readGenAndGolden(ClientSourcePath(libCfg.ProductPath, svc))
				assertBlockMatch("client.cc constructor destructor", got, golden, clientClass+"::"+clientClass+"(\n    std::shared_ptr<"+connClass+"> connection, Options opts)\n    : connection_(std::move(connection)),", clientClass+"::~"+clientClass+"() = default;\n")
			}

			// 3. connection.h
			{
				got, golden := readGenAndGolden(ConnectionHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("connection.h retry policy", got, golden, "class "+retryPolicy+" : public ::google::cloud::RetryPolicy {", "};\n")
				assertBlockMatch("connection.h limited error count", got, golden, "class "+limitedErrorCount+" : public "+retryPolicy+" {", "};\n")
				assertBlockMatch("connection.h limited time", got, golden, "class "+limitedTime+" : public "+retryPolicy+" {", "};\n")
				assertBlockMatch("connection.h connection class", got, golden, "class "+connClass+" {\n public:\n  virtual ~"+connClass+"() = 0;\n\n  virtual Options options() { return Options{}; }\n", "virtual Options options() { return Options{}; }\n")
				assertBlockMatch("connection.h make connection", got, golden, "std::shared_ptr<"+connClass+"> Make"+connClass+"(\n    Options options = {});", "Options options = {});")
			}

			// 4. connection.cc
			{
				got, golden := readGenAndGolden(ConnectionSourcePath(libCfg.ProductPath, svc))
				assertBlockMatch("connection.cc destructor", got, golden, connClass+"::~"+connClass+"() = default;\n", connClass+"::~"+connClass+"() = default;\n")
			}

			// 5. connection_idempotency_policy.h
			{
				got, golden := readGenAndGolden(IdempotencyPolicyHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("connection_idempotency_policy.h class", got, golden, "class "+idempotencyClass+" {\n public:\n  virtual ~"+idempotencyClass+"();\n\n  /// Create a new copy of this object.\n  virtual std::unique_ptr<"+idempotencyClass+"> clone() const;", "clone() const;")
				assertBlockMatch("connection_idempotency_policy.h factory", got, golden, "std::unique_ptr<"+idempotencyClass+">\n    MakeDefault"+idempotencyClass+"();", "();")
			}

			// 6. connection_idempotency_policy.cc
			{
				got, golden := readGenAndGolden(IdempotencyPolicySourcePath(libCfg.ProductPath, svc))
				assertBlockMatch("connection_idempotency_policy.cc using", got, golden, "using ::google::cloud::Idempotency;\n", "using ::google::cloud::Idempotency;\n")
				assertBlockMatch("connection_idempotency_policy.cc destructor", got, golden, idempotencyClass+"::~"+idempotencyClass+"() = default;\n", idempotencyClass+"::~"+idempotencyClass+"() = default;\n")
				assertBlockMatch("connection_idempotency_policy.cc clone", got, golden, "std::unique_ptr<"+idempotencyClass+">\n"+idempotencyClass+"::clone() const {\n  return std::make_unique<"+idempotencyClass+">(*this);\n}", "}")
				assertBlockMatch("connection_idempotency_policy.cc factory", got, golden, "std::unique_ptr<"+idempotencyClass+">\n    MakeDefault"+idempotencyClass+"() {\n  return std::make_unique<"+idempotencyClass+">();\n}", "}")
			}

			// 7. options.h
			{
				got, golden := readGenAndGolden(OptionsHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("options.h retry option", got, golden, "struct "+retryPolicy+"Option {\n  using Type = std::shared_ptr<"+retryPolicy+">;\n};", "};")
				assertBlockMatch("options.h backoff option", got, golden, "struct "+svc+"BackoffPolicyOption {\n  using Type = std::shared_ptr<BackoffPolicy>;\n};", "};")
				assertBlockMatch("options.h idempotency option", got, golden, "struct "+idempotencyClass+"Option {\n  using Type = std::shared_ptr<"+idempotencyClass+">;\n};", "};")
				if test.hasLRO {
					assertBlockMatch("options.h polling option", got, golden, "struct "+svc+"PollingPolicyOption {\n  using Type = std::shared_ptr<PollingPolicy>;\n};", "};")
					assertBlockMatch("options.h policy option list", got, golden, "using "+svc+"PolicyOptionList =\n    OptionList<"+retryPolicy+"Option,\n               "+svc+"BackoffPolicyOption,\n               "+svc+"PollingPolicyOption,\n               "+idempotencyClass+"Option>;", ">;")
				} else {
					assertBlockMatch("options.h policy option list", got, golden, "using "+svc+"PolicyOptionList =\n    OptionList<"+retryPolicy+"Option,\n               "+svc+"BackoffPolicyOption,\n               "+idempotencyClass+"Option>;", ">;")
				}
			}

			// 8. mock_connection.h
			{
				got, golden := readGenAndGolden(MockConnectionHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("mock_connection.h class", got, golden, "class "+mockConnClass+" : public "+test.ns+"::"+connClass+" {\n public:\n  MOCK_METHOD(Options, options, (), (override));\n", "MOCK_METHOD(Options, options, (), (override));\n")
			}

			// 9. retry_traits.h
			{
				got, golden := readGenAndGolden(RetryTraitsHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("retry_traits.h struct", got, golden, "struct "+retryTraits+" {\n  static bool IsPermanentFailure(google::cloud::Status const& status) {", "}")
			}

			// 10. option_defaults.h
			{
				got, golden := readGenAndGolden(OptionDefaultsHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("option_defaults.h decl", got, golden, "Options "+svc+"DefaultOptions(Options options);", ";")
			}

			// 11. connection_impl.h
			{
				got, golden := readGenAndGolden(ConnectionImplHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("connection_impl.h class", got, golden, "class "+connImplClass+"\n    : public "+test.ns+"::"+connClass+" {\n public:\n  ~"+connImplClass+"() override = default;", "~"+connImplClass+"() override = default;")
				assertBlockMatch("connection_impl.h constructor", got, golden, "  "+connImplClass+"(\n    std::unique_ptr<google::cloud::BackgroundThreads> background,\n    std::shared_ptr<"+test.internalNs+"::"+stubClass+"> stub,\n    Options options);", "options);")
				assertBlockMatch("connection_impl.h options", got, golden, "  Options options() override { return options_; }\n", "options_; }\n")
				connImplPrivate := " private:\n  std::unique_ptr<google::cloud::BackgroundThreads> background_;\n  std::shared_ptr<" + test.internalNs + "::" + stubClass + "> stub_;\n  Options options_;\n};"
				if test.hasRequestId {
					connImplPrivate = " private:\n  std::unique_ptr<google::cloud::BackgroundThreads> background_;\n  std::shared_ptr<" + test.internalNs + "::" + stubClass + "> stub_;\n  Options options_;\n  std::shared_ptr<google::cloud::internal::InvocationIdGenerator>\n      invocation_id_generator_ =\n          std::make_shared<google::cloud::internal::InvocationIdGenerator>();\n};"
				}
				assertBlockMatch("connection_impl.h private members", got, golden, connImplPrivate, "};")
			}

			// 12. stub.h
			{
				got, golden := readGenAndGolden(StubHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("stub.h stub class", got, golden, "class "+stubClass+" {\n public:\n  virtual ~"+stubClass+"() = 0;", "virtual ~"+stubClass+"() = 0;")
				assertBlockMatch("stub.h default stub class", got, golden, "class "+defaultStubClass+" : public "+stubClass+" {", stubClass+" {")
			}

			// 13. stub.cc
			{
				got, golden := readGenAndGolden(StubSourcePath(libCfg.ProductPath, svc))
				assertBlockMatch("stub.cc destructor", got, golden, stubClass+"::~"+stubClass+"() = default;\n", stubClass+"::~"+stubClass+"() = default;\n")
			}

			// 14. stub_factory.h
			{
				got, golden := readGenAndGolden(StubFactoryHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("stub_factory.h decl", got, golden, "std::shared_ptr<"+stubClass+"> CreateDefault"+stubClass+"(\n    std::shared_ptr<internal::GrpcAuthenticationStrategy> auth,\n    Options const& options);", ");")
			}

			// 15. auth_decorator.h
			{
				got, golden := readGenAndGolden(AuthDecoratorHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("auth_decorator.h class", got, golden, "class "+authClass+" : public "+stubClass+" {\n public:\n  ~"+authClass+"() override = default;\n  "+authClass+"(\n      std::shared_ptr<google::cloud::internal::GrpcAuthenticationStrategy> auth,\n      std::shared_ptr<"+stubClass+"> child);", "child);")
				assertBlockMatch("auth_decorator.h private members", got, golden, " private:\n  std::shared_ptr<google::cloud::internal::GrpcAuthenticationStrategy> auth_;\n  std::shared_ptr<"+stubClass+"> child_;\n};", "};")
			}

			// 16. logging_decorator.h
			{
				got, golden := readGenAndGolden(LoggingDecoratorHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("logging_decorator.h class", got, golden, "class "+loggingClass+" : public "+stubClass+" {\n public:\n  ~"+loggingClass+"() override = default;\n  "+loggingClass+"(std::shared_ptr<"+stubClass+"> child,\n                       TracingOptions tracing_options,\n                       std::set<std::string> const& components);", "components);")
				loggingPrivateMembers := " private:\n  std::shared_ptr<" + stubClass + "> child_;\n  TracingOptions tracing_options_;\n};  // " + loggingClass
				if test.hasStreaming {
					loggingPrivateMembers = " private:\n  std::shared_ptr<" + stubClass + "> child_;\n  TracingOptions tracing_options_;\n  bool stream_logging_;\n};  // " + loggingClass
				}
				assertBlockMatch("logging_decorator.h private members", got, golden, loggingPrivateMembers, loggingClass)
			}

			// 17. metadata_decorator.h
			{
				got, golden := readGenAndGolden(MetadataDecoratorHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("metadata_decorator.h class", got, golden, "class "+metadataClass+" : public "+stubClass+" {\n public:\n  ~"+metadataClass+"() override = default;\n  "+metadataClass+"(\n      std::shared_ptr<"+stubClass+"> child,\n      std::multimap<std::string, std::string> fixed_metadata,\n      std::string api_client_header = \"\");", "api_client_header = \"\");")
				assertBlockMatch("metadata_decorator.h set metadata", got, golden, "  void SetMetadata(grpc::ClientContext& context,\n                   Options const& options,\n                   std::string const& request_params);\n  void SetMetadata(grpc::ClientContext& context, Options const& options);", "Options const& options);")
				assertBlockMatch("metadata_decorator.h private members", got, golden, "  std::shared_ptr<"+stubClass+"> child_;\n  std::multimap<std::string, std::string> fixed_metadata_;\n  std::string api_client_header_;\n};", "};")
			}

			// 18. tracing_connection.h
			{
				got, golden := readGenAndGolden(TracingConnectionHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("tracing_connection.h class", got, golden, "#ifdef GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n\nclass "+tracingConnClass+"\n    : public "+test.ns+"::"+connClass+" {\n public:\n  ~"+tracingConnClass+"() override = default;\n\n  explicit "+tracingConnClass+"(\n    std::shared_ptr<"+test.ns+"::"+connClass+"> child);\n\n  Options options() override { return child_->options(); }", "child_->options(); }")
				assertBlockMatch("tracing_connection.h private members", got, golden, " private:\n  std::shared_ptr<"+test.ns+"::"+connClass+"> child_;\n};\n\n#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY", "#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY")
				assertBlockMatch("tracing_connection.h factory", got, golden, "std::shared_ptr<"+test.ns+"::"+connClass+">\nMake"+tracingConnClass+"(\n    std::shared_ptr<"+test.ns+"::"+connClass+"> conn);", "conn);")
			}

			// 19. tracing_stub.h
			{
				got, golden := readGenAndGolden(TracingStubHeaderPath(libCfg.ProductPath, svc))
				assertBlockMatch("tracing_stub.h class", got, golden, "#ifdef GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n\nclass "+tracingStubClass+" : public "+stubClass+" {\n public:\n  ~"+tracingStubClass+"() override = default;\n\n  explicit "+tracingStubClass+"(std::shared_ptr<"+stubClass+"> child);", "child);")
				assertBlockMatch("tracing_stub.h private members", got, golden, " private:\n  std::shared_ptr<"+stubClass+"> child_;\n  std::shared_ptr<opentelemetry::context::propagation::TextMapPropagator> propagator_;\n};\n\n#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY", "#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY")
				assertBlockMatch("tracing_stub.h factory", got, golden, "std::shared_ptr<"+stubClass+"> Make"+tracingStubClass+"(\n    std::shared_ptr<"+stubClass+"> stub);", "stub);")
			}
		})
	}
}
