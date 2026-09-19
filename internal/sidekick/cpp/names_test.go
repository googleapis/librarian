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
			if got != test.want {
				t.Errorf("CamelCaseToSnakeCase(%q) = %q, want %q", test.input, got, test.want)
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
			if got != test.want {
				t.Errorf("ServiceNameToFilePath(%q) = %q, want %q", test.serviceName, got, test.want)
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
		got  string
		want string
	}{
		{
			name: "client header",
			got:  ClientHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/request_id_client.h",
		},
		{
			name: "client source",
			got:  ClientSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/request_id_client.cc",
		},
		{
			name: "connection header",
			got:  ConnectionHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/request_id_connection.h",
		},
		{
			name: "connection source",
			got:  ConnectionSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/request_id_connection.cc",
		},
		{
			name: "idempotency policy header",
			got:  IdempotencyPolicyHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/request_id_connection_idempotency_policy.h",
		},
		{
			name: "idempotency policy source",
			got:  IdempotencyPolicySourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/request_id_connection_idempotency_policy.cc",
		},
		{
			name: "options header",
			got:  OptionsHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/request_id_options.h",
		},
		{
			name: "mock connection header",
			got:  MockConnectionHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/mocks/mock_request_id_connection.h",
		},
		{
			name: "option defaults header",
			got:  OptionDefaultsHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_option_defaults.h",
		},
		{
			name: "option defaults source",
			got:  OptionDefaultsSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_option_defaults.cc",
		},
		{
			name: "retry traits header",
			got:  RetryTraitsHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_retry_traits.h",
		},
		{
			name: "tracing connection header",
			got:  TracingConnectionHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_connection.h",
		},
		{
			name: "tracing connection source",
			got:  TracingConnectionSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_connection.cc",
		},
		{
			name: "connection impl header",
			got:  ConnectionImplHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_connection_impl.h",
		},
		{
			name: "connection impl source",
			got:  ConnectionImplSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_connection_impl.cc",
		},
		{
			name: "stub factory header",
			got:  StubFactoryHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_stub_factory.h",
		},
		{
			name: "stub factory source",
			got:  StubFactorySourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_stub_factory.cc",
		},
		{
			name: "auth decorator header",
			got:  AuthDecoratorHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_auth_decorator.h",
		},
		{
			name: "auth decorator source",
			got:  AuthDecoratorSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_auth_decorator.cc",
		},
		{
			name: "logging decorator header",
			got:  LoggingDecoratorHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_logging_decorator.h",
		},
		{
			name: "logging decorator source",
			got:  LoggingDecoratorSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_logging_decorator.cc",
		},
		{
			name: "metadata decorator header",
			got:  MetadataDecoratorHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_metadata_decorator.h",
		},
		{
			name: "metadata decorator source",
			got:  MetadataDecoratorSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_metadata_decorator.cc",
		},
		{
			name: "stub header",
			got:  StubHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_stub.h",
		},
		{
			name: "stub source",
			got:  StubSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_stub.cc",
		},
		{
			name: "tracing stub header",
			got:  TracingStubHeaderPath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_stub.h",
		},
		{
			name: "tracing stub source",
			got:  TracingStubSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_tracing_stub.cc",
		},
		{
			name: "sources source",
			got:  SourcesSourcePath(productPath, serviceName),
			want: "generator/integration_tests/golden/v1/internal/request_id_sources.cc",
		},
		{
			name: "forwarding client header",
			got:  ForwardingClientHeaderPath(forwardingPath, serviceName),
			want: "generator/integration_tests/golden/request_id_client.h",
		},
		{
			name: "forwarding connection header",
			got:  ForwardingConnectionHeaderPath(forwardingPath, serviceName),
			want: "generator/integration_tests/golden/request_id_connection.h",
		},
		{
			name: "forwarding idempotency policy header",
			got:  ForwardingIdempotencyPolicyHeaderPath(forwardingPath, serviceName),
			want: "generator/integration_tests/golden/request_id_connection_idempotency_policy.h",
		},
		{
			name: "forwarding options header",
			got:  ForwardingOptionsHeaderPath(forwardingPath, serviceName),
			want: "generator/integration_tests/golden/request_id_options.h",
		},
		{
			name: "forwarding mock connection header",
			got:  ForwardingMockConnectionHeaderPath(forwardingPath, serviceName),
			want: "generator/integration_tests/golden/mocks/mock_request_id_connection.h",
		},
		{
			name: "empty product path",
			got:  ClientHeaderPath("", serviceName),
			want: "request_id_client.h",
		},
		{
			name: "leading slash product path trimmed",
			got:  ClientHeaderPath("/foo/bar", serviceName),
			want: "foo/bar/request_id_client.h",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("%s: got %q, want %q", test.name, test.got, test.want)
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
