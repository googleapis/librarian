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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	defaultAuthScope          = "https://www.googleapis.com/auth/cloud-platform"
	operationsServiceIDPrefix = ".google.longrunning.Operations"
	locationsServiceIDPrefix  = ".google.cloud.location.Locations"

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
	RestAsyncIOEnabled bool

	DocLines    []string
	HasDocLines bool

	// Mixin method availability for property stubs
	HasListOperations  bool
	HasGetOperation    bool
	HasCancelOperation bool
	HasDeleteOperation bool
	HasWaitOperation   bool
	HasGetLocation     bool
	HasListLocations   bool

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

type methodConfigName struct {
	Service string `json:"service"`
	Method  string `json:"method"`
}

type grpcServiceConfig struct {
	MethodConfig []struct {
		Names       []methodConfigName `json:"name"`
		Timeout     string             `json:"timeout"`
		RetryPolicy *struct {
			MaxAttempts          int      `json:"maxAttempts"`
			InitialBackoff       string   `json:"initialBackoff"`
			MaxBackoff           string   `json:"maxBackoff"`
			BackoffMultiplier    float64  `json:"backoffMultiplier"`
			RetryableStatusCodes []string `json:"retryableStatusCodes"`
		} `json:"retryPolicy"`
	} `json:"methodConfig"`
}

type authRule struct {
	Selector string `yaml:"selector"`
	OAuth    struct {
		CanonicalScopes string `yaml:"canonical_scopes"`
	} `yaml:"oauth"`
}

type librarySetting struct {
	Version        string `yaml:"version"`
	PythonSettings struct {
		ExperimentalFeatures struct {
			RestAsyncIOEnabled bool `yaml:"rest_async_io_enabled"`
		} `yaml:"experimental_features"`
	} `yaml:"python_settings"`
}

type serviceConfigSchema struct {
	Authentication struct {
		Rules []authRule `yaml:"rules"`
	} `yaml:"authentication"`
	Publishing struct {
		LibrarySettings []librarySetting `yaml:"library_settings"`
	} `yaml:"publishing"`
}

var (
	// errLoadServiceConfig is returned when loading a service configuration YAML fails.
	errLoadServiceConfig = errors.New("loading service config")
	// errLoadGRPCConfig is returned when loading a gRPC service configuration JSON fails.
	errLoadGRPCConfig = errors.New("loading gRPC service config")

	locationOrder = []string{"GetLocation", "ListLocations"}
	opOrder       = []string{"CancelOperation", "DeleteOperation", "GetOperation", "ListOperations", "WaitOperation"}

	grpcStatusToException = map[string]string{
		"CANCELLED":           "Cancelled",
		"UNKNOWN":             "Unknown",
		"INVALID_ARGUMENT":    "InvalidArgument",
		"DEADLINE_EXCEEDED":   "DeadlineExceeded",
		"NOT_FOUND":           "NotFound",
		"ALREADY_EXISTS":      "AlreadyExists",
		"PERMISSION_DENIED":   "PermissionDenied",
		"RESOURCE_EXHAUSTED":  "ResourceExhausted",
		"FAILED_PRECONDITION": "FailedPrecondition",
		"ABORTED":             "Aborted",
		"OUT_OF_RANGE":        "OutOfRange",
		"UNIMPLEMENTED":       "MethodNotImplemented",
		"INTERNAL":            "InternalServerError",
		"UNAVAILABLE":         "ServiceUnavailable",
		"DATA_LOSS":           "DataLoss",
		"UNAUTHENTICATED":     "Unauthenticated",
	}
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
		return nil, fmt.Errorf("%w for %s: %w", errLoadServiceConfig, service.Name, err)
	}
	scopes := c.findAuthScopes(svcConfig)
	restAsyncIOEnabled := c.isRestAsyncIOEnabled(svcConfig, service)

	cfg, err := c.loadGRPCServiceConfig(service)
	if err != nil {
		return nil, fmt.Errorf("%w for %s: %w", errLoadGRPCConfig, service.Name, err)
	}

	var (
		hasLRO             bool
		hasLocationMixin   bool
		hasOperationsMixin bool
		hasListOperations  bool
		hasGetOperation    bool
		hasCancelOperation bool
		hasDeleteOperation bool
		hasWaitOperation   bool
		hasGetLocation     bool
		hasListLocations   bool

		nativeMethods []*api.Method
		mixinMethods  []*api.Method
	)

	for _, m := range service.Methods {
		if m.OperationInfo != nil || m.OutputTypeID == ".google.longrunning.Operation" {
			hasLRO = true
		}
		if isMixinMethod(m, service) {
			mixinMethods = append(mixinMethods, m)
			if strings.HasPrefix(m.SourceServiceID, operationsServiceIDPrefix) {
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
			} else if strings.HasPrefix(m.SourceServiceID, locationsServiceIDPrefix) {
				hasLocationMixin = true
				switch m.Name {
				case "GetLocation":
					hasGetLocation = true
				case "ListLocations":
					hasListLocations = true
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

	for _, m := range nativeMethods {
		inModule := c.resolveTypeModule(m.InputTypeID, service)
		if inModule != "" {
			typeModules[inModule] = true
		}
		inTypeName := typeNameFromID(m.InputTypeID)
		if m.InputType != nil && m.InputType.Name != "" {
			inTypeName = m.InputType.Name
		}
		inputIdent := inModule + "." + inTypeName
		inputTypeShortIdent := "~." + inTypeName
		outputTypeShortIdent := ""
		requestSerializer := inputIdent + ".serialize"
		responseDeserializer := ""

		outIdent := ""
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

	// Order mixins for wrapped methods: Location -> Operations
	for _, reqName := range locationOrder {
		for _, m := range mixinMethods {
			if strings.HasPrefix(m.SourceServiceID, locationsServiceIDPrefix) && m.Name == reqName {
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
			if strings.HasPrefix(m.SourceServiceID, operationsServiceIDPrefix) && m.Name == reqName {
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
	slices.SortFunc(imports, func(a, b *transportImport) int {
		return strings.Compare(transportImportKey(a), transportImportKey(b))
	})

	return &transportAnnotations{
		Name:               name,
		TransportClassName: transportClassName,
		ServiceFQN:         serviceFQN,
		DefaultHost:        defaultHost,
		VersionPackage:     versionPackage,
		Scopes:             scopes,
		HasLRO:             hasLRO,
		HasLocationMixin:   hasLocationMixin,
		HasOperationsMixin: hasOperationsMixin,
		RestAsyncIOEnabled: restAsyncIOEnabled,
		DocLines:           docLines,
		HasDocLines:        len(docLines) > 0,
		HasListOperations:  hasListOperations,
		HasGetOperation:    hasGetOperation,
		HasCancelOperation: hasCancelOperation,
		HasDeleteOperation: hasDeleteOperation,
		HasWaitOperation:   hasWaitOperation,
		HasGetLocation:     hasGetLocation,
		HasListLocations:   hasListLocations,
		Imports:            imports,
		WrappedMethods:     wrappedMethods,
		ServiceMethods:     serviceMethods,
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

func (c *codec) buildWrappedMethod(m *api.Method, service *api.Service, cfg *grpcServiceConfig) *wrappedMethodAnnotations {
	methName := snakeCase(m.Name)
	wAnn := &wrappedMethodAnnotations{
		Name:           methName,
		DefaultTimeout: "None",
	}

	if cfg == nil {
		return wAnn
	}

	targetServiceFQN := service.Package + "." + service.Name
	if fqn, ok := strings.CutPrefix(service.ID, "."); ok {
		targetServiceFQN = fqn
	}

	for _, mc := range cfg.MethodConfig {
		matched := slices.ContainsFunc(mc.Names, func(nameEntry methodConfigName) bool {
			return (nameEntry.Service == targetServiceFQN || nameEntry.Service == "") && nameEntry.Method == m.Name
		})
		if !matched {
			continue
		}

		if mc.Timeout != "" {
			if timeout, err := parseFloatDuration(mc.Timeout); err == nil {
				wAnn.DefaultTimeout = formatFloat(timeout)
			}
		}
		if mc.RetryPolicy != nil {
			wAnn.HasRetry = true
			if initB, err := parseFloatDuration(mc.RetryPolicy.InitialBackoff); err == nil {
				wAnn.InitialBackoff = formatFloat(initB)
			}
			if maxB, err := parseFloatDuration(mc.RetryPolicy.MaxBackoff); err == nil {
				wAnn.MaxBackoff = formatFloat(maxB)
			}
			if mc.RetryPolicy.BackoffMultiplier > 0 {
				wAnn.BackoffMultiplier = formatFloat(mc.RetryPolicy.BackoffMultiplier)
			}
			wAnn.Deadline = wAnn.DefaultTimeout

			var exceptions []string
			for _, code := range mc.RetryPolicy.RetryableStatusCodes {
				if ex, ok := grpcStatusToException[code]; ok {
					exceptions = append(exceptions, ex)
				}
			}
			slices.Sort(exceptions)
			wAnn.RetryExceptions = exceptions
		}
		break
	}

	return wAnn
}

func isMixinMethod(m *api.Method, service *api.Service) bool {
	return m.SourceServiceID != "" && m.SourceServiceID != service.ID
}

func (c *codec) resolveTypeModule(typeID string, service *api.Service) string {
	if c.Model != nil {
		for curID := typeID; curID != ""; {
			if msg := c.Model.Message(curID); msg != nil && msg.SourceLocation != nil && msg.SourceLocation.File != "" {
				return strings.TrimSuffix(filepath.Base(msg.SourceLocation.File), ".proto")
			}
			if enum := c.Model.Enum(curID); enum != nil && enum.SourceLocation != nil && enum.SourceLocation.File != "" {
				return strings.TrimSuffix(filepath.Base(enum.SourceLocation.File), ".proto")
			}
			idx := strings.LastIndex(curID, ".")
			if idx <= 0 {
				break
			}
			curID = curID[:idx]
		}
	}
	if service != nil {
		return snakeCase(service.Name)
	}
	return "common"
}

func (c *codec) loadGRPCServiceConfig(service *api.Service) (*grpcServiceConfig, error) {
	configPath := c.findConfigFileForService(service, "*_grpc_service_config.json")
	if configPath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg grpcServiceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing gRPC service config %s: %w", configPath, err)
	}
	return &cfg, nil
}

func (c *codec) loadServiceConfig(service *api.Service) (*serviceConfigSchema, error) {
	var version string
	if parts := strings.Split(service.Package, "."); len(parts) > 0 {
		version = parts[len(parts)-1]
	}
	var configPath string
	if version != "" {
		configPath = c.findConfigFileForService(service, "*_"+version+".yaml")
	}
	if configPath == "" {
		configPath = c.findConfigFileForService(service, "*.yaml")
	}
	if configPath == "" {
		return nil, nil
	}
	return yaml.Read[serviceConfigSchema](configPath)
}

func (c *codec) isRestAsyncIOEnabled(cfg *serviceConfigSchema, service *api.Service) bool {
	if cfg == nil {
		return false
	}
	return slices.ContainsFunc(cfg.Publishing.LibrarySettings, func(ls librarySetting) bool {
		return (ls.Version == service.Package || ls.Version == "") && ls.PythonSettings.ExperimentalFeatures.RestAsyncIOEnabled
	})
}

func (c *codec) findAuthScopes(cfg *serviceConfigSchema) []string {
	if cfg == nil {
		return []string{defaultAuthScope}
	}
	var scopes []string
	for _, rule := range cfg.Authentication.Rules {
		if rule.OAuth.CanonicalScopes == "" {
			continue
		}
		rawScopes := strings.FieldsFunc(rule.OAuth.CanonicalScopes, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r'
		})
		for _, s := range rawScopes {
			s = strings.TrimSpace(s)
			if s != "" && !slices.Contains(scopes, s) {
				scopes = append(scopes, s)
			}
		}
	}
	if len(scopes) == 0 {
		return []string{defaultAuthScope}
	}
	return scopes
}

func (c *codec) findConfigFileForService(service *api.Service, globPattern string) string {
	pkgRelDir := strings.ReplaceAll(service.Package, ".", "/")
	var searchDirs []string

	if c.Library != nil {
		for _, root := range c.Library.Roots {
			searchDirs = append(searchDirs, filepath.Join(root, pkgRelDir))
			for _, apiCfg := range c.Library.APIs {
				if len(c.Library.APIs) == 1 || strings.ReplaceAll(apiCfg.Path, "/", ".") == service.Package || apiCfg.Path == pkgRelDir {
					searchDirs = append(searchDirs, filepath.Join(root, apiCfg.Path))
				}
			}
		}
	}

	if service != nil && service.SourceLocation != nil && service.SourceLocation.File != "" {
		dir := filepath.Dir(service.SourceLocation.File)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			searchDirs = append(searchDirs, dir)
		}
	}

	searchDirs = append(searchDirs, pkgRelDir)

	for _, dir := range searchDirs {
		matches, err := filepath.Glob(filepath.Join(dir, globPattern))
		if err == nil && len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func parseFloatDuration(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if rem, ok := strings.CutSuffix(s, "s"); ok {
		s = rem
	}
	return strconv.ParseFloat(s, 64)
}

func formatFloat(val float64) string {
	if val == float64(int64(val)) {
		return fmt.Sprintf("%.1f", val)
	}
	return strconv.FormatFloat(val, 'f', -1, 64)
}

func typeNameFromID(id string) string {
	parts := strings.Split(id, ".")
	return parts[len(parts)-1]
}

func getGRPCStubType(m *api.Method) string {
	if m.ClientSideStreaming && m.ServerSideStreaming {
		return "stream_stream"
	}
	if m.ClientSideStreaming {
		return "stream_unary"
	}
	if m.ServerSideStreaming {
		return "unary_stream"
	}
	return "unary_unary"
}
