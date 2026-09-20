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
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type serviceAnnotations struct {
	Name          string
	CopyrightYear string
	BoilerPlate   []string
	Model         *modelAnnotations
	SourceFile    string

	// Namespaces
	Namespace                string
	InternalNamespace        string
	MocksNamespace           string
	ForwardingNamespace      string
	ForwardingMocksNamespace string

	// Header include guards
	ClientHeaderIncludeGuard                      string
	ConnectionHeaderIncludeGuard                  string
	IdempotencyPolicyHeaderIncludeGuard           string
	OptionsHeaderIncludeGuard                     string
	MockConnectionHeaderIncludeGuard              string
	OptionDefaultsHeaderIncludeGuard              string
	RetryTraitsHeaderIncludeGuard                 string
	TracingConnectionHeaderIncludeGuard           string
	ConnectionImplHeaderIncludeGuard              string
	StubFactoryHeaderIncludeGuard                 string
	AuthDecoratorHeaderIncludeGuard               string
	LoggingDecoratorHeaderIncludeGuard            string
	MetadataDecoratorHeaderIncludeGuard           string
	StubHeaderIncludeGuard                        string
	TracingStubHeaderIncludeGuard                 string
	ForwardingClientHeaderIncludeGuard            string
	ForwardingConnectionHeaderIncludeGuard        string
	ForwardingIdempotencyPolicyHeaderIncludeGuard string
	ForwardingOptionsHeaderIncludeGuard           string
	ForwardingMockConnectionHeaderIncludeGuard    string

	// REST Header include guards
	RestConnectionHeaderIncludeGuard        string
	RestConnectionImplHeaderIncludeGuard    string
	RestStubHeaderIncludeGuard              string
	RestStubFactoryHeaderIncludeGuard       string
	RestLoggingDecoratorHeaderIncludeGuard  string
	RestMetadataDecoratorHeaderIncludeGuard string
	RoundRobinHeaderIncludeGuard            string

	// Header paths
	ClientHeaderPath                      string
	ConnectionHeaderPath                  string
	IdempotencyPolicyHeaderPath           string
	OptionsHeaderPath                     string
	MockConnectionHeaderPath              string
	OptionDefaultsHeaderPath              string
	RetryTraitsHeaderPath                 string
	TracingConnectionHeaderPath           string
	ConnectionImplHeaderPath              string
	StubFactoryHeaderPath                 string
	AuthDecoratorHeaderPath               string
	LoggingDecoratorHeaderPath            string
	MetadataDecoratorHeaderPath           string
	StubHeaderPath                        string
	TracingStubHeaderPath                 string
	ForwardingClientHeaderPath            string
	ForwardingConnectionHeaderPath        string
	ForwardingIdempotencyPolicyHeaderPath string
	ForwardingOptionsHeaderPath           string
	ForwardingMockConnectionHeaderPath    string

	// REST Header paths
	RestConnectionHeaderPath        string
	RestConnectionImplHeaderPath    string
	RestStubHeaderPath              string
	RestStubFactoryHeaderPath       string
	RestLoggingDecoratorHeaderPath  string
	RestMetadataDecoratorHeaderPath string
	RoundRobinHeaderPath            string

	// Proto header paths
	ProtoHeaderPath     string
	ProtoGrpcHeaderPath string

	// Class names
	ClientClassName                      string
	ConnectionClassName                  string
	ConnectionIdempotencyPolicyClassName string
	MockConnectionClassName              string
	ConnectionImplClassName              string
	StubClassName                        string
	DefaultStubClassName                 string
	AuthDecoratorClassName               string
	LoggingDecoratorClassName            string
	MetadataDecoratorClassName           string
	TracingConnectionClassName           string
	TracingStubClassName                 string
	RetryPolicyName                      string
	LimitedErrorCountRetryPolicyName     string
	LimitedTimeRetryPolicyName           string
	RetryTraitsName                      string

	// REST Class names
	RestStubClassName                 string
	DefaultRestStubClassName          string
	RestLoggingDecoratorClassName     string
	RestMetadataDecoratorClassName    string
	RestConnectionImplClassName       string
	RoundRobinClassName               string
	MakeRestConnectionFunctionName    string
	CreateDefaultRestStubFunctionName string

	// Options and helpers
	RetryPolicyOptionName                              string
	BackoffPolicyOptionName                            string
	ConnectionIdempotencyPolicyOptionName              string
	PollingPolicyOptionName                            string
	ServicePolicyOptionListName                        string
	ServiceDefaultOptionsFunctionName                  string
	CreateDefaultStubFunctionName                      string
	MakeDefaultConnectionIdempotencyPolicyFunctionName string

	// Deprecation
	IsDeprecated bool

	// Retry status codes
	RetryStatusCodes []string

	// Feature flags
	HasLongrunningMethod         bool
	HasLRO                       bool
	HasBidirStreamingMethod      bool
	HasStreamingReadMethod       bool
	HasStreamingWriteMethod      bool
	HasAsyncStreamingReadMethod  bool
	HasAsyncStreamingWriteMethod bool
	HasStreamingMethod           bool
	HasPaginatedMethod           bool
	HasAsyncMethod               bool
	HasAsyncRpcs                 bool
	HasRequestId                 bool
	HasStreamRange               bool
	HasCompletionQueue           bool
	HasExplicitRoutingMethod     bool

	// REST & Transport configuration
	GenerateRestTransport         bool
	GenerateGrpcTransport         bool
	GenerateRoundRobinDecorator   bool
	HasGrpcLRO                    bool
	IsLocationOptionallyDependent bool
	ApiVersion                    string
	HasApiVersion                 bool

	// Header and source local includes
	ConnectionHeaderIncludes     []string
	ConnectionSourceIncludes     []string
	ConnectionImplHeaderIncludes []string
	SourcesCcIncludes            []string

	// Options & Endpoints
	ServiceEndpointEnvVar  string
	ServiceAuthorityEnvVar string
	EmulatorEndpointEnvVar string
	DefaultHost            string

	// Grpc Stub Names
	ServiceGrpcName      string
	ServiceGrpcProtoName string

	// Proto includes
	ConnectionProtoIncludes        []string
	IdempotencyPolicyProtoIncludes []string
	StubProtoIncludes              []string
	SourcesCcCopyrightYear         string
	SourcesContext                 *sourcesContextAnnotation

	// Mixins
	HasLocationMixin   bool
	HasIamMixin        bool
	HasOperationsMixin bool
	HasOperationsStub  bool

	// Signature and type checks
	HasMap                        bool
	HasDuration                   bool
	HasDeprecatedFieldInSignature bool

	// Methods
	Methods                     []*methodAnnotations
	AsyncMethods                []*methodAnnotations
	StubAsyncMethods            []*methodAnnotations
	RestMethods                 []*methodAnnotations
	RestAsyncMethods            []*methodAnnotations
	RestStubProtoIncludes       []string
	HasIamUpdater               bool
	DescriptionLines            []docLine
	ProductOptionsPage          string
	ClientCommentReferenceLines []string
	HasClientCommentReferences  bool
}

type docLine struct {
	Text       string
	HasContent bool
}

type sourcesContextAnnotation struct {
	UseSourcesYear bool
}

func (c *codec) annotateService(s *api.Service, modelAnn *modelAnnotations, model *api.API) error {
	year := modelAnn.CopyrightYear
	boilerPlate := modelAnn.BoilerPlate
	var sourceFile string
	if loc, ok := model.DefinitionLocation(s.ID); ok {
		sourceFile = loc.Filename
	}

	var productPath string
	var forwardingProductPath string
	if c.config != nil {
		productPath = c.config.ProductPath
		forwardingProductPath = c.config.ForwardingProductPath
	}

	clientHeaderPath := ClientHeaderPath(productPath, s.Name)
	connectionHeaderPath := ConnectionHeaderPath(productPath, s.Name)
	idempotencyPolicyHeaderPath := IdempotencyPolicyHeaderPath(productPath, s.Name)
	optionsHeaderPath := OptionsHeaderPath(productPath, s.Name)
	mockConnectionHeaderPath := MockConnectionHeaderPath(productPath, s.Name)
	optionDefaultsHeaderPath := OptionDefaultsHeaderPath(productPath, s.Name)
	retryTraitsHeaderPath := RetryTraitsHeaderPath(productPath, s.Name)
	tracingConnectionHeaderPath := TracingConnectionHeaderPath(productPath, s.Name)
	connectionImplHeaderPath := ConnectionImplHeaderPath(productPath, s.Name)
	stubFactoryHeaderPath := StubFactoryHeaderPath(productPath, s.Name)
	authDecoratorHeaderPath := AuthDecoratorHeaderPath(productPath, s.Name)
	loggingDecoratorHeaderPath := LoggingDecoratorHeaderPath(productPath, s.Name)
	metadataDecoratorHeaderPath := MetadataDecoratorHeaderPath(productPath, s.Name)
	stubHeaderPath := StubHeaderPath(productPath, s.Name)
	tracingStubHeaderPath := TracingStubHeaderPath(productPath, s.Name)

	restConnectionHeaderPath := RestConnectionHeaderPath(productPath, s.Name)
	restConnectionImplHeaderPath := RestConnectionImplHeaderPath(productPath, s.Name)
	restStubHeaderPath := RestStubHeaderPath(productPath, s.Name)
	restStubFactoryHeaderPath := RestStubFactoryHeaderPath(productPath, s.Name)
	restLoggingDecoratorHeaderPath := RestLoggingDecoratorHeaderPath(productPath, s.Name)
	restMetadataDecoratorHeaderPath := RestMetadataDecoratorHeaderPath(productPath, s.Name)
	roundRobinHeaderPath := RoundRobinHeaderPath(productPath, s.Name)

	restConnectionHeaderGuard := FormatHeaderIncludeGuard(restConnectionHeaderPath)
	restConnectionImplHeaderGuard := FormatHeaderIncludeGuard(restConnectionImplHeaderPath)
	restStubHeaderGuard := FormatHeaderIncludeGuard(restStubHeaderPath)
	restStubFactoryHeaderGuard := FormatHeaderIncludeGuard(restStubFactoryHeaderPath)
	restLoggingDecoratorHeaderGuard := FormatHeaderIncludeGuard(restLoggingDecoratorHeaderPath)
	restMetadataDecoratorHeaderGuard := FormatHeaderIncludeGuard(restMetadataDecoratorHeaderPath)
	roundRobinHeaderGuard := FormatHeaderIncludeGuard(roundRobinHeaderPath)

	var (
		forwardingNamespace      string
		forwardingMocksNamespace string

		forwardingClientHeaderPath            string
		forwardingConnectionHeaderPath        string
		forwardingIdempotencyPolicyHeaderPath string
		forwardingOptionsHeaderPath           string
		forwardingMockConnectionHeaderPath    string

		forwardingClientHeaderGuard            string
		forwardingConnectionHeaderGuard        string
		forwardingIdempotencyPolicyHeaderGuard string
		forwardingOptionsHeaderGuard           string
		forwardingMockConnectionHeaderGuard    string
	)
	if forwardingProductPath != "" {
		forwardingNamespace = Namespace(forwardingProductPath)
		forwardingMocksNamespace = MocksNamespace(forwardingProductPath)

		forwardingClientHeaderPath = ForwardingClientHeaderPath(forwardingProductPath, s.Name)
		forwardingConnectionHeaderPath = ForwardingConnectionHeaderPath(forwardingProductPath, s.Name)
		forwardingIdempotencyPolicyHeaderPath = ForwardingIdempotencyPolicyHeaderPath(forwardingProductPath, s.Name)
		forwardingOptionsHeaderPath = ForwardingOptionsHeaderPath(forwardingProductPath, s.Name)
		forwardingMockConnectionHeaderPath = ForwardingMockConnectionHeaderPath(forwardingProductPath, s.Name)

		forwardingClientHeaderGuard = FormatHeaderIncludeGuard(forwardingClientHeaderPath)
		forwardingConnectionHeaderGuard = FormatHeaderIncludeGuard(forwardingConnectionHeaderPath)
		forwardingIdempotencyPolicyHeaderGuard = FormatHeaderIncludeGuard(forwardingIdempotencyPolicyHeaderPath)
		forwardingOptionsHeaderGuard = FormatHeaderIncludeGuard(forwardingOptionsHeaderPath)
		forwardingMockConnectionHeaderGuard = FormatHeaderIncludeGuard(forwardingMockConnectionHeaderPath)
	}

	var protoHeaderPath, protoGrpcHeaderPath string
	if sourceFile != "" {
		base := strings.TrimSuffix(sourceFile, ".proto")
		protoHeaderPath = base + ".pb.h"
		protoGrpcHeaderPath = base + ".grpc.pb.h"
	}

	var (
		hasLongrunningMethod         bool
		hasBidirStreamingMethod      bool
		hasStreamingReadMethod       bool
		hasStreamingWriteMethod      bool
		hasAsyncStreamingReadMethod  bool
		hasAsyncStreamingWriteMethod bool
		hasPaginatedMethod           bool
		hasAsyncMethod               bool
		hasRequestId                 bool
		hasExplicitRoutingMethod     bool
	)

	isLongrunningPoller := func(m *api.Method) bool {
		if !strings.HasSuffix(m.SourceServiceID, "google.longrunning.Operations") {
			return false
		}
		if m.Name != "GetOperation" && m.Name != "CancelOperation" && m.Name != "WaitOperation" {
			return false
		}
		if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 || m.PathInfo.Bindings[0].PathTemplate == nil {
			return true
		}
		for _, seg := range m.PathInfo.Bindings[0].PathTemplate.Segments {
			if seg.Variable != nil && len(seg.Variable.Segments) > 0 && seg.Variable.Segments[0] == "operations" {
				return true
			}
		}
		return false
	}

	for _, m := range s.Methods {
		if isLongrunningPoller(m) {
			continue
		}
		isAsync := c.config != nil && (slices.Contains(c.config.GenAsyncRPCs, m.Name) || slices.Contains(c.config.GenAsyncRPCs, s.Name+"."+m.Name))
		if m.OperationInfo != nil || m.IsLRO {
			hasLongrunningMethod = true
		}
		if m.ClientSideStreaming && m.ServerSideStreaming {
			hasBidirStreamingMethod = true
		} else if m.ServerSideStreaming {
			hasStreamingReadMethod = true
			if isAsync {
				hasAsyncStreamingReadMethod = true
			}
		} else if m.ClientSideStreaming {
			hasStreamingWriteMethod = true
			if isAsync {
				hasAsyncStreamingWriteMethod = true
			}
		}
		if m.Pagination != nil {
			hasPaginatedMethod = true
		}
		if len(m.AutoPopulated) > 0 {
			hasRequestId = true
		}
		if len(m.Routing) > 0 {
			hasExplicitRoutingMethod = true
		}
		if isAsync {
			hasAsyncMethod = true
		}
	}
	if hasLongrunningMethod {
		hasAsyncMethod = true
	}

	generateRestTransport := false
	generateGrpcTransport := true
	generateRoundRobinDecorator := false
	isLocationOptionallyDependent := false
	if c.config != nil {
		generateRestTransport = c.config.GenerateRestTransport
		if c.config.GenerateGrpcTransport != nil {
			generateGrpcTransport = *c.config.GenerateGrpcTransport
		}
		generateRoundRobinDecorator = c.config.GenerateRoundRobinDecorator
		if c.config.EndpointLocationStyle == "LOCATION_OPTIONALLY_DEPENDENT" {
			isLocationOptionallyDependent = true
		}
	}
	hasGrpcLRO := hasLongrunningMethod && generateGrpcTransport

	var apiVersion string
	for _, m := range s.Methods {
		if m.APIVersion != "" {
			apiVersion = m.APIVersion
			break
		}
	}
	hasApiVersion := apiVersion != ""

	connectionHeaderIncludes := []string{
		retryTraitsHeaderPath,
		idempotencyPolicyHeaderPath,
	}
	slices.Sort(connectionHeaderIncludes)

	var connectionSourceIncludes []string
	connectionSourceIncludes = append(connectionSourceIncludes,
		optionDefaultsHeaderPath,
		tracingConnectionHeaderPath,
		optionsHeaderPath,
	)
	if generateGrpcTransport {
		connectionSourceIncludes = append(connectionSourceIncludes,
			connectionImplHeaderPath,
			stubFactoryHeaderPath,
		)
	}
	slices.Sort(connectionSourceIncludes)

	connectionImplHeaderIncludes := []string{
		retryTraitsHeaderPath,
		stubHeaderPath,
		connectionHeaderPath,
		idempotencyPolicyHeaderPath,
		optionsHeaderPath,
	}
	slices.Sort(connectionImplHeaderIncludes)

	sourcesCcIncludes := []string{
		ClientSourcePath(productPath, s.Name),
		ConnectionSourcePath(productPath, s.Name),
		IdempotencyPolicySourcePath(productPath, s.Name),
		OptionDefaultsSourcePath(productPath, s.Name),
		TracingConnectionSourcePath(productPath, s.Name),
	}
	if generateRestTransport {
		sourcesCcIncludes = append(sourcesCcIncludes,
			RestConnectionSourcePath(productPath, s.Name),
			RestConnectionImplSourcePath(productPath, s.Name),
			RestLoggingDecoratorSourcePath(productPath, s.Name),
			RestMetadataDecoratorSourcePath(productPath, s.Name),
			RestStubSourcePath(productPath, s.Name),
			RestStubFactorySourcePath(productPath, s.Name),
		)
	}
	if generateGrpcTransport {
		sourcesCcIncludes = append(sourcesCcIncludes,
			AuthDecoratorSourcePath(productPath, s.Name),
			ConnectionImplSourcePath(productPath, s.Name),
			LoggingDecoratorSourcePath(productPath, s.Name),
			MetadataDecoratorSourcePath(productPath, s.Name),
			StubSourcePath(productPath, s.Name),
			StubFactorySourcePath(productPath, s.Name),
			TracingStubSourcePath(productPath, s.Name),
		)
	}
	if generateRoundRobinDecorator {
		sourcesCcIncludes = append(sourcesCcIncludes,
			RoundRobinSourcePath(productPath, s.Name),
		)
	}
	slices.Sort(sourcesCcIncludes)

	var retryStatusCodes []string
	if c.config != nil && len(c.config.RetryableStatusCodes) > 0 {
		codeSet := make(map[string]bool)
		for _, rawCode := range c.config.RetryableStatusCodes {
			parts := strings.Split(rawCode, ".")
			if len(parts) == 1 {
				codeSet[parts[0]] = true
				continue
			}
			if len(parts) == 2 && parts[0] == s.Name {
				codeSet[parts[1]] = true
			}
		}
		for code := range codeSet {
			retryStatusCodes = append(retryStatusCodes, code)
		}
		slices.Sort(retryStatusCodes)
	}
	if len(retryStatusCodes) == 0 {
		retryStatusCodes = []string{"kDeadlineExceeded", "kUnavailable"}
	}

	serviceEndpointEnvVar := "GOOGLE_CLOUD_CPP_" + strings.ToUpper(CamelCaseToSnakeCase(s.Name)) + "_ENDPOINT"
	if c.config != nil && c.config.ServiceEndpointEnvVar != "" {
		serviceEndpointEnvVar = c.config.ServiceEndpointEnvVar
	}
	serviceAuthorityEnvVar := strings.TrimSuffix(serviceEndpointEnvVar, "_ENDPOINT") + "_AUTHORITY"
	var emulatorEndpointEnvVar string
	if c.config != nil {
		emulatorEndpointEnvVar = c.config.EmulatorEndpointEnvVar
	}
	defaultHost := s.DefaultHost
	serviceGrpcName := ProtoNameToCppName("." + s.Package + "." + s.Name)
	serviceGrpcProtoName := s.Package + "." + s.Name

	var restStubProtoIncludes []string
	if c.config != nil {
		for _, proto := range c.config.AdditionalProtoFiles {
			h := strings.TrimSuffix(proto, ".proto") + ".pb.h"
			if !slices.Contains(restStubProtoIncludes, h) {
				restStubProtoIncludes = append(restStubProtoIncludes, h)
			}
		}
	}
	var hasLocationMixin, hasIamMixin, hasOperationsMixin bool
	for _, m := range s.Methods {
		if isLongrunningPoller(m) {
			continue
		}
		if strings.Contains(m.SourceServiceID, "google.cloud.location.Locations") {
			hasLocationMixin = true
		}
		if strings.Contains(m.SourceServiceID, "google.iam.v1.IAMPolicy") {
			hasIamMixin = true
		}
		if strings.Contains(m.SourceServiceID, "google.longrunning.Operations") {
			hasOperationsMixin = true
		}
	}
	if hasLocationMixin && !slices.Contains(restStubProtoIncludes, "google/cloud/location/locations.pb.h") {
		restStubProtoIncludes = append(restStubProtoIncludes, "google/cloud/location/locations.pb.h")
	}
	if hasIamMixin && !slices.Contains(restStubProtoIncludes, "google/iam/v1/iam_policy.pb.h") {
		restStubProtoIncludes = append(restStubProtoIncludes, "google/iam/v1/iam_policy.pb.h")
	}
	if hasOperationsMixin || hasLongrunningMethod {
		if !slices.Contains(restStubProtoIncludes, "google/longrunning/operations.pb.h") {
			restStubProtoIncludes = append(restStubProtoIncludes, "google/longrunning/operations.pb.h")
		}
	}
	if protoHeaderPath != "" && !slices.Contains(restStubProtoIncludes, protoHeaderPath) {
		restStubProtoIncludes = append(restStubProtoIncludes, protoHeaderPath)
	}

	hasOperationsStub := hasOperationsMixin || hasLongrunningMethod

	sourcesCcCopyrightYear := max("2024", year)

	var connectionProtoIncludes []string
	if protoHeaderPath != "" {
		connectionProtoIncludes = append(connectionProtoIncludes, protoHeaderPath)
	}
	if c.config != nil {
		for _, proto := range c.config.AdditionalProtoFiles {
			h := strings.TrimSuffix(proto, ".proto") + ".pb.h"
			if !slices.Contains(connectionProtoIncludes, h) {
				connectionProtoIncludes = append(connectionProtoIncludes, h)
			}
		}
	}
	slices.Sort(connectionProtoIncludes)

	var idempotencyPolicyProtoIncludes []string
	if generateGrpcTransport {
		if protoGrpcHeaderPath != "" {
			idempotencyPolicyProtoIncludes = append(idempotencyPolicyProtoIncludes, protoGrpcHeaderPath)
		}
		if hasLocationMixin {
			idempotencyPolicyProtoIncludes = append(idempotencyPolicyProtoIncludes, "google/cloud/location/locations.grpc.pb.h")
		}
		if hasIamMixin {
			idempotencyPolicyProtoIncludes = append(idempotencyPolicyProtoIncludes, "google/iam/v1/iam_policy.grpc.pb.h")
		}
		if hasOperationsMixin {
			idempotencyPolicyProtoIncludes = append(idempotencyPolicyProtoIncludes, "google/longrunning/operations.grpc.pb.h")
		}
	} else {
		if protoHeaderPath != "" {
			idempotencyPolicyProtoIncludes = append(idempotencyPolicyProtoIncludes, protoHeaderPath)
		}
	}
	slices.Sort(idempotencyPolicyProtoIncludes)
	idempotencyPolicyProtoIncludes = slices.Compact(idempotencyPolicyProtoIncludes)

	var stubProtoIncludes []string
	var additionalPbHeaders []string
	if c.config != nil {
		for _, proto := range c.config.AdditionalProtoFiles {
			h := strings.TrimSuffix(proto, ".proto") + ".pb.h"
			if !slices.Contains(additionalPbHeaders, h) {
				additionalPbHeaders = append(additionalPbHeaders, h)
			}
		}
		slices.Sort(additionalPbHeaders)
		stubProtoIncludes = append(stubProtoIncludes, additionalPbHeaders...)
	}
	var mixinHeaders []string
	if hasLocationMixin {
		mixinHeaders = append(mixinHeaders, "google/cloud/location/locations.grpc.pb.h")
	}
	if hasIamMixin {
		mixinHeaders = append(mixinHeaders, "google/iam/v1/iam_policy.grpc.pb.h")
	}
	if hasOperationsMixin {
		mixinHeaders = append(mixinHeaders, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(mixinHeaders)
	stubProtoIncludes = append(stubProtoIncludes, mixinHeaders...)

	includeLroHeader := hasLongrunningMethod && !hasOperationsMixin
	var mainProtoGrpcHeaders []string
	if protoGrpcHeaderPath != "" {
		mainProtoGrpcHeaders = append(mainProtoGrpcHeaders, protoGrpcHeaderPath)
	}
	if includeLroHeader {
		mainProtoGrpcHeaders = append(mainProtoGrpcHeaders, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(mainProtoGrpcHeaders)
	stubProtoIncludes = append(stubProtoIncludes, mainProtoGrpcHeaders...)

	doc := s.Documentation
	refLines := formatClientCommentReferenceLines(doc, model, s)

	sAnn := &serviceAnnotations{
		Name:          s.Name,
		CopyrightYear: year,
		BoilerPlate:   boilerPlate,
		Model:         modelAnn,
		SourceFile:    sourceFile,

		ServiceEndpointEnvVar:  serviceEndpointEnvVar,
		ServiceAuthorityEnvVar: serviceAuthorityEnvVar,
		EmulatorEndpointEnvVar: emulatorEndpointEnvVar,
		DefaultHost:            defaultHost,

		ServiceGrpcName:      serviceGrpcName,
		ServiceGrpcProtoName: serviceGrpcProtoName,

		Namespace:                Namespace(productPath),
		InternalNamespace:        InternalNamespace(productPath),
		MocksNamespace:           MocksNamespace(productPath),
		ForwardingNamespace:      forwardingNamespace,
		ForwardingMocksNamespace: forwardingMocksNamespace,

		ClientHeaderIncludeGuard:            FormatHeaderIncludeGuard(clientHeaderPath),
		ConnectionHeaderIncludeGuard:        FormatHeaderIncludeGuard(connectionHeaderPath),
		IdempotencyPolicyHeaderIncludeGuard: FormatHeaderIncludeGuard(idempotencyPolicyHeaderPath),
		OptionsHeaderIncludeGuard:           FormatHeaderIncludeGuard(optionsHeaderPath),
		MockConnectionHeaderIncludeGuard:    FormatHeaderIncludeGuard(mockConnectionHeaderPath),
		OptionDefaultsHeaderIncludeGuard:    FormatHeaderIncludeGuard(optionDefaultsHeaderPath),
		RetryTraitsHeaderIncludeGuard:       FormatHeaderIncludeGuard(retryTraitsHeaderPath),
		TracingConnectionHeaderIncludeGuard: FormatHeaderIncludeGuard(tracingConnectionHeaderPath),
		ConnectionImplHeaderIncludeGuard:    FormatHeaderIncludeGuard(connectionImplHeaderPath),
		StubFactoryHeaderIncludeGuard:       FormatHeaderIncludeGuard(stubFactoryHeaderPath),
		AuthDecoratorHeaderIncludeGuard:     FormatHeaderIncludeGuard(authDecoratorHeaderPath),
		LoggingDecoratorHeaderIncludeGuard:  FormatHeaderIncludeGuard(loggingDecoratorHeaderPath),
		MetadataDecoratorHeaderIncludeGuard: FormatHeaderIncludeGuard(metadataDecoratorHeaderPath),
		StubHeaderIncludeGuard:              FormatHeaderIncludeGuard(stubHeaderPath),
		TracingStubHeaderIncludeGuard:       FormatHeaderIncludeGuard(tracingStubHeaderPath),

		ForwardingClientHeaderIncludeGuard:            forwardingClientHeaderGuard,
		ForwardingConnectionHeaderIncludeGuard:        forwardingConnectionHeaderGuard,
		ForwardingIdempotencyPolicyHeaderIncludeGuard: forwardingIdempotencyPolicyHeaderGuard,
		ForwardingOptionsHeaderIncludeGuard:           forwardingOptionsHeaderGuard,
		ForwardingMockConnectionHeaderIncludeGuard:    forwardingMockConnectionHeaderGuard,

		RestConnectionHeaderIncludeGuard:        restConnectionHeaderGuard,
		RestConnectionImplHeaderIncludeGuard:    restConnectionImplHeaderGuard,
		RestStubHeaderIncludeGuard:              restStubHeaderGuard,
		RestStubFactoryHeaderIncludeGuard:       restStubFactoryHeaderGuard,
		RestLoggingDecoratorHeaderIncludeGuard:  restLoggingDecoratorHeaderGuard,
		RestMetadataDecoratorHeaderIncludeGuard: restMetadataDecoratorHeaderGuard,
		RoundRobinHeaderIncludeGuard:            roundRobinHeaderGuard,

		ClientHeaderPath:            clientHeaderPath,
		ConnectionHeaderPath:        connectionHeaderPath,
		IdempotencyPolicyHeaderPath: idempotencyPolicyHeaderPath,
		OptionsHeaderPath:           optionsHeaderPath,
		MockConnectionHeaderPath:    mockConnectionHeaderPath,
		OptionDefaultsHeaderPath:    optionDefaultsHeaderPath,
		RetryTraitsHeaderPath:       retryTraitsHeaderPath,
		TracingConnectionHeaderPath: tracingConnectionHeaderPath,
		ConnectionImplHeaderPath:    connectionImplHeaderPath,
		StubFactoryHeaderPath:       stubFactoryHeaderPath,
		AuthDecoratorHeaderPath:     authDecoratorHeaderPath,
		LoggingDecoratorHeaderPath:  loggingDecoratorHeaderPath,
		MetadataDecoratorHeaderPath: metadataDecoratorHeaderPath,
		StubHeaderPath:              stubHeaderPath,
		TracingStubHeaderPath:       tracingStubHeaderPath,

		RestConnectionHeaderPath:        restConnectionHeaderPath,
		RestConnectionImplHeaderPath:    restConnectionImplHeaderPath,
		RestStubHeaderPath:              restStubHeaderPath,
		RestStubFactoryHeaderPath:       restStubFactoryHeaderPath,
		RestLoggingDecoratorHeaderPath:  restLoggingDecoratorHeaderPath,
		RestMetadataDecoratorHeaderPath: restMetadataDecoratorHeaderPath,
		RoundRobinHeaderPath:            roundRobinHeaderPath,

		ForwardingClientHeaderPath:            forwardingClientHeaderPath,
		ForwardingConnectionHeaderPath:        forwardingConnectionHeaderPath,
		ForwardingIdempotencyPolicyHeaderPath: forwardingIdempotencyPolicyHeaderPath,
		ForwardingOptionsHeaderPath:           forwardingOptionsHeaderPath,
		ForwardingMockConnectionHeaderPath:    forwardingMockConnectionHeaderPath,

		ProtoHeaderPath:     protoHeaderPath,
		ProtoGrpcHeaderPath: protoGrpcHeaderPath,

		ClientClassName:                      ClientClassName(s.Name),
		ConnectionClassName:                  ConnectionClassName(s.Name),
		ConnectionIdempotencyPolicyClassName: ConnectionIdempotencyPolicyClassName(s.Name),
		MockConnectionClassName:              MockConnectionClassName(s.Name),
		ConnectionImplClassName:              ConnectionImplClassName(s.Name),
		StubClassName:                        StubClassName(s.Name),
		DefaultStubClassName:                 DefaultStubClassName(s.Name),
		AuthDecoratorClassName:               AuthDecoratorClassName(s.Name),
		LoggingDecoratorClassName:            LoggingDecoratorClassName(s.Name),
		MetadataDecoratorClassName:           MetadataDecoratorClassName(s.Name),
		TracingConnectionClassName:           TracingConnectionClassName(s.Name),
		TracingStubClassName:                 TracingStubClassName(s.Name),
		RetryPolicyName:                      RetryPolicyName(s.Name),
		LimitedErrorCountRetryPolicyName:     LimitedErrorCountRetryPolicyName(s.Name),
		LimitedTimeRetryPolicyName:           LimitedTimeRetryPolicyName(s.Name),
		RetryTraitsName:                      RetryTraitsName(s.Name),

		RestStubClassName:                 RestStubClassName(s.Name),
		DefaultRestStubClassName:          DefaultRestStubClassName(s.Name),
		RestLoggingDecoratorClassName:     RestLoggingDecoratorClassName(s.Name),
		RestMetadataDecoratorClassName:    RestMetadataDecoratorClassName(s.Name),
		RestConnectionImplClassName:       RestConnectionImplClassName(s.Name),
		RoundRobinClassName:               RoundRobinClassName(s.Name),
		MakeRestConnectionFunctionName:    MakeRestConnectionFunctionName(s.Name),
		CreateDefaultRestStubFunctionName: CreateDefaultRestStubFunctionName(s.Name),

		RetryPolicyOptionName:                              s.Name + "RetryPolicyOption",
		BackoffPolicyOptionName:                            s.Name + "BackoffPolicyOption",
		ConnectionIdempotencyPolicyOptionName:              s.Name + "ConnectionIdempotencyPolicyOption",
		PollingPolicyOptionName:                            s.Name + "PollingPolicyOption",
		ServicePolicyOptionListName:                        s.Name + "PolicyOptionList",
		ServiceDefaultOptionsFunctionName:                  s.Name + "DefaultOptions",
		CreateDefaultStubFunctionName:                      "CreateDefault" + s.Name + "Stub",
		MakeDefaultConnectionIdempotencyPolicyFunctionName: "MakeDefault" + s.Name + "ConnectionIdempotencyPolicy",
		DescriptionLines:                                   formatServiceDescriptionLines(s),
		ProductOptionsPage:                                 OptionsGroup(productPath),
		ClientCommentReferenceLines:                        refLines,
		HasClientCommentReferences:                         len(refLines) > 0,

		IsDeprecated:     s.Deprecated,
		RetryStatusCodes: retryStatusCodes,

		HasLongrunningMethod:         hasLongrunningMethod,
		HasLRO:                       hasLongrunningMethod,
		HasBidirStreamingMethod:      hasBidirStreamingMethod,
		HasStreamingReadMethod:       hasStreamingReadMethod,
		HasStreamingWriteMethod:      hasStreamingWriteMethod,
		HasAsyncStreamingReadMethod:  hasAsyncStreamingReadMethod,
		HasAsyncStreamingWriteMethod: hasAsyncStreamingWriteMethod,
		HasStreamingMethod:           hasStreamingReadMethod || hasStreamingWriteMethod || hasBidirStreamingMethod,
		HasPaginatedMethod:           hasPaginatedMethod,
		HasAsyncMethod:               hasAsyncMethod,
		HasRequestId:                 hasRequestId,
		HasStreamRange:               hasStreamingReadMethod || hasPaginatedMethod,
		HasCompletionQueue:           hasAsyncMethod || hasBidirStreamingMethod,
		HasExplicitRoutingMethod:     hasExplicitRoutingMethod,

		GenerateRestTransport:         generateRestTransport,
		GenerateGrpcTransport:         generateGrpcTransport,
		GenerateRoundRobinDecorator:   generateRoundRobinDecorator,
		HasGrpcLRO:                    hasGrpcLRO,
		IsLocationOptionallyDependent: isLocationOptionallyDependent,
		ApiVersion:                    apiVersion,
		HasApiVersion:                 hasApiVersion,

		ConnectionHeaderIncludes:     connectionHeaderIncludes,
		ConnectionSourceIncludes:     connectionSourceIncludes,
		ConnectionImplHeaderIncludes: connectionImplHeaderIncludes,
		SourcesCcIncludes:            sourcesCcIncludes,
		RestStubProtoIncludes:        restStubProtoIncludes,

		ConnectionProtoIncludes:        connectionProtoIncludes,
		IdempotencyPolicyProtoIncludes: idempotencyPolicyProtoIncludes,
		StubProtoIncludes:              stubProtoIncludes,
		SourcesCcCopyrightYear:         sourcesCcCopyrightYear,
		SourcesContext:                 &sourcesContextAnnotation{UseSourcesYear: true},
		HasLocationMixin:               hasLocationMixin,
		HasIamMixin:                    hasIamMixin,
		HasOperationsMixin:             hasOperationsMixin,
		HasOperationsStub:              hasOperationsStub,
	}
	var getIamPolicyMethod, setIamPolicyMethod *api.Method
	for _, m := range s.Methods {
		respType := strings.TrimPrefix(m.OutputTypeID, ".")
		inputType := strings.TrimPrefix(m.InputTypeID, ".")
		if respType == "google.iam.v1.Policy" {
			for _, sig := range m.Signatures {
				sigStr := strings.Join(sig.Names, ",")
				if inputType == "google.iam.v1.GetIamPolicyRequest" && sigStr == "resource" {
					getIamPolicyMethod = m
				}
				if inputType == "google.iam.v1.SetIamPolicyRequest" && sigStr == "resource,policy" {
					setIamPolicyMethod = m
				}
			}
		}
	}
	hasIamUpdater := getIamPolicyMethod != nil && setIamPolicyMethod != nil

	var omittedRPCs []string
	var genAsyncRPCs []string
	if c.config != nil {
		omittedRPCs = c.config.OmittedRPCs
		genAsyncRPCs = c.config.GenAsyncRPCs
	}

	isOmitted := func(m *api.Method) bool {
		for _, rpc := range omittedRPCs {
			if rpc == m.Name || rpc == s.Name+"."+m.Name {
				return true
			}
		}
		return false
	}

	isGenAsync := func(m *api.Method) bool {
		for _, rpc := range genAsyncRPCs {
			if rpc == m.Name || rpc == s.Name+"."+m.Name {
				return true
			}
		}
		return false
	}

	var methods []*methodAnnotations
	var asyncMethods []*methodAnnotations
	var stubAsyncMethods []*methodAnnotations

	for _, m := range s.Methods {
		if isLongrunningPoller(m) {
			continue
		}
		if isOmitted(m) {
			continue
		}
		mAnn := c.annotateMethod(m, sAnn, model, hasIamUpdater && m == setIamPolicyMethod)
		methods = append(methods, mAnn)
		if isGenAsync(m) && !mAnn.IsLongrunning && !mAnn.IsBidirStreaming {
			stubAsyncMethods = append(stubAsyncMethods, mAnn)
			if mAnn.IsUnary {
				asyncMethods = append(asyncMethods, mAnn)
			}
		}
	}

	var hasDuration, hasDeprecatedFieldInSignature bool
	for _, mAnn := range methods {
		if mAnn.HasDeprecatedFieldInSignature {
			hasDeprecatedFieldInSignature = true
		}
		for _, sig := range mAnn.Signatures {
			if slices.ContainsFunc(sig.Parameters, func(p *parameterAnnotation) bool {
				return strings.Contains(p.Type, "google::protobuf::Duration")
			}) {
				hasDuration = true
			}
		}
	}
	hasMapField := func(msgID string) bool {
		msg := model.Message(msgID)
		return msg != nil && slices.ContainsFunc(msg.Fields, func(f *api.Field) bool { return f.Map })
	}
	hasMap := slices.ContainsFunc(s.Methods, func(m *api.Method) bool {
		return hasMapField(m.InputTypeID) || hasMapField(m.OutputTypeID)
	})

	var restMethods []*methodAnnotations
	for _, m := range methods {
		if m.IsRestRpc {
			restMethods = append(restMethods, m)
		}
	}
	var restAsyncMethods []*methodAnnotations
	for _, m := range asyncMethods {
		if m.IsRestRpc {
			restAsyncMethods = append(restAsyncMethods, m)
		}
	}

	sAnn.HasMap = hasMap
	sAnn.HasDuration = hasDuration
	sAnn.HasDeprecatedFieldInSignature = hasDeprecatedFieldInSignature
	sAnn.Methods = methods
	sAnn.AsyncMethods = asyncMethods
	sAnn.HasAsyncRpcs = len(stubAsyncMethods) > 0
	sAnn.StubAsyncMethods = stubAsyncMethods
	sAnn.RestMethods = restMethods
	sAnn.RestAsyncMethods = restAsyncMethods
	sAnn.HasIamUpdater = hasIamUpdater
	s.Codec = sAnn
	return nil
}

func formatServiceDescriptionLines(s *api.Service) []docLine {
	doc := strings.TrimSpace(s.Documentation)
	if doc == "" {
		return []docLine{{
			Text:       s.Name + "Client",
			HasContent: true,
		}}
	}
	rawLines := strings.Split(doc, "\n")
	lines := make([]docLine, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		lines = append(lines, docLine{
			Text:       trimmed,
			HasContent: trimmed != "",
		})
	}
	return lines
}

func serviceDependencyFiles(svc *api.Service, model *api.API) (map[string]bool, bool) {
	if svc == nil || model == nil {
		return nil, false
	}
	svcLoc, hasSvcLoc := model.DefinitionLocation(svc.ID)
	if !hasSvcLoc {
		return nil, false
	}
	depFiles := map[string]bool{
		svcLoc.Filename: true,
	}
	if deps, err := api.FindDependencies(model, []string{svc.ID}); err == nil {
		for d := range deps {
			if dLoc, ok := findSymbolLocation(model, d); ok {
				depFiles[dLoc.Filename] = true
			}
		}
	}
	return depFiles, true
}

func formatClientCommentReferenceLines(doc string, model *api.API, svc *api.Service) []string {
	depFiles, hasSvcLoc := serviceDependencyFiles(svc, model)

	refMap := make(map[string]api.SourceLocation)
	matches := commentRefRegex.FindAllStringSubmatch(doc, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		sym := match[1]
		loc, ok := findSymbolLocation(model, sym)
		if !ok {
			continue
		}
		if hasSvcLoc && !depFiles[loc.Filename] {
			continue
		}
		refMap[sym] = loc
	}
	refKeys := make([]string, 0, len(refMap))
	for k := range refMap {
		refKeys = append(refKeys, k)
	}
	slices.Sort(refKeys)

	refLines := make([]string, 0, len(refKeys))
	for _, k := range refKeys {
		loc := refMap[k]
		refLines = append(refLines, fmt.Sprintf("[%s]: @googleapis_reference_link{%s#L%d}", k, loc.Filename, loc.Line))
	}
	if len(refLines) == 0 {
		return nil
	}
	return refLines
}
