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
)

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
		wantRestStub                     string
		wantDefaultRestStub              string
		wantRestLogging                  string
		wantRestMetadata                 string
		wantRestConnectionImpl           string
		wantRoundRobin                   string
		wantMakeRestConnection           string
		wantCreateDefaultRestStub        string
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
			wantRestStub:                     "GoldenThingAdminRestStub",
			wantDefaultRestStub:              "DefaultGoldenThingAdminRestStub",
			wantRestLogging:                  "GoldenThingAdminRestLogging",
			wantRestMetadata:                 "GoldenThingAdminRestMetadata",
			wantRestConnectionImpl:           "GoldenThingAdminRestConnectionImpl",
			wantRoundRobin:                   "GoldenThingAdminRoundRobin",
			wantMakeRestConnection:           "MakeGoldenThingAdminConnectionRest",
			wantCreateDefaultRestStub:        "CreateDefaultGoldenThingAdminRestStub",
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
			wantRestStub:                     "EchoServiceRestStub",
			wantDefaultRestStub:              "DefaultEchoServiceRestStub",
			wantRestLogging:                  "EchoServiceRestLogging",
			wantRestMetadata:                 "EchoServiceRestMetadata",
			wantRestConnectionImpl:           "EchoServiceRestConnectionImpl",
			wantRoundRobin:                   "EchoServiceRoundRobin",
			wantMakeRestConnection:           "MakeEchoServiceConnectionRest",
			wantCreateDefaultRestStub:        "CreateDefaultEchoServiceRestStub",
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
			wantRestStub:                     "DeprecatedServiceRestStub",
			wantDefaultRestStub:              "DefaultDeprecatedServiceRestStub",
			wantRestLogging:                  "DeprecatedServiceRestLogging",
			wantRestMetadata:                 "DeprecatedServiceRestMetadata",
			wantRestConnectionImpl:           "DeprecatedServiceRestConnectionImpl",
			wantRoundRobin:                   "DeprecatedServiceRoundRobin",
			wantMakeRestConnection:           "MakeDeprecatedServiceConnectionRest",
			wantCreateDefaultRestStub:        "CreateDefaultDeprecatedServiceRestStub",
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
			wantRestStub:                     "RestStub",
			wantDefaultRestStub:              "DefaultRestStub",
			wantRestLogging:                  "RestLogging",
			wantRestMetadata:                 "RestMetadata",
			wantRestConnectionImpl:           "RestConnectionImpl",
			wantRoundRobin:                   "RoundRobin",
			wantMakeRestConnection:           "MakeConnectionRest",
			wantCreateDefaultRestStub:        "CreateDefaultRestStub",
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
			if diff := cmp.Diff(test.wantRestStub, RestStubClassName(test.serviceName)); diff != "" {
				t.Logf("RestStubClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantDefaultRestStub, DefaultRestStubClassName(test.serviceName)); diff != "" {
				t.Logf("DefaultRestStubClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantRestLogging, RestLoggingDecoratorClassName(test.serviceName)); diff != "" {
				t.Logf("RestLoggingDecoratorClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantRestMetadata, RestMetadataDecoratorClassName(test.serviceName)); diff != "" {
				t.Logf("RestMetadataDecoratorClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantRestConnectionImpl, RestConnectionImplClassName(test.serviceName)); diff != "" {
				t.Logf("RestConnectionImplClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantRoundRobin, RoundRobinClassName(test.serviceName)); diff != "" {
				t.Logf("RoundRobinClassName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantMakeRestConnection, MakeRestConnectionFunctionName(test.serviceName)); diff != "" {
				t.Logf("MakeRestConnectionFunctionName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantCreateDefaultRestStub, CreateDefaultRestStubFunctionName(test.serviceName)); diff != "" {
				t.Logf("CreateDefaultRestStubFunctionName")
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOptionsGroup(t *testing.T) {
	for _, test := range []struct {
		name        string
		productPath string
		want        string
	}{
		{
			name:        "empty product path",
			productPath: "",
			want:        "options",
		},
		{
			name:        "google cloud product path",
			productPath: "google/cloud/echo/v1",
			want:        "google-cloud-echo-options",
		},
		{
			name:        "golden integration test product path",
			productPath: "generator/integration_tests/golden/v1",
			want:        "generator-integration_tests-golden-options",
		},
		{
			name:        "simple single segment path",
			productPath: "echo",
			want:        "echo-options",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := OptionsGroup(test.productPath)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
