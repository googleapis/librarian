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

	// Feature flags
	HasLongrunningMethod         bool
	HasBidirStreamingMethod      bool
	HasStreamingReadMethod       bool
	HasStreamingWriteMethod      bool
	HasAsyncStreamingReadMethod  bool
	HasAsyncStreamingWriteMethod bool
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

	for _, m := range s.Methods {
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

	sAnn := &serviceAnnotations{
		Name:          s.Name,
		CopyrightYear: year,
		BoilerPlate:   boilerPlate,
		Model:         modelAnn,
		SourceFile:    sourceFile,

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

		HasLongrunningMethod:         hasLongrunningMethod,
		HasBidirStreamingMethod:      hasBidirStreamingMethod,
		HasStreamingReadMethod:       hasStreamingReadMethod,
		HasStreamingWriteMethod:      hasStreamingWriteMethod,
		HasAsyncStreamingReadMethod:  hasAsyncStreamingReadMethod,
		HasAsyncStreamingWriteMethod: hasAsyncStreamingWriteMethod,
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
	s.Codec = sAnn
	for _, m := range s.Methods {
		if err := c.annotateMethod(m, sAnn); err != nil {
			return err
		}
	}
	return nil
}
