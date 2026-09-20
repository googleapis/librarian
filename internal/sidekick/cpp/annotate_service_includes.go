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

type sourcesContextAnnotation struct {
	UseSourcesYear bool
}

type serviceMixins struct {
	hasLocation   bool
	hasIam        bool
	hasOperations bool
}

func isLongrunningPoller(m *api.Method) bool {
	if m == nil {
		return false
	}
	if !strings.HasSuffix(m.SourceServiceID, "google.longrunning.Operations") {
		return false
	}
	if m.Name != "GetOperation" && m.Name != "CancelOperation" && m.Name != "WaitOperation" {
		return false
	}
	if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 || m.PathInfo.Bindings[0] == nil || m.PathInfo.Bindings[0].PathTemplate == nil {
		return true
	}
	return slices.ContainsFunc(m.PathInfo.Bindings[0].PathTemplate.Segments, func(seg api.PathSegment) bool {
		return seg.Variable != nil && len(seg.Variable.Segments) > 0 && seg.Variable.Segments[0] == "operations"
	})
}

func detectServiceMixins(s *api.Service) serviceMixins {
	var mixins serviceMixins
	if s == nil {
		return mixins
	}
	for _, m := range s.Methods {
		if m == nil {
			continue
		}
		if isLongrunningPoller(m) {
			continue
		}
		if strings.Contains(m.SourceServiceID, "google.cloud.location.Locations") {
			mixins.hasLocation = true
		}
		if strings.Contains(m.SourceServiceID, "google.iam.v1.IAMPolicy") {
			mixins.hasIam = true
		}
		if strings.Contains(m.SourceServiceID, "google.longrunning.Operations") {
			mixins.hasOperations = true
		}
	}
	return mixins
}

func connectionHeaderIncludes(retryTraitsHeaderPath, idempotencyPolicyHeaderPath string) []string {
	includes := []string{
		retryTraitsHeaderPath,
		idempotencyPolicyHeaderPath,
	}
	slices.Sort(includes)
	return includes
}

func connectionSourceIncludes(
	optionDefaultsHeaderPath,
	tracingConnectionHeaderPath,
	optionsHeaderPath,
	connectionImplHeaderPath,
	stubFactoryHeaderPath string,
	generateGrpcTransport bool,
) []string {
	includes := []string{
		optionDefaultsHeaderPath,
		tracingConnectionHeaderPath,
		optionsHeaderPath,
	}
	if generateGrpcTransport {
		includes = append(includes,
			connectionImplHeaderPath,
			stubFactoryHeaderPath,
		)
	}
	slices.Sort(includes)
	return includes
}

func connectionImplHeaderIncludes(
	retryTraitsHeaderPath,
	stubHeaderPath,
	connectionHeaderPath,
	idempotencyPolicyHeaderPath,
	optionsHeaderPath string,
) []string {
	includes := []string{
		retryTraitsHeaderPath,
		stubHeaderPath,
		connectionHeaderPath,
		idempotencyPolicyHeaderPath,
		optionsHeaderPath,
	}
	slices.Sort(includes)
	return includes
}

func sourcesCcIncludes(
	productPath, serviceName string,
	generateRestTransport, generateGrpcTransport, generateRoundRobinDecorator bool,
) []string {
	includes := []string{
		ClientSourcePath(productPath, serviceName),
		ConnectionSourcePath(productPath, serviceName),
		IdempotencyPolicySourcePath(productPath, serviceName),
		OptionDefaultsSourcePath(productPath, serviceName),
		TracingConnectionSourcePath(productPath, serviceName),
	}
	if generateRestTransport {
		includes = append(includes,
			RestConnectionSourcePath(productPath, serviceName),
			RestConnectionImplSourcePath(productPath, serviceName),
			RestLoggingDecoratorSourcePath(productPath, serviceName),
			RestMetadataDecoratorSourcePath(productPath, serviceName),
			RestStubSourcePath(productPath, serviceName),
			RestStubFactorySourcePath(productPath, serviceName),
		)
	}
	if generateGrpcTransport {
		includes = append(includes,
			AuthDecoratorSourcePath(productPath, serviceName),
			ConnectionImplSourcePath(productPath, serviceName),
			LoggingDecoratorSourcePath(productPath, serviceName),
			MetadataDecoratorSourcePath(productPath, serviceName),
			StubSourcePath(productPath, serviceName),
			StubFactorySourcePath(productPath, serviceName),
			TracingStubSourcePath(productPath, serviceName),
		)
	}
	if generateRoundRobinDecorator {
		includes = append(includes,
			RoundRobinSourcePath(productPath, serviceName),
		)
	}
	slices.Sort(includes)
	return includes
}

func restStubProtoIncludes(
	additionalProtoFiles []string,
	hasLocationMixin, hasIamMixin, hasOperationsMixin, hasLongrunningMethod bool,
	protoHeaderPath string,
) []string {
	var includes []string
	for _, proto := range additionalProtoFiles {
		h := strings.TrimSuffix(proto, ".proto") + ".pb.h"
		if !slices.Contains(includes, h) {
			includes = append(includes, h)
		}
	}
	if hasLocationMixin && !slices.Contains(includes, "google/cloud/location/locations.pb.h") {
		includes = append(includes, "google/cloud/location/locations.pb.h")
	}
	if hasIamMixin && !slices.Contains(includes, "google/iam/v1/iam_policy.pb.h") {
		includes = append(includes, "google/iam/v1/iam_policy.pb.h")
	}
	if (hasOperationsMixin || hasLongrunningMethod) && !slices.Contains(includes, "google/longrunning/operations.pb.h") {
		includes = append(includes, "google/longrunning/operations.pb.h")
	}
	if protoHeaderPath != "" && !slices.Contains(includes, protoHeaderPath) {
		includes = append(includes, protoHeaderPath)
	}
	return includes
}

func sourcesCcCopyrightYear(year string) string {
	return max("2024", year)
}

func connectionProtoIncludes(protoHeaderPath string, additionalProtoFiles []string) []string {
	var includes []string
	if protoHeaderPath != "" {
		includes = append(includes, protoHeaderPath)
	}
	for _, proto := range additionalProtoFiles {
		h := strings.TrimSuffix(proto, ".proto") + ".pb.h"
		if !slices.Contains(includes, h) {
			includes = append(includes, h)
		}
	}
	slices.Sort(includes)
	return includes
}

func idempotencyPolicyProtoIncludes(
	generateGrpcTransport bool,
	protoGrpcHeaderPath, protoHeaderPath string,
	hasLocationMixin, hasIamMixin, hasOperationsMixin bool,
) []string {
	if !generateGrpcTransport {
		if protoHeaderPath != "" {
			return []string{protoHeaderPath}
		}
		return nil
	}

	var includes []string
	if protoGrpcHeaderPath != "" {
		includes = append(includes, protoGrpcHeaderPath)
	}
	if hasLocationMixin {
		includes = append(includes, "google/cloud/location/locations.grpc.pb.h")
	}
	if hasIamMixin {
		includes = append(includes, "google/iam/v1/iam_policy.grpc.pb.h")
	}
	if hasOperationsMixin {
		includes = append(includes, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(includes)
	return slices.Compact(includes)
}

func stubProtoIncludes(
	additionalProtoFiles []string,
	hasLocationMixin, hasIamMixin, hasOperationsMixin, hasLongrunningMethod bool,
	protoGrpcHeaderPath string,
) []string {
	var includes []string
	var additionalPbHeaders []string
	for _, proto := range additionalProtoFiles {
		h := strings.TrimSuffix(proto, ".proto") + ".pb.h"
		if !slices.Contains(additionalPbHeaders, h) {
			additionalPbHeaders = append(additionalPbHeaders, h)
		}
	}
	slices.Sort(additionalPbHeaders)
	includes = append(includes, additionalPbHeaders...)

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
	includes = append(includes, mixinHeaders...)

	includeLroHeader := hasLongrunningMethod && !hasOperationsMixin
	var mainProtoGrpcHeaders []string
	if protoGrpcHeaderPath != "" {
		mainProtoGrpcHeaders = append(mainProtoGrpcHeaders, protoGrpcHeaderPath)
	}
	if includeLroHeader {
		mainProtoGrpcHeaders = append(mainProtoGrpcHeaders, "google/longrunning/operations.grpc.pb.h")
	}
	slices.Sort(mainProtoGrpcHeaders)
	includes = append(includes, mainProtoGrpcHeaders...)

	return includes
}

func populateServiceIncludes(
	sAnn *serviceAnnotations,
	s *api.Service,
	productPath string,
	additionalProtoFiles []string,
	year string,
) *serviceAnnotations {
	if sAnn == nil {
		sAnn = &serviceAnnotations{}
	}
	mixins := detectServiceMixins(s)
	sAnn.HasLocationMixin = mixins.hasLocation
	sAnn.HasIamMixin = mixins.hasIam
	sAnn.HasOperationsMixin = mixins.hasOperations
	sAnn.HasOperationsStub = mixins.hasOperations || sAnn.HasLongrunningMethod
	sAnn.SourcesCcCopyrightYear = sourcesCcCopyrightYear(year)
	sAnn.SourcesContext = &sourcesContextAnnotation{UseSourcesYear: true}

	sAnn.ConnectionHeaderIncludes = connectionHeaderIncludes(
		sAnn.RetryTraitsHeaderPath,
		sAnn.IdempotencyPolicyHeaderPath,
	)
	sAnn.ConnectionSourceIncludes = connectionSourceIncludes(
		sAnn.OptionDefaultsHeaderPath,
		sAnn.TracingConnectionHeaderPath,
		sAnn.OptionsHeaderPath,
		sAnn.ConnectionImplHeaderPath,
		sAnn.StubFactoryHeaderPath,
		sAnn.GenerateGrpcTransport,
	)
	sAnn.ConnectionImplHeaderIncludes = connectionImplHeaderIncludes(
		sAnn.RetryTraitsHeaderPath,
		sAnn.StubHeaderPath,
		sAnn.ConnectionHeaderPath,
		sAnn.IdempotencyPolicyHeaderPath,
		sAnn.OptionsHeaderPath,
	)
	serviceName := sAnn.Name
	if serviceName == "" && s != nil {
		serviceName = s.Name
	}
	sAnn.SourcesCcIncludes = sourcesCcIncludes(
		productPath,
		serviceName,
		sAnn.GenerateRestTransport,
		sAnn.GenerateGrpcTransport,
		sAnn.GenerateRoundRobinDecorator,
	)
	includeLroInRestStub := sAnn.HasLongrunningMethod && !sAnn.HasComputeLRO
	sAnn.RestStubProtoIncludes = restStubProtoIncludes(
		additionalProtoFiles,
		sAnn.HasLocationMixin,
		sAnn.HasIamMixin,
		sAnn.HasOperationsMixin,
		includeLroInRestStub,
		sAnn.ProtoHeaderPath,
	)
	if sAnn.HasLongrunningMethod && sAnn.HasComputeLRO && sAnn.LongrunningOperationIncludeHeader != "" {
		if !slices.Contains(sAnn.RestStubProtoIncludes, sAnn.LongrunningOperationIncludeHeader) {
			sAnn.RestStubProtoIncludes = append(sAnn.RestStubProtoIncludes, sAnn.LongrunningOperationIncludeHeader)
			slices.Sort(sAnn.RestStubProtoIncludes)
		}
	}
	sAnn.ConnectionProtoIncludes = connectionProtoIncludes(
		sAnn.ProtoHeaderPath,
		additionalProtoFiles,
	)
	sAnn.IdempotencyPolicyProtoIncludes = idempotencyPolicyProtoIncludes(
		sAnn.GenerateGrpcTransport,
		sAnn.ProtoGrpcHeaderPath,
		sAnn.ProtoHeaderPath,
		sAnn.HasLocationMixin,
		sAnn.HasIamMixin,
		sAnn.HasOperationsMixin,
	)
	sAnn.StubProtoIncludes = stubProtoIncludes(
		additionalProtoFiles,
		sAnn.HasLocationMixin,
		sAnn.HasIamMixin,
		sAnn.HasOperationsMixin,
		sAnn.HasLongrunningMethod,
		sAnn.ProtoGrpcHeaderPath,
	)
	return sAnn
}
