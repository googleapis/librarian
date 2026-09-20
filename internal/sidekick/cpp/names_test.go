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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

func TestCamelCaseToSnakeCase(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "simple word",
			input: "Simple",
			want:  "simple",
		},
		{
			name:  "camel case",
			input: "CamelCase",
			want:  "camel_case",
		},
		{
			name:  "big query exception",
			input: "BigQueryRead",
			want:  "bigquery_read",
		},
		{
			name:  "big query standalone",
			input: "BigQuery",
			want:  "bigquery",
		},
		{
			name:  "acronym prefix",
			input: "HTTPServer",
			want:  "http_server",
		},
		{
			name:  "all uppercase",
			input: "ABC",
			want:  "abc",
		},
		{
			name:  "all uppercase longer",
			input: "ABCD",
			want:  "abcd",
		},
		{
			name:  "uppercase with trailing camel",
			input: "ABCDef",
			want:  "abc_def",
		},
		{
			name:  "digit boundary",
			input: "Foo2Bar",
			want:  "foo2_bar",
		},
		{
			name:  "multiple digits",
			input: "Foo22Bar",
			want:  "foo22_bar",
		},
		{
			name:  "already snake case",
			input: "already_snake_case",
			want:  "already_snake_case",
		},
		{
			name:  "mixed snake and camel",
			input: "Already_SnakeCase",
			want:  "already_snake_case",
		},
		{
			name:  "database admin",
			input: "DatabaseAdmin",
			want:  "database_admin",
		},
		{
			name:  "database service component",
			input: "databaseService",
			want:  "database_service",
		},
		{
			name:  "golden kitchen sink",
			input: "GoldenKitchenSink",
			want:  "golden_kitchen_sink",
		},
		{
			name:  "golden thing admin",
			input: "GoldenThingAdmin",
			want:  "golden_thing_admin",
		},
		{
			name:  "golden rest only",
			input: "GoldenRestOnly",
			want:  "golden_rest_only",
		},
		{
			name:  "request id",
			input: "RequestId",
			want:  "request_id",
		},
		{
			name:  "deprecated",
			input: "Deprecated",
			want:  "deprecated",
		},
		{
			name:  "upstream FooBarB",
			input: "FooBarB",
			want:  "foo_bar_b",
		},
		{
			name:  "upstream FooBarBaz",
			input: "FooBarBaz",
			want:  "foo_bar_baz",
		},
		{
			name:  "upstream fooBarBaz",
			input: "fooBarBaz",
			want:  "foo_bar_baz",
		},
		{
			name:  "upstream fooBarAb",
			input: "fooBarAb",
			want:  "foo_bar_ab",
		},
		{
			name:  "upstream fooBarBAAAAA",
			input: "fooBarBAAAAA",
			want:  "foo_bar_baaaaa",
		},
		{
			name:  "upstream foo_BarB",
			input: "foo_BarB",
			want:  "foo_bar_b",
		},
		{
			name:  "upstream v1",
			input: "v1",
			want:  "v1",
		},
		{
			name:  "upstream A",
			input: "A",
			want:  "a",
		},
		{
			name:  "upstream aB",
			input: "aB",
			want:  "a_b",
		},
		{
			name:  "upstream Foo123",
			input: "Foo123",
			want:  "foo123",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := CamelCaseToSnakeCase(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServiceNameToFilePath(t *testing.T) {
	for _, test := range []struct {
		name        string
		serviceName string
		want        string
	}{
		{
			name:        "golden request id with service suffix",
			serviceName: "RequestIdService",
			want:        "request_id",
		},
		{
			name:        "golden deprecated with service suffix",
			serviceName: "DeprecatedService",
			want:        "deprecated",
		},
		{
			name:        "golden kitchen sink without service suffix",
			serviceName: "GoldenKitchenSink",
			want:        "golden_kitchen_sink",
		},
		{
			name:        "golden thing admin without service suffix",
			serviceName: "GoldenThingAdmin",
			want:        "golden_thing_admin",
		},
		{
			name:        "golden rest only without service suffix",
			serviceName: "GoldenRestOnly",
			want:        "golden_rest_only",
		},
		{
			name:        "trailing service in last component",
			serviceName: "google.spanner.admin.database.v1.DatabaseAdminService",
			want:        "google/spanner/admin/database/v1/database_admin",
		},
		{
			name:        "no trailing service in last component",
			serviceName: "google.spanner.admin.database.v1.DatabaseAdmin",
			want:        "google/spanner/admin/database/v1/database_admin",
		},
		{
			name:        "trailing service in intermediate component",
			serviceName: "google.spanner.admin.databaseService.v1.DatabaseAdminService",
			want:        "google/spanner/admin/database_service/v1/database_admin",
		},
		{
			name:        "service only",
			serviceName: "Service",
			want:        "",
		},
		{
			name:        "service service suffix",
			serviceName: "ServiceService",
			want:        "service",
		},
		{
			name:        "empty service name",
			serviceName: "",
			want:        "",
		},
		{
			name:        "single component no suffix",
			serviceName: "Echo",
			want:        "echo",
		},
		{
			name:        "big query service",
			serviceName: "BigQueryService",
			want:        "bigquery",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ServiceNameToFilePath(test.serviceName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPathHelpers(t *testing.T) {
	const (
		productPath    = "generator/integration_tests/golden/v1"
		forwardingPath = "generator/integration_tests/golden"
		serviceName    = "RequestIdService"
	)

	for _, test := range []struct {
		name string
		fn   func() string
		want string
	}{
		{
			name: "client header",
			fn:   func() string { return ClientHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/request_id_client.h",
		},
		{
			name: "client source",
			fn:   func() string { return ClientSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/request_id_client.cc",
		},
		{
			name: "connection header",
			fn:   func() string { return ConnectionHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/request_id_connection.h",
		},
		{
			name: "connection source",
			fn:   func() string { return ConnectionSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/request_id_connection.cc",
		},
		{
			name: "idempotency policy header",
			fn:   func() string { return IdempotencyPolicyHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/request_id_connection_idempotency_policy.h",
		},
		{
			name: "idempotency policy source",
			fn:   func() string { return IdempotencyPolicySourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/request_id_connection_idempotency_policy.cc",
		},
		{
			name: "options header",
			fn:   func() string { return OptionsHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/request_id_options.h",
		},
		{
			name: "mock connection header",
			fn:   func() string { return MockConnectionHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/mocks/mock_request_id_connection.h",
		},
		{
			name: "option defaults header",
			fn:   func() string { return OptionDefaultsHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_option_defaults.h",
		},
		{
			name: "option defaults source",
			fn:   func() string { return OptionDefaultsSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_option_defaults.cc",
		},
		{
			name: "retry traits header",
			fn:   func() string { return RetryTraitsHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_retry_traits.h",
		},
		{
			name: "tracing connection header",
			fn:   func() string { return TracingConnectionHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_connection.h",
		},
		{
			name: "tracing connection source",
			fn:   func() string { return TracingConnectionSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_connection.cc",
		},
		{
			name: "connection impl header",
			fn:   func() string { return ConnectionImplHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_connection_impl.h",
		},
		{
			name: "connection impl source",
			fn:   func() string { return ConnectionImplSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_connection_impl.cc",
		},
		{
			name: "stub factory header",
			fn:   func() string { return StubFactoryHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_stub_factory.h",
		},
		{
			name: "stub factory source",
			fn:   func() string { return StubFactorySourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_stub_factory.cc",
		},
		{
			name: "auth decorator header",
			fn:   func() string { return AuthDecoratorHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_auth_decorator.h",
		},
		{
			name: "auth decorator source",
			fn:   func() string { return AuthDecoratorSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_auth_decorator.cc",
		},
		{
			name: "logging decorator header",
			fn:   func() string { return LoggingDecoratorHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_logging_decorator.h",
		},
		{
			name: "logging decorator source",
			fn:   func() string { return LoggingDecoratorSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_logging_decorator.cc",
		},
		{
			name: "metadata decorator header",
			fn:   func() string { return MetadataDecoratorHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_metadata_decorator.h",
		},
		{
			name: "metadata decorator source",
			fn:   func() string { return MetadataDecoratorSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_metadata_decorator.cc",
		},
		{
			name: "stub header",
			fn:   func() string { return StubHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_stub.h",
		},
		{
			name: "stub source",
			fn:   func() string { return StubSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_stub.cc",
		},
		{
			name: "tracing stub header",
			fn:   func() string { return TracingStubHeaderPath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_stub.h",
		},
		{
			name: "tracing stub source",
			fn:   func() string { return TracingStubSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_stub.cc",
		},
		{
			name: "sources source",
			fn:   func() string { return SourcesSourcePath(productPath, serviceName) },
			want: "generator/integration_tests/golden/v1/internal/request_id_sources.cc",
		},
		{
			name: "forwarding client header",
			fn:   func() string { return ForwardingClientHeaderPath(forwardingPath, serviceName) },
			want: "generator/integration_tests/golden/request_id_client.h",
		},
		{
			name: "forwarding connection header",
			fn:   func() string { return ForwardingConnectionHeaderPath(forwardingPath, serviceName) },
			want: "generator/integration_tests/golden/request_id_connection.h",
		},
		{
			name: "forwarding idempotency policy header",
			fn:   func() string { return ForwardingIdempotencyPolicyHeaderPath(forwardingPath, serviceName) },
			want: "generator/integration_tests/golden/request_id_connection_idempotency_policy.h",
		},
		{
			name: "forwarding options header",
			fn:   func() string { return ForwardingOptionsHeaderPath(forwardingPath, serviceName) },
			want: "generator/integration_tests/golden/request_id_options.h",
		},
		{
			name: "forwarding mock connection header",
			fn:   func() string { return ForwardingMockConnectionHeaderPath(forwardingPath, serviceName) },
			want: "generator/integration_tests/golden/mocks/mock_request_id_connection.h",
		},
		{
			name: "empty product path",
			fn:   func() string { return ClientHeaderPath("", serviceName) },
			want: "request_id_client.h",
		},
		{
			name: "leading slash product path trimmed",
			fn:   func() string { return ClientHeaderPath("/foo/bar", serviceName) },
			want: "foo/bar/request_id_client.h",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.fn()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServiceGeneratedFiles(t *testing.T) {
	files := ServiceGeneratedFiles("v1", "RequestIdService")
	const wantCount = 28
	if len(files) != wantCount {
		t.Fatalf("unexpected number of service files: want %d, got %d", wantCount, len(files))
	}

	seenOutput := make(map[string]bool)
	for _, f := range files {
		if f.TemplatePath == "" {
			t.Errorf("empty TemplatePath for %+v", f)
		}
		if f.OutputPath == "" {
			t.Errorf("empty OutputPath for %+v", f)
		}
		if seenOutput[f.OutputPath] {
			t.Errorf("duplicate OutputPath %q", f.OutputPath)
		}
		seenOutput[f.OutputPath] = true
	}

	wantFiles := []language.GeneratedFile{
		{TemplatePath: "templates/service/client.h.mustache", OutputPath: "v1/request_id_client.h"},
		{TemplatePath: "templates/service/client.cc.mustache", OutputPath: "v1/request_id_client.cc"},
		{TemplatePath: "templates/service/connection.h.mustache", OutputPath: "v1/request_id_connection.h"},
		{TemplatePath: "templates/service/connection.cc.mustache", OutputPath: "v1/request_id_connection.cc"},
		{TemplatePath: "templates/service/connection_idempotency_policy.h.mustache", OutputPath: "v1/request_id_connection_idempotency_policy.h"},
		{TemplatePath: "templates/service/connection_idempotency_policy.cc.mustache", OutputPath: "v1/request_id_connection_idempotency_policy.cc"},
		{TemplatePath: "templates/service/options.h.mustache", OutputPath: "v1/request_id_options.h"},
		{TemplatePath: "templates/service/mock_connection.h.mustache", OutputPath: "v1/mocks/mock_request_id_connection.h"},
		{TemplatePath: "templates/service/option_defaults.h.mustache", OutputPath: "v1/internal/request_id_option_defaults.h"},
		{TemplatePath: "templates/service/option_defaults.cc.mustache", OutputPath: "v1/internal/request_id_option_defaults.cc"},
		{TemplatePath: "templates/service/retry_traits.h.mustache", OutputPath: "v1/internal/request_id_retry_traits.h"},
		{TemplatePath: "templates/service/tracing_connection.h.mustache", OutputPath: "v1/internal/request_id_tracing_connection.h"},
		{TemplatePath: "templates/service/tracing_connection.cc.mustache", OutputPath: "v1/internal/request_id_tracing_connection.cc"},
		{TemplatePath: "templates/service/connection_impl.h.mustache", OutputPath: "v1/internal/request_id_connection_impl.h"},
		{TemplatePath: "templates/service/connection_impl.cc.mustache", OutputPath: "v1/internal/request_id_connection_impl.cc"},
		{TemplatePath: "templates/service/stub_factory.h.mustache", OutputPath: "v1/internal/request_id_stub_factory.h"},
		{TemplatePath: "templates/service/stub_factory.cc.mustache", OutputPath: "v1/internal/request_id_stub_factory.cc"},
		{TemplatePath: "templates/service/auth_decorator.h.mustache", OutputPath: "v1/internal/request_id_auth_decorator.h"},
		{TemplatePath: "templates/service/auth_decorator.cc.mustache", OutputPath: "v1/internal/request_id_auth_decorator.cc"},
		{TemplatePath: "templates/service/logging_decorator.h.mustache", OutputPath: "v1/internal/request_id_logging_decorator.h"},
		{TemplatePath: "templates/service/logging_decorator.cc.mustache", OutputPath: "v1/internal/request_id_logging_decorator.cc"},
		{TemplatePath: "templates/service/metadata_decorator.h.mustache", OutputPath: "v1/internal/request_id_metadata_decorator.h"},
		{TemplatePath: "templates/service/metadata_decorator.cc.mustache", OutputPath: "v1/internal/request_id_metadata_decorator.cc"},
		{TemplatePath: "templates/service/stub.h.mustache", OutputPath: "v1/internal/request_id_stub.h"},
		{TemplatePath: "templates/service/stub.cc.mustache", OutputPath: "v1/internal/request_id_stub.cc"},
		{TemplatePath: "templates/service/tracing_stub.h.mustache", OutputPath: "v1/internal/request_id_tracing_stub.h"},
		{TemplatePath: "templates/service/tracing_stub.cc.mustache", OutputPath: "v1/internal/request_id_tracing_stub.cc"},
		{TemplatePath: "templates/service/sources.cc.mustache", OutputPath: "v1/internal/request_id_sources.cc"},
	}

	if diff := cmp.Diff(wantFiles, files); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestForwardingGeneratedFiles(t *testing.T) {
	files := ForwardingGeneratedFiles("golden", "GoldenKitchenSink")
	const wantCount = 5
	if len(files) != wantCount {
		t.Fatalf("unexpected number of forwarding files: want %d, got %d", wantCount, len(files))
	}

	wantFiles := []language.GeneratedFile{
		{TemplatePath: "templates/service/forwarding_client.h.mustache", OutputPath: "golden/golden_kitchen_sink_client.h"},
		{TemplatePath: "templates/service/forwarding_connection.h.mustache", OutputPath: "golden/golden_kitchen_sink_connection.h"},
		{TemplatePath: "templates/service/forwarding_connection_idempotency_policy.h.mustache", OutputPath: "golden/golden_kitchen_sink_connection_idempotency_policy.h"},
		{TemplatePath: "templates/service/forwarding_options.h.mustache", OutputPath: "golden/golden_kitchen_sink_options.h"},
		{TemplatePath: "templates/service/forwarding_mock_connection.h.mustache", OutputPath: "golden/mocks/mock_golden_kitchen_sink_connection.h"},
	}

	if diff := cmp.Diff(wantFiles, files); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatHeaderIncludeGuard(t *testing.T) {
	for _, test := range []struct {
		name       string
		headerPath string
		want       string
	}{
		{
			name:       "empty path",
			headerPath: "",
			want:       "",
		},
		{
			name:       "dot path",
			headerPath: ".",
			want:       "",
		},
		{
			name:       "slash path",
			headerPath: "/",
			want:       "",
		},
		{
			name:       "multiple slashes only",
			headerPath: "///",
			want:       "",
		},
		{
			name:       "dot slash only",
			headerPath: "./",
			want:       "",
		},
		{
			name:       "single file",
			headerPath: "client.h",
			want:       "GOOGLE_CLOUD_CPP_CLIENT_H",
		},
		{
			name:       "relative path with directories",
			headerPath: "google/cloud/test/v1/client.h",
			want:       "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_TEST_V1_CLIENT_H",
		},
		{
			name:       "integration test golden path",
			headerPath: "generator/integration_tests/golden/v1/golden_kitchen_sink_client.h",
			want:       "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			name:       "forwarding header path",
			headerPath: "generator/integration_tests/golden/golden_kitchen_sink_client.h",
			want:       "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_GOLDEN_KITCHEN_SINK_CLIENT_H",
		},
		{
			name:       "forwarding mock header path",
			headerPath: "generator/integration_tests/golden/mocks/mock_golden_kitchen_sink_connection.h",
			want:       "GOOGLE_CLOUD_CPP_GENERATOR_INTEGRATION_TESTS_GOLDEN_MOCKS_MOCK_GOLDEN_KITCHEN_SINK_CONNECTION_H",
		},
		{
			name:       "leading slash is stripped",
			headerPath: "/leading/slash/path.h",
			want:       "GOOGLE_CLOUD_CPP_LEADING_SLASH_PATH_H",
		},
		{
			name:       "redundant dots are cleaned",
			headerPath: "path/with/../clean/file.h",
			want:       "GOOGLE_CLOUD_CPP_PATH_CLEAN_FILE_H",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := FormatHeaderIncludeGuard(test.headerPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNamespace(t *testing.T) {
	for _, test := range []struct {
		name        string
		productPath string
		want        string
	}{
		{
			name:        "google cloud test",
			productPath: "google/cloud/test",
			want:        "test",
		},
		{
			name:        "google cloud test with trailing slash",
			productPath: "google/cloud/test/",
			want:        "test",
		},
		{
			name:        "google cloud test with version",
			productPath: "google/cloud/test/v1",
			want:        "test_v1",
		},
		{
			name:        "google cloud test with version and trailing slash",
			productPath: "google/cloud/test/v1/",
			want:        "test_v1",
		},
		{
			name:        "google cloud test with nested subdir",
			productPath: "google/cloud/test/foo/v1",
			want:        "test_foo_v1",
		},
		{
			name:        "golden integration tests",
			productPath: "generator/integration_tests/golden/v1",
			want:        "golden_v1",
		},
		{
			name:        "golden integration tests with trailing slash",
			productPath: "generator/integration_tests/golden/v1/",
			want:        "golden_v1",
		},
		{
			name:        "blah golden",
			productPath: "blah/golden",
			want:        "golden",
		},
		{
			name:        "blah golden v1",
			productPath: "blah/golden/v1",
			want:        "golden_v1",
		},
		{
			name:        "generic service path",
			productPath: "foo/bar/service",
			want:        "service",
		},
		{
			name:        "single component",
			productPath: "myservice",
			want:        "myservice",
		},
		{
			name:        "empty string",
			productPath: "",
			want:        "",
		},
		{
			name:        "only slashes",
			productPath: "///",
			want:        "",
		},
		{
			name:        "leading dot slash",
			productPath: "./google/cloud/test",
			want:        "test",
		},
		{
			name:        "dot slash only",
			productPath: "./",
			want:        "",
		},
		{
			name:        "dot only",
			productPath: ".",
			want:        "",
		},
		{
			name:        "dot slash with version",
			productPath: "./google/cloud/test/v1",
			want:        "test_v1",
		},
		{
			name:        "double slash with leading dot",
			productPath: ".//google/cloud/test",
			want:        "test",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Namespace(test.productPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInternalNamespace(t *testing.T) {
	for _, test := range []struct {
		name        string
		productPath string
		want        string
	}{
		{
			name:        "google cloud test v1",
			productPath: "google/cloud/test/v1",
			want:        "test_v1_internal",
		},
		{
			name:        "golden v1",
			productPath: "generator/integration_tests/golden/v1",
			want:        "golden_v1_internal",
		},
		{
			name:        "empty product path",
			productPath: "",
			want:        "internal",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := InternalNamespace(test.productPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMocksNamespace(t *testing.T) {
	for _, test := range []struct {
		name        string
		productPath string
		want        string
	}{
		{
			name:        "google cloud test v1",
			productPath: "google/cloud/test/v1",
			want:        "test_v1_mocks",
		},
		{
			name:        "golden v1",
			productPath: "generator/integration_tests/golden/v1",
			want:        "golden_v1_mocks",
		},
		{
			name:        "empty product path",
			productPath: "",
			want:        "mocks",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := MocksNamespace(test.productPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClassNames(t *testing.T) {
	for _, test := range []struct {
		serviceName                      string
		wantClient                       string
		wantConnection                   string
		wantConnectionIdempotencyPolicy  string
		wantMockConnection               string
		wantConnectionImpl               string
		wantStub                         string
		wantDefaultStub                  string
		wantAuthDecorator                string
		wantLoggingDecorator             string
		wantMetadataDecorator            string
		wantTracingConnection            string
		wantTracingStub                  string
		wantRetryPolicy                  string
		wantLimitedErrorCountRetryPolicy string
		wantLimitedTimeRetryPolicy       string
		wantRetryTraits                  string
	}{
		{
			serviceName:                      "GoldenThingAdmin",
			wantClient:                       "GoldenThingAdminClient",
			wantConnection:                   "GoldenThingAdminConnection",
			wantConnectionIdempotencyPolicy:  "GoldenThingAdminConnectionIdempotencyPolicy",
			wantMockConnection:               "MockGoldenThingAdminConnection",
			wantConnectionImpl:               "GoldenThingAdminConnectionImpl",
			wantStub:                         "GoldenThingAdminStub",
			wantDefaultStub:                  "DefaultGoldenThingAdminStub",
			wantAuthDecorator:                "GoldenThingAdminAuth",
			wantLoggingDecorator:             "GoldenThingAdminLogging",
			wantMetadataDecorator:            "GoldenThingAdminMetadata",
			wantTracingConnection:            "GoldenThingAdminTracingConnection",
			wantTracingStub:                  "GoldenThingAdminTracingStub",
			wantRetryPolicy:                  "GoldenThingAdminRetryPolicy",
			wantLimitedErrorCountRetryPolicy: "GoldenThingAdminLimitedErrorCountRetryPolicy",
			wantLimitedTimeRetryPolicy:       "GoldenThingAdminLimitedTimeRetryPolicy",
			wantRetryTraits:                  "GoldenThingAdminRetryTraits",
		},
		{
			serviceName:                      "EchoService",
			wantClient:                       "EchoServiceClient",
			wantConnection:                   "EchoServiceConnection",
			wantConnectionIdempotencyPolicy:  "EchoServiceConnectionIdempotencyPolicy",
			wantMockConnection:               "MockEchoServiceConnection",
			wantConnectionImpl:               "EchoServiceConnectionImpl",
			wantStub:                         "EchoServiceStub",
			wantDefaultStub:                  "DefaultEchoServiceStub",
			wantAuthDecorator:                "EchoServiceAuth",
			wantLoggingDecorator:             "EchoServiceLogging",
			wantMetadataDecorator:            "EchoServiceMetadata",
			wantTracingConnection:            "EchoServiceTracingConnection",
			wantTracingStub:                  "EchoServiceTracingStub",
			wantRetryPolicy:                  "EchoServiceRetryPolicy",
			wantLimitedErrorCountRetryPolicy: "EchoServiceLimitedErrorCountRetryPolicy",
			wantLimitedTimeRetryPolicy:       "EchoServiceLimitedTimeRetryPolicy",
			wantRetryTraits:                  "EchoServiceRetryTraits",
		},
		{
			serviceName:                      "DeprecatedService",
			wantClient:                       "DeprecatedServiceClient",
			wantConnection:                   "DeprecatedServiceConnection",
			wantConnectionIdempotencyPolicy:  "DeprecatedServiceConnectionIdempotencyPolicy",
			wantMockConnection:               "MockDeprecatedServiceConnection",
			wantConnectionImpl:               "DeprecatedServiceConnectionImpl",
			wantStub:                         "DeprecatedServiceStub",
			wantDefaultStub:                  "DefaultDeprecatedServiceStub",
			wantAuthDecorator:                "DeprecatedServiceAuth",
			wantLoggingDecorator:             "DeprecatedServiceLogging",
			wantMetadataDecorator:            "DeprecatedServiceMetadata",
			wantTracingConnection:            "DeprecatedServiceTracingConnection",
			wantTracingStub:                  "DeprecatedServiceTracingStub",
			wantRetryPolicy:                  "DeprecatedServiceRetryPolicy",
			wantLimitedErrorCountRetryPolicy: "DeprecatedServiceLimitedErrorCountRetryPolicy",
			wantLimitedTimeRetryPolicy:       "DeprecatedServiceLimitedTimeRetryPolicy",
			wantRetryTraits:                  "DeprecatedServiceRetryTraits",
		},
		{
			serviceName:                      "",
			wantClient:                       "Client",
			wantConnection:                   "Connection",
			wantConnectionIdempotencyPolicy:  "ConnectionIdempotencyPolicy",
			wantMockConnection:               "MockConnection",
			wantConnectionImpl:               "ConnectionImpl",
			wantStub:                         "Stub",
			wantDefaultStub:                  "DefaultStub",
			wantAuthDecorator:                "Auth",
			wantLoggingDecorator:             "Logging",
			wantMetadataDecorator:            "Metadata",
			wantTracingConnection:            "TracingConnection",
			wantTracingStub:                  "TracingStub",
			wantRetryPolicy:                  "RetryPolicy",
			wantLimitedErrorCountRetryPolicy: "LimitedErrorCountRetryPolicy",
			wantLimitedTimeRetryPolicy:       "LimitedTimeRetryPolicy",
			wantRetryTraits:                  "RetryTraits",
		},
	} {
		t.Run(test.serviceName, func(t *testing.T) {
			if diff := cmp.Diff(test.wantClient, ClientClassName(test.serviceName)); diff != "" {
				t.Logf("ClientClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantConnection, ConnectionClassName(test.serviceName)); diff != "" {
				t.Logf("ConnectionClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantConnectionIdempotencyPolicy, ConnectionIdempotencyPolicyClassName(test.serviceName)); diff != "" {
				t.Logf("ConnectionIdempotencyPolicyClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantMockConnection, MockConnectionClassName(test.serviceName)); diff != "" {
				t.Logf("MockConnectionClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantConnectionImpl, ConnectionImplClassName(test.serviceName)); diff != "" {
				t.Logf("ConnectionImplClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantStub, StubClassName(test.serviceName)); diff != "" {
				t.Logf("StubClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantDefaultStub, DefaultStubClassName(test.serviceName)); diff != "" {
				t.Logf("DefaultStubClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantAuthDecorator, AuthDecoratorClassName(test.serviceName)); diff != "" {
				t.Logf("AuthDecoratorClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantLoggingDecorator, LoggingDecoratorClassName(test.serviceName)); diff != "" {
				t.Logf("LoggingDecoratorClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantMetadataDecorator, MetadataDecoratorClassName(test.serviceName)); diff != "" {
				t.Logf("MetadataDecoratorClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantTracingConnection, TracingConnectionClassName(test.serviceName)); diff != "" {
				t.Logf("TracingConnectionClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantTracingStub, TracingStubClassName(test.serviceName)); diff != "" {
				t.Logf("TracingStubClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantRetryPolicy, RetryPolicyName(test.serviceName)); diff != "" {
				t.Logf("RetryPolicyName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantLimitedErrorCountRetryPolicy, LimitedErrorCountRetryPolicyName(test.serviceName)); diff != "" {
				t.Logf("LimitedErrorCountRetryPolicyName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantLimitedTimeRetryPolicy, LimitedTimeRetryPolicyName(test.serviceName)); diff != "" {
				t.Logf("LimitedTimeRetryPolicyName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantRetryTraits, RetryTraitsName(test.serviceName)); diff != "" {
				t.Logf("RetryTraitsName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
