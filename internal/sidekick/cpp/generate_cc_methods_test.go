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

func TestGenerate_CcMethods_RequestIdService(t *testing.T) {
	requireProtoc(t)

	srcs := &sources.Sources{
		Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
		ProtobufSrc: filepath.Join("testdata", "protos"),
	}
	sourceConfig := &sources.SourceConfig{
		Sources:     srcs,
		ActiveRoots: []string{"protobuf-src", "googleapis"},
		IncludeList: []string{"test_request_id.proto"},
	}
	modelConfig := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: "generator/integration_tests",
		ServiceConfig:       "generator/integration_tests/test_request_id.yaml",
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
		InitialCopyrightYear:          "2024",
		RetryableStatusCodes:          []string{"kUnavailable"},
		GenAsyncRPCs:                  []string{"CreateFoo"},
		OverrideServiceConfigYAMLName: "generator/integration_tests/test_request_id.yaml",
	}

	if err := Generate(t.Context(), model, outdir, libCfg); err != nil {
		t.Fatal(err)
	}

	// formatWithClangFormat(t, outdir)

	goldenDir := filepath.Join("testdata", "golden", "v1")
	prefix := libCfg.ProductPath + "/"
	svc := "RequestIdService"

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

	// 1. client.cc
	{
		got, golden := readGenAndGolden(ClientSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("client.cc methods", got, golden,
			"RequestIdServiceClient::RequestIdServiceClient(",
			"return connection_->AsyncCreateFoo(request);\n}\n")
	}

	// 2. connection.cc
	{
		got, golden := readGenAndGolden(ConnectionSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("connection.cc methods", got, golden,
			"RequestIdServiceConnection::~RequestIdServiceConnection() = default;\n",
			"MakeRequestIdServiceTracingConnection(\n      std::make_shared<golden_v1_internal::RequestIdServiceConnectionImpl>(\n      std::move(background), std::move(stub), std::move(options)));\n}\n")
	}

	// 3. connection_idempotency_policy.cc
	{
		got, golden := readGenAndGolden(IdempotencyPolicySourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_idempotency_policy.cc methods", got, golden,
			"RequestIdServiceConnectionIdempotencyPolicy::~RequestIdServiceConnectionIdempotencyPolicy() = default;\n",
			"return std::make_unique<RequestIdServiceConnectionIdempotencyPolicy>();\n}\n")
	}

	// 4. option_defaults.cc
	{
		got, golden := readGenAndGolden(OptionDefaultsSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("option_defaults.cc methods", got, golden,
			"namespace {\n",
			"return options;\n}\n")
	}

	// 5. stub.cc
	{
		got, golden := readGenAndGolden(StubSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("stub.cc methods", got, golden,
			"RequestIdServiceStub::~RequestIdServiceStub() = default;\n",
			"      .then([](future<StatusOr<google::protobuf::Empty>> f) {\n        return f.get().status();\n      });\n}\n")
	}

	// 6. stub_factory.cc
	{
		got, golden := readGenAndGolden(StubFactorySourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("stub_factory.cc methods", got, golden,
			"std::shared_ptr<RequestIdServiceStub>\nCreateDefaultRequestIdServiceStub(",
			"return stub;\n}\n")
	}

	// 7. auth_decorator.cc
	{
		got, golden := readGenAndGolden(AuthDecoratorSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("auth_decorator.cc methods", got, golden,
			"RequestIdServiceAuth::RequestIdServiceAuth(",
			"            cq, *std::move(context), std::move(options), request);\n      });\n}\n")
	}

	// 8. logging_decorator.cc
	{
		got, golden := readGenAndGolden(LoggingDecoratorSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("logging_decorator.cc methods", got, golden,
			"RequestIdServiceLogging::RequestIdServiceLogging(",
			"        return child_->AsyncCancelOperation(\n            cq, std::move(context), std::move(options), request);\n      },\n      cq, std::move(context), std::move(options), request, __func__,\n      tracing_options_);\n}\n")
	}

	// 9. metadata_decorator.cc
	{
		got, golden := readGenAndGolden(MetadataDecoratorSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("metadata_decorator.cc methods", got, golden,
			"RequestIdServiceMetadata::RequestIdServiceMetadata(",
			"google::cloud::internal::SetMetadata(\n      context, options, fixed_metadata_, api_client_header_);\n}\n")
	}

	// 10. tracing_connection.cc
	{
		got, golden := readGenAndGolden(TracingConnectionSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_connection.cc methods", got, golden,
			"RequestIdServiceTracingConnection::RequestIdServiceTracingConnection(",
			"    conn = std::make_shared<RequestIdServiceTracingConnection>(std::move(conn));\n  }\n#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n  return conn;\n}\n")
	}

	// 11. tracing_stub.cc
	{
		got, golden := readGenAndGolden(TracingStubSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_stub.cc methods", got, golden,
			"RequestIdServiceTracingStub::RequestIdServiceTracingStub(",
			"MakeRequestIdServiceTracingStub(\n    std::shared_ptr<RequestIdServiceStub> stub) {\n#ifdef GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n  return std::make_shared<RequestIdServiceTracingStub>(std::move(stub));\n#else\n  return stub;\n#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n}\n")
	}

	// 12. connection_impl.cc
	{
		got, golden := readGenAndGolden(ConnectionImplSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_impl.cc methods", got, golden,
			"namespace {\n",
			"std::move(current), std::move(request_copy), __func__);\n}\n")
	}

	// 13. sources.cc
	{
		got, golden := readGenAndGolden(SourcesSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("sources.cc includes", got, golden,
			"// NOLINTBEGIN(bugprone-suspicious-include)\n",
			"// NOLINTEND(bugprone-suspicious-include)\n")
	}
}

func TestGenerate_CcMethods_DeprecatedService(t *testing.T) {
	requireProtoc(t)

	srcs := &sources.Sources{
		Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
		ProtobufSrc: filepath.Join("testdata", "protos"),
	}
	sourceConfig := &sources.SourceConfig{
		Sources:     srcs,
		ActiveRoots: []string{"protobuf-src", "googleapis"},
		IncludeList: []string{"test_deprecated.proto"},
	}
	modelConfig := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: "generator/integration_tests",
		ServiceConfig:       "",
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
		ProductPath:          "generator/integration_tests/golden/v1",
		InitialCopyrightYear: "2024",
		RetryableStatusCodes: []string{"kUnavailable"},
	}

	if err := Generate(t.Context(), model, outdir, libCfg); err != nil {
		t.Fatal(err)
	}

	goldenDir := filepath.Join("testdata", "golden", "v1")
	prefix := libCfg.ProductPath + "/"
	svc := "DeprecatedService"

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

	// 1. client.cc
	{
		got, golden := readGenAndGolden(ClientSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("deprecated_client.cc methods", got, golden,
			"DeprecatedServiceClient::DeprecatedServiceClient(",
			"return connection_->Noop(request);\n}\n")
	}

	// 2. connection.cc
	{
		got, golden := readGenAndGolden(ConnectionSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("deprecated_connection.cc methods", got, golden,
			"DeprecatedServiceConnection::~DeprecatedServiceConnection() = default;\n",
			"MakeDeprecatedServiceTracingConnection(\n      std::make_shared<golden_v1_internal::DeprecatedServiceConnectionImpl>(\n      std::move(background), std::move(stub), std::move(options)));\n}\n")
	}

	// 3. connection_idempotency_policy.cc
	{
		got, golden := readGenAndGolden(IdempotencyPolicySourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("deprecated_connection_idempotency_policy.cc methods", got, golden,
			"DeprecatedServiceConnectionIdempotencyPolicy::~DeprecatedServiceConnectionIdempotencyPolicy() = default;\n",
			"return std::make_unique<DeprecatedServiceConnectionIdempotencyPolicy>();\n}\n")
	}

	// 4. option_defaults.cc
	{
		got, golden := readGenAndGolden(OptionDefaultsSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_option_defaults.cc methods", got, golden,
			"namespace {\n",
			"return options;\n}\n")
	}

	// 5. stub.cc
	{
		got, golden := readGenAndGolden(StubSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_stub.cc methods", got, golden,
			"DeprecatedServiceStub::~DeprecatedServiceStub() = default;\n",
			"return google::cloud::Status();\n}\n")
	}

	// 6. stub_factory.cc
	{
		got, golden := readGenAndGolden(StubFactorySourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_stub_factory.cc methods", got, golden,
			"std::shared_ptr<DeprecatedServiceStub>\nCreateDefaultDeprecatedServiceStub(",
			"return stub;\n}\n")
	}

	// 7. auth_decorator.cc
	{
		got, golden := readGenAndGolden(AuthDecoratorSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_auth_decorator.cc methods", got, golden,
			"DeprecatedServiceAuth::DeprecatedServiceAuth(",
			"return child_->Noop(context, options, request);\n}\n")
	}

	// 8. logging_decorator.cc
	{
		got, golden := readGenAndGolden(LoggingDecoratorSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_logging_decorator.cc methods", got, golden,
			"DeprecatedServiceLogging::DeprecatedServiceLogging(",
			"return child_->Noop(context, options, request);\n      },\n      context, options, request, __func__, tracing_options_);\n}\n")
	}

	// 9. metadata_decorator.cc
	{
		got, golden := readGenAndGolden(MetadataDecoratorSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_metadata_decorator.cc methods", got, golden,
			"DeprecatedServiceMetadata::DeprecatedServiceMetadata(",
			"google::cloud::internal::SetMetadata(\n      context, options, fixed_metadata_, api_client_header_);\n}\n")
	}

	// 10. tracing_connection.cc
	{
		got, golden := readGenAndGolden(TracingConnectionSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_tracing_connection.cc methods", got, golden,
			"DeprecatedServiceTracingConnection::DeprecatedServiceTracingConnection(",
			"    conn = std::make_shared<DeprecatedServiceTracingConnection>(std::move(conn));\n  }\n#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n  return conn;\n}\n")
	}

	// 11. tracing_stub.cc
	{
		got, golden := readGenAndGolden(TracingStubSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_tracing_stub.cc methods", got, golden,
			"DeprecatedServiceTracingStub::DeprecatedServiceTracingStub(",
			"MakeDeprecatedServiceTracingStub(\n    std::shared_ptr<DeprecatedServiceStub> stub) {\n#ifdef GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n  return std::make_shared<DeprecatedServiceTracingStub>(std::move(stub));\n#else\n  return stub;\n#endif  // GOOGLE_CLOUD_CPP_HAVE_OPENTELEMETRY\n}\n")
	}

	// 12. connection_impl.cc
	{
		got, golden := readGenAndGolden(ConnectionImplSourcePath(libCfg.ProductPath, svc))
		assertBlockMatch("internal/deprecated_connection_impl.cc methods", got, golden,
			"namespace {\n",
			"return stub_->Noop(context, options, request);\n      },\n      *current, request, __func__);\n}\n")
	}

}
