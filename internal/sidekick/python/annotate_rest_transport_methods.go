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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

var iamRestPolicyDocLines = []string{
	"An Identity and Access Management (IAM) policy, which",
	"specifies access controls for Google Cloud resources.",
	"",
	"A ``Policy`` is a collection of ``bindings``. A",
	"``binding`` binds one or more ``members``, or",
	"principals, to a single ``role``. Principals can be user",
	"accounts, service accounts, Google groups, and domains",
	"(such as G Suite). A ``role`` is a named list of",
	"permissions; each ``role`` can be an IAM predefined role",
	"or a user-created custom role.",
	"",
	"For some types of Google Cloud resources, a ``binding``",
	"can also specify a ``condition``, which is a logical",
	"expression that allows access to a resource only if the",
	"expression evaluates to ``true``. A condition can add",
	"constraints based on attributes of the request, the",
	"resource, or both. To learn which resources support",
	"conditions in their IAM policies, see the `IAM",
	"documentation <https://cloud.google.com/iam/help/conditions/resource-policies>`__.",
	"",
	"**JSON example:**",
	"",
	"::",
	"",
	"       {",
	"         \"bindings\": [",
	"           {",
	"             \"role\": \"roles/resourcemanager.organizationAdmin\",",
	"             \"members\": [",
	"               \"user:mike@example.com\",",
	"               \"group:admins@example.com\",",
	"               \"domain:google.com\",",
	"               \"serviceAccount:my-project-id@appspot.gserviceaccount.com\"",
	"             ]",
	"           },",
	"           {",
	"             \"role\": \"roles/resourcemanager.organizationViewer\",",
	"             \"members\": [",
	"               \"user:eve@example.com\"",
	"             ],",
	"             \"condition\": {",
	"               \"title\": \"expirable access\",",
	"               \"description\": \"Does not grant access after Sep 2020\",",
	"               \"expression\": \"request.time <",
	"               timestamp('2020-10-01T00:00:00.000Z')\",",
	"             }",
	"           }",
	"         ],",
	"         \"etag\": \"BwWWja0YfJA=\",",
	"         \"version\": 3",
	"       }",
	"",
	"**YAML example:**",
	"",
	"::",
	"",
	"       bindings:",
	"       - members:",
	"         - user:mike@example.com",
	"         - group:admins@example.com",
	"         - domain:google.com",
	"         - serviceAccount:my-project-id@appspot.gserviceaccount.com",
	"         role: roles/resourcemanager.organizationAdmin",
	"       - members:",
	"         - user:eve@example.com",
	"         role: roles/resourcemanager.organizationViewer",
	"         condition:",
	"           title: expirable access",
	"           description: Does not grant access after Sep 2020",
	"           expression: request.time < timestamp('2020-10-01T00:00:00.000Z')",
	"       etag: BwWWja0YfJA=",
	"       version: 3",
	"",
	"For a description of IAM and its features, see the `IAM",
	"documentation <https://cloud.google.com/iam/docs/>`__.",
}

// buildRestBaseMethods constructs base method class annotations for native and mixin methods.
func buildRestBaseMethods(nativeMethods, mixinMethods []*api.Method) []*restBaseMethodAnnotation {
	var result []*restBaseMethodAnnotation
	for _, m := range nativeMethods {
		var (
			hasReq  bool
			reqVals []*requiredFieldDefaultAnnotation
			httpOpt []*httpOptionAnnotation
		)
		if mAnn, ok := m.Codec.(*methodAnnotations); ok && mAnn.RestMethod != nil {
			hasReq = mAnn.RestMethod.HasRequiredFields
			reqVals = mAnn.RestMethod.RequiredFieldsDefaultValues
			httpOpt = mAnn.RestMethod.HTTPOptions
		}
		result = append(result, &restBaseMethodAnnotation{
			BaseClassName:               "_Base" + pascalCase(m.Name),
			IsMixin:                     false,
			HasRequiredFields:           hasReq,
			RequiredFieldsDefaultValues: reqVals,
			HTTPOptions:                 httpOpt,
		})
	}
	for _, entry := range mixinGroups {
		for _, reqName := range entry.order {
			for _, m := range mixinMethods {
				if strings.HasPrefix(m.SourceServiceID, entry.prefix) && m.Name == reqName {
					var httpOpt []*httpOptionAnnotation
					if mAnn, ok := m.Codec.(*methodAnnotations); ok && mAnn.RestMethod != nil {
						httpOpt = mAnn.RestMethod.HTTPOptions
					}
					result = append(result, &restBaseMethodAnnotation{
						BaseClassName: "_Base" + pascalCase(m.Name),
						IsMixin:       true,
						HTTPOptions:   httpOpt,
					})
				}
			}
		}
	}
	return result
}

// buildRestPrimaryMethods constructs method implementation annotations for native service methods.
func (c *codec) buildRestPrimaryMethods(nativeMethods []*api.Method, service *api.Service) []*restMethodDetailAnnotation {
	var result []*restMethodDetailAnnotation
	for _, m := range nativeMethods {
		var inputIdent string
		if iamMod, iamType, ok := isIAMType(m.InputTypeID); ok {
			inputIdent = iamMod + "." + iamType
		} else {
			inModule := c.resolveTypeModule(m.InputTypeID, service)
			inTypeName := typeNameFromID(m.InputTypeID)
			if m.InputType != nil && m.InputType.Name != "" {
				inTypeName = m.InputType.Name
			}
			inputIdent = inModule + "." + inTypeName
		}

		isVoid := m.ReturnsEmpty || m.OutputTypeID == api.WktEmptyID
		isLRO := m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation"

		outIdent := ""
		if isVoid {
			outIdent = "empty_pb2.Empty"
		} else if isLRO {
			outIdent = "operations_pb2.Operation"
		} else if iamMod, iamType, ok := isIAMType(m.OutputTypeID); ok {
			outIdent = iamMod + "." + iamType
		} else {
			outModule := c.resolveTypeModule(m.OutputTypeID, service)
			outTypeName := typeNameFromID(m.OutputTypeID)
			if m.OutputType != nil && m.OutputType.Name != "" {
				outTypeName = m.OutputType.Name
			}
			outIdent = outModule + "." + outTypeName
		}

		hasBody := false
		if mAnn, ok := m.Codec.(*methodAnnotations); ok && mAnn.RestMethod != nil && len(mAnn.RestMethod.HTTPOptions) > 0 {
			hasBody = mAnn.RestMethod.HTTPOptions[0].HasBody
		}

		docLead, docRest, docWrap := docSummaryForRest(m.Name)
		var (
			hasInDoc, hasOutDoc       bool
			inFirstLine, outFirstLine string
			inRestLines, outRestLines []string
		)
		inputType := m.InputType
		if inputType == nil && m.InputTypeID != "" && c.Model != nil {
			inputType = c.resolveMessageType(m.InputTypeID)
		}
		outputType := m.OutputType
		if outputType == nil && m.OutputTypeID != "" && c.Model != nil {
			outputType = c.resolveMessageType(m.OutputTypeID)
		}
		if _, _, ok := isIAMType(m.InputTypeID); ok {
			inFirstLine = fmt.Sprintf("Request message for ``%s`` method.", pascalCase(m.Name))
			hasInDoc = true
		} else if inputType != nil && inputType.Documentation != "" {
			inFirstLine, inRestLines, hasInDoc = methodArgsDoc(inputType.Documentation)
		}
		if isLRO {
			outFirstLine, outRestLines, hasOutDoc = methodArgsDoc("This resource represents a long-running operation that is the result of a network API call.")
		} else if _, iamType, ok := isIAMType(m.OutputTypeID); ok {
			if iamType == "Policy" {
				outFirstLine = iamRestPolicyDocLines[0]
				outRestLines = iamRestPolicyDocLines[1:]
				hasOutDoc = true
			} else {
				outFirstLine = fmt.Sprintf("Response message for ``%s`` method.", pascalCase(m.Name))
				hasOutDoc = true
			}
		} else if outputType != nil && outputType.Documentation != "" && !isVoid {
			outFirstLine, outRestLines, hasOutDoc = methodArgsDoc(outputType.Documentation)
		}

		isInputProtoPlus := isProtoPlusType(m.InputTypeID, service.Package)
		isOutputProtoPlus := !isVoid && !isLRO && isProtoPlusType(m.OutputTypeID, service.Package)

		result = append(result, &restMethodDetailAnnotation{
			Name:                          snakeCase(m.Name),
			MethodPascalName:              pascalCase(m.Name),
			MethodSnakeName:               snakeCase(m.Name),
			InputTypeIdent:                inputIdent,
			OutputTypeIdent:               outIdent,
			PropertyOutputTypeIdent:       outIdent,
			IsVoid:                        isVoid,
			IsLRO:                         isLRO,
			HasBody:                       hasBody,
			DocSummaryLead:                docLead,
			DocSummaryRest:                docRest,
			DocSummaryWrap:                docWrap,
			HasInputDoc:                   hasInDoc,
			InputDocFirstLine:             inFirstLine,
			InputDocRestLines:             inRestLines,
			HasOutputDoc:                  hasOutDoc,
			OutputDocFirstLine:            outFirstLine,
			OutputDocRestLines:            outRestLines,
			OutputDocNeedsTrailingNewline: !hasOutDoc || len(outRestLines) > 0,
			IsInputProtoPlus:              isInputProtoPlus,
			IsOutputProtoPlus:             isOutputProtoPlus,
		})
	}
	return result
}

// isProtoPlusType determines whether a protobuf type identifier refers to a proto-plus type.
func isProtoPlusType(typeID string, servicePkg string) bool {
	trimmed := strings.TrimPrefix(typeID, ".")
	if strings.HasPrefix(trimmed, "google.iam.v1.") ||
		strings.HasPrefix(trimmed, "google.longrunning.") ||
		strings.HasPrefix(trimmed, "google.protobuf.") ||
		strings.HasPrefix(trimmed, "google.rpc.") {
		return false
	}
	if servicePkg != "" && strings.HasPrefix(trimmed, servicePkg) {
		return true
	}
	return false
}

// buildRestMixinMethods constructs method implementation annotations for API mixins.
func buildRestMixinMethods(mixinMethods []*api.Method) []*restMixinMethodAnnotation {
	var result []*restMixinMethodAnnotation
	for _, entry := range mixinGroups {
		for _, reqName := range entry.order {
			for _, m := range mixinMethods {
				if strings.HasPrefix(m.SourceServiceID, entry.prefix) && m.Name == reqName {
					spec := mixinSpecMap[reqName]
					hasBody := false
					if mAnn, ok := m.Codec.(*methodAnnotations); ok && mAnn.RestMethod != nil && len(mAnn.RestMethod.HTTPOptions) > 0 {
						hasBody = mAnn.RestMethod.HTTPOptions[0].HasBody
					}
					result = append(result, &restMixinMethodAnnotation{
						Name:             snakeCase(m.Name),
						MethodPascalName: pascalCase(m.Name),
						MethodSnakeName:  snakeCase(m.Name),
						InputTypeIdent:   spec.inputIdent,
						OutputTypeIdent:  spec.outputIdent,
						DocName:          strings.ReplaceAll(snakeCase(m.Name), "_", " "),
						IsVoid:           spec.isVoid,
						HasBody:          hasBody,
					})
				}
			}
		}
	}
	return result
}

// buildRestLROOperations constructs metadata for long-running operations supported by the transport.
func buildRestLROOperations(mixinMethods []*api.Method) []*restLROOperationAnnotation {
	var result []*restLROOperationAnnotation
	for _, reqName := range opOrder {
		for _, m := range mixinMethods {
			if strings.HasPrefix(m.SourceServiceID, operationsServiceIDPrefix) && m.Name == reqName {
				var httpOpt []*httpOptionAnnotation
				if mAnn, ok := m.Codec.(*methodAnnotations); ok && mAnn.RestMethod != nil {
					httpOpt = mAnn.RestMethod.HTTPOptions
				}
				result = append(result, &restLROOperationAnnotation{
					Selector:    "google.longrunning.Operations." + reqName,
					HTTPOptions: httpOpt,
				})
			}
		}
	}
	return result
}

// buildRestTypeImports computes and sorts the module imports required by REST transport methods.
func (c *codec) buildRestTypeImports(nativeMethods []*api.Method, service *api.Service, hasLRO, hasOperationsMixin bool, versionPackage string) []*restTypeImport {
	typeModules := make(map[string]bool)
	usesEmpty := false
	usesOperations := hasOperationsMixin || hasLRO
	usesIAMPolicy := false
	usesPolicy := false

	for _, m := range nativeMethods {
		if m.ReturnsEmpty || m.OutputTypeID == api.WktEmptyID {
			usesEmpty = true
		} else if m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation" {
			usesOperations = true
		} else if iamMod, _, ok := isIAMType(m.OutputTypeID); ok {
			switch iamMod {
			case "iam_policy_pb2":
				usesIAMPolicy = true
			case "policy_pb2":
				usesPolicy = true
			}
		} else if outMod := c.resolveTypeModule(m.OutputTypeID, service); outMod != "" {
			typeModules[outMod] = true
		}

		if iamMod, _, ok := isIAMType(m.InputTypeID); ok {
			switch iamMod {
			case "iam_policy_pb2":
				usesIAMPolicy = true
			case "policy_pb2":
				usesPolicy = true
			}
		} else if inMod := c.resolveTypeModule(m.InputTypeID, service); inMod != "" {
			typeModules[inMod] = true
		}
	}

	var methodImports []*restTypeImport
	for mod := range typeModules {
		methodImports = append(methodImports, &restTypeImport{
			From:   versionPackage + ".types",
			Import: mod,
		})
	}
	if usesEmpty {
		methodImports = append(methodImports, &restTypeImport{
			From:   "google.protobuf.empty_pb2",
			As:     "empty_pb2",
			Ignore: true,
		})
	}
	if usesIAMPolicy {
		methodImports = append(methodImports, &restTypeImport{
			From:   "google.iam.v1.iam_policy_pb2",
			As:     "iam_policy_pb2",
			Ignore: true,
		})
	}
	if usesPolicy {
		methodImports = append(methodImports, &restTypeImport{
			From:   "google.iam.v1.policy_pb2",
			As:     "policy_pb2",
			Ignore: true,
		})
	}

	slices.SortFunc(methodImports, func(a, b *restTypeImport) int {
		return strings.Compare(restImportKey(a), restImportKey(b))
	})

	if usesOperations {
		methodImports = append(methodImports, &restTypeImport{
			From:   "google.longrunning",
			Import: "operations_pb2",
			Ignore: true,
		})
	}
	return methodImports
}

// restImportKey returns a sort key string for a REST type import.
func restImportKey(imp *restTypeImport) string {
	if imp.As != "" {
		return fmt.Sprintf("import %s as %s", imp.From, imp.As)
	}
	return fmt.Sprintf("from %s import %s", imp.From, imp.Import)
}
