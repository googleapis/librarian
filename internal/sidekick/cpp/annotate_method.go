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

type methodAnnotations struct {
	Service     *serviceAnnotations
	ServiceName string
	Name        string

	// Types
	CppRequestType  string
	CppResponseType string
	CppReturnType   string

	// Method classifications
	IsDeprecated                   bool
	IsUnary                        bool
	IsPaginated                    bool
	RangeOutputType                string
	IsLongrunning                  bool
	LongrunningDeducedResponseType string
	LongrunningReturnsEmpty        bool
	LongrunningOperationType       string
	IsComputeLRO                   bool
	IsStreamingRead                bool
	IsStreamingWrite               bool
	IsBidirStreaming               bool
	IsStreaming                    bool
	IsResponseTypeEmpty            bool
	IsAsync                        bool

	// Routing
	HasRouting           bool
	HasHttpRouting       bool
	HttpRoutingParams    []*httpRoutingParamAnnotation
	RoutingParamsCount   int
	RoutingParamMatchers []*routingMatcherAnnotation

	// Stub
	StubMemberName string

	// REST
	HasRestPath             bool
	RestVerb                string
	RestPathSegments        []*restPathSegmentAnnotation
	RestPathVerb            string
	HasRestPathVerb         bool
	RestQueryParams         []*queryParamAnnotation
	RestHasQueryParams      bool
	RestRequestBodyAccessor string
	RestPayloadType         string
	RestReturnTypeName      string
	IsRestRpc               bool

	// Request ID
	HasRequestId       bool
	RequestIdFieldName string

	// Idempotency
	Idempotency string

	// IAM Policy
	IsSetIamPolicy bool

	// LRO & Pagination
	LongrunningMetadataType       string
	LongrunningResponseIsEmpty    bool
	PaginationElementsFieldName   string
	HasDeprecatedFieldInSignature bool

	// Comments
	Comments          string
	RequestComments   string
	NoAwaitComments   string
	OperationComments string
	DocLines          []string

	// Signatures (overloads)
	Signatures []*methodSignatureAnnotations
}

type httpRoutingParamAnnotation struct {
	Key           string
	FieldAccessor string
	HasNext       bool
}

type restPathSegmentAnnotation struct {
	IsApiVersion  bool
	ApiVersion    string
	IsLiteral     bool
	Literal       string
	IsField       bool
	FieldAccessor string
	HasNext       bool
}

type queryParamAnnotation struct {
	ParamKey      string
	FieldAccessor string
	IsString      bool
	IsNumber      bool
	IsBool        bool
}

type routingPatternAnnotation struct {
	FieldAccessor string
	Pattern       string
	HasRegex      bool
}

type routingSimpleFieldAnnotation struct {
	FieldName string
	IsFirst   bool
}

type routingMatcherAnnotation struct {
	ParamKey     string
	MatcherName  string
	IsSimple     bool
	SimpleFields []*routingSimpleFieldAnnotation
	Patterns     []*routingPatternAnnotation
}

type parameterAnnotation struct {
	Type            string
	Name            string
	FieldName       string
	IsMap           bool
	IsRepeated      bool
	IsMapOrRepeated bool
	IsMessage       bool
	IsScalar        bool
}

type methodSignatureAnnotations struct {
	Method                         *methodAnnotations
	Name                           string
	Parameters                     []*parameterAnnotation
	Comments                       string
	NoAwaitComments                string
	IsDeprecated                   bool
	IsUnary                        bool
	IsPaginated                    bool
	RangeOutputType                string
	IsLongrunning                  bool
	IsComputeLRO                   bool
	LongrunningOperationType       string
	LongrunningDeducedResponseType string
	IsStreamingRead                bool
	IsResponseTypeEmpty            bool
	IsAsync                        bool
	CppReturnType                  string
	CppResponseType                string
	HasIamUpdater                  bool
}

func (c *codec) annotateMethod(m *api.Method, sAnn *serviceAnnotations, model *api.API, setIamPolicyWithUpdater bool) *methodAnnotations {
	cppReqType := ProtoNameToCppName(m.InputTypeID)
	cppRespType := ProtoNameToCppName(m.OutputTypeID)
	isRespEmpty := m.OutputTypeID == ".google.protobuf.Empty" || m.OutputTypeID == "google.protobuf.Empty"
	cppReturnType := "StatusOr<" + cppRespType + ">"
	if isRespEmpty {
		cppReturnType = "Status"
	}

	isLongrunning := m.OperationInfo != nil || m.IsLRO || m.OperationService != ""
	isComputeLRO := m.OperationService != ""
	var lroDeducedType string
	var lroReturnsEmpty bool
	lroOpType := "google::longrunning::Operation"
	if isComputeLRO {
		lroOpType = ProtoNameToCppName(m.OutputTypeID)
		if lroOpType == "" && sAnn.LongrunningOperationType != "" {
			lroOpType = sAnn.LongrunningOperationType
		}
		lroDeducedType = lroOpType
		lroReturnsEmpty = false
	} else if isLongrunning {
		if m.OperationInfo != nil {
			respID := m.OperationInfo.ResponseTypeID
			if respID == "" || respID == ".google.protobuf.Empty" || respID == "google.protobuf.Empty" {
				respID = m.OperationInfo.MetadataTypeID
			}
			if respID != "" && respID != ".google.protobuf.Empty" && respID != "google.protobuf.Empty" {
				lroDeducedType = ProtoNameToCppName(respID)
			}
		}
		if lroDeducedType == "" {
			lroReturnsEmpty = true
		}
	}

	rangeOutputField, isPaginated := isMethodPaginated(m, model)
	var rangeOutputType string
	if isPaginated {
		if rangeOutputField != nil && rangeOutputField.Map {
			mapEntry := model.Message(rangeOutputField.TypezID)
			var keyField *api.Field
			if mapEntry != nil {
				for _, f := range mapEntry.Fields {
					if f.Name == "key" {
						keyField = f
						break
					}
				}
				if keyField == nil && len(mapEntry.Fields) > 0 {
					keyField = mapEntry.Fields[0]
				}
			}
			valField := mapEntryValueField(rangeOutputField, model)
			keyType := "std::string"
			if keyField != nil && keyField.Typez != api.TypezString {
				keyType = cppBaseTypeToString(keyField)
			}
			var valType string
			if valField != nil {
				valType = ProtoNameToCppName(valField.TypezID)
				if valType == "" {
					valType = cppBaseTypeToString(valField)
				}
			}
			rangeOutputType = fmt.Sprintf("std::pair<%s, %s>", keyType, valType)
		} else if rangeOutputField != nil && rangeOutputField.Typez == api.TypezString {
			rangeOutputType = "std::string"
		} else if rangeOutputField != nil {
			rangeOutputType = ProtoNameToCppName(rangeOutputField.TypezID)
		}
	}

	isStreamingRead := m.ServerSideStreaming && !m.ClientSideStreaming
	isStreamingWrite := m.ClientSideStreaming && !m.ServerSideStreaming
	isBidirStreaming := m.ClientSideStreaming && m.ServerSideStreaming
	isStreaming := isStreamingRead || isStreamingWrite || isBidirStreaming
	isUnary := !isStreaming && !isLongrunning && !isPaginated

	if isLongrunning {
		if lroReturnsEmpty {
			cppReturnType = "Status"
		} else {
			cppReturnType = "StatusOr<" + lroDeducedType + ">"
		}
	}

	isGenAsync := c.config != nil && (slices.Contains(c.config.GenAsyncRPCs, m.Name) || slices.Contains(c.config.GenAsyncRPCs, sAnn.Name+"."+m.Name))
	isAsync := isGenAsync || isLongrunning

	var lroMetadataType string
	if m.OperationInfo != nil && m.OperationInfo.MetadataTypeID != "" {
		lroMetadataType = ProtoNameToCppName(m.OperationInfo.MetadataTypeID)
	}
	var paginationElementsFieldName string
	if rangeOutputField != nil {
		paginationElementsFieldName = CppParamName(rangeOutputField.Name)
	}

	hasRequestId := len(m.AutoPopulated) > 0
	var requestIdFieldName string
	if hasRequestId {
		requestIdFieldName = CppParamName(m.AutoPopulated[0].Name)
	}

	idempotency := c.determineIdempotency(m, sAnn.Name)
	isSetIamPolicy := (m.OutputTypeID == "google.iam.v1.Policy" || m.OutputTypeID == ".google.iam.v1.Policy") &&
		(m.InputTypeID == "google.iam.v1.SetIamPolicyRequest" || m.InputTypeID == ".google.iam.v1.SetIamPolicyRequest")

	routingMatchers := annotateRoutingInfo(m)
	hasRouting := len(routingMatchers) > 0

	var hasHttpRouting bool
	var httpRoutingParams []*httpRoutingParamAnnotation
	if !hasRouting && m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 && m.PathInfo.Bindings[0].PathTemplate != nil {
		for _, seg := range m.PathInfo.Bindings[0].PathTemplate.Segments {
			if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
				var fieldCalls []string
				for _, fp := range seg.Variable.FieldPath {
					fieldCalls = append(fieldCalls, CppParamName(CamelCaseToSnakeCase(fp))+"()")
				}
				vName := strings.Join(seg.Variable.FieldPath, ".")
				vAccessor := strings.Join(fieldCalls, ".")
				httpRoutingParams = append(httpRoutingParams, &httpRoutingParamAnnotation{
					Key:           vName,
					FieldAccessor: vAccessor,
				})
			}
		}
		if len(httpRoutingParams) > 0 {
			hasHttpRouting = true
			for i := range len(httpRoutingParams) - 1 {
				httpRoutingParams[i].HasNext = true
			}
		}
	}

	var stubMemberName string
	if strings.Contains(m.SourceServiceID, "google.cloud.location.Locations") {
		stubMemberName = "locations_stub_"
	} else if strings.Contains(m.SourceServiceID, "google.iam.v1.IAMPolicy") {
		stubMemberName = "iampolicy_stub_"
	} else if strings.Contains(m.SourceServiceID, "google.longrunning.Operations") {
		stubMemberName = "operations_stub_"
	} else {
		stubMemberName = "grpc_stub_"
	}

	lroResponseIsEmpty := isLongrunning && m.OperationInfo != nil && (m.OperationInfo.ResponseTypeID == "" || m.OperationInfo.ResponseTypeID == ".google.protobuf.Empty" || m.OperationInfo.ResponseTypeID == "google.protobuf.Empty")

	var b *api.PathBinding
	if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
		b = m.PathInfo.Bindings[0]
	}
	hasRestPath, restVerb, restPathSegments, restPathVerb := buildRestPath(m)
	restQueryParams := buildRestQueryParams(m, b, model)
	var restRequestBodyAccessor string
	var restReturnTypeName string
	var restPayloadType string
	if hasRestPath {
		var bodyField string
		if m.PathInfo != nil {
			bodyField = m.PathInfo.BodyFieldPath
		}
		restRequestBodyAccessor = buildRestRequestBodyAccessor(bodyField)
		restReturnTypeName = buildRestReturnTypeName(m)
		restPayloadType = buildRestPayloadType(m)
	}
	isRestRpc := !isStreamingRead && !isStreamingWrite && !isBidirStreaming && hasRestPath

	mAnn := &methodAnnotations{
		Service:                        sAnn,
		ServiceName:                    sAnn.Name,
		Name:                           m.Name,
		CppRequestType:                 cppReqType,
		CppResponseType:                cppRespType,
		CppReturnType:                  cppReturnType,
		IsDeprecated:                   m.Deprecated,
		IsUnary:                        isUnary,
		IsPaginated:                    isPaginated,
		RangeOutputType:                rangeOutputType,
		IsLongrunning:                  isLongrunning,
		IsComputeLRO:                   isComputeLRO,
		LongrunningDeducedResponseType: lroDeducedType,
		LongrunningReturnsEmpty:        lroReturnsEmpty,
		LongrunningResponseIsEmpty:     lroResponseIsEmpty,
		LongrunningOperationType:       lroOpType,
		LongrunningMetadataType:        lroMetadataType,
		PaginationElementsFieldName:    paginationElementsFieldName,
		IsStreamingRead:                isStreamingRead,
		IsStreamingWrite:               isStreamingWrite,
		IsBidirStreaming:               isBidirStreaming,
		IsStreaming:                    isStreaming,
		IsResponseTypeEmpty:            isRespEmpty,
		IsAsync:                        isAsync,
		HasRouting:                     hasRouting,
		HasHttpRouting:                 hasHttpRouting,
		HttpRoutingParams:              httpRoutingParams,
		RoutingParamsCount:             len(routingMatchers),
		RoutingParamMatchers:           routingMatchers,
		StubMemberName:                 stubMemberName,
		HasRestPath:                    hasRestPath,
		RestVerb:                       restVerb,
		RestPathSegments:               restPathSegments,
		RestPathVerb:                   restPathVerb,
		HasRestPathVerb:                restPathVerb != "",
		RestQueryParams:                restQueryParams,
		RestHasQueryParams:             len(restQueryParams) > 0,
		RestRequestBodyAccessor:        restRequestBodyAccessor,
		RestPayloadType:                restPayloadType,
		RestReturnTypeName:             restReturnTypeName,
		IsRestRpc:                      isRestRpc,
		HasRequestId:                   hasRequestId,
		RequestIdFieldName:             requestIdFieldName,
		Idempotency:                    idempotency,
		IsSetIamPolicy:                 isSetIamPolicy,
	}

	if isLongrunning {
		mAnn.NoAwaitComments = formatStartMethodComments(m.Name, lroOpType, m.Deprecated)
		mAnn.OperationComments = formatAwaitMethodComments(m.Name, lroOpType, m.Deprecated)
	}

	reqParamComment := formatProtobufRequestParamComment(m)
	mAnn.RequestComments = formatMethodDoxygenComments(m, reqParamComment, model, sAnn.SourceFile, isPaginated, rangeOutputField, isLongrunning, isRespEmpty, isComputeLRO)
	mAnn.DocLines = strings.Split(mAnn.RequestComments, "\n")

	if isBidirStreaming {
		mAnn.Comments = formatMethodDoxygenComments(m, "", model, sAnn.SourceFile, false, nil, false, false, false)
	}

	reqMsg := model.Message(m.InputTypeID)
	isOmitted := func(name string) bool {
		return c.config != nil && slices.Contains(c.config.OmittedRPCs, name)
	}

	sigs := append([]*api.MethodSignature(nil), m.Signatures...)
	if hasEmptyMethodSignature(m) {
		hasEmpty := false
		for _, s := range sigs {
			if len(s.Names) == 0 && len(s.Fields) == 0 {
				hasEmpty = true
				break
			}
		}
		if !hasEmpty {
			sigs = append([]*api.MethodSignature{{}}, sigs...)
		}
	}

	var hasDeprecatedFieldInSignature bool
	seenSigUIDs := make(map[string]bool)
	for _, sig := range sigs {
		var params []*parameterAnnotation
		var sigUIDBuilder strings.Builder
		var paramCommentsBuilder strings.Builder
		for _, f := range sig.Fields {
			if f.Deprecated {
				hasDeprecatedFieldInSignature = true
			}
			paramType := CppParamTypeToString(f)
			paramName := CppParamName(f.Name)
			isMap := f.Map
			isRepeated := f.Repeated && !f.Map
			isMapOrRepeated := f.Map || f.Repeated
			isMessage := !f.Map && !f.Repeated && f.Typez == api.TypezMessage
			isScalar := !f.Map && !f.Repeated && f.Typez != api.TypezMessage
			params = append(params, &parameterAnnotation{
				Type:            paramType,
				Name:            paramName,
				FieldName:       paramName,
				IsMap:           isMap,
				IsRepeated:      isRepeated,
				IsMapOrRepeated: isMapOrRepeated,
				IsMessage:       isMessage,
				IsScalar:        isScalar,
			})
			sigUIDBuilder.WriteString(paramType + ", ")
			paramCommentsBuilder.WriteString(formatParameterComment(reqMsg, f))
		}
		sigUID := sigUIDBuilder.String()
		if len(sigUID) >= 2 {
			sigUID = sigUID[:len(sigUID)-2]
		}
		if seenSigUIDs[sigUID] {
			continue
		}
		seenSigUIDs[sigUID] = true

		signature := fmt.Sprintf("%s(%s)", m.Name, sigUID)
		qualifiedSignature := fmt.Sprintf("%s.%s", sAnn.Name, signature)
		sigStr := strings.Join(sig.Names, ",")

		if (sigStr != "" && isOmitted(sigStr)) || isOmitted(signature) || isOmitted(qualifiedSignature) || isOmitted(m.Name) || isOmitted(sAnn.Name+"."+m.Name) {
			continue
		}

		sigComments := formatMethodDoxygenComments(m, paramCommentsBuilder.String(), model, sAnn.SourceFile, isPaginated, rangeOutputField, isLongrunning, isRespEmpty, isComputeLRO)

		hasIamUpdater := setIamPolicyWithUpdater && m.Name == "SetIamPolicy" && sigStr == "resource,policy"

		sigAnn := &methodSignatureAnnotations{
			Method:                         mAnn,
			Name:                           m.Name,
			Parameters:                     params,
			Comments:                       sigComments,
			NoAwaitComments:                mAnn.NoAwaitComments,
			IsDeprecated:                   m.Deprecated,
			IsUnary:                        isUnary,
			IsPaginated:                    isPaginated,
			RangeOutputType:                rangeOutputType,
			IsLongrunning:                  isLongrunning,
			IsComputeLRO:                   isComputeLRO,
			LongrunningOperationType:       lroOpType,
			LongrunningDeducedResponseType: lroDeducedType,
			IsStreamingRead:                isStreamingRead,
			IsResponseTypeEmpty:            isRespEmpty,
			IsAsync:                        isAsync,
			CppReturnType:                  cppReturnType,
			CppResponseType:                cppRespType,
			HasIamUpdater:                  hasIamUpdater,
		}
		mAnn.Signatures = append(mAnn.Signatures, sigAnn)
	}
	mAnn.HasDeprecatedFieldInSignature = hasDeprecatedFieldInSignature

	m.Codec = mAnn
	return mAnn
}

func isMethodPaginated(m *api.Method, model *api.API) (*api.Field, bool) {
	if m.Pagination != nil {
		if respMsg := model.Message(m.OutputTypeID); respMsg != nil && respMsg.Pagination != nil {
			return respMsg.Pagination.PageableItem, true
		}
	}
	reqMsg := model.Message(m.InputTypeID)
	respMsg := model.Message(m.OutputTypeID)
	if reqMsg == nil || respMsg == nil {
		return nil, false
	}
	hasPageSize := false
	hasPageToken := false
	for _, f := range reqMsg.Fields {
		if (f.Name == "page_size" || f.Name == "max_results") && (f.Typez == api.TypezInt32 || f.Typez == api.TypezUint32) {
			hasPageSize = true
		}
		if f.Name == "page_token" && f.Typez == api.TypezString {
			hasPageToken = true
		}
	}
	if !hasPageSize || !hasPageToken {
		return nil, false
	}
	hasNextPageToken := false
	var firstRepeated *api.Field
	for _, f := range respMsg.Fields {
		if f.Name == "next_page_token" && f.Typez == api.TypezString {
			hasNextPageToken = true
		}
		if (f.Repeated || f.Map) && firstRepeated == nil {
			firstRepeated = f
		}
	}
	if hasNextPageToken && firstRepeated != nil {
		return firstRepeated, true
	}
	return nil, false
}

func mapEntryValueField(field *api.Field, model *api.API) *api.Field {
	if field == nil || !field.Map || model == nil {
		return nil
	}
	mapEntry := model.Message(field.TypezID)
	if mapEntry == nil {
		return nil
	}
	for _, f := range mapEntry.Fields {
		if f.Name == "value" {
			return f
		}
	}
	if len(mapEntry.Fields) > 1 {
		return mapEntry.Fields[1]
	}
	return nil
}

func hasEmptyMethodSignature(m *api.Method) bool {
	for _, sig := range m.Signatures {
		if len(sig.Fields) == 0 {
			return true
		}
	}
	return false
}

func (c *codec) determineIdempotency(m *api.Method, serviceName string) string {
	if c.config != nil && len(c.config.IdempotencyOverrides) > 0 {
		for _, rule := range c.config.IdempotencyOverrides {
			if rule.RPCName == m.Name || rule.RPCName == serviceName+"."+m.Name {
				if strings.EqualFold(rule.Idempotency, "IDEMPOTENT") || rule.Idempotency == "kIdempotent" {
					return "kIdempotent"
				}
				if strings.EqualFold(rule.Idempotency, "NON_IDEMPOTENT") || rule.Idempotency == "kNonIdempotent" {
					return "kNonIdempotent"
				}
			}
		}
	}
	if isKnownIdempotentMethod(m) {
		return "kIdempotent"
	}
	if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 {
		switch m.PathInfo.Bindings[0].Verb {
		case "GET", "PUT":
			return "kIdempotent"
		case "POST", "DELETE", "PATCH":
			return "kNonIdempotent"
		}
	}
	return "kNonIdempotent"
}

func isKnownIdempotentMethod(m *api.Method) bool {
	return (m.Name == "GetIamPolicy" &&
		(m.OutputTypeID == "google.iam.v1.Policy" || m.OutputTypeID == ".google.iam.v1.Policy") &&
		(m.InputTypeID == "google.iam.v1.GetIamPolicyRequest" || m.InputTypeID == ".google.iam.v1.GetIamPolicyRequest")) ||
		(m.Name == "TestIamPermissions" &&
			(m.OutputTypeID == "google.iam.v1.TestIamPermissionsResponse" || m.OutputTypeID == ".google.iam.v1.TestIamPermissionsResponse") &&
			(m.InputTypeID == "google.iam.v1.TestIamPermissionsRequest" || m.InputTypeID == ".google.iam.v1.TestIamPermissionsRequest"))
}

func annotateRoutingInfo(m *api.Method) []*routingMatcherAnnotation {
	if len(m.Routing) == 0 {
		return nil
	}
	var matchers []*routingMatcherAnnotation
	for _, r := range m.Routing {
		if r.Name == "" {
			continue
		}
		allSimple := true
		var simpleFields []*routingSimpleFieldAnnotation
		var patterns []*routingPatternAnnotation
		variants := slices.Clone(r.Variants)
		slices.Reverse(variants)
		for i, v := range variants {
			fieldAccessor := strings.Join(v.FieldPath, "().")
			isFirst := i == 0
			isSimplePattern := len(v.Prefix.Segments) == 0 && len(v.Suffix.Segments) == 0 &&
				(len(v.Matching.Segments) == 0 || (len(v.Matching.Segments) == 1 && (v.Matching.Segments[0] == api.MultiSegmentWildcard || v.Matching.Segments[0] == "*")))
			if isSimplePattern {
				simpleFields = append(simpleFields, &routingSimpleFieldAnnotation{
					FieldName: fieldAccessor,
					IsFirst:   isFirst,
				})
			} else {
				allSimple = false
			}

			var pBuilder strings.Builder
			if len(v.Prefix.Segments) > 0 {
				pBuilder.WriteString(strings.Join(v.Prefix.Segments, "/"))
				pBuilder.WriteString("/")
			}
			pBuilder.WriteString("(")
			pBuilder.WriteString(strings.Join(v.Matching.Segments, "/"))
			pBuilder.WriteString(")")
			if len(v.Suffix.Segments) > 0 {
				pBuilder.WriteString("/")
				pBuilder.WriteString(strings.Join(v.Suffix.Segments, "/"))
			}
			rawPattern := pBuilder.String()
			const doubleStarPlaceholder = "\x00"
			regexPattern := strings.ReplaceAll(rawPattern, "**", doubleStarPlaceholder)
			regexPattern = strings.ReplaceAll(regexPattern, "*):", "[^:]+):")
			regexPattern = strings.ReplaceAll(regexPattern, "*:", "[^:]+:")
			regexPattern = strings.ReplaceAll(regexPattern, "*", "[^/]+")
			regexPattern = strings.ReplaceAll(regexPattern, doubleStarPlaceholder, ".*")

			hasRegex := regexPattern != "(.*)"
			patterns = append(patterns, &routingPatternAnnotation{
				FieldAccessor: fieldAccessor,
				Pattern:       regexPattern,
				HasRegex:      hasRegex,
			})
		}

		matchers = append(matchers, &routingMatcherAnnotation{
			ParamKey:     r.Name,
			MatcherName:  r.Name,
			IsSimple:     allSimple,
			SimpleFields: simpleFields,
			Patterns:     patterns,
		})
	}
	return matchers
}

func buildRestPath(m *api.Method) (bool, string, []*restPathSegmentAnnotation, string) {
	if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 || m.PathInfo.Bindings[0].PathTemplate == nil {
		return false, "", nil, ""
	}
	b := m.PathInfo.Bindings[0]
	tmpl := b.PathTemplate

	var verb string
	switch strings.ToUpper(b.Verb) {
	case "GET":
		verb = "Get"
	case "POST":
		verb = "Post"
	case "PUT":
		verb = "Put"
	case "DELETE":
		verb = "Delete"
	case "PATCH":
		verb = "Patch"
	default:
		verb = "Post"
	}

	var apiVersion string
	for _, seg := range tmpl.Segments {
		if strings.HasPrefix(seg.Literal, "v") && len(seg.Literal) > 1 {
			allDigits := true
			for _, r := range seg.Literal[1:] {
				if r < '0' || r > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				apiVersion = seg.Literal
				break
			}
		}
	}

	var segments []*restPathSegmentAnnotation
	for _, seg := range tmpl.Segments {
		if seg.Literal != "" {
			if apiVersion != "" && seg.Literal == apiVersion {
				segments = append(segments, &restPathSegmentAnnotation{
					IsApiVersion: true,
					ApiVersion:   apiVersion,
				})
				continue
			}
			segments = append(segments, &restPathSegmentAnnotation{
				IsLiteral: true,
				Literal:   seg.Literal,
			})
			continue
		}
		if seg.Variable != nil {
			var fieldCalls []string
			for _, fp := range seg.Variable.FieldPath {
				fieldCalls = append(fieldCalls, CppParamName(CamelCaseToSnakeCase(fp))+"()")
			}
			segments = append(segments, &restPathSegmentAnnotation{
				IsField:       true,
				FieldAccessor: strings.Join(fieldCalls, "."),
			})
		}
	}
	if len(segments) > 0 {
		for i := range len(segments) - 1 {
			segments[i].HasNext = true
		}
	}

	return true, verb, segments, tmpl.Verb
}

func buildRestQueryParams(m *api.Method, b *api.PathBinding, model *api.API) []*queryParamAnnotation {
	if b == nil || len(b.QueryParameters) == 0 || m.InputTypeID == "" {
		return nil
	}
	msg := model.Message(m.InputTypeID)
	if msg == nil {
		return nil
	}
	var params []*queryParamAnnotation
	for _, f := range msg.Fields {
		if !b.QueryParameters[f.Name] {
			continue
		}
		if f.Deprecated || f.Repeated {
			continue
		}
		// In newer googleapis specifications (AIP-158 / AIP-127), google.longrunning.ListOperationsRequest
		// added field 5 bool return_partial_success. Upstream google-cloud-cpp's generator baseline
		// predates this field and omits it from REST query parameters to maintain compatibility.
		if m.Name == "ListOperations" && f.Name == "return_partial_success" {
			continue
		}
		snakeName := CppParamName(CamelCaseToSnakeCase(f.Name)) + "()"
		switch f.Typez {
		case api.TypezString:
			params = append(params, &queryParamAnnotation{
				ParamKey:      f.Name,
				FieldAccessor: snakeName,
				IsString:      true,
			})
		case api.TypezBool:
			params = append(params, &queryParamAnnotation{
				ParamKey:      f.Name,
				FieldAccessor: snakeName,
				IsBool:        true,
			})
		case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64, api.TypezFloat, api.TypezDouble, api.TypezEnum:
			params = append(params, &queryParamAnnotation{
				ParamKey:      f.Name,
				FieldAccessor: snakeName,
				IsNumber:      true,
			})
		}
	}
	return params
}

func buildRestRequestBodyAccessor(bodyField string) string {
	if bodyField == "" || bodyField == "*" {
		return "request"
	}
	return "request." + CppParamName(CamelCaseToSnakeCase(bodyField)) + "()"
}

func buildRestReturnTypeName(m *api.Method) string {
	if m.OutputTypeID == "" || m.OutputTypeID == ".google.protobuf.Empty" || m.OutputTypeID == "google.protobuf.Empty" {
		return "Status"
	}
	return "StatusOr<" + ProtoNameToCppName(m.OutputTypeID) + ">"
}

func buildRestPayloadType(m *api.Method) string {
	if m.OutputTypeID == "" || m.OutputTypeID == ".google.protobuf.Empty" || m.OutputTypeID == "google.protobuf.Empty" {
		return "google::cloud::rest_internal::EmptyResponseType"
	}
	return ProtoNameToCppName(m.OutputTypeID)
}
