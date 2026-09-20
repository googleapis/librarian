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

	// Methods
	Methods            []*methodAnnotations
	AsyncMethods       []*methodAnnotations
	HasIamUpdater      bool
	DescriptionLines   []docLine
	ProductOptionsPage string
}

type docLine struct {
	Text       string
	HasContent bool
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
		return strings.HasSuffix(m.SourceServiceID, "google.longrunning.Operations") &&
			(m.Name == "GetOperation" || m.Name == "CancelOperation" || m.Name == "WaitOperation")
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
	serviceAuthorityEnvVar := "GOOGLE_CLOUD_CPP_" + strings.ToUpper(CamelCaseToSnakeCase(s.Name)) + "_AUTHORITY"
	var emulatorEndpointEnvVar string
	if c.config != nil {
		emulatorEndpointEnvVar = c.config.EmulatorEndpointEnvVar
	}
	defaultHost := s.DefaultHost
	serviceGrpcName := ProtoNameToCppName("." + s.Package + "." + s.Name)
	serviceGrpcProtoName := s.Package + "." + s.Name

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
			asyncMethods = append(asyncMethods, mAnn)
		}
	}

	sAnn.Methods = methods
	sAnn.AsyncMethods = asyncMethods
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
