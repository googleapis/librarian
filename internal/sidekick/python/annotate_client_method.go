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
	"path/filepath"
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
	SamplePreInitLines    []string
	HasSamplePreInit      bool
	SampleRequestArgs     []*sampleRequestArg
	PackageImport         string
	VersionSegment        string
	ClientName            string
	AsyncReturnType       string
	AsyncReturnSphinxType string
	AsyncPagerClassName   string
}

func (c *codec) annotateClientMethod(m *api.Method, service *api.Service, ann *clientAnnotations) *clientMethodAnnotations {
	name := pythonIdentifier(snakeCase(m.Name))
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
		docText := strings.TrimSpace(m.InputType.Documentation)
		if docText == "" {
			reqDocLines = []string{"The request object."}
		} else {
			converted := convertMarkdownToRst("The request object. " + docText)
			reqDocLines = wrapCommonMark(converted, 56)
		}
	}
	reqTypeHint := reqStem + "." + reqTypeName
	reqSphinxType := ann.VersionPackage + ".types." + reqTypeName

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
			for _, dl := range rawLines[1:] {
				trimmed := strings.TrimRight(dl, " \t\r")
				if strings.TrimSpace(trimmed) == "" {
					lroLines = append(lroLines, "")
				} else {
					lroLines = append(lroLines, "   "+trimmed)
				}
			}
		} else {
			rawLRODoc := "The result type for the operation will be :class:`" + respSphinx + "`"
			if strings.TrimSpace(respDoc) != "" {
				rawLRODoc += " " + respDoc
			}
			lroLines = formatRstDoc(rawLRODoc, 56, 16)
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
	} else if !returnsEmpty {
		outStem := "common"
		outName := ""
		if m.OutputType != nil {
			outStem = c.resolveMessageStem(m.OutputType)
			outName = m.OutputType.Name
			returnDocLines = formatMethodReturnDoc(m.OutputType.Documentation)
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
		if len(fields) == 0 && len(sig.Names) > 0 && m.InputType != nil {
			for _, fn := range sig.Names {
				if f := findFieldInMessage(m.InputType, fn); f != nil {
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

	// Routing header
	var routingKey, routingAccessor string
	if len(m.Routing) > 0 && len(m.Routing[0].Variants) > 0 {
		routingKey = m.Routing[0].Name
		routingAccessor = m.Routing[0].Variants[0].FieldName()
		if routingKey == "" {
			routingKey = routingAccessor
		}
	} else if m.PathInfo != nil && len(m.PathInfo.Bindings) > 0 && m.PathInfo.Bindings[0].PathTemplate != nil {
		for _, seg := range m.PathInfo.Bindings[0].PathTemplate.Segments {
			if seg.Variable != nil && len(seg.Variable.FieldPath) > 0 {
				field := strings.Join(seg.Variable.FieldPath, ".")
				routingKey = field
				routingAccessor = field
				break
			}
		}
	}

	// Sample generation
	samplePreInitLines, sampleArgs := c.buildSampleCode(m, ann.VersionSegment)

	hasReturnDocTrailer := len(returnDocLines) != 1
	var routingHeaderAccessor string
	if routingKey != "" {
		routingHeaderAccessor = "request." + routingAccessor
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
		HasRoutingHeader:      routingKey != "",
		SamplePreInitLines:    samplePreInitLines,
		HasSamplePreInit:      len(samplePreInitLines) > 0,
		SampleRequestArgs:     sampleArgs,
		PackageImport:         ann.PackageImport,
		VersionSegment:        ann.VersionSegment,
		ClientName:            ann.ClientName,
		AsyncReturnType:       asyncReturnType,
		AsyncReturnSphinxType: asyncReturnSphinxType,
		AsyncPagerClassName:   asyncPagerClassName,
	}
}

func findFieldInMessage(msg *api.Message, name string) *api.Field {
	if msg == nil {
		return nil
	}
	idx := slices.IndexFunc(msg.Fields, func(f *api.Field) bool {
		return f.Name == name
	})
	if idx < 0 {
		return nil
	}
	return msg.Fields[idx]
}

func (c *codec) resolveFieldTypeHintAndSphinx(f *api.Field, versionPackage string) (string, string) {
	if f == nil {
		return "str", "str"
	}

	var baseHint, baseSphinx string
	switch f.Typez {
	case api.TypezMessage:
		target := f.MessageType
		if target == nil && f.TypezID != "" && c.Model != nil {
			target = c.Model.Message(f.TypezID)
		}
		if target != nil {
			modelPkg := ""
			if c.Model != nil {
				modelPkg = c.Model.PackageName
			}
			targetPkg := target.Package
			if targetPkg == "" {
				targetPkg = modelPkg
			}
			targetFile := resolveProtoFile(target.SourceLocation, targetPkg, target.Name)
			if modelPkg != "" && targetPkg != modelPkg {
				stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
				importAlias := stem + "_pb2"
				baseHint = importAlias + "." + relativeTypeName(target)
				module := pythonModuleFromProto(targetFile)
				baseSphinx = module + "." + relativeTypeName(target)
			} else {
				stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
				baseHint = stem + "." + relativeTypeName(target)
				baseSphinx = versionPackage + ".types." + relativeTypeName(target)
			}
		} else {
			baseHint = "object"
			baseSphinx = "object"
		}
	case api.TypezEnum:
		target := f.EnumType
		if target == nil && f.TypezID != "" && c.Model != nil {
			target = c.Model.Enum(f.TypezID)
		}
		if target != nil {
			targetFile := resolveProtoFile(target.SourceLocation, target.Package, target.Name)
			stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
			baseHint = stem + "." + relativeTypeName(target)
			baseSphinx = versionPackage + ".types." + relativeTypeName(target)
		} else {
			baseHint = "int"
			baseSphinx = "int"
		}
	default:
		prim := primitiveTypeHint(f.Typez)
		baseHint = prim
		baseSphinx = prim
	}

	if f.Repeated {
		return fmt.Sprintf("MutableSequence[%s]", baseHint), fmt.Sprintf("MutableSequence[%s]", baseSphinx)
	}
	return baseHint, baseSphinx
}

func (c *codec) resolveMessageStem(msg *api.Message) string {
	if msg == nil {
		return "common"
	}
	src := c.sourceFileForLocation(msg.SourceLocation)
	return strings.TrimSuffix(filepath.Base(src), ".proto")
}

func (c *codec) resolveMessageType(id string) *api.Message {
	if id == "" || c.Model == nil {
		return nil
	}
	if msg := c.Model.Message(id); msg != nil {
		return msg
	}
	if cut, ok := strings.CutPrefix(id, "."); ok {
		return c.Model.Message(cut)
	}
	return c.Model.Message("." + id)
}
