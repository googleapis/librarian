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

	"github.com/googleapis/librarian/internal/yaml"
)

const (
	// LangAll is the identifier for all languages.
	LangAll = "all"
	// LangCsharp is the language identifier for C#.
	LangCsharp = "csharp"
	// LangDart is the language identifier for Dart.
	LangDart = "dart"
	// LangGo is the language identifier for Go.
	LangGo = "go"
	// LangJava is the language identifier for Java.
	LangJava = "java"
	// LangNodejs is the language identifier for Node.js.
	LangNodejs = "nodejs"
	// LangPhp is the language identifier for PHP.
	LangPhp = "php"
	// LangPython is the language identifier for Python.
	LangPython = "python"
	// LangRuby is the language identifier for Ruby.
	LangRuby = "ruby"
	// LangRust is the language identifier for Rust.
	LangRust = "rust"
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
	// Empty means all languages can use this API.
	//
	// Restrictions exist for several reasons:
	//   - Newer languages (Rust, Dart) skip older beta versions when stable versions exist
	//   - Python has historical legacy APIs not available to other languages
	//   - Some APIs (like DIREGAPIC protos) are only used by specific languages
	Languages []string `yaml:"languages,omitempty"`

	// NewIssueURI overrides the new issue URI from the service config's
	// publishing section.
	NewIssueURI string `yaml:"new_issue_uri,omitempty"`

	// OpenAPI is the file path to an OpenAPI spec, currently in internal/testdata.
	// This is not an official spec yet and exists only for Rust to validate OpenAPI support.
	OpenAPI string `yaml:"open_api,omitempty"`

	// Path is the proto directory path in github.com/googleapis/googleapis.
	// If ServiceConfig is empty, the service config is assumed to live at this path.
	Path string `yaml:"path,omitempty"`

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
// and then to GRPCRest.
func (api *API) Transport(language string) string {
	if trans, ok := api.Transports[language]; ok {
		return string(trans)
	}
	if trans, ok := api.Transports[LangAll]; ok {
		return string(trans)
	}

	return string(GRPCRest)
}

var (
	//go:embed apis.yaml
	apiYaml []byte
	// APIs defines API paths that require explicit configurations.
	// APIs not in this list are implicitly allowed if
	// they start with "google/cloud/".
	APIs = unmarshalAPIsOrPanic()
)

func unmarshalAPIsOrPanic() []API {
	apis, err := yaml.Unmarshal[[]API](apiYaml)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal api.yaml: %v", err))
	}
	return *apis
}
