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

func TestGenerate_Methods_RequestIdService(t *testing.T) {
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

	svc := "RequestIdService"

	// 1. client.h
	{
		got, golden := readGenAndGolden(ClientHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("client.h methods", got, golden,
			"  // clang-format off\n  ///\n  /// Creates a `Foo` resource.",
			"AsyncCreateFoo(google::test::requestid::v1::CreateFooRequest const& request, Options opts = {});\n")
	}

	// 2. connection.h
	{
		got, golden := readGenAndGolden(ConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection.h methods", got, golden,
			"  virtual StatusOr<google::test::requestid::v1::Foo>\n  CreateFoo",
			"AsyncCreateFoo(google::test::requestid::v1::CreateFooRequest const& request);\n};\n")
	}

	// 3. connection_impl.h
	{
		got, golden := readGenAndGolden(ConnectionImplHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_impl.h methods", got, golden,
			"  StatusOr<google::test::requestid::v1::Foo>\n  CreateFoo",
			"AsyncCreateFoo(google::test::requestid::v1::CreateFooRequest const& request) override;\n\n private:")
	}

	// 4. tracing_connection.h
	{
		got, golden := readGenAndGolden(TracingConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_connection.h methods", got, golden,
			"  StatusOr<google::test::requestid::v1::Foo>\n  CreateFoo",
			"AsyncCreateFoo(google::test::requestid::v1::CreateFooRequest const& request) override;\n\n private:")
	}

	// 5. mock_connection.h
	{
		got, golden := readGenAndGolden(MockConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("mock_connection.h methods", got, golden,
			"  MOCK_METHOD(StatusOr<google::test::requestid::v1::Foo>,\n  CreateFoo,",
			"AsyncCreateFoo,\n  (google::test::requestid::v1::CreateFooRequest const& request), (override));\n};\n")
	}

	// 6. connection_idempotency_policy.h
	{
		got, golden := readGenAndGolden(IdempotencyPolicyHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_idempotency_policy.h methods", got, golden,
			"  virtual google::cloud::Idempotency\n  CreateFoo",
			"ListFoos(google::test::requestid::v1::ListFoosRequest request);\n};\n")
	}

	// 7. stub.h (Stub interface and DefaultStub)
	{
		got, golden := readGenAndGolden(StubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("stub.h Stub methods", got, golden,
			"  virtual StatusOr<google::test::requestid::v1::Foo> CreateFoo(",
			"CancelOperationRequest const& request) = 0;\n};\n")
		assertBlockMatch("stub.h DefaultStub methods", got, golden,
			"  StatusOr<google::test::requestid::v1::Foo> CreateFoo(",
			"CancelOperationRequest const& request) override;")
	}

	// 8. auth_decorator.h
	{
		got, golden := readGenAndGolden(AuthDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("auth_decorator.h methods", got, golden,
			"  StatusOr<google::test::requestid::v1::Foo> CreateFoo(",
			"CancelOperationRequest const& request) override;\n\n private:")
	}

	// 9. logging_decorator.h
	{
		got, golden := readGenAndGolden(LoggingDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("logging_decorator.h methods", got, golden,
			"  StatusOr<google::test::requestid::v1::Foo> CreateFoo(",
			"CancelOperationRequest const& request) override;\n\n private:")
	}

	// 10. metadata_decorator.h
	{
		got, golden := readGenAndGolden(MetadataDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("metadata_decorator.h methods", got, golden,
			"  StatusOr<google::test::requestid::v1::Foo> CreateFoo(",
			"CancelOperationRequest const& request) override;\n\n private:")
	}

	// 11. tracing_stub.h
	{
		got, golden := readGenAndGolden(TracingStubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_stub.h methods", got, golden,
			"  StatusOr<google::test::requestid::v1::Foo> CreateFoo(",
			"CancelOperationRequest const& request) override;\n\n private:")
	}
}

func TestGenerate_Methods_GoldenKitchenSink(t *testing.T) {
	requireProtoc(t)

	srcs := &sources.Sources{
		Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
		ProtobufSrc: filepath.Join("testdata", "protos"),
	}
	sourceConfig := &sources.SourceConfig{
		Sources:     srcs,
		ActiveRoots: []string{"protobuf-src", "googleapis"},
		IncludeList: []string{"test.proto", "backup.proto"},
	}
	modelConfig := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: "generator/integration_tests",
		ServiceConfig:       "generator/integration_tests/test.yaml",
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
		InitialCopyrightYear:          "2022",
		RetryableStatusCodes:          []string{"GoldenKitchenSink.kInternal", "kUnavailable", "GoldenThingAdmin.kDeadlineExceeded"},
		OverrideServiceConfigYAMLName: "generator/integration_tests/test.yaml",
		OmittedRPCs: []string{
			"Omitted1",
			"GoldenKitchenSink.Omitted2",
			"Deprecated1",
			"Deprecated2(std::string const&)",
		},
		GenAsyncRPCs: []string{
			"GetDatabase",
			"DropDatabase",
			"StreamingRead",
			"StreamingWrite",
		},
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

	svc := "GoldenKitchenSink"

	// 1. client.h
	{
		got, golden := readGenAndGolden(ClientHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("client.h methods", got, golden,
			"  // clang-format off\n  ///\n  /// Generates an OAuth 2.0 access token for a service account.",
			"ListOperations(google::longrunning::ListOperationsRequest request, Options opts = {});\n")
	}

	// 2. connection.h
	{
		got, golden := readGenAndGolden(ConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection.h methods", got, golden,
			"  virtual StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\n  GenerateAccessToken",
			"ListOperations(google::longrunning::ListOperationsRequest request);\n};\n")
	}

	// 3. connection_impl.h
	{
		got, golden := readGenAndGolden(ConnectionImplHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_impl.h methods", got, golden,
			"void GoldenKitchenSinkStreamingReadStreamingUpdater(\n    google::test::admin::database::v1::Response const& response,\n    google::test::admin::database::v1::Request& request);\n\nclass GoldenKitchenSinkConnectionImpl",
			"ListOperations(google::longrunning::ListOperationsRequest request) override;\n\n private:")
	}

	// 4. tracing_connection.h
	{
		got, golden := readGenAndGolden(TracingConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_connection.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>\n  GenerateAccessToken",
			"ListOperations(google::longrunning::ListOperationsRequest request) override;\n\n private:")
	}

	// 5. mock_connection.h
	{
		got, golden := readGenAndGolden(MockConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("mock_connection.h methods", got, golden,
			"  MOCK_METHOD(StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse>,\n  GenerateAccessToken,",
			"ListOperations,\n  (google::longrunning::ListOperationsRequest request), (override));\n};\n")
	}

	// 6. connection_idempotency_policy.h
	{
		got, golden := readGenAndGolden(IdempotencyPolicyHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_idempotency_policy.h methods", got, golden,
			"  virtual google::cloud::Idempotency\n  GenerateAccessToken",
			"ListOperations(google::longrunning::ListOperationsRequest request);\n};\n")
	}

	// 7. stub.h (Stub interface and DefaultStub)
	{
		got, golden := readGenAndGolden(StubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("stub.h Stub methods", got, golden,
			"  virtual StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(",
			"    google::cloud::internal::ImmutableOptions options) = 0;\n};\n")
		assertBlockMatch("stub.h DefaultStub methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(",
			"    google::cloud::internal::ImmutableOptions options) override;")
	}

	// 8. auth_decorator.h
	{
		got, golden := readGenAndGolden(AuthDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("auth_decorator.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(",
			"    google::cloud::internal::ImmutableOptions options) override;\n\n private:")
	}

	// 9. logging_decorator.h
	{
		got, golden := readGenAndGolden(LoggingDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("logging_decorator.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(",
			"    google::cloud::internal::ImmutableOptions options) override;\n\n private:")
	}

	// 10. metadata_decorator.h
	{
		got, golden := readGenAndGolden(MetadataDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("metadata_decorator.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(",
			"    google::cloud::internal::ImmutableOptions options) override;\n\n private:")
	}

	// 11. tracing_stub.h
	{
		got, golden := readGenAndGolden(TracingStubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_stub.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::GenerateAccessTokenResponse> GenerateAccessToken(",
			"    google::cloud::internal::ImmutableOptions options) override;\n\n private:")
	}
}

func TestGenerate_Methods_GoldenThingAdmin(t *testing.T) {
	requireProtoc(t)

	srcs := &sources.Sources{
		Googleapis:  filepath.Join("..", "..", "testdata", "googleapis"),
		ProtobufSrc: filepath.Join("testdata", "protos"),
	}
	sourceConfig := &sources.SourceConfig{
		Sources:     srcs,
		ActiveRoots: []string{"protobuf-src", "googleapis"},
		IncludeList: []string{"test.proto", "backup.proto"},
	}
	modelConfig := &parser.ModelConfig{
		Language:            config.LanguageCpp,
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: "generator/integration_tests",
		ServiceConfig:       "generator/integration_tests/test.yaml",
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
		InitialCopyrightYear:          "2022",
		RetryableStatusCodes:          []string{"GoldenKitchenSink.kInternal", "kUnavailable", "GoldenThingAdmin.kDeadlineExceeded"},
		OverrideServiceConfigYAMLName: "generator/integration_tests/test.yaml",
		OmittedRPCs: []string{
			"Omitted1",
			"GoldenKitchenSink.Omitted2",
			"Deprecated1",
			"Deprecated2(std::string const&)",
		},
		GenAsyncRPCs: []string{
			"GetDatabase",
			"DropDatabase",
			"StreamingRead",
			"StreamingWrite",
		},
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

	svc := "GoldenThingAdmin"

	// 1. client.h
	{
		got, golden := readGenAndGolden(ClientHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("client.h methods", got, golden,
			"  StreamRange<google::test::admin::database::v1::Database>\n  ListDatabases(",
			"  AsyncDropDatabase(google::test::admin::database::v1::DropDatabaseRequest const& request, Options opts = {});\n\n private:")
	}

	// 2. connection.h
	{
		got, golden := readGenAndGolden(ConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection.h methods", got, golden,
			"  virtual StreamRange<google::test::admin::database::v1::Database>\n  ListDatabases(",
			"  virtual future<Status>\n  AsyncDropDatabase(google::test::admin::database::v1::DropDatabaseRequest const& request);\n};\n")
	}

	// 3. connection_impl.h
	{
		got, golden := readGenAndGolden(ConnectionImplHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_impl.h methods", got, golden,
			"  StreamRange<google::test::admin::database::v1::Database>\n  ListDatabases(",
			"  future<Status>\n  AsyncDropDatabase(google::test::admin::database::v1::DropDatabaseRequest const& request) override;\n\n private:")
	}

	// 4. tracing_connection.h
	{
		got, golden := readGenAndGolden(TracingConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_connection.h methods", got, golden,
			"  StreamRange<google::test::admin::database::v1::Database>\n  ListDatabases(",
			"  future<Status>\n  AsyncDropDatabase(google::test::admin::database::v1::DropDatabaseRequest const& request) override;\n\n private:")
	}

	// 5. mock_connection.h
	{
		got, golden := readGenAndGolden(MockConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("mock_connection.h methods", got, golden,
			"  MOCK_METHOD((StreamRange<google::test::admin::database::v1::Database>),\n  ListDatabases,",
			"AsyncDropDatabase,\n  (google::test::admin::database::v1::DropDatabaseRequest const& request), (override));\n};\n")
	}

	// 6. connection_idempotency_policy.h
	{
		got, golden := readGenAndGolden(IdempotencyPolicyHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_idempotency_policy.h methods", got, golden,
			"  virtual google::cloud::Idempotency\n  ListDatabases",
			"ListOperations(google::longrunning::ListOperationsRequest request);\n};\n")
	}

	// 7. stub.h (Stub interface and DefaultStub)
	{
		got, golden := readGenAndGolden(StubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("stub.h Stub methods", got, golden,
			"  virtual StatusOr<google::test::admin::database::v1::ListDatabasesResponse> ListDatabases(",
			"      google::longrunning::CancelOperationRequest const& request) = 0;\n};\n")
		assertBlockMatch("stub.h DefaultStub methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::ListDatabasesResponse> ListDatabases(",
			"      google::longrunning::CancelOperationRequest const& request) override;")
	}

	// 8. auth_decorator.h
	{
		got, golden := readGenAndGolden(AuthDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("auth_decorator.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::ListDatabasesResponse> ListDatabases(",
			"      google::longrunning::CancelOperationRequest const& request) override;\n\n private:")
	}

	// 9. logging_decorator.h
	{
		got, golden := readGenAndGolden(LoggingDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("logging_decorator.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::ListDatabasesResponse> ListDatabases(",
			"      google::longrunning::CancelOperationRequest const& request) override;\n\n private:")
	}

	// 10. metadata_decorator.h
	{
		got, golden := readGenAndGolden(MetadataDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("metadata_decorator.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::ListDatabasesResponse> ListDatabases(",
			"      google::longrunning::CancelOperationRequest const& request) override;\n\n private:")
	}

	// 11. tracing_stub.h
	{
		got, golden := readGenAndGolden(TracingStubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_stub.h methods", got, golden,
			"  StatusOr<google::test::admin::database::v1::ListDatabasesResponse> ListDatabases(",
			"      google::longrunning::CancelOperationRequest const& request) override;\n\n private:")
	}
}

func TestGenerate_Methods_DeprecatedService(t *testing.T) {
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

	svc := "DeprecatedService"

	// 1. client.h
	{
		got, golden := readGenAndGolden(ClientHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("client.h methods", got, golden,
			"  // clang-format off\n  ///\n  /// Does nothing.",
			"Noop(google::test::deprecated::v1::DeprecatedServiceRequest const& request, Options opts = {});\n\n private:")
	}

	// 2. connection.h
	{
		got, golden := readGenAndGolden(ConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection.h methods", got, golden,
			"  virtual Status\n  Noop",
			"Noop(google::test::deprecated::v1::DeprecatedServiceRequest const& request);\n};\n")
	}

	// 3. connection_impl.h
	{
		got, golden := readGenAndGolden(ConnectionImplHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_impl.h methods", got, golden,
			"  Status\n  Noop",
			"Noop(google::test::deprecated::v1::DeprecatedServiceRequest const& request) override;\n\n private:")
	}

	// 4. tracing_connection.h
	{
		got, golden := readGenAndGolden(TracingConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_connection.h methods", got, golden,
			"  Status\n  Noop",
			"Noop(google::test::deprecated::v1::DeprecatedServiceRequest const& request) override;\n\n private:")
	}

	// 5. mock_connection.h
	{
		got, golden := readGenAndGolden(MockConnectionHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("mock_connection.h methods", got, golden,
			"  MOCK_METHOD(Status,\n  Noop,",
			"(google::test::deprecated::v1::DeprecatedServiceRequest const& request), (override));\n};\n")
	}

	// 6. connection_idempotency_policy.h
	{
		got, golden := readGenAndGolden(IdempotencyPolicyHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("connection_idempotency_policy.h methods", got, golden,
			"  virtual google::cloud::Idempotency\n  Noop",
			"Noop(google::test::deprecated::v1::DeprecatedServiceRequest const& request);\n};\n")
	}

	// 7. stub.h (Stub interface and DefaultStub)
	{
		got, golden := readGenAndGolden(StubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("stub.h Stub methods", got, golden,
			"  virtual Status Noop(",
			"google::test::deprecated::v1::DeprecatedServiceRequest const& request) = 0;\n};\n")
		assertBlockMatch("stub.h DefaultStub methods", got, golden,
			"  Status Noop(",
			"google::test::deprecated::v1::DeprecatedServiceRequest const& request) override;")
	}

	// 8. auth_decorator.h
	{
		got, golden := readGenAndGolden(AuthDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("auth_decorator.h methods", got, golden,
			"  Status Noop(",
			"google::test::deprecated::v1::DeprecatedServiceRequest const& request) override;\n\n private:")
	}

	// 9. logging_decorator.h
	{
		got, golden := readGenAndGolden(LoggingDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("logging_decorator.h methods", got, golden,
			"  Status Noop(",
			"google::test::deprecated::v1::DeprecatedServiceRequest const& request) override;\n\n private:")
	}

	// 10. metadata_decorator.h
	{
		got, golden := readGenAndGolden(MetadataDecoratorHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("metadata_decorator.h methods", got, golden,
			"  Status Noop(",
			"google::test::deprecated::v1::DeprecatedServiceRequest const& request) override;\n\n private:")
	}

	// 11. tracing_stub.h
	{
		got, golden := readGenAndGolden(TracingStubHeaderPath(libCfg.ProductPath, svc))
		assertBlockMatch("tracing_stub.h methods", got, golden,
			"  Status Noop(",
			"google::test::deprecated::v1::DeprecatedServiceRequest const& request) override;\n\n private:")
	}
}
