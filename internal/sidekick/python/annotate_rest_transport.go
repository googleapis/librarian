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

// restTransportAnnotation contains all metadata needed to render rest_base.py, rest.py, and rest_asyncio.py.
type restTransportAnnotation struct {
	Name, BaseTransportClassName, TransportClassName                    string
	ClientName, AsyncClientName                                         string
	ServiceFQN, ServiceFQNClient, ServiceFQNAsyncClient                 string
	ServiceProtoName, DefaultHost, VersionPackage, ClientPackageVersion string
	PackageName                                                         string
	Scopes                                                              []string
	HasLRO, HasLocationMixin, HasOperationsMixin, HasDocLines           bool
	RestAsyncIOEnabled                                                  bool
	DocLines                                                            []string
	TypeImports                                                         []*restTypeImport
	BaseMethods                                                         []*restBaseMethodAnnotation
	PrimaryMethods                                                      []*restMethodDetailAnnotation
	MixinMethods                                                        []*restMixinMethodAnnotation
	LROOperations                                                       []*restLROOperationAnnotation
	WrappedMethods                                                      []*wrappedMethodAnnotations
}

// restTypeImport represents an imported module or symbol used in REST transport type hints.
type restTypeImport struct {
	From, Import, As string
	Ignore           bool
}

// restBaseMethodAnnotation contains metadata for rendering a base REST transport method class.
type restBaseMethodAnnotation struct {
	BaseClassName               string
	IsMixin, HasRequiredFields  bool
	RequiredFieldsDefaultValues []*requiredFieldDefaultAnnotation
	HTTPOptions                 []*httpOptionAnnotation
}

// restMethodDetailAnnotation contains metadata for a service method implementation in a REST transport.
type restMethodDetailAnnotation struct {
	Name, MethodPascalName, MethodSnakeName                  string
	InputTypeIdent, OutputTypeIdent, PropertyOutputTypeIdent string
	DocSummaryLead, DocSummaryRest                           string
	InputDocFirstLine, OutputDocFirstLine                    string
	InputDocRestLines, OutputDocRestLines                    []string
	IsVoid, IsLRO, HasBody, DocSummaryWrap                   bool
	HasInputDoc, HasOutputDoc, OutputDocNeedsTrailingNewline bool
}

// restMixinMethodAnnotation contains metadata for an API mixin method in a REST transport.
type restMixinMethodAnnotation struct {
	Name, MethodPascalName, MethodSnakeName  string
	InputTypeIdent, OutputTypeIdent, DocName string
	IsVoid, HasBody                          bool
}

// restLROOperationAnnotation contains HTTP option metadata for a long-running operation selector.
type restLROOperationAnnotation struct {
	Selector    string
	HTTPOptions []*httpOptionAnnotation
}

// mixinSpec specifies the protobuf input/output type identifiers and void status for a mixin method.
type mixinSpec struct {
	inputIdent  string
	outputIdent string
	isVoid      bool
}

var (
	mixinSpecMap = map[string]mixinSpec{
		"GetLocation":     {"locations_pb2.GetLocationRequest", "locations_pb2.Location", false},
		"ListLocations":   {"locations_pb2.ListLocationsRequest", "locations_pb2.ListLocationsResponse", false},
		"CancelOperation": {"operations_pb2.CancelOperationRequest", "None", true},
		"DeleteOperation": {"operations_pb2.DeleteOperationRequest", "None", true},
		"GetOperation":    {"operations_pb2.GetOperationRequest", "operations_pb2.Operation", false},
		"ListOperations":  {"operations_pb2.ListOperationsRequest", "operations_pb2.ListOperationsResponse", false},
		"WaitOperation":   {"operations_pb2.WaitOperationRequest", "operations_pb2.Operation", false},
	}
	mixinGroups = []struct {
		order  []string
		prefix string
	}{
		{locationOrder, locationsServiceIDPrefix},
		{opOrder, operationsServiceIDPrefix},
	}
)

// annotateRestTransport generates transport metadata for REST client and base transport generation.
func (c *codec) annotateRestTransport(service *api.Service, svcAnn *serviceAnnotations) (*restTransportAnnotation, error) {
	name := service.Name
	baseTransportClassName := name + "Transport"
	transportClassName := name + "RestTransport"
	versionPackage := c.pythonPackage()
	defaultHost := service.DefaultHost
	serviceFQN := service.Package + "." + service.Name
	if fqn, ok := strings.CutPrefix(service.ID, "."); ok {
		serviceFQN = fqn
	}

	docLines := formatRstDocLines(service.Documentation, serviceDocWidth, serviceDocIndent)
	svcConfig, err := c.loadServiceConfig(service)
	if err != nil {
		return nil, fmt.Errorf("%w for %s: %w", errLoadServiceConfig, service.Name, err)
	}
	scopes := c.findAuthScopes(svcConfig)

	var (
		hasLRO, hasLocationMixin, hasOperationsMixin bool
		nativeMethods, mixinMethods                  []*api.Method
	)

	for _, m := range service.Methods {
		if m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation" {
			hasLRO = true
		}
		if isMixin(m, service) {
			mixinMethods = append(mixinMethods, m)
			if strings.HasPrefix(m.SourceServiceID, operationsServiceIDPrefix) {
				hasOperationsMixin = true
			} else if strings.HasPrefix(m.SourceServiceID, locationsServiceIDPrefix) {
				hasLocationMixin = true
			}
		} else {
			nativeMethods = append(nativeMethods, m)
		}
	}

	slices.SortFunc(nativeMethods, func(a, b *api.Method) int {
		return strings.Compare(a.Name, b.Name)
	})

	clientPkgVer := ""
	if parts := strings.Split(service.Package, "."); len(parts) > 0 {
		clientPkgVer = parts[len(parts)-1]
	}

	typeImports := c.buildRestTypeImports(nativeMethods, service, hasLRO, hasOperationsMixin, versionPackage)
	baseMethods := buildRestBaseMethods(nativeMethods, mixinMethods)
	primaryMethods := c.buildRestPrimaryMethods(nativeMethods, service)
	orderedMixinMethods := buildRestMixinMethods(mixinMethods)

	var lroOps []*restLROOperationAnnotation
	if hasLRO {
		lroOps = buildRestLROOperations(mixinMethods)
	}

	clientName := name + "Client"
	asyncClientName := name + "AsyncClient"
	if svcAnn != nil {
		if svcAnn.ClientName != "" {
			clientName = svcAnn.ClientName
		}
		if svcAnn.AsyncClientName != "" {
			asyncClientName = svcAnn.AsyncClientName
		}
	}

	packageName := c.pypiPackageName()
	restAsyncIOEnabled := c.isRestAsyncIOEnabled(svcConfig, service)

	var wrappedMethods []*wrappedMethodAnnotations
	if svcAnn != nil && svcAnn.Transport != nil {
		wrappedMethods = svcAnn.Transport.WrappedMethods
	} else if tAnn, err := c.annotateTransport(service); err != nil {
		return nil, err
	} else if tAnn != nil {
		wrappedMethods = tAnn.WrappedMethods
	}

	return &restTransportAnnotation{
		Name:                   name,
		BaseTransportClassName: baseTransportClassName,
		TransportClassName:     transportClassName,
		ClientName:             clientName,
		AsyncClientName:        asyncClientName,
		ServiceFQN:             serviceFQN,
		ServiceFQNClient:       versionPackage + "." + clientName,
		ServiceFQNAsyncClient:  versionPackage + "." + asyncClientName,
		ServiceProtoName:       service.Name,
		DefaultHost:            defaultHost,
		VersionPackage:         versionPackage,
		ClientPackageVersion:   clientPkgVer,
		PackageName:            packageName,
		Scopes:                 scopes,
		HasLRO:                 hasLRO,
		HasLocationMixin:       hasLocationMixin,
		HasOperationsMixin:     hasOperationsMixin,
		RestAsyncIOEnabled:     restAsyncIOEnabled,
		DocLines:               docLines,
		HasDocLines:            len(docLines) > 0,
		TypeImports:            typeImports,
		BaseMethods:            baseMethods,
		PrimaryMethods:         primaryMethods,
		MixinMethods:           orderedMixinMethods,
		LROOperations:          lroOps,
		WrappedMethods:         wrappedMethods,
	}, nil
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
		inModule := c.resolveTypeModule(m.InputTypeID, service)
		inTypeName := typeNameFromID(m.InputTypeID)
		if m.InputType != nil && m.InputType.Name != "" {
			inTypeName = m.InputType.Name
		}
		inputIdent := inModule + "." + inTypeName

		isVoid := m.ReturnsEmpty || m.OutputTypeID == api.WktEmptyID
		isLRO := m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation"

		outIdent := ""
		if isVoid {
			outIdent = "empty_pb2.Empty"
		} else if isLRO {
			outIdent = "operations_pb2.Operation"
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
		if m.InputType != nil && m.InputType.Documentation != "" {
			inFirstLine, inRestLines, hasInDoc = methodArgsDoc(m.InputType.Documentation)
		}
		if isLRO {
			outFirstLine, outRestLines, hasOutDoc = methodArgsDoc("This resource represents a long-running operation that is the result of a network API call.")
		} else if m.OutputType != nil && m.OutputType.Documentation != "" && !isVoid {
			outFirstLine, outRestLines, hasOutDoc = methodArgsDoc(m.OutputType.Documentation)
		}

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
		})
	}
	return result
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

	for _, m := range nativeMethods {
		if m.ReturnsEmpty || m.OutputTypeID == api.WktEmptyID {
			usesEmpty = true
		} else if m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation" {
			usesOperations = true
		} else if outMod := c.resolveTypeModule(m.OutputTypeID, service); outMod != "" {
			typeModules[outMod] = true
		}
		if inMod := c.resolveTypeModule(m.InputTypeID, service); inMod != "" {
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

// docSummaryForRest splits a method docstring summary line into leading and wrapped segments.
func docSummaryForRest(methodName string) (lead, rest string, wrap bool) {
	humanized := strings.ReplaceAll(snakeCase(methodName), "_", " ")
	const (
		totalWidth, firstLineOffset = 70, 45
		firstLineAvail              = totalWidth - firstLineOffset // 25
	)
	if len(humanized) <= firstLineAvail {
		return humanized, "", false
	}
	words := strings.Fields(humanized)
	var (
		line1Words        []string
		currLen, splitIdx int
	)
	for i, w := range words {
		addedLen := len(w)
		if len(line1Words) > 0 {
			addedLen++
		}
		if len(line1Words) > 0 && currLen+addedLen > firstLineAvail {
			splitIdx = i
			break
		}
		line1Words = append(line1Words, w)
		currLen += addedLen
	}
	if splitIdx == 0 && len(words) > 0 {
		line1Words = []string{words[0]}
		splitIdx = 1
	}
	lead = strings.Join(line1Words, " ")
	restWords := words[splitIdx:]
	if len(restWords) == 0 {
		return lead, "", false
	}
	return lead, strings.Join(restWords, " "), true
}

// methodArgsDoc formats method input or output documentation into first-line and subsequent lines.
func methodArgsDoc(doc string) (string, []string, bool) {
	lines := formatRstDoc(doc, 56, 16)
	if len(lines) == 0 {
		return "", nil, false
	}
	if len(lines) > 1 {
		return lines[0], lines[1:], true
	}
	return lines[0], nil, true
}
