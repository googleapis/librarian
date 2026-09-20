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
	Methods             []*methodAnnotations
	AsyncMethods        []*methodAnnotations
	HasIamUpdater       bool
	ClientClassComments string
	ProductOptionsPage  string
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

	connectionHeaderIncludes := []string{
		retryTraitsHeaderPath,
		idempotencyPolicyHeaderPath,
	}
	slices.Sort(connectionHeaderIncludes)

	connectionSourceIncludes := []string{
		connectionImplHeaderPath,
		optionDefaultsHeaderPath,
		stubFactoryHeaderPath,
		tracingConnectionHeaderPath,
		optionsHeaderPath,
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
		ConnectionImplSourcePath(productPath, s.Name),
		StubFactorySourcePath(productPath, s.Name),
		AuthDecoratorSourcePath(productPath, s.Name),
		LoggingDecoratorSourcePath(productPath, s.Name),
		MetadataDecoratorSourcePath(productPath, s.Name),
		StubSourcePath(productPath, s.Name),
		TracingStubSourcePath(productPath, s.Name),
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

		RetryPolicyOptionName:                              s.Name + "RetryPolicyOption",
		BackoffPolicyOptionName:                            s.Name + "BackoffPolicyOption",
		ConnectionIdempotencyPolicyOptionName:              s.Name + "ConnectionIdempotencyPolicyOption",
		PollingPolicyOptionName:                            s.Name + "PollingPolicyOption",
		ServicePolicyOptionListName:                        s.Name + "PolicyOptionList",
		ServiceDefaultOptionsFunctionName:                  s.Name + "DefaultOptions",
		CreateDefaultStubFunctionName:                      "CreateDefault" + s.Name + "Stub",
		MakeDefaultConnectionIdempotencyPolicyFunctionName: "MakeDefault" + s.Name + "ConnectionIdempotencyPolicy",
		ClientClassComments:                                formatClassCommentsFromServiceComments(s),
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

const fixedClientComment = `///
/// @par Equality
///
/// Instances of this class created via copy-construction or copy-assignment
/// always compare equal. Instances created with equal
/// ` + "`std::shared_ptr<*Connection>`" + ` objects compare equal. Objects that compare
/// equal share the same underlying resources.
///
/// @par Performance
///
/// Creating a new instance of this class is a relatively expensive operation,
/// new objects establish new connections to the service. In contrast,
/// copy-construction, move-construction, and the corresponding assignment
/// operations are relatively efficient as the copies share all underlying
/// resources.
///
/// @par Thread Safety
///
/// Concurrent access to different instances of this class, even if they compare
/// equal, is guaranteed to work. Two or more threads operating on the same
/// instance of this class is not guaranteed to work. Since copy-construction
/// and move-construction is a relatively efficient operation, consider using
/// such a copy when using this class from multiple threads.
///`

func formatClassCommentsFromServiceComments(s *api.Service) string {
	doc := strings.TrimSpace(s.Documentation)
	var formattedComments string
	if doc == "" {
		formattedComments = " " + s.Name + "Client"
	} else {
		r := strings.NewReplacer(
			"\n\n", "\n///\n/// ",
			"\n", "\n/// ",
			"[groups](#google.monitoring.v3.Group)", "[groups][google.monitoring.v3.Group]",
		)
		formattedComments = " " + r.Replace(doc)
	}
	res := "///\n///" + formattedComments + "\n" + fixedClientComment
	return strings.ReplaceAll(res, "///  ", "/// ")
}
