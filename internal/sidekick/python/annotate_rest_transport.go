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
	Name, BaseTransportClassName, TransportClassName                             string
	ClientName, AsyncClientName                                                  string
	ServiceFQN, ServiceFQNClient, ServiceFQNAsyncClient                          string
	ServiceProtoName, DefaultHost, VersionPackage, ClientPackageVersion          string
	PackageName, RestNumericEnumsBool                                            string
	Scopes                                                                       []string
	HasLRO, HasLocationMixin, HasOperationsMixin, HasIAMPolicyMixin, HasDocLines bool
	RestAsyncIOEnabled, ShowRestBetaPreview                                      bool
	DocLines                                                                     []string
	TypeImports                                                                  []*restTypeImport
	BaseMethods                                                                  []*restBaseMethodAnnotation
	PrimaryMethods                                                               []*restMethodDetailAnnotation
	MixinMethods                                                                 []*restMixinMethodAnnotation
	LROOperations                                                                []*restLROOperationAnnotation
	WrappedMethods                                                               []*wrappedMethodAnnotations
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
	IsInputProtoPlus, IsOutputProtoPlus                      bool
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
		"GetLocation":        {"locations_pb2.GetLocationRequest", "locations_pb2.Location", false},
		"ListLocations":      {"locations_pb2.ListLocationsRequest", "locations_pb2.ListLocationsResponse", false},
		"GetIamPolicy":       {"iam_policy_pb2.GetIamPolicyRequest", "policy_pb2.Policy", false},
		"SetIamPolicy":       {"iam_policy_pb2.SetIamPolicyRequest", "policy_pb2.Policy", false},
		"TestIamPermissions": {"iam_policy_pb2.TestIamPermissionsRequest", "iam_policy_pb2.TestIamPermissionsResponse", false},
		"CancelOperation":    {"operations_pb2.CancelOperationRequest", "None", true},
		"DeleteOperation":    {"operations_pb2.DeleteOperationRequest", "None", true},
		"GetOperation":       {"operations_pb2.GetOperationRequest", "operations_pb2.Operation", false},
		"ListOperations":     {"operations_pb2.ListOperationsRequest", "operations_pb2.ListOperationsResponse", false},
		"WaitOperation":      {"operations_pb2.WaitOperationRequest", "operations_pb2.Operation", false},
	}
	mixinGroups = []struct {
		order  []string
		prefix string
	}{
		{locationOrder, locationsServiceIDPrefix},
		{iamOrder, iamServiceIDPrefix},
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
		return nil, fmt.Errorf("%w for %s: %w", ErrLoadServiceConfig, service.Name, err)
	}
	scopes := c.findAuthScopes(svcConfig)

	var (
		hasLRO, hasLocationMixin, hasOperationsMixin, hasIAMPolicyMixin bool
		nativeMethods, mixinMethods                                     []*api.Method
	)

	for _, m := range service.Methods {
		if !isMixin(m, service) && (m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation") {
			hasLRO = true
		}
		if isMixin(m, service) {
			mixinMethods = append(mixinMethods, m)
			srcID := strings.TrimPrefix(m.SourceServiceID, ".")
			if strings.HasPrefix(srcID, strings.TrimPrefix(operationsServiceIDPrefix, ".")) {
				hasOperationsMixin = true
			} else if strings.HasPrefix(srcID, strings.TrimPrefix(locationsServiceIDPrefix, ".")) {
				hasLocationMixin = true
			} else if strings.HasPrefix(srcID, strings.TrimPrefix(iamServiceIDPrefix, ".")) {
				hasIAMPolicyMixin = true
			}
		} else {
			nativeMethods = append(nativeMethods, m)
		}
	}

	slices.SortFunc(nativeMethods, func(a, b *api.Method) int {
		return caseInsensitiveCompare(a.Name, b.Name)
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

	ann := &restTransportAnnotation{
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
		HasIAMPolicyMixin:      hasIAMPolicyMixin,
		RestAsyncIOEnabled:     restAsyncIOEnabled,
		ShowRestBetaPreview:    !c.hasRestNumericEnums(),
		RestNumericEnumsBool:   "False",
		DocLines:               docLines,
		HasDocLines:            len(docLines) > 0,
		TypeImports:            typeImports,
		BaseMethods:            baseMethods,
		PrimaryMethods:         primaryMethods,
		MixinMethods:           orderedMixinMethods,
		LROOperations:          lroOps,
		WrappedMethods:         wrappedMethods,
	}
	if c.hasRestNumericEnums() {
		ann.RestNumericEnumsBool = "True"
	}
	return ann, nil
}
