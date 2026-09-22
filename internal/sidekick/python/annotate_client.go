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

// clientAnnotations contains all metadata needed to render client.py for a service.
type clientAnnotations struct {
	Service             *api.Service
	Name                string
	ClientName          string
	AsyncClientName     string
	TransportClassName  string
	DirectoryName       string
	ServiceDocTitle     string
	DocHead             string
	DocBody             []string
	HasMultiLineDoc     bool
	VersionPackage      string
	PackageImport       string
	VersionSegment      string
	DefaultHost         string
	EndpointTemplate    string
	ServiceFQN          string
	CustomResourcePaths []*clientResourcePath
	TypeImports         []*clientTypeImport
	ExternalImports     []*clientExternalImport
	Methods             []*clientMethodAnnotations

	HasLocationMixin      bool
	HasIAMPolicyMixin     bool
	HasSetIamPolicy       bool
	HasGetIamPolicy       bool
	HasTestIamPermissions bool
	HasOperations         bool
	HasOperationsMixin    bool
	HasLRO                bool
	HasListOperations     bool
	HasGetOperation       bool
	HasDeleteOperation    bool
	HasCancelOperation    bool
	HasWaitOperation      bool
	HasGetLocation        bool
	HasListLocations      bool
	RestAsyncIOEnabled    bool
	ShowRestBetaPreview   bool
	HasPagers             bool
	CopyrightYear         string
}

func (c *codec) annotateClient(service *api.Service) (*clientAnnotations, error) {
	name := service.Name
	clientName := name
	if !strings.HasSuffix(name, "Client") {
		clientName = name + "Client"
	}
	asyncClientName := strings.TrimSuffix(clientName, "Client") + "AsyncClient"
	transportClassName := name + "Transport"
	directoryName := snakeCase(name)
	versionPackage := c.pythonPackage()

	lastDot := strings.LastIndex(versionPackage, ".")
	packageImport := versionPackage
	versionSegment := ""
	if lastDot >= 0 {
		packageImport = versionPackage[:lastDot]
		versionSegment = versionPackage[lastDot+1:]
	}

	defaultHost := service.DefaultHost
	endpointPrefix := strings.TrimSuffix(defaultHost, ".googleapis.com")
	endpointTemplate := endpointPrefix + ".{UNIVERSE_DOMAIN}"

	serviceFQN := service.Package + "." + service.Name
	if fqn, ok := strings.CutPrefix(service.ID, "."); ok {
		serviceFQN = fqn
	}

	serviceDocTitle := strings.ToLower(strings.ReplaceAll(snakeCase(name), "_", " "))
	docLines := formatRstDocLines(service.Documentation, 72, 4)
	var docHead string
	var docBody []string
	if len(docLines) > 0 {
		docHead = docLines[0]
		if len(docLines) > 1 {
			docBody = docLines[1:]
		}
	}

	customPaths := c.annotateCustomResourcePaths(service)
	typeImports, externalImports := c.collectClientImports(service)

	svcConfig, err := c.loadServiceConfig(service)
	if err != nil {
		return nil, fmt.Errorf("%w for %s: %w", ErrLoadServiceConfig, service.Name, err)
	}
	restAsyncIOEnabled := c.isRestAsyncIOEnabled(svcConfig, service)

	ann := &clientAnnotations{
		Service:             service,
		Name:                name,
		ClientName:          clientName,
		AsyncClientName:     asyncClientName,
		TransportClassName:  transportClassName,
		DirectoryName:       directoryName,
		ServiceDocTitle:     serviceDocTitle,
		DocHead:             docHead,
		DocBody:             docBody,
		HasMultiLineDoc:     len(docLines) > 1,
		VersionPackage:      versionPackage,
		PackageImport:       packageImport,
		VersionSegment:      versionSegment,
		DefaultHost:         defaultHost,
		EndpointTemplate:    endpointTemplate,
		ServiceFQN:          serviceFQN,
		CustomResourcePaths: customPaths,
		TypeImports:         typeImports,
		ExternalImports:     externalImports,
		RestAsyncIOEnabled:  restAsyncIOEnabled,
		ShowRestBetaPreview: !c.hasRestNumericEnums(),
		CopyrightYear:       c.GenerationYear,
	}

	var hasLRO bool
	for _, m := range service.Methods {
		if !isMixin(m, service) && (m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation") {
			hasLRO = true
		}
		if isMixin(m, service) {
			srcID := strings.TrimPrefix(m.SourceServiceID, ".")
			if strings.HasPrefix(srcID, strings.TrimPrefix(operationsServiceIDPrefix, ".")) {
				ann.HasOperationsMixin = true
				switch m.Name {
				case "ListOperations":
					ann.HasListOperations = true
				case "GetOperation":
					ann.HasGetOperation = true
				case "CancelOperation":
					ann.HasCancelOperation = true
				case "DeleteOperation":
					ann.HasDeleteOperation = true
				case "WaitOperation":
					ann.HasWaitOperation = true
				}
			} else if strings.HasPrefix(srcID, strings.TrimPrefix(locationsServiceIDPrefix, ".")) {
				ann.HasLocationMixin = true
				switch m.Name {
				case "GetLocation":
					ann.HasGetLocation = true
				case "ListLocations":
					ann.HasListLocations = true
				}
			} else if strings.HasPrefix(srcID, strings.TrimPrefix(iamServiceIDPrefix, ".")) {
				ann.HasIAMPolicyMixin = true
				switch m.Name {
				case "SetIamPolicy":
					ann.HasSetIamPolicy = true
				case "GetIamPolicy":
					ann.HasGetIamPolicy = true
				case "TestIamPermissions":
					ann.HasTestIamPermissions = true
				}
			}
			continue
		}

		mAnn := c.annotateClientMethod(m, service, ann)
		if mAnn.IsPaged {
			ann.HasPagers = true
		}
		ann.Methods = append(ann.Methods, mAnn)
	}

	ann.HasLRO = hasLRO
	ann.HasOperations = ann.HasOperationsMixin || hasLRO
	return ann, nil
}

// hasRestNumericEnums reports whether the rest-numeric-enums generator option is enabled
// for the library, checking package-level and library-level generator flags in OptArgsByAPI.
func (c *codec) hasRestNumericEnums() bool {
	if c.Library != nil && c.Library.Python != nil {
		for _, optList := range c.Library.Python.OptArgsByAPI {
			if slices.Contains(optList, "rest-numeric-enums") || slices.Contains(optList, "rest_numeric_enums") {
				return true
			}
		}
	}
	return false
}
