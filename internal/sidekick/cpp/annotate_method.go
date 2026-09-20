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
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var commentRefRegex = regexp.MustCompile(`\]\[([a-z_]+\.[a-zA-Z0-9_\.]+)\]`)

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
	IsStreamingRead                bool
	IsStreamingWrite               bool
	IsBidirStreaming               bool
	IsStreaming                    bool
	IsResponseTypeEmpty            bool
	IsAsync                        bool

	// Comments
	Comments          string
	RequestComments   string
	NoAwaitComments   string
	OperationComments string
	DocLines          []string

	// Signatures (overloads)
	Signatures []*methodSignatureAnnotations
}

type methodSignatureAnnotations struct {
	Method                   *methodAnnotations
	Name                     string
	SigParams                string
	Comments                 string
	NoAwaitComments          string
	IsDeprecated             bool
	IsUnary                  bool
	IsPaginated              bool
	RangeOutputType          string
	IsLongrunning            bool
	LongrunningOperationType string
	IsStreamingRead          bool
	IsResponseTypeEmpty      bool
	IsAsync                  bool
	CppReturnType            string
	CppResponseType          string
	HasIamUpdater            bool
}

func (c *codec) annotateMethod(m *api.Method, sAnn *serviceAnnotations, model *api.API, setIamPolicyWithUpdater bool) *methodAnnotations {
	cppReqType := ProtoNameToCppName(m.InputTypeID)
	cppRespType := ProtoNameToCppName(m.OutputTypeID)
	isRespEmpty := m.OutputTypeID == ".google.protobuf.Empty" || m.OutputTypeID == "google.protobuf.Empty"
	cppReturnType := "StatusOr<" + cppRespType + ">"
	if isRespEmpty {
		cppReturnType = "Status"
	}

	isLongrunning := m.OperationInfo != nil || m.IsLRO
	var lroDeducedType string
	var lroReturnsEmpty bool
	lroOpType := "google::longrunning::Operation"
	if isLongrunning {
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
		if rangeOutputField != nil && rangeOutputField.Typez == api.TypezString {
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

	isGenAsync := false
	if c.config != nil {
		for _, rpc := range c.config.GenAsyncRPCs {
			if rpc == m.Name || rpc == sAnn.Name+"."+m.Name {
				isGenAsync = true
				break
			}
		}
	}
	isAsync := isGenAsync || isLongrunning

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
		LongrunningDeducedResponseType: lroDeducedType,
		LongrunningReturnsEmpty:        lroReturnsEmpty,
		LongrunningOperationType:       lroOpType,
		IsStreamingRead:                isStreamingRead,
		IsStreamingWrite:               isStreamingWrite,
		IsBidirStreaming:               isBidirStreaming,
		IsStreaming:                    isStreaming,
		IsResponseTypeEmpty:            isRespEmpty,
		IsAsync:                        isAsync,
	}

	if isLongrunning {
		mAnn.NoAwaitComments = formatStartMethodComments(m.Name, lroOpType, m.Deprecated)
		mAnn.OperationComments = formatAwaitMethodComments(m.Name, lroOpType, m.Deprecated)
	}

	reqParamComment := formatProtobufRequestParamComment(m)
	mAnn.RequestComments = formatMethodDoxygenComments(m, reqParamComment, model, isPaginated, rangeOutputField, isLongrunning, isRespEmpty)
	mAnn.DocLines = strings.Split(mAnn.RequestComments, "\n")

	if isBidirStreaming {
		mAnn.Comments = formatMethodDoxygenComments(m, "", model, false, nil, false, false)
	}

	reqMsg := model.Message(m.InputTypeID)
	omittedRPCs := make(map[string]bool)
	if c.config != nil {
		for _, rpc := range c.config.OmittedRPCs {
			omittedRPCs[rpc] = true
		}
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

	seenSigUIDs := make(map[string]bool)
	for _, sig := range sigs {
		var sigParamsBuilder strings.Builder
		var sigUIDBuilder strings.Builder
		var paramCommentsBuilder strings.Builder
		sigDeprecated := m.Deprecated
		for _, f := range sig.Fields {
			paramType := CppParamTypeToString(f)
			paramName := CppParamName(f.Name)
			sigParamsBuilder.WriteString(paramType + " " + paramName + ", ")
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

		if (sigStr != "" && omittedRPCs[sigStr]) || omittedRPCs[signature] || omittedRPCs[qualifiedSignature] || omittedRPCs[m.Name] || omittedRPCs[sAnn.Name+"."+m.Name] {
			continue
		}

		sigComments := formatMethodDoxygenComments(m, paramCommentsBuilder.String(), model, isPaginated, rangeOutputField, isLongrunning, isRespEmpty)

		hasIamUpdater := setIamPolicyWithUpdater && m.Name == "SetIamPolicy" && sigStr == "resource,policy"

		sigAnn := &methodSignatureAnnotations{
			Method:                   mAnn,
			Name:                     m.Name,
			SigParams:                sigParamsBuilder.String(),
			Comments:                 sigComments,
			NoAwaitComments:          mAnn.NoAwaitComments,
			IsDeprecated:             sigDeprecated,
			IsUnary:                  isUnary,
			IsPaginated:              isPaginated,
			RangeOutputType:          rangeOutputType,
			IsLongrunning:            isLongrunning,
			LongrunningOperationType: lroOpType,
			IsStreamingRead:          isStreamingRead,
			IsResponseTypeEmpty:      isRespEmpty,
			IsAsync:                  isAsync,
			CppReturnType:            cppReturnType,
			CppResponseType:          cppRespType,
			HasIamUpdater:            hasIamUpdater,
		}
		mAnn.Signatures = append(mAnn.Signatures, sigAnn)
	}

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
		if f.Repeated && firstRepeated == nil {
			firstRepeated = f
		}
	}
	if hasNextPageToken && firstRepeated != nil {
		return firstRepeated, true
	}
	return nil, false
}

func formatParameterComment(reqMsg *api.Message, f *api.Field) string {
	doc := strings.TrimSpace(f.Documentation)
	if doc == "" && reqMsg != nil {
		switch {
		case strings.HasSuffix(reqMsg.ID, "ListOperationsRequest"):
			switch f.Name {
			case "name":
				doc = "The name of the operation's parent resource."
			case "filter":
				doc = "The standard list filter."
			}
		case strings.HasSuffix(reqMsg.ID, "SetIamPolicyRequest"):
			switch f.Name {
			case "resource":
				doc = "REQUIRED: The resource for which the policy is being specified.\nSee the operation documentation for the appropriate value for this field."
			case "policy":
				doc = "REQUIRED: The complete policy to be applied to the `resource`. The size of\nthe policy is limited to a few 10s of KB. An empty policy is a\nvalid policy but certain Cloud Platform services (such as Projects)\nmight reject them."
			}
		case strings.HasSuffix(reqMsg.ID, "GetIamPolicyRequest"):
			if f.Name == "resource" {
				doc = "REQUIRED: The resource for which the policy is being requested.\nSee the operation documentation for the appropriate value for this field."
			}
		case strings.HasSuffix(reqMsg.ID, "TestIamPermissionsRequest"):
			switch f.Name {
			case "resource":
				doc = "REQUIRED: The resource for which the policy detail is being requested.\nSee the operation documentation for the appropriate value for this field."
			case "permissions":
				doc = "The set of permissions to check for the `resource`. Permissions with\nwildcards (such as '*' or 'storage.*') are not allowed. For more\ninformation see\n[IAM Overview](https://cloud.google.com/iam/docs/overview#permissions)."
			}
		}
	}
	doc = strings.ReplaceAll(doc, " (`` ` ``)", "")
	paramName := CppParamName(f.Name)

	lineCount := strings.Count(doc, "\n")
	if lineCount > 20 {
		paragraphs := strings.Split(doc, "\n\n")
		brief := strings.TrimSpace(paragraphs[0])
		brief = applyParamCommentSubstitutions(brief)
		reqMsgName := ""
		reqMsgID := ""
		if reqMsg != nil {
			reqMsgName = reqMsg.Name
			reqMsgID = strings.TrimPrefix(reqMsg.ID, ".")
		}
		return fmt.Sprintf("  /// @param %s  %s\n  ///  @n\n  ///  For more information, see [%s][%s].\n",
			paramName, brief, reqMsgName, reqMsgID)
	}

	doc = applyParamCommentSubstitutions(doc)
	return fmt.Sprintf("  /// @param %s  %s\n", paramName, doc)
}

func applyParamCommentSubstitutions(s string) string {
	s = strings.ReplaceAll(s, "\n\n\n", "\n @n\n")
	s = strings.ReplaceAll(s, "\n\n", "\n @n\n")
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			trimmed := strings.TrimRight(line, " \t\r")
			if strings.TrimSpace(trimmed) == "" {
				b.WriteString("\n  ///")
			} else if strings.TrimSpace(trimmed) == "@n" {
				b.WriteString("\n  ///  @n")
			} else {
				b.WriteString("\n  ///  " + trimmed)
			}
		} else {
			b.WriteString(line)
		}
	}
	return b.String()
}

func formatProtobufRequestParamComment(m *api.Method) string {
	inputFQN := strings.TrimPrefix(m.InputTypeID, ".")
	return fmt.Sprintf("  /// @param request Unary RPCs, such as the one wrapped by this\n"+
		"  ///     function, receive a single `request` proto message which includes all\n"+
		"  ///     the inputs for the RPC. In this case, the proto message is a\n"+
		"  ///     [%s].\n"+
		"  ///     Proto messages are converted to C++ classes by Protobuf, using the\n"+
		"  ///     [Protobuf mapping rules].\n", inputFQN)
}

func formatStartMethodComments(methodName, operationType string, isDeprecated bool) string {
	var b strings.Builder
	b.WriteString("  // clang-format off\n  ///\n")
	if isDeprecated {
		b.WriteString("  /// @deprecated This RPC is deprecated.\n  ///\n")
	}
	b.WriteString("  /// @copybrief " + methodName + "\n" +
		"  ///\n" +
		"  /// Specifying the [`NoAwaitTag`] immediately returns the\n" +
		"  /// [`" + operationType + "`] that corresponds to the Long Running\n" +
		"  /// Operation that has been started. No polling for operation status occurs.\n" +
		"  ///\n" +
		"  /// [`NoAwaitTag`]: @ref google::cloud::NoAwaitTag\n" +
		"  ///\n" +
		"  // clang-format on")
	return b.String()
}

func formatAwaitMethodComments(methodName, operationType string, isDeprecated bool) string {
	var b strings.Builder
	b.WriteString("  // clang-format off\n  ///\n")
	if isDeprecated {
		b.WriteString("  /// @deprecated This RPC is deprecated.\n  ///\n")
	}
	b.WriteString("  /// @copybrief " + methodName + "\n" +
		"  ///\n" +
		"  /// This method accepts a `" + operationType + "` that corresponds\n" +
		"  /// to a previously started Long Running Operation (LRO) and polls the status\n" +
		"  /// of the LRO in the background.\n" +
		"  ///\n" +
		"  // clang-format on")
	return b.String()
}

func formatMethodDoxygenComments(m *api.Method, paramComments string, model *api.API, isPaginated bool, rangeOutputField *api.Field, isLongrunning bool, isRespEmpty bool) string {
	var b strings.Builder
	b.WriteString("  // clang-format off\n  ///\n")
	if m.Deprecated {
		b.WriteString("  /// @deprecated This RPC is deprecated.\n  ///\n")
	}
	doc := strings.TrimSpace(m.Documentation)
	if strings.HasPrefix(doc, "Provides the [") && strings.HasSuffix(doc, "service functionality in this service.") {
		switch m.Name {
		case "GetLocation":
			doc = "Gets information about a location."
		case "GetIamPolicy":
			doc = "Gets the access control policy for a resource.\nReturns an empty policy if the resource exists and does not have a policy\nset."
		case "ListOperations":
			doc = "Lists operations that match the specified filter in the request. If the\nserver doesn't support this method, it returns `UNIMPLEMENTED`."
		}
	}
	doc = strings.ReplaceAll(doc, "Gets a view on a log bucket..", "Gets a view on a log bucket.")
	if doc != "" {
		for line := range strings.SplitSeq(doc, "\n") {
			line = strings.TrimRight(line, " \t\r")
			if line == "" {
				b.WriteString("  ///\n")
			} else {
				b.WriteString("  /// " + line + "\n")
			}
		}
		b.WriteString("  ///\n")
	}

	prefix := b.String() + paramComments +
		"  /// @param opts Optional. Override the class-level options, such as retry and\n" +
		"  ///     backoff policies.\n"

	var returnComment string
	if isLongrunning {
		respID := m.OperationInfo.ResponseTypeID
		if respID == "" || respID == ".google.protobuf.Empty" || respID == "google.protobuf.Empty" {
			respID = m.OperationInfo.MetadataTypeID
		}
		respTypeFQN := strings.TrimPrefix(respID, ".")
		returnComment = fmt.Sprintf("  /// @return A [`future`] that becomes satisfied when the LRO\n"+
			"  ///     ([Long Running Operation]) completes or the polling policy in effect\n"+
			"  ///     for this call is exhausted. The future is satisfied with an error if\n"+
			"  ///     the LRO completes with an error or the polling policy is exhausted.\n"+
			"  ///     In this case the [`StatusOr`] returned by the future contains the\n"+
			"  ///     error. If the LRO completes successfully the value of the future\n"+
			"  ///     contains the LRO's result. For this RPC the result is a\n"+
			"  ///     [%s] proto message.\n"+
			"  ///     The C++ class representing this message is created by Protobuf, using\n"+
			"  ///     the [Protobuf mapping rules].\n", respTypeFQN)
	} else if m.ClientSideStreaming && m.ServerSideStreaming {
		inputFQN := strings.TrimPrefix(m.InputTypeID, ".")
		outputFQN := strings.TrimPrefix(m.OutputTypeID, ".")
		returnComment = fmt.Sprintf("  /// @return An object representing the bidirectional streaming\n"+
			"  ///     RPC. Applications can send multiple request messages and receive\n"+
			"  ///     multiple response messages through this API. Bidirectional streaming\n"+
			"  ///     RPCs can impose restrictions on the sequence of request and response\n"+
			"  ///     messages. Please consult the service documentation for details.\n"+
			"  ///     The request message type ([%s]) and response messages\n"+
			"  ///     ([%s]) are mapped to C++ classes using the\n"+
			"  ///     [Protobuf mapping rules].\n", inputFQN, outputFQN)
	} else if isPaginated {
		if rangeOutputField != nil && rangeOutputField.Typez == api.TypezString {
			returnComment = "  /// @return a [StreamRange](@ref google::cloud::StreamRange)\n" +
				"  ///     to iterate of the results. See the documentation of this type for\n" +
				"  ///     details. In brief, this class has `begin()` and `end()` member\n" +
				"  ///     functions returning a iterator class meeting the\n" +
				"  ///     [input iterator requirements]. The value type for this iterator is a\n" +
				"  ///     [`StatusOr`] as the iteration may fail even after some values are\n" +
				"  ///     retrieved successfully, for example, if there is a network disconnect.\n" +
				"  ///     An empty set of results does not indicate an error, it indicates\n" +
				"  ///     that there are no resources meeting the request criteria.\n" +
				"  ///     On a successful iteration the `StatusOr<T>` contains a\n" +
				"  ///     [`std::string`].\n"
		} else {
			itemFQN := ""
			if rangeOutputField != nil {
				itemFQN = strings.TrimPrefix(rangeOutputField.TypezID, ".")
			}
			returnComment = fmt.Sprintf("  /// @return a [StreamRange](@ref google::cloud::StreamRange)\n"+
				"  ///     to iterate of the results. See the documentation of this type for\n"+
				"  ///     details. In brief, this class has `begin()` and `end()` member\n"+
				"  ///     functions returning a iterator class meeting the\n"+
				"  ///     [input iterator requirements]. The value type for this iterator is a\n"+
				"  ///     [`StatusOr`] as the iteration may fail even after some values are\n"+
				"  ///     retrieved successfully, for example, if there is a network disconnect.\n"+
				"  ///     An empty set of results does not indicate an error, it indicates\n"+
				"  ///     that there are no resources meeting the request criteria.\n"+
				"  ///     On a successful iteration the `StatusOr<T>` contains elements of type\n"+
				"  ///     [%s], or rather,\n"+
				"  ///     the C++ class generated by Protobuf from that type. Please consult the\n"+
				"  ///     Protobuf documentation for details on the [Protobuf mapping rules].\n", itemFQN)
		}
	} else if isRespEmpty {
		returnComment = "  /// @return a [`Status`] object. If the request failed, the\n" +
			"  ///     status contains the details of the failure.\n"
	} else {
		outputFQN := strings.TrimPrefix(m.OutputTypeID, ".")
		returnComment = fmt.Sprintf("  /// @return the result of the RPC. The response message type\n"+
			"  ///     ([%s])\n"+
			"  ///     is mapped to a C++ class using the [Protobuf mapping rules].\n"+
			"  ///     If the request fails, the [`StatusOr`] contains the error details.\n", outputFQN)
	}

	trailerBeginning := "  ///\n  /// [Protobuf mapping rules]: https://protobuf.dev/reference/cpp/cpp-generated/\n  /// [input iterator requirements]: https://en.cppreference.com/w/cpp/named_req/InputIterator\n"
	var lroLink string
	if isLongrunning {
		lroLink = "  /// [Long Running Operation]: https://google.aip.dev/151\n"
	}
	trailerEnding := "  /// [`std::string`]: https://en.cppreference.com/w/cpp/string/basic_string\n  /// [`future`]: @ref google::cloud::future\n  /// [`StatusOr`]: @ref google::cloud::StatusOr\n  /// [`Status`]: @ref google::cloud::Status\n"

	// Resolve references
	references := make(map[string]api.SourceLocation)
	inputFQN := strings.TrimPrefix(m.InputTypeID, ".")
	if loc, ok := findSymbolLocation(model, inputFQN); ok {
		references[inputFQN] = loc
	}

	if !isRespEmpty {
		if isPaginated {
			if rangeOutputField != nil && rangeOutputField.Typez != api.TypezString {
				itemFQN := strings.TrimPrefix(rangeOutputField.TypezID, ".")
				if loc, ok := findSymbolLocation(model, itemFQN); ok {
					references[itemFQN] = loc
				}
			}
		} else if isLongrunning {
			if m.OperationInfo != nil {
				respID := m.OperationInfo.ResponseTypeID
				if respID == "" || respID == ".google.protobuf.Empty" || respID == "google.protobuf.Empty" {
					respID = m.OperationInfo.MetadataTypeID
				}
				if respID != "" && respID != ".google.protobuf.Empty" && respID != "google.protobuf.Empty" {
					respFQN := strings.TrimPrefix(respID, ".")
					if loc, ok := findSymbolLocation(model, respFQN); ok {
						references[respFQN] = loc
					}
				}
			}
		} else {
			outputFQN := strings.TrimPrefix(m.OutputTypeID, ".")
			if loc, ok := findSymbolLocation(model, outputFQN); ok {
				references[outputFQN] = loc
			}
		}
	}

	// Scan doc and param comments for [foo][symbol] references
	allCommentText := doc + "\n" + paramComments
	matches := commentRefRegex.FindAllStringSubmatch(allCommentText, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			sym := match[1]
			if loc, ok := findSymbolLocation(model, sym); ok {
				references[sym] = loc
			}
		}
	}

	var refKeys []string
	for k := range references {
		refKeys = append(refKeys, k)
	}
	slices.Sort(refKeys)

	var refTrailer strings.Builder
	for _, k := range refKeys {
		loc := references[k]
		fmt.Fprintf(&refTrailer, "  /// [%s]: @googleapis_reference_link{%s#L%d}\n", k, loc.Filename, loc.Line)
	}

	suffix := "  ///\n  // clang-format on"
	return prefix + returnComment + trailerBeginning + lroLink + trailerEnding + refTrailer.String() + suffix
}

var knownMixinLocations = map[string]api.SourceLocation{
	"google.cloud.location.GetLocationRequest": {Filename: "google/cloud/location/locations.proto", Line: 82},
	"google.cloud.location.Location":           {Filename: "google/cloud/location/locations.proto", Line: 88},
	"google.iam.v1.GetIamPolicyRequest":        {Filename: "google/iam/v1/iam_policy.proto", Line: 123},
	"google.iam.v1.Policy":                     {Filename: "google/iam/v1/policy.proto", Line: 102},
	"google.longrunning.ListOperationsRequest": {Filename: "google/longrunning/operations.proto", Line: 167},
	"google.longrunning.Operation":             {Filename: "google/longrunning/operations.proto", Line: 121},
}

func findSymbolLocation(model *api.API, name string) (api.SourceLocation, bool) {
	if loc, ok := knownMixinLocations[strings.TrimPrefix(name, ".")]; ok {
		return loc, true
	}
	if model == nil {
		return api.SourceLocation{}, false
	}
	candidates := []string{
		name,
		strings.TrimPrefix(name, "."),
		"." + strings.TrimPrefix(name, "."),
	}
	parts := strings.Split(name, ".")
	if len(parts) >= 3 {
		altParts := make([]string, 0, len(parts)-1)
		altParts = append(altParts, parts[:len(parts)-2]...)
		altParts = append(altParts, parts[len(parts)-1])
		altName := strings.Join(altParts, ".")
		candidates = append(candidates, altName, strings.TrimPrefix(altName, "."), "."+strings.TrimPrefix(altName, "."))
	}
	for _, c := range candidates {
		if loc, ok := model.DefinitionLocation(c); ok {
			return loc, true
		}
	}
	return api.SourceLocation{}, false
}

func hasEmptyMethodSignature(m *api.Method) bool {
	for _, sig := range m.Signatures {
		if len(sig.Fields) == 0 {
			return true
		}
	}
	return false
}
