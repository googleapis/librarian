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

//go:generate go run -tags configdocgen ../../cmd/config_doc_generate.go -input . -output ../../doc/api-allowlist-schema.md -root API -root-title API -title "API Allowlist"

package serviceconfig

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// Transport defines the supported transport protocol.
type Transport string

const (
	// GRPC indicates gRPC transport.
	GRPC Transport = "grpc"
	// Rest indicates REST transport.
	Rest Transport = "rest"
	// GRPCRest indicates both gRPC and REST transports.
	// This is the default if not specified.
	GRPCRest Transport = "grpc+rest"
)

// API describes an API path and its availability across languages.
type API struct {
	// Note: Properties should typically be added in alphabetical order, but
	// because this order impacts YAML serialization, we keep Path at the top
	// for ease of consumption in file-form.

	// Path is the proto directory path in github.com/googleapis/googleapis.
	// If ServiceConfig is empty, the service config is assumed to live at this
	// path.
	Path string `yaml:"path,omitempty"`

	// Description provides the information for describing an API.
	Description string `yaml:"description,omitempty"`

	// Discovery is the file path to a discovery document in
	// github.com/googleapis/discovery-artifact-manager.
	// Used by sidekick languages (Rust, Dart) as an alternative to proto files.
	Discovery string `yaml:"discovery,omitempty"`

	// DocumentationURI overrides the product documentation URI from the service
	// config's publishing section.
	DocumentationURI string `yaml:"documentation_uri,omitempty"`

	// Languages restricts which languages can generate client libraries for this API.
	// Use "all" to indicate all languages can use this API.
	//
	// Restrictions exist for several reasons:
	//   - Newer languages (Rust, Dart) skip older beta versions when stable versions exist
	//   - Python has historical legacy APIs not available to other languages
	//   - Some APIs (like DIREGAPIC protos) are only used by specific languages
	Languages []string `yaml:"languages,omitempty"`

	// NewIssueURI overrides the new issue URI from the service config's
	// publishing section.
	NewIssueURI string `yaml:"new_issue_uri,omitempty"`

	// SkipRESTNumericEnums lists languages that should not pass the
	// rest-numeric-enums flag to the generator. The special value "all"
	// skips it for every language. If empty, all languages use numeric enums.
	SkipRESTNumericEnums []string `yaml:"skip_rest_numeric_enums,omitempty"`

	// OpenAPI is the file path to an OpenAPI spec, currently in internal/testdata.
	// This is not an official spec yet and exists only for Rust to validate OpenAPI support.
	OpenAPI string `yaml:"open_api,omitempty"`

	// ReleaseLevels is the release level per language.
	// Map key is the language name (e.g., "python", "rust").
	// Optional. If omitted, the generator default is used.
	//
	// TODO(https://github.com/googleapis/librarian/issues/4834): Go uses
	// "alpha", "beta", and "ga" instead of "preview" and "stable". We should
	// standardize release level vocabulary across languages.
	ReleaseLevels map[string]string `yaml:"release_level,omitempty"`

	// ShortName overrides the API short name from the service config's
	// publishing section.
	ShortName string `yaml:"short_name,omitempty"`

	// ServiceConfig is the service config file path override.
	// If empty, the service config is discovered in the directory specified by Path.
	ServiceConfig string `yaml:"service_config,omitempty"`

	// ServiceName is a DNS-like logical identifier for the service, such as `calendar.googleapis.com`.
	ServiceName string `yaml:"service_name,omitempty"`

	// Title overrides the API title from the service config.
	Title string `yaml:"title,omitempty"`

	// Transports defines the supported transports per language.
	// Map key is the language name (e.g., "python", "rust").
	// Optional. If omitted, all languages use GRPCRest by default.
	Transports map[string]Transport `yaml:"transports,omitempty"`
}

// Transport gets transport for a given language.
//
// If language-specific transport is not defined, it falls back to the "all" language setting,
// and then to the default GRPCRest for all languages.
func (api *API) Transport(language string) Transport {
	if trans, ok := api.Transports[language]; ok {
		return trans
	}
	if trans, ok := api.Transports[config.LanguageAll]; ok {
		return trans
	}
	return GRPCRest
}

// HasRESTNumericEnums reports whether the generator should pass the
// rest-numeric-enums option for the given language. The default (when
// SkipRESTNumericEnums is empty) is true.
func (api *API) HasRESTNumericEnums(language string) bool {
	if len(api.SkipRESTNumericEnums) == 0 {
		return true
	}
	if slices.Contains(api.SkipRESTNumericEnums, config.LanguageAll) {
		return false
	}
	return !slices.Contains(api.SkipRESTNumericEnums, language)
}

// ReleaseLevel gets the release level for a given language.
//
// If language-specific release level is not defined, it falls back to the "all" language setting,
// and then it is derived based on the library version and API path maturity.
func (api *API) ReleaseLevel(language, version string) string {
	if rl, ok := api.ReleaseLevels[language]; ok {
		return rl
	}
	if rl, ok := api.ReleaseLevels[config.LanguageAll]; ok {
		return rl
	}

	// Not explicitly set, derive release level from API and client stability.
	apiVersion := ExtractVersion(api.Path)
	isAlpha := strings.Contains(apiVersion, "alpha")
	isBeta := strings.Contains(apiVersion, "beta")

	level := "stable"
	if isAlpha || isBeta || strings.HasPrefix(version, "0.") {
		level = "preview"
	}

	// TODO(https://github.com/googleapis/librarian/issues/4834): standardize
	// release level vocabulary across languages.
	if language == config.LanguageGo {
		if isAlpha {
			return "alpha"
		}
		if isBeta || level == "preview" {
			return "beta"
		}
		if level == "stable" {
			return "ga"
		}
	}
	return level
}

// RepoMetadataReleaseLevel returns the release level for repo metadata.
//
// TODO(https://github.com/googleapis/librarian/issues/4834): delete this
// function once the issue is resolved.
// For Go, it maps the raw release level to "stable" or "preview".
// For other languages, it returns the raw release level.
func (api *API) RepoMetadataReleaseLevel(language, version string) string {
	if language == config.LanguageGo {
		if api.ReleaseLevel(language, version) == "ga" {
			return "stable"
		}
		return "preview"
	}
	return api.ReleaseLevel(language, version)
}

// RepoMetadataTransport returns the transport for repo metadata.
//
// TODO(https://github.com/googleapis/librarian/issues/4854): delete
// once the issue is resolved.
// For Java, it maps the transport to "grpc", "http", or "both".
// TODO(https://github.com/googleapis/librarian/issues/5000): remove the
// Python exclusion, or change the comments.
// For Python, transport is currently excluded to reduce the difference during
// migration.
func (api *API) RepoMetadataTransport(language string, library *config.Library) string {
	transport := api.Transport(language)
	if language == config.LanguageJava && library != nil && library.Java != nil && library.Java.TransportOverride != "" {
		transport = Transport(library.Java.TransportOverride)
	}
	if language == config.LanguagePython {
		return ""
	}
	if language == config.LanguageJava {
		switch transport {
		case GRPC:
			return "grpc"
		case Rest:
			return "http"
		default:
			return "both"
		}
	}
	return string(transport)
}

var (
	//go:embed sdk.yaml
	sdkYaml []byte
	// APIs defines API paths that require explicit configurations.
	// APIs not in this list are implicitly allowed if
	// they start with "google/cloud/".
	// This is unmarshaled from sdk.yaml, which is embedded into the librarian
	// executable. The file can be edited by hand or via tooling. To change
	// the file in tooling:
	// 1. Access serviceconfig.APIs to implicitly load the existing file.
	// 2. Modify the data in memory.
	// 3. Call yaml.Write("internal/serviceconfig/sdk.yaml", serviceconfig.APIs)
	//    within the tool.
	// 4. Run `go tool yamlfmt .` from the root of the repository to reformat
	//    the file as per repository conventions.
	APIs = unmarshalAPIsOrPanic()
)

var (
	apisByPath     map[string]*API
	apisByPathOnce sync.Once
)

// HasAPIPath reports whether path matches the Path field of any API in
// sdk.yaml that is available for the given language.
func HasAPIPath(path, language string) bool {
	apisByPathOnce.Do(func() {
		apisByPath = make(map[string]*API, len(APIs))
		for i := range APIs {
			apisByPath[APIs[i].Path] = &APIs[i]
		}
	})
	api, ok := apisByPath[path]
	if !ok {
		return false
	}
	return slices.Contains(api.Languages, config.LanguageAll) || slices.Contains(api.Languages, language)
}

func unmarshalAPIsOrPanic() []API {
	apis, err := yaml.Unmarshal[[]API](sdkYaml)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal sdk.yaml: %v", err))
	}
	return *apis
}
