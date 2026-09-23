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

package python

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var (
	reClassSphinx = regexp.MustCompile(":class:``([^`]+)``")
)

const (
	// emptyProtoDoc is the standard documentation string for google.protobuf.Empty.
	emptyProtoDoc = `A generic empty message that you can re-use to avoid defining duplicated
empty messages in your APIs. A typical example is to
use it as the request or the response type of an API
method. For instance:

   service Foo {
      rpc Bar(google.protobuf.Empty) returns
      (google.protobuf.Empty);

   }`
)

// flattenedParam represents a flattened keyword parameter for a client method.
type flattenedParam struct {
	Name       string
	TypeHint   string
	SphinxType string
	DocLines   []string
	IsRepeated bool
}

// clientRoutingHeader represents an HTTP routing parameter extracted for gRPC metadata.
type clientRoutingHeader struct {
	Key      string
	Accessor string
}

// clientMethodAnnotations contains metadata needed to render an RPC method on the client.
type clientMethodAnnotations struct {
	Method                *api.Method
	Name                  string
	DocLines              []string
	DocHead               string
	DocBody               []string
	HasDocHead            bool
	HasDocBody            bool
	RequestTypeHint       string
	RequestSphinxType     string
	RequestTypeName       string
	RequestDocLines       []string
	ReturnType            string
	ReturnSphinxType      string
	ReturnDocLines        []string
	HasReturnDoc          bool
	HasReturnDocTrailer   bool
	ReturnsEmpty          bool
	IsPaged               bool
	PagerClassName        string
	IsLRO                 bool
	IsSimpleUnary         bool
	OperationResponseType string
	OperationMetadataType string
	FlattenedParams       []*flattenedParam
	FlattenedParamsList   string
	HasFlattenedParams    bool
	RoutingHeaderKey      string
	RoutingHeaderAccessor string
	HasRoutingHeader      bool
	RoutingHeaders        []*clientRoutingHeader
	SamplePreInitLines    []string
	HasSamplePreInit      bool
	SampleRequestArgs     []*sampleRequestArg
	IsExternalRequest     bool
	ExternalRequestModule string
	ExternalRequestAlias  string
	PackageImport         string
	VersionSegment        string
	ClientName            string
	AsyncReturnType       string
	AsyncReturnSphinxType string
	AsyncPagerClassName   string
	IsExtendedOperation   bool
	OpServiceSnake        string
	OpServiceSingular     string
	OpHasProject          bool
	OpHasZone             bool
	OpHasRegion           bool
}

// formatRequestDocLines formats the parameter documentation lines for the request object.
func formatRequestDocLines(docText string) []string {
	if docText == "" {
		return []string{"The request object."}
	}
	lines := formatRstDoc(docText, 56, 16)
	if len(lines) > 0 {
		lines[0] = "The request object. " + lines[0]
		return lines
	}
	return []string{"The request object."}
}

// extractRoutingHeaders extracts explicit and path-derived routing header key-accessor pairs for method m.
func extractRoutingHeaders(m *api.Method) []*clientRoutingHeader {
	var headers []*clientRoutingHeader
	seen := make(map[string]bool)
	add := func(k, a string) {
		if k != "" && !seen[k] {
			seen[k] = true
			headers = append(headers, &clientRoutingHeader{Key: k, Accessor: a})
		}
	}
	if len(m.Routing) > 0 {
		for _, r := range m.Routing {
			if len(r.Variants) > 0 {
				k := r.Name
				a := r.Variants[0].FieldName()
				if k == "" {
					k = a
				}
				add(k, "request."+a)
			}
		}
	} else if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 && m.PathInfo.Bindings[0].PathTemplate != nil {
		for _, seg := range m.PathInfo.Bindings[0].PathTemplate.Segments {
			if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
				f := strings.Join(seg.Variable.FieldPath, ".")
				add(f, "request."+f)
			}
		}
	}
	return headers
}

func (c *codec) annotateClientMethod(m *api.Method, service *api.Service, ann *clientAnnotations) *clientMethodAnnotations {
	name := pythonMethodIdentifier(snakeCase(m.Name))
	docLines := formatRstDocLines(m.Documentation, 72, 8)
	var docHead string
	var docBody []string
	if len(docLines) > 0 {
		docHead = docLines[0]
		if len(docLines) > 1 {
			docBody = docLines[1:]
		}
	}

	reqStem := "common"
	reqTypeName := ""
	var reqDocLines []string
	if m.InputType != nil {
		reqTypeName = m.InputType.Name
		reqStem = c.resolveMessageStem(m.InputType)
		reqDocLines = formatRequestDocLines(strings.TrimSpace(m.InputType.Documentation))
	} else if m.InputTypeID != "" && c.Model != nil {
		if inMsg := c.resolveMessageType(m.InputTypeID); inMsg != nil {
			reqTypeName = inMsg.Name
			reqStem = c.resolveMessageStem(inMsg)
			reqDocLines = formatRequestDocLines(strings.TrimSpace(inMsg.Documentation))
		}
	}
	reqTypeHint := reqStem + "." + reqTypeName
	reqSphinxType := ann.VersionPackage + ".types." + reqTypeName

	isExternalRequest := false
	var extReqMod, extReqAlias string
	if iamMod, iamType, ok := isIAMType(m.InputTypeID); ok {
		isExternalRequest = true
		extReqMod = "google.iam.v1." + iamMod
		extReqAlias = iamMod
		reqTypeHint = iamMod + "." + iamType
		reqSphinxType = "google.iam.v1." + iamMod + "." + iamType
		reqTypeName = iamType
		reqDocLines = []string{fmt.Sprintf("The request object. Request message for ``%s`` method.", pascalCase(m.Name))}
	}

	isLRO := m.OperationInfo != nil || m.IsLRO
	isPaged := m.Pagination != nil && !isMixin(m, service)
	pagerClassName := pascalCase(m.Name) + "Pager"
	returnsEmpty := !isLRO && (m.ReturnsEmpty || strings.TrimPrefix(m.OutputTypeID, ".") == "google.protobuf.Empty" || (m.OutputType != nil && strings.TrimPrefix(m.OutputType.ID, ".") == "google.protobuf.Empty"))
	isSimpleUnary := !returnsEmpty && !isPaged && !isLRO

	returnType := "None"
	returnSphinxType := ""
	var returnDocLines []string
	hasReturnDoc := isLRO || !returnsEmpty

	var opRespType, opMetaType string
	if isLRO {
		returnType = "operation.Operation"
		returnSphinxType = "google.api_core.operation.Operation"

		var respTypeID, metaTypeID string
		if m.OperationInfo != nil {
			respTypeID = m.OperationInfo.ResponseTypeID
			metaTypeID = m.OperationInfo.MetadataTypeID
		}

		respMsg := c.resolveMessageType(respTypeID)
		metaMsg := c.resolveMessageType(metaTypeID)
		respStem := "common"
		respName := typeNameFromID(respTypeID)
		var respDoc string
		if respMsg != nil {
			respStem = c.resolveMessageStem(respMsg)
			respName = respMsg.Name
			respDoc = respMsg.Documentation
		} else if strings.TrimPrefix(respTypeID, ".") == "google.protobuf.Empty" {
			respName = "Empty"
		}
		if strings.TrimPrefix(respTypeID, ".") == "google.protobuf.Empty" && respDoc == "" {
			respDoc = emptyProtoDoc
		}
		metaStem := "common"
		metaName := typeNameFromID(metaTypeID)
		if metaMsg != nil {
			metaStem = c.resolveMessageStem(metaMsg)
			metaName = metaMsg.Name
		}
		if strings.TrimPrefix(respTypeID, ".") == "google.protobuf.Empty" {
			opRespType = "empty_pb2.Empty"
		} else {
			opRespType = respStem + "." + respName
		}
		opMetaType = metaStem + "." + metaName

		modelPkg := ""
		if c.Model != nil {
			modelPkg = c.Model.PackageName
		}
		respSphinx := ann.VersionPackage + ".types." + respName
		if respMsg != nil && respMsg.Package != "" && modelPkg != "" && respMsg.Package != modelPkg {
			targetFile := resolveProtoFile(respMsg.SourceLocation, respMsg.Package, respMsg.Name)
			respSphinx = pythonModuleFromProto(targetFile) + "." + respMsg.Name
		} else if strings.TrimPrefix(respTypeID, ".") == "google.protobuf.Empty" {
			respSphinx = "google.protobuf.empty_pb2.Empty"
		}

		var lroLines []string
		if strings.Contains(respDoc, "\n") {
			rawLines := strings.Split(respDoc, "\n")
			firstLine := strings.TrimSpace(rawLines[0])
			lroLines = append(lroLines, fmt.Sprintf("The result type for the operation will be :class:`%s` %s", respSphinx, firstLine))
			hasIndentedBlock := false
			for _, dl := range rawLines[1:] {
				if strings.HasPrefix(dl, "    ") || strings.HasPrefix(dl, "\t") {
					hasIndentedBlock = true
					break
				}
			}
			if hasIndentedBlock {
				for _, dl := range rawLines[1:] {
					trimmed := strings.TrimRight(dl, " \t\r")
					if strings.TrimSpace(trimmed) == "" {
						lroLines = append(lroLines, "")
					} else {
						lroLines = append(lroLines, "   "+trimmed)
					}
				}
			} else {
				rest := strings.TrimSpace(strings.Join(rawLines[1:], "\n"))
				wrappedRest := formatReturnRstDoc(rest, 56-len(definitionIndent), 0)
				for _, rl := range wrappedRest {
					if rl == "" {
						lroLines = append(lroLines, "")
					} else {
						lroLines = append(lroLines, definitionIndent+rl)
					}
				}
			}
		} else {
			rawLRODoc := "The result type for the operation will be :class:`" + respSphinx + "`"
			if strings.TrimSpace(respDoc) != "" {
				rawLRODoc += " " + respDoc
			}
			lroLines = formatReturnRstDoc(rawLRODoc, 56, 16)
			for i, line := range lroLines {
				lroLines[i] = reClassSphinx.ReplaceAllString(line, ":class:`$1`")
			}
		}
		returnDocLines = append([]string{"An object representing a long-running operation.", ""}, lroLines...)
	} else if isPaged {
		returnType = "pagers." + pagerClassName
		returnSphinxType = ann.VersionPackage + ".services." + ann.DirectoryName + ".pagers." + pagerClassName
		pagerDoc := "Iterating over this object will yield results and resolve additional pages automatically."
		if m.OutputType != nil && strings.TrimSpace(m.OutputType.Documentation) != "" {
			pagerDoc = m.OutputType.Documentation + "\n\n" + pagerDoc
		}
		returnDocLines = formatMethodReturnDoc(pagerDoc)
	} else if iamMod, iamType, ok := isIAMType(m.OutputTypeID); ok {
		returnType = iamMod + "." + iamType
		returnSphinxType = "google.iam.v1." + iamMod + "." + iamType
		hasReturnDoc = true
		if iamType == "Policy" {
			returnDocLines = iamClientPolicyReturnDocLines
		} else {
			returnDocLines = []string{fmt.Sprintf("Response message for %s method.", pascalCase(m.Name))}
		}
	} else if !returnsEmpty {
		outStem := "common"
		outName := ""
		outMsg := m.OutputType
		if outMsg == nil && m.OutputTypeID != "" && c.Model != nil {
			outMsg = c.resolveMessageType(m.OutputTypeID)
		}
		if outMsg != nil {
			outStem = c.resolveMessageStem(outMsg)
			outName = outMsg.Name
			if outName == "Policy" && strings.HasPrefix(strings.TrimSpace(outMsg.Documentation), "An Identity and Access Management") {
				returnDocLines = iamComputePolicyReturnDocLines
			} else {
				returnDocLines = formatMethodReturnDoc(outMsg.Documentation)
			}
		}
		returnType = outStem + "." + outName
		returnSphinxType = ann.VersionPackage + ".types." + outName
	}

	asyncPagerClassName := ""
	if isPaged {
		asyncPagerClassName = pascalCase(m.Name) + "AsyncPager"
	}
	asyncReturnType := returnType
	asyncReturnSphinxType := returnSphinxType
	if isLRO {
		asyncReturnType = "operation_async.AsyncOperation"
		asyncReturnSphinxType = "google.api_core.operation_async.AsyncOperation"
	} else if isPaged {
		asyncReturnType = "pagers." + asyncPagerClassName
		asyncReturnSphinxType = ann.VersionPackage + ".services." + ann.DirectoryName + ".pagers." + asyncPagerClassName
	}

	// Flattened parameters from signatures
	var flattenedParams []*flattenedParam
	seenFields := make(map[string]bool)
	var paramNames []string

	for _, sig := range m.Signatures {
		fields := sig.Fields
		inputMsg := m.InputType
		if inputMsg == nil && m.InputTypeID != "" {
			inputMsg = c.resolveMessageType(m.InputTypeID)
		}
		if len(fields) == 0 && len(sig.Names) > 0 && inputMsg != nil {
			for _, fn := range sig.Names {
				if f := findFieldInMessage(inputMsg, fn); f != nil {
					fields = append(fields, f)
				}
			}
		}
		for _, f := range fields {
			if f == nil || seenFields[f.Name] {
				continue
			}
			seenFields[f.Name] = true
			pName := pythonIdentifier(snakeCase(f.Name))
			paramNames = append(paramNames, pName)

			tHint, sType := c.resolveFieldTypeHintAndSphinx(f, ann.VersionPackage)
			fDocs := formatRstDoc(f.Documentation, 56, 16)
			var pDocLines []string
			if len(fDocs) > 0 {
				pDocLines = append(pDocLines, fDocs...)
				if len(fDocs) > 1 {
					pDocLines = append(pDocLines, "")
				}
			}
			pDocLines = append(pDocLines,
				fmt.Sprintf("This corresponds to the ``%s`` field", f.Name),
				"on the ``request`` instance; if ``request`` is provided, this",
				"should not be set.",
			)

			flattenedParams = append(flattenedParams, &flattenedParam{
				Name:       pName,
				TypeHint:   tHint,
				SphinxType: sType,
				DocLines:   pDocLines,
				IsRepeated: f.Repeated,
			})
		}
	}

	// Routing headers
	routingHeaders := extractRoutingHeaders(m)
	var routingKey, routingHeaderAccessor string
	if len(routingHeaders) > 0 {
		routingKey = routingHeaders[0].Key
		routingHeaderAccessor = routingHeaders[0].Accessor
	}

	// Sample generation
	samplePreInitLines, sampleArgs := c.buildSampleCode(m, ann.VersionSegment)

	hasReturnDocTrailer := len(returnDocLines) != 1

	var isExtendedOp, opHasProject, opHasZone, opHasRegion bool
	var opServiceSnake, opServiceSingular string
	protoOpts := c.loadProtoServiceOptions(service)
	if opService, ok := protoOpts.MethodOperationServices[m.Name]; ok && opService != "" {
		isExtendedOp = true
		opServiceSnake = snakeCase(opService)
		opServiceSingular = strings.TrimSuffix(opService, "s")
		if m.InputType != nil {
			opHasProject = slices.ContainsFunc(m.InputType.Fields, func(f *api.Field) bool { return f.Name == "project" })
			opHasZone = slices.ContainsFunc(m.InputType.Fields, func(f *api.Field) bool { return f.Name == "zone" })
			opHasRegion = slices.ContainsFunc(m.InputType.Fields, func(f *api.Field) bool { return f.Name == "region" })
		}
	}

	return &clientMethodAnnotations{
		Method:                m,
		Name:                  name,
		DocLines:              docLines,
		DocHead:               docHead,
		DocBody:               docBody,
		HasDocHead:            docHead != "",
		HasDocBody:            len(docBody) > 0,
		RequestTypeHint:       reqTypeHint,
		RequestSphinxType:     reqSphinxType,
		RequestTypeName:       reqTypeName,
		RequestDocLines:       reqDocLines,
		ReturnType:            returnType,
		ReturnSphinxType:      returnSphinxType,
		ReturnDocLines:        returnDocLines,
		HasReturnDoc:          hasReturnDoc,
		HasReturnDocTrailer:   hasReturnDocTrailer,
		ReturnsEmpty:          returnsEmpty,
		IsPaged:               isPaged,
		PagerClassName:        pagerClassName,
		IsLRO:                 isLRO,
		IsSimpleUnary:         isSimpleUnary,
		OperationResponseType: opRespType,
		OperationMetadataType: opMetaType,
		FlattenedParams:       flattenedParams,
		FlattenedParamsList:   strings.Join(paramNames, ", "),
		HasFlattenedParams:    len(flattenedParams) > 0,
		RoutingHeaderKey:      routingKey,
		RoutingHeaderAccessor: routingHeaderAccessor,
		HasRoutingHeader:      len(routingHeaders) > 0,
		RoutingHeaders:        routingHeaders,
		SamplePreInitLines:    samplePreInitLines,
		HasSamplePreInit:      len(samplePreInitLines) > 0,
		SampleRequestArgs:     sampleArgs,
		IsExternalRequest:     isExternalRequest,
		ExternalRequestModule: extReqMod,
		ExternalRequestAlias:  extReqAlias,
		PackageImport:         ann.PackageImport,
		VersionSegment:        ann.VersionSegment,
		ClientName:            ann.ClientName,
		AsyncReturnType:       asyncReturnType,
		AsyncReturnSphinxType: asyncReturnSphinxType,
		AsyncPagerClassName:   asyncPagerClassName,
		IsExtendedOperation:   isExtendedOp,
		OpServiceSnake:        opServiceSnake,
		OpServiceSingular:     opServiceSingular,
		OpHasProject:          opHasProject,
		OpHasZone:             opHasZone,
		OpHasRegion:           opHasRegion,
	}
}
