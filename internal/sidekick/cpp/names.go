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
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/sidekick/language"
)

var includeGuardReplacer = strings.NewReplacer("/", "_", ".", "_")

// CamelCaseToSnakeCase converts a CamelCase string to snake_case, matching
// google-cloud-cpp's CamelCaseToSnakeCase in generator/internal/codegen_utils.cc.
func CamelCaseToSnakeCase(input string) string {
	var b strings.Builder
	for i := 0; i < len(input); i++ {
		c := input[i]
		lower := byte(unicode.ToLower(rune(c)))
		if c != '_' && i+2 < len(input) {
			if unicode.IsUpper(rune(input[i+1])) && unicode.IsLower(rune(input[i+2])) {
				b.WriteByte(lower)
				b.WriteByte('_')
				continue
			}
		}
		if c != '_' && i+1 < len(input) {
			if (unicode.IsLower(rune(c)) || unicode.IsDigit(rune(c))) && unicode.IsUpper(rune(input[i+1])) {
				b.WriteByte(lower)
				b.WriteByte('_')
				continue
			}
		}
		b.WriteByte(lower)
	}
	res := b.String()
	res = strings.ReplaceAll(res, "big_query", "bigquery")
	return res
}

// ServiceNameToFilePath converts a service name to its file path representation,
// mirroring google-cloud-cpp/generator/internal/codegen_utils.cc.
func ServiceNameToFilePath(serviceName string) string {
	components := strings.Split(serviceName, ".")
	if len(components) > 0 {
		components[len(components)-1] = strings.TrimSuffix(components[len(components)-1], "Service")
	}
	parts := make([]string, len(components))
	for i, c := range components {
		parts[i] = CamelCaseToSnakeCase(c)
	}
	return strings.Join(parts, "/")
}

// ClientHeaderPath returns the relative output path for a service client header.
func ClientHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_client.h")
}

// ClientSourcePath returns the relative output path for a service client source file.
func ClientSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_client.cc")
}

// ConnectionHeaderPath returns the relative output path for a service connection header.
func ConnectionHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_connection.h")
}

// ConnectionSourcePath returns the relative output path for a service connection source file.
func ConnectionSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_connection.cc")
}

// IdempotencyPolicyHeaderPath returns the relative output path for a service idempotency policy header.
func IdempotencyPolicyHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_connection_idempotency_policy.h")
}

// IdempotencyPolicySourcePath returns the relative output path for a service idempotency policy source file.
func IdempotencyPolicySourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_connection_idempotency_policy.cc")
}

// OptionsHeaderPath returns the relative output path for a service options header.
func OptionsHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_options.h")
}

// MockConnectionHeaderPath returns the relative output path for a mock service connection header.
func MockConnectionHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("mocks", "mock_"+ServiceNameToFilePath(serviceName)+"_connection.h"))
}

// OptionDefaultsHeaderPath returns the relative output path for an internal option defaults header.
func OptionDefaultsHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_option_defaults.h"))
}

// OptionDefaultsSourcePath returns the relative output path for an internal option defaults source file.
func OptionDefaultsSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_option_defaults.cc"))
}

// RetryTraitsHeaderPath returns the relative output path for an internal retry traits header.
func RetryTraitsHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_retry_traits.h"))
}

// TracingConnectionHeaderPath returns the relative output path for an internal tracing connection header.
func TracingConnectionHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_tracing_connection.h"))
}

// TracingConnectionSourcePath returns the relative output path for an internal tracing connection source file.
func TracingConnectionSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_tracing_connection.cc"))
}

// ConnectionImplHeaderPath returns the relative output path for an internal connection impl header.
func ConnectionImplHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_connection_impl.h"))
}

// ConnectionImplSourcePath returns the relative output path for an internal connection impl source file.
func ConnectionImplSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_connection_impl.cc"))
}

// StubFactoryHeaderPath returns the relative output path for an internal stub factory header.
func StubFactoryHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_stub_factory.h"))
}

// StubFactorySourcePath returns the relative output path for an internal stub factory source file.
func StubFactorySourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_stub_factory.cc"))
}

// AuthDecoratorHeaderPath returns the relative output path for an internal auth decorator header.
func AuthDecoratorHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_auth_decorator.h"))
}

// AuthDecoratorSourcePath returns the relative output path for an internal auth decorator source file.
func AuthDecoratorSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_auth_decorator.cc"))
}

// LoggingDecoratorHeaderPath returns the relative output path for an internal logging decorator header.
func LoggingDecoratorHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_logging_decorator.h"))
}

// LoggingDecoratorSourcePath returns the relative output path for an internal logging decorator source file.
func LoggingDecoratorSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_logging_decorator.cc"))
}

// MetadataDecoratorHeaderPath returns the relative output path for an internal metadata decorator header.
func MetadataDecoratorHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_metadata_decorator.h"))
}

// MetadataDecoratorSourcePath returns the relative output path for an internal metadata decorator source file.
func MetadataDecoratorSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_metadata_decorator.cc"))
}

// StubHeaderPath returns the relative output path for an internal stub header.
func StubHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_stub.h"))
}

// StubSourcePath returns the relative output path for an internal stub source file.
func StubSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_stub.cc"))
}

// TracingStubHeaderPath returns the relative output path for an internal tracing stub header.
func TracingStubHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_tracing_stub.h"))
}

// TracingStubSourcePath returns the relative output path for an internal tracing stub source file.
func TracingStubSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_tracing_stub.cc"))
}

// SourcesSourcePath returns the relative output path for an internal sources conglomerate file.
func SourcesSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_sources.cc"))
}

// RestConnectionHeaderPath returns the relative output path for a REST service connection header.
func RestConnectionHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_rest_connection.h")
}

// RestConnectionSourcePath returns the relative output path for a REST service connection source file.
func RestConnectionSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, ServiceNameToFilePath(serviceName)+"_rest_connection.cc")
}

// RestConnectionImplHeaderPath returns the relative output path for an internal REST connection impl header.
func RestConnectionImplHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_connection_impl.h"))
}

// RestConnectionImplSourcePath returns the relative output path for an internal REST connection impl source file.
func RestConnectionImplSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_connection_impl.cc"))
}

// RestStubFactoryHeaderPath returns the relative output path for an internal REST stub factory header.
func RestStubFactoryHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_stub_factory.h"))
}

// RestStubFactorySourcePath returns the relative output path for an internal REST stub factory source file.
func RestStubFactorySourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_stub_factory.cc"))
}

// RestLoggingDecoratorHeaderPath returns the relative output path for an internal REST logging decorator header.
func RestLoggingDecoratorHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_logging_decorator.h"))
}

// RestLoggingDecoratorSourcePath returns the relative output path for an internal REST logging decorator source file.
func RestLoggingDecoratorSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_logging_decorator.cc"))
}

// RestMetadataDecoratorHeaderPath returns the relative output path for an internal REST metadata decorator header.
func RestMetadataDecoratorHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_metadata_decorator.h"))
}

// RestMetadataDecoratorSourcePath returns the relative output path for an internal REST metadata decorator source file.
func RestMetadataDecoratorSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_metadata_decorator.cc"))
}

// RestStubHeaderPath returns the relative output path for an internal REST stub header.
func RestStubHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_stub.h"))
}

// RestStubSourcePath returns the relative output path for an internal REST stub source file.
func RestStubSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_rest_stub.cc"))
}

// RoundRobinHeaderPath returns the relative output path for an internal round-robin decorator header.
func RoundRobinHeaderPath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_round_robin_decorator.h"))
}

// RoundRobinSourcePath returns the relative output path for an internal round-robin decorator source file.
func RoundRobinSourcePath(productPath, serviceName string) string {
	return joinProductPath(productPath, path.Join("internal", ServiceNameToFilePath(serviceName)+"_round_robin_decorator.cc"))
}

// ForwardingClientHeaderPath returns the relative output path for a forwarding client header.
func ForwardingClientHeaderPath(forwardingPath, serviceName string) string {
	return joinProductPath(forwardingPath, ServiceNameToFilePath(serviceName)+"_client.h")
}

// ForwardingConnectionHeaderPath returns the relative output path for a forwarding connection header.
func ForwardingConnectionHeaderPath(forwardingPath, serviceName string) string {
	return joinProductPath(forwardingPath, ServiceNameToFilePath(serviceName)+"_connection.h")
}

// ForwardingIdempotencyPolicyHeaderPath returns the relative output path for a forwarding idempotency policy header.
func ForwardingIdempotencyPolicyHeaderPath(forwardingPath, serviceName string) string {
	return joinProductPath(forwardingPath, ServiceNameToFilePath(serviceName)+"_connection_idempotency_policy.h")
}

// ForwardingOptionsHeaderPath returns the relative output path for a forwarding options header.
func ForwardingOptionsHeaderPath(forwardingPath, serviceName string) string {
	return joinProductPath(forwardingPath, ServiceNameToFilePath(serviceName)+"_options.h")
}

// ForwardingMockConnectionHeaderPath returns the relative output path for a forwarding mock connection header.
func ForwardingMockConnectionHeaderPath(forwardingPath, serviceName string) string {
	return joinProductPath(forwardingPath, path.Join("mocks", "mock_"+ServiceNameToFilePath(serviceName)+"_connection.h"))
}

func joinProductPath(productPath, relPath string) string {
	cleanProduct := strings.TrimPrefix(path.Clean(productPath), "/")
	if cleanProduct == "." || cleanProduct == "" {
		return relPath
	}
	return path.Join(cleanProduct, relPath)
}

// CommonGeneratedFiles returns the 14 common generated files for a service.
func CommonGeneratedFiles(productPath, serviceName string) []language.GeneratedFile {
	return []language.GeneratedFile{
		{TemplatePath: "templates/service/client.h.mustache", OutputPath: ClientHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/client.cc.mustache", OutputPath: ClientSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/connection.h.mustache", OutputPath: ConnectionHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/connection.cc.mustache", OutputPath: ConnectionSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/connection_idempotency_policy.h.mustache", OutputPath: IdempotencyPolicyHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/connection_idempotency_policy.cc.mustache", OutputPath: IdempotencyPolicySourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/options.h.mustache", OutputPath: OptionsHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/mock_connection.h.mustache", OutputPath: MockConnectionHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/option_defaults.h.mustache", OutputPath: OptionDefaultsHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/option_defaults.cc.mustache", OutputPath: OptionDefaultsSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/retry_traits.h.mustache", OutputPath: RetryTraitsHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/tracing_connection.h.mustache", OutputPath: TracingConnectionHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/tracing_connection.cc.mustache", OutputPath: TracingConnectionSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/sources.cc.mustache", OutputPath: SourcesSourcePath(productPath, serviceName)},
	}
}

// GrpcGeneratedFiles returns the 14 gRPC-specific generated files for a service.
func GrpcGeneratedFiles(productPath, serviceName string) []language.GeneratedFile {
	return []language.GeneratedFile{
		{TemplatePath: "templates/service/connection_impl.h.mustache", OutputPath: ConnectionImplHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/connection_impl.cc.mustache", OutputPath: ConnectionImplSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/stub_factory.h.mustache", OutputPath: StubFactoryHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/stub_factory.cc.mustache", OutputPath: StubFactorySourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/auth_decorator.h.mustache", OutputPath: AuthDecoratorHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/auth_decorator.cc.mustache", OutputPath: AuthDecoratorSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/logging_decorator.h.mustache", OutputPath: LoggingDecoratorHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/logging_decorator.cc.mustache", OutputPath: LoggingDecoratorSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/metadata_decorator.h.mustache", OutputPath: MetadataDecoratorHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/metadata_decorator.cc.mustache", OutputPath: MetadataDecoratorSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/stub.h.mustache", OutputPath: StubHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/stub.cc.mustache", OutputPath: StubSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/tracing_stub.h.mustache", OutputPath: TracingStubHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/tracing_stub.cc.mustache", OutputPath: TracingStubSourcePath(productPath, serviceName)},
	}
}

// RestGeneratedFiles returns the 12 REST-specific generated files for a service.
func RestGeneratedFiles(productPath, serviceName string) []language.GeneratedFile {
	return []language.GeneratedFile{
		{TemplatePath: "templates/service/rest_connection.h.mustache", OutputPath: RestConnectionHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_connection.cc.mustache", OutputPath: RestConnectionSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_connection_impl.h.mustache", OutputPath: RestConnectionImplHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_connection_impl.cc.mustache", OutputPath: RestConnectionImplSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_stub_factory.h.mustache", OutputPath: RestStubFactoryHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_stub_factory.cc.mustache", OutputPath: RestStubFactorySourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_logging_decorator.h.mustache", OutputPath: RestLoggingDecoratorHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_logging_decorator.cc.mustache", OutputPath: RestLoggingDecoratorSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_metadata_decorator.h.mustache", OutputPath: RestMetadataDecoratorHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_metadata_decorator.cc.mustache", OutputPath: RestMetadataDecoratorSourcePath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_stub.h.mustache", OutputPath: RestStubHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/rest_stub.cc.mustache", OutputPath: RestStubSourcePath(productPath, serviceName)},
	}
}

// RoundRobinGeneratedFiles returns the 2 round-robin generated files for a service.
func RoundRobinGeneratedFiles(productPath, serviceName string) []language.GeneratedFile {
	return []language.GeneratedFile{
		{TemplatePath: "templates/service/round_robin_decorator.h.mustache", OutputPath: RoundRobinHeaderPath(productPath, serviceName)},
		{TemplatePath: "templates/service/round_robin_decorator.cc.mustache", OutputPath: RoundRobinSourcePath(productPath, serviceName)},
	}
}

// ServiceGeneratedFiles returns the 28 generated files for a gRPC service.
func ServiceGeneratedFiles(productPath, serviceName string) []language.GeneratedFile {
	var files []language.GeneratedFile
	files = append(files, CommonGeneratedFiles(productPath, serviceName)...)
	files = append(files, GrpcGeneratedFiles(productPath, serviceName)...)
	return files
}

// ForwardingGeneratedFiles returns the 5 forwarding header files for a service.
func ForwardingGeneratedFiles(forwardingPath, serviceName string) []language.GeneratedFile {
	return []language.GeneratedFile{
		{TemplatePath: "templates/service/forwarding_client.h.mustache", OutputPath: ForwardingClientHeaderPath(forwardingPath, serviceName)},
		{TemplatePath: "templates/service/forwarding_connection.h.mustache", OutputPath: ForwardingConnectionHeaderPath(forwardingPath, serviceName)},
		{TemplatePath: "templates/service/forwarding_connection_idempotency_policy.h.mustache", OutputPath: ForwardingIdempotencyPolicyHeaderPath(forwardingPath, serviceName)},
		{TemplatePath: "templates/service/forwarding_options.h.mustache", OutputPath: ForwardingOptionsHeaderPath(forwardingPath, serviceName)},
		{TemplatePath: "templates/service/forwarding_mock_connection.h.mustache", OutputPath: ForwardingMockConnectionHeaderPath(forwardingPath, serviceName)},
	}
}

// FormatHeaderIncludeGuard generates the C++ include guard macro for a header path.
// It matches google-cloud-cpp's FormatHeaderIncludeGuard in generator/internal/codegen_utils.cc.
func FormatHeaderIncludeGuard(headerPath string) string {
	if headerPath == "" {
		return ""
	}
	clean := strings.TrimPrefix(path.Clean(headerPath), "/")
	if clean == "." || clean == "" {
		return ""
	}
	return "GOOGLE_CLOUD_CPP_" + strings.ToUpper(includeGuardReplacer.Replace(clean))
}

// parseProductPath splits a product path into prefix, library name, and service subdirectory,
// matching google-cloud-cpp's ParseProductPath in generator/internal/scaffold_generator.cc.
func parseProductPath(productPath string) (prefix, libraryName, serviceSubdir string) {
	cleaned := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(filepath.ToSlash(productPath))), "./")
	if cleaned == "." || cleaned == "" {
		return "", "", ""
	}
	raw := strings.Split(cleaned, "/")
	var v []string
	for _, part := range raw {
		if part != "" {
			v = append(v, part)
		}
	}
	if len(v) == 0 {
		return "", "", ""
	}
	it := len(v) - 1
	if len(v) > 2 && v[0] == "google" && v[1] == "cloud" {
		it = 2
	} else {
		// Parity with google-cloud-cpp's ParseProductPath in
		// generator/internal/scaffold_generator.cc:53, which checks for "golden"
		// to support integration test golden directory structures.
		for i, part := range v {
			if part == "golden" {
				it = i
				break
			}
		}
	}
	prefix = strings.Join(v[:it], "/")
	libraryName = v[it]
	serviceSubdir = strings.Join(v[it+1:], "/")
	return prefix, libraryName, serviceSubdir
}

// Namespace returns the C++ namespace for the given product path, matching
// google-cloud-cpp's Namespace(product_path, NamespaceType::kNormal).
func Namespace(productPath string) string {
	_, lib, subdir := parseProductPath(productPath)
	if lib == "" {
		return ""
	}
	if subdir == "" {
		return lib
	}
	return lib + "_" + strings.ReplaceAll(subdir, "/", "_")
}

// InternalNamespace returns the C++ internal namespace for the given product path, matching
// google-cloud-cpp's Namespace(product_path, NamespaceType::kInternal).
func InternalNamespace(productPath string) string {
	ns := Namespace(productPath)
	if ns == "" {
		return "internal"
	}
	return ns + "_internal"
}

// MocksNamespace returns the C++ mocks namespace for the given product path, matching
// google-cloud-cpp's Namespace(product_path, NamespaceType::kMocks).
func MocksNamespace(productPath string) string {
	ns := Namespace(productPath)
	if ns == "" {
		return "mocks"
	}
	return ns + "_mocks"
}

// OptionsGroup returns the Doxygen options group name for a product path,
// matching google-cloud-cpp's OptionsGroup(product_path).
func OptionsGroup(productPath string) string {
	prefix, lib, _ := parseProductPath(productPath)
	if lib == "" {
		return "options"
	}
	libPath := lib + "/"
	if prefix != "" {
		libPath = prefix + "/" + lib + "/"
	}
	return strings.ReplaceAll(libPath, "/", "-") + "options"
}

// ClientClassName returns the C++ client class name for a service.
func ClientClassName(serviceName string) string {
	return serviceName + "Client"
}

// ConnectionClassName returns the C++ connection class name for a service.
func ConnectionClassName(serviceName string) string {
	return serviceName + "Connection"
}

// ConnectionIdempotencyPolicyClassName returns the C++ idempotency policy class name for a service.
func ConnectionIdempotencyPolicyClassName(serviceName string) string {
	return serviceName + "ConnectionIdempotencyPolicy"
}

// MockConnectionClassName returns the C++ mock connection class name for a service.
func MockConnectionClassName(serviceName string) string {
	return "Mock" + serviceName + "Connection"
}

// ConnectionImplClassName returns the C++ connection implementation class name for a service.
func ConnectionImplClassName(serviceName string) string {
	return serviceName + "ConnectionImpl"
}

// StubClassName returns the C++ stub class name for a service.
func StubClassName(serviceName string) string {
	return serviceName + "Stub"
}

// DefaultStubClassName returns the C++ default stub class name for a service.
func DefaultStubClassName(serviceName string) string {
	return "Default" + serviceName + "Stub"
}

// AuthDecoratorClassName returns the C++ auth decorator class name for a service.
func AuthDecoratorClassName(serviceName string) string {
	return serviceName + "Auth"
}

// LoggingDecoratorClassName returns the C++ logging decorator class name for a service.
func LoggingDecoratorClassName(serviceName string) string {
	return serviceName + "Logging"
}

// MetadataDecoratorClassName returns the C++ metadata decorator class name for a service.
func MetadataDecoratorClassName(serviceName string) string {
	return serviceName + "Metadata"
}

// TracingConnectionClassName returns the C++ tracing connection class name for a service.
func TracingConnectionClassName(serviceName string) string {
	return serviceName + "TracingConnection"
}

// TracingStubClassName returns the C++ tracing stub class name for a service.
func TracingStubClassName(serviceName string) string {
	return serviceName + "TracingStub"
}

// RetryPolicyName returns the C++ retry policy class name for a service.
func RetryPolicyName(serviceName string) string {
	return serviceName + "RetryPolicy"
}

// LimitedErrorCountRetryPolicyName returns the C++ limited error count retry policy class name for a service.
func LimitedErrorCountRetryPolicyName(serviceName string) string {
	return serviceName + "LimitedErrorCountRetryPolicy"
}

// LimitedTimeRetryPolicyName returns the C++ limited time retry policy class name for a service.
func LimitedTimeRetryPolicyName(serviceName string) string {
	return serviceName + "LimitedTimeRetryPolicy"
}

// RetryTraitsName returns the C++ retry traits struct name for a service.
func RetryTraitsName(serviceName string) string {
	return serviceName + "RetryTraits"
}

// RestStubClassName returns the C++ REST stub class name for a service.
func RestStubClassName(serviceName string) string {
	return serviceName + "RestStub"
}

// DefaultRestStubClassName returns the C++ default REST stub class name for a service.
func DefaultRestStubClassName(serviceName string) string {
	return "Default" + serviceName + "RestStub"
}

// RestLoggingDecoratorClassName returns the C++ REST logging decorator class name for a service.
func RestLoggingDecoratorClassName(serviceName string) string {
	return serviceName + "RestLogging"
}

// RestMetadataDecoratorClassName returns the C++ REST metadata decorator class name for a service.
func RestMetadataDecoratorClassName(serviceName string) string {
	return serviceName + "RestMetadata"
}

// RestConnectionImplClassName returns the C++ REST connection implementation class name for a service.
func RestConnectionImplClassName(serviceName string) string {
	return serviceName + "RestConnectionImpl"
}

// RoundRobinClassName returns the C++ round-robin decorator class name for a service.
func RoundRobinClassName(serviceName string) string {
	return serviceName + "RoundRobin"
}

// MakeRestConnectionFunctionName returns the factory function name for a REST connection.
func MakeRestConnectionFunctionName(serviceName string) string {
	return "Make" + serviceName + "ConnectionRest"
}

// CreateDefaultRestStubFunctionName returns the factory function name for a default REST stub.
func CreateDefaultRestStubFunctionName(serviceName string) string {
	return "CreateDefault" + serviceName + "RestStub"
}

// ProtoNameToCppName converts a fully qualified protobuf name to its C++ equivalent,
// stripping any leading dot and replacing dots with '::'.
// For example, ".google.protobuf.Empty" becomes "google::protobuf::Empty".
func ProtoNameToCppName(protoName string) string {
	protoName = strings.TrimPrefix(protoName, ".")
	return strings.ReplaceAll(protoName, ".", "::")
}
