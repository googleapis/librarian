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

const (
	defaultAuthScope          = "https://www.googleapis.com/auth/cloud-platform"
	operationsServiceIDPrefix = ".google.longrunning.Operations"
	locationsServiceIDPrefix  = ".google.cloud.location.Locations"
	iamServiceIDPrefix        = ".google.iam.v1.IAMPolicy"

	serviceDocWidth  = 72
	serviceDocIndent = 4
	methodDocWidth   = 72
	methodDocIndent  = 8
)

// transportImport represents an import statement in base.py.
type transportImport struct {
	From   string
	Import string
	As     string
	Ignore bool
}

// transportAnnotations contains all data needed to render base.py, __init__.py, and README.rst for a service transport.
type transportAnnotations struct {
	Name               string
	TransportClassName string
	ServiceFQN         string
	DefaultHost        string
	VersionPackage     string
	Scopes             []string
	HasLRO             bool
	HasLocationMixin   bool
	HasOperationsMixin bool
	HasIAMPolicyMixin  bool
	RestAsyncIOEnabled bool

	DocLines    []string
	HasDocLines bool

	// Mixin method availability for property stubs
	HasListOperations     bool
	HasGetOperation       bool
	HasCancelOperation    bool
	HasDeleteOperation    bool
	HasWaitOperation      bool
	HasGetLocation        bool
	HasListLocations      bool
	HasSetIamPolicy       bool
	HasGetIamPolicy       bool
	HasTestIamPermissions bool

	Imports        []*transportImport
	WrappedMethods []*wrappedMethodAnnotations
	ServiceMethods []*transportMethodAnnotations
}

// wrappedMethodAnnotations decorates a method callable in _prep_wrapped_messages.
type wrappedMethodAnnotations struct {
	Name              string
	HasRetry          bool
	InitialBackoff    string
	MaxBackoff        string
	BackoffMultiplier string
	RetryExceptions   []string
	Deadline          string
	DefaultTimeout    string
}

// transportMethodAnnotations decorates an abstract property callable signature.
type transportMethodAnnotations struct {
	Name                 string
	InputTypeIdent       string
	OutputTypeIdent      string
	InputTypeShortIdent  string
	OutputTypeShortIdent string
	RPCPath              string
	GRPCStubType         string
	RequestSerializer    string
	ResponseDeserializer string
	DocSummaryLead       string
	DocSummaryRest       string
	DocSummaryWrap       bool
	DocLines             []string
	HasDocLines          bool
}

var (
	locationOrder = []string{"GetLocation", "ListLocations"}
	iamOrder      = []string{"GetIamPolicy", "SetIamPolicy", "TestIamPermissions"}
	opOrder       = []string{"CancelOperation", "DeleteOperation", "GetOperation", "ListOperations", "WaitOperation"}
)

func (c *codec) annotateTransport(service *api.Service) (*transportAnnotations, error) {
	name := service.Name
	transportClassName := name + "Transport"
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
	restAsyncIOEnabled := c.isRestAsyncIOEnabled(svcConfig, service)

	cfg, err := c.loadGRPCServiceConfig(service)
	if err != nil {
		return nil, fmt.Errorf("%w for %s: %w", ErrLoadGRPCConfig, service.Name, err)
	}

	var (
		hasLRO                bool
		hasLocationMixin      bool
		hasOperationsMixin    bool
		hasIAMPolicyMixin     bool
		hasListOperations     bool
		hasGetOperation       bool
		hasCancelOperation    bool
		hasDeleteOperation    bool
		hasWaitOperation      bool
		hasGetLocation        bool
		hasListLocations      bool
		hasSetIamPolicy       bool
		hasGetIamPolicy       bool
		hasTestIamPermissions bool

		nativeMethods []*api.Method
		mixinMethods  []*api.Method
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
				switch m.Name {
				case "ListOperations":
					hasListOperations = true
				case "GetOperation":
					hasGetOperation = true
				case "CancelOperation":
					hasCancelOperation = true
				case "DeleteOperation":
					hasDeleteOperation = true
				case "WaitOperation":
					hasWaitOperation = true
				}
			} else if strings.HasPrefix(srcID, strings.TrimPrefix(locationsServiceIDPrefix, ".")) {
				hasLocationMixin = true
				switch m.Name {
				case "GetLocation":
					hasGetLocation = true
				case "ListLocations":
					hasListLocations = true
				}
			} else if strings.HasPrefix(srcID, strings.TrimPrefix(iamServiceIDPrefix, ".")) {
				hasIAMPolicyMixin = true
				switch m.Name {
				case "SetIamPolicy":
					hasSetIamPolicy = true
				case "GetIamPolicy":
					hasGetIamPolicy = true
				case "TestIamPermissions":
					hasTestIamPermissions = true
				}
			}
		} else {
			nativeMethods = append(nativeMethods, m)
		}
	}

	var serviceMethods []*transportMethodAnnotations
	typeModules := make(map[string]bool)
	usesEmpty := false
	usesOperations := hasOperationsMixin
	usesIAMPolicy := false
	usesPolicy := false

	for _, m := range nativeMethods {
		var (
			inputIdent           string
			inputTypeShortIdent  string
			requestSerializer    string
			outIdent             string
			outputTypeShortIdent string
			responseDeserializer string
		)

		if iamMod, iamType, ok := isIAMType(m.InputTypeID); ok {
			inputIdent = iamMod + "." + iamType
			inputTypeShortIdent = "~." + iamType
			requestSerializer = inputIdent + ".SerializeToString"
			switch iamMod {
			case "iam_policy_pb2":
				usesIAMPolicy = true
			case "policy_pb2":
				usesPolicy = true
			}
		} else {
			inModule := c.resolveTypeModule(m.InputTypeID, service)
			if inModule != "" {
				typeModules[inModule] = true
			}
			inTypeName := typeNameFromID(m.InputTypeID)
			if m.InputType != nil && m.InputType.Name != "" {
				inTypeName = m.InputType.Name
			}
			inputIdent = inModule + "." + inTypeName
			inputTypeShortIdent = "~." + inTypeName
			requestSerializer = inputIdent + ".serialize"
		}

		if m.ReturnsEmpty || m.OutputTypeID == api.WktEmptyID {
			outIdent = "empty_pb2.Empty"
			outputTypeShortIdent = "~.Empty"
			responseDeserializer = "empty_pb2.Empty.FromString"
			usesEmpty = true
		} else if m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation" {
			outIdent = "operations_pb2.Operation"
			outputTypeShortIdent = "~.Operation"
			responseDeserializer = "operations_pb2.Operation.FromString"
			usesOperations = true
		} else if iamMod, iamType, ok := isIAMType(m.OutputTypeID); ok {
			outIdent = iamMod + "." + iamType
			outputTypeShortIdent = "~." + iamType
			responseDeserializer = outIdent + ".FromString"
			switch iamMod {
			case "iam_policy_pb2":
				usesIAMPolicy = true
			case "policy_pb2":
				usesPolicy = true
			}
		} else {
			outModule := c.resolveTypeModule(m.OutputTypeID, service)
			if outModule != "" {
				typeModules[outModule] = true
			}
			outTypeName := typeNameFromID(m.OutputTypeID)
			if m.OutputType != nil && m.OutputType.Name != "" {
				outTypeName = m.OutputType.Name
			}
			outIdent = outModule + "." + outTypeName
			outputTypeShortIdent = "~." + outTypeName
			responseDeserializer = outIdent + ".deserialize"
		}

		rpcPath := "/" + serviceFQN + "/" + m.Name
		docSummary := formatMethodDocSummary(m.Name)
		mDocLines := formatRstDocLines(m.Documentation, methodDocWidth, methodDocIndent)

		serviceMethods = append(serviceMethods, &transportMethodAnnotations{
			Name:                 snakeCase(m.Name),
			InputTypeIdent:       inputIdent,
			OutputTypeIdent:      outIdent,
			InputTypeShortIdent:  inputTypeShortIdent,
			OutputTypeShortIdent: outputTypeShortIdent,
			RPCPath:              rpcPath,
			GRPCStubType:         getGRPCStubType(m),
			RequestSerializer:    requestSerializer,
			ResponseDeserializer: responseDeserializer,
			DocSummaryLead:       docSummary.Lead,
			DocSummaryRest:       docSummary.Rest,
			DocSummaryWrap:       docSummary.Wrap,
			DocLines:             mDocLines,
			HasDocLines:          len(mDocLines) > 0,
		})
	}

	var wrappedMethods []*wrappedMethodAnnotations
	for _, m := range nativeMethods {
		wrappedMethods = append(wrappedMethods, c.buildWrappedMethod(m, service, cfg))
	}

	// Order mixins for wrapped methods: Location -> IAM -> Operations
	for _, reqName := range locationOrder {
		for _, m := range mixinMethods {
			srcID := strings.TrimPrefix(m.SourceServiceID, ".")
			if strings.HasPrefix(srcID, strings.TrimPrefix(locationsServiceIDPrefix, ".")) && m.Name == reqName {
				wrappedMethods = append(wrappedMethods, &wrappedMethodAnnotations{
					Name:           snakeCase(m.Name),
					HasRetry:       false,
					DefaultTimeout: "None",
				})
			}
		}
	}
	for _, reqName := range iamOrder {
		for _, m := range mixinMethods {
			srcID := strings.TrimPrefix(m.SourceServiceID, ".")
			if strings.HasPrefix(srcID, strings.TrimPrefix(iamServiceIDPrefix, ".")) && m.Name == reqName {
				wrappedMethods = append(wrappedMethods, &wrappedMethodAnnotations{
					Name:           snakeCase(m.Name),
					HasRetry:       false,
					DefaultTimeout: "None",
				})
			}
		}
	}
	for _, reqName := range opOrder {
		for _, m := range mixinMethods {
			srcID := strings.TrimPrefix(m.SourceServiceID, ".")
			if strings.HasPrefix(srcID, strings.TrimPrefix(operationsServiceIDPrefix, ".")) && m.Name == reqName {
				wrappedMethods = append(wrappedMethods, &wrappedMethodAnnotations{
					Name:           snakeCase(m.Name),
					HasRetry:       false,
					DefaultTimeout: "None",
				})
			}
		}
	}

	var imports []*transportImport
	for mod := range typeModules {
		imports = append(imports, &transportImport{
			From:   versionPackage + ".types",
			Import: mod,
		})
	}
	if usesOperations {
		imports = append(imports, &transportImport{
			From:   "google.longrunning",
			Import: "operations_pb2",
			Ignore: true,
		})
	}
	if usesEmpty {
		imports = append(imports, &transportImport{
			From:   "google.protobuf.empty_pb2",
			As:     "empty_pb2",
			Ignore: true,
		})
	}
	if hasLocationMixin {
		imports = append(imports, &transportImport{
			From:   "google.cloud.location",
			Import: "locations_pb2",
			Ignore: true,
		})
	}
	if hasIAMPolicyMixin {
		imports = append(imports, &transportImport{
			From:   "google.iam.v1",
			Import: "iam_policy_pb2",
			Ignore: true,
		}, &transportImport{
			From:   "google.iam.v1",
			Import: "policy_pb2",
			Ignore: true,
		})
	} else {
		if usesIAMPolicy {
			imports = append(imports, &transportImport{
				From:   "google.iam.v1.iam_policy_pb2",
				As:     "iam_policy_pb2",
				Ignore: true,
			})
		}
		if usesPolicy {
			imports = append(imports, &transportImport{
				From:   "google.iam.v1.policy_pb2",
				As:     "policy_pb2",
				Ignore: true,
			})
		}
	}
	slices.SortFunc(imports, func(a, b *transportImport) int {
		return strings.Compare(transportImportKey(a), transportImportKey(b))
	})

	return &transportAnnotations{
		Name:                  name,
		TransportClassName:    transportClassName,
		ServiceFQN:            serviceFQN,
		DefaultHost:           defaultHost,
		VersionPackage:        versionPackage,
		Scopes:                scopes,
		HasLRO:                hasLRO,
		HasLocationMixin:      hasLocationMixin,
		HasOperationsMixin:    hasOperationsMixin,
		HasIAMPolicyMixin:     hasIAMPolicyMixin,
		RestAsyncIOEnabled:    restAsyncIOEnabled,
		DocLines:              docLines,
		HasDocLines:           len(docLines) > 0,
		HasListOperations:     hasListOperations,
		HasGetOperation:       hasGetOperation,
		HasCancelOperation:    hasCancelOperation,
		HasDeleteOperation:    hasDeleteOperation,
		HasWaitOperation:      hasWaitOperation,
		HasGetLocation:        hasGetLocation,
		HasListLocations:      hasListLocations,
		HasSetIamPolicy:       hasSetIamPolicy,
		HasGetIamPolicy:       hasGetIamPolicy,
		HasTestIamPermissions: hasTestIamPermissions,
		Imports:               imports,
		WrappedMethods:        wrappedMethods,
		ServiceMethods:        serviceMethods,
	}, nil
}

func transportImportKey(imp *transportImport) string {
	if imp.As != "" {
		if imp.Ignore {
			return fmt.Sprintf("import %s as %s  # type: ignore", imp.From, imp.As)
		}
		return fmt.Sprintf("import %s as %s", imp.From, imp.As)
	}
	if imp.Ignore {
		return fmt.Sprintf("from %s import %s # type: ignore", imp.From, imp.Import)
	}
	return fmt.Sprintf("from %s import %s", imp.From, imp.Import)
}
