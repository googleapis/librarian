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

// methodConfigName maps a service and method name to its gRPC method configuration.
type methodConfigName struct {
	Service string `json:"service"`
	Method  string `json:"method"`
}

// grpcServiceConfig represents the JSON schema of a gRPC service configuration file.
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

// authRule represents an authentication rule with OAuth canonical scopes in a service YAML configuration.
type authRule struct {
	Selector string `yaml:"selector"`
	OAuth    struct {
		CanonicalScopes string `yaml:"canonical_scopes"`
	} `yaml:"oauth"`
}

// librarySetting represents Python library settings in a service YAML configuration.
type librarySetting struct {
	Version        string `yaml:"version"`
	PythonSettings struct {
		ExperimentalFeatures struct {
			RestAsyncIOEnabled bool `yaml:"rest_async_io_enabled"`
		} `yaml:"experimental_features"`
	} `yaml:"python_settings"`
}

// serviceConfigSchema represents the subset of a service configuration YAML required for transport and client annotations.
type serviceConfigSchema struct {
	Authentication struct {
		Rules []authRule `yaml:"rules"`
	} `yaml:"authentication"`
	Publishing struct {
		LibrarySettings []librarySetting `yaml:"library_settings"`
	} `yaml:"publishing"`
}

var (
	// ErrLoadServiceConfig is returned when loading a service configuration YAML fails.
	ErrLoadServiceConfig = errors.New("loading service config")
	// ErrLoadGRPCConfig is returned when loading a gRPC service configuration JSON fails.
	ErrLoadGRPCConfig = errors.New("loading gRPC service config")

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

// buildWrappedMethod builds the wrapped method annotations for a service method
// using the provided gRPC service configuration.
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

// loadGRPCServiceConfig loads and parses the gRPC service configuration JSON for a service.
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

// loadServiceConfig loads and parses the service configuration YAML for a service.
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

// isRestAsyncIOEnabled reports whether REST async IO is enabled in library settings for the service.
func (c *codec) isRestAsyncIOEnabled(cfg *serviceConfigSchema, service *api.Service) bool {
	if cfg == nil {
		return false
	}
	return slices.ContainsFunc(cfg.Publishing.LibrarySettings, func(ls librarySetting) bool {
		return (ls.Version == service.Package || ls.Version == "") && ls.PythonSettings.ExperimentalFeatures.RestAsyncIOEnabled
	})
}

// findAuthScopes extracts OAuth scopes from the service configuration YAML, falling back to the default auth scope.
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

// findConfigFileForService locates a configuration file matching globPattern
// for the given service, searching library roots and the service source location.
func (c *codec) findConfigFileForService(service *api.Service, globPattern string) string {
	if service == nil {
		return ""
	}
	pkgRelDir := strings.ReplaceAll(service.Package, ".", "/")
	var searchDirs []string

	addSearchDir := func(dir string) {
		if dir != "" && !slices.Contains(searchDirs, dir) {
			searchDirs = append(searchDirs, dir)
		}
	}

	if c.Library != nil {
		for _, root := range c.Library.Roots {
			addSearchDir(filepath.Join(root, pkgRelDir))
			for _, apiCfg := range c.Library.APIs {
				if len(c.Library.APIs) == 1 || strings.ReplaceAll(apiCfg.Path, "/", ".") == service.Package || apiCfg.Path == pkgRelDir {
					addSearchDir(filepath.Join(root, apiCfg.Path))
				}
			}
		}
	}

	if service.SourceLocation != nil && service.SourceLocation.File != "" {
		dir := filepath.Dir(service.SourceLocation.File)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			addSearchDir(dir)
		}
	}

	addSearchDir(pkgRelDir)

	for _, dir := range searchDirs {
		matches, err := filepath.Glob(filepath.Join(dir, globPattern))
		if err == nil && len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// parseFloatDuration parses a duration string (e.g. "60.0s") into seconds as a float64.
func parseFloatDuration(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if rem, ok := strings.CutSuffix(s, "s"); ok {
		s = rem
	}
	return strconv.ParseFloat(s, 64)
}

// formatFloat formats a float64 into a string representation for Python code (e.g. 60.0 -> "60.0", 1.5 -> "1.5").
func formatFloat(val float64) string {
	if val == float64(int64(val)) {
		return fmt.Sprintf("%.1f", val)
	}
	return strconv.FormatFloat(val, 'f', -1, 64)
}

// getGRPCStubType returns the gRPC stub method type (e.g. "unary_unary", "unary_stream") for a method.
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
