// Copyright 2025 Google LLC
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

// Package discovery parses Google Discovery Documents into the sidekick
// abstract API model (`api.API`).
package discovery

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser/svcconfig"
	"google.golang.org/genproto/googleapis/api/serviceconfig"
)

// NewAPI parses the discovery doc in `contents` and returns the corresponding `api.API` model.
func NewAPI(serviceConfig *serviceconfig.Service, contents []byte, cfg *config.Config) (*api.API, error) {
	doc, err := newDiscoDocument(contents)
	if err != nil {
		return nil, err
	}
	result := &api.API{
		Name:        doc.Name,
		Title:       doc.Title,
		Description: doc.Description,
		Revision:    doc.Revision,
		Messages:    make([]*api.Message, 0),
		State: &api.APIState{
			ServiceByID: make(map[string]*api.Service),
			MethodByID:  make(map[string]*api.Method),
			MessageByID: make(map[string]*api.Message),
			EnumByID:    make(map[string]*api.Enum),
		},
	}
	// Discovery docs use some well-known types inspired by Protobuf. With
	// protoc these types are automatically included via `import` statements.
	// In the Discovery docs JSON inputs, these types are not automatically
	// included.
	result.LoadWellKnownTypes()

	// Discovery docs do not define a service name or package name. The service
	// config may provide one.
	packageName := ""
	if serviceConfig != nil {
		result.Name = strings.TrimSuffix(serviceConfig.Name, ".googleapis.com")
		result.Title = serviceConfig.Title
		if serviceConfig.Documentation != nil {
			result.Description = serviceConfig.Documentation.Summary
		}
		names := svcconfig.ExtractPackageName(serviceConfig)
		if names != nil {
			packageName, _ = names.PackageName, names.ServiceName
			result.PackageName = packageName
		}
	}

	for name, schema := range doc.Schemas {
		id := fmt.Sprintf(".%s.%s", packageName, schema.ID)
		if schema.Type != "object" {
			return nil, fmt.Errorf("schema %s is not an object: %q", id, schema.Type)
		}
		message := &api.Message{
			Name:          name,
			ID:            id,
			Package:       packageName,
			Documentation: schema.Description,
		}
		err := makeMessageFields(result, message, schema)
		if err != nil {
			return nil, err
		}
		result.Messages = append(result.Messages, message)
		result.State.MessageByID[id] = message
	}
	// The messages must be sorted otherwise the generated code gets different
	// output on each run.
	slices.SortStableFunc(result.Messages, compareMessages)

	for _, resource := range doc.Resources {
		if err := addServiceRecursive(result, doc, resource); err != nil {
			return nil, err
		}
	}
	// The services must be sorted otherwise the generated code gets different
	// output on each run.
	slices.SortStableFunc(result.Services, compareServices)

	lroAnnotations(result, cfg)

	return result, nil
}

// A document is an API discovery document.
type document struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Version           string             `json:"version"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	Revision          string             `json:"revision"`
	RootURL           string             `json:"rootUrl"`
	MTLSRootURL       string             `json:"mtlsRootUrl"`
	ServicePath       string             `json:"servicePath"`
	BasePath          string             `json:"basePath"`
	DocumentationLink string             `json:"documentationLink"`
	Auth              auth               `json:"auth"`
	Features          []string           `json:"features"`
	Methods           methodList         `json:"methods"`
	Schemas           map[string]*schema `json:"schemas"`
	Resources         resourceList       `json:"resources"`
}

// init performs additional initialization and checks that
// were not done during unmarshaling.
func (d *document) init() error {
	schemasByID := map[string]*schema{}
	for _, s := range d.Schemas {
		schemasByID[s.ID] = s
	}
	for name, s := range d.Schemas {
		if s.Ref != "" {
			return fmt.Errorf("top level schema %q is a reference", name)
		}
		s.Name = name
		if err := s.init(schemasByID); err != nil {
			return err
		}
	}
	for _, m := range d.Methods {
		if err := m.init(schemasByID); err != nil {
			return err
		}
	}
	for _, r := range d.Resources {
		if err := r.init("", schemasByID); err != nil {
			return err
		}
	}
	return nil
}

// newDocument unmarshals the bytes into a Document.
// It also validates the document to make sure it is error-free.
func newDiscoDocument(bytes []byte) (*document, error) {
	var doc document
	if err := json.Unmarshal(bytes, &doc); err != nil {
		return nil, err
	}
	if err := doc.init(); err != nil {
		return nil, err
	}
	return &doc, nil
}

// auth represents the auth section of a discovery document.
// Only OAuth2 information is retained.
type auth struct {
	OAuth2Scopes []scope
}

// A scope is an OAuth2 scope.
type scope struct {
	ID          string
	Description string
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *auth) UnmarshalJSON(data []byte) error {
	// Pull out the oauth2 scopes and turn them into nice structs.
	// Ignore other auth information.
	var m struct {
		OAuth2 struct {
			Scopes map[string]struct {
				Description string
			}
		}
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	// Sort keys to provide a deterministic ordering, mainly for testing.
	for _, k := range sortedKeys(m.OAuth2.Scopes) {
		a.OAuth2Scopes = append(a.OAuth2Scopes, scope{
			ID:          k,
			Description: m.OAuth2.Scopes[k].Description,
		})
	}
	return nil
}

// A schema holds a JSON schema as defined by
// https://tools.ietf.org/html/draft-zyp-json-schema-03#section-5.1.
// We only support the subset of JSON schema needed for Google API generation.
type schema struct {
	ID                   string // union types not supported
	Type                 string // union types not supported
	Format               string
	Description          string
	Properties           propertyList
	ItemSchema           *schema `json:"items"` // array of schemas not supported
	AdditionalProperties *schema // boolean not supported
	Ref                  string  `json:"$ref"`
	Default              string
	Pattern              string
	Deprecated           bool
	Enums                []string `json:"enum"`
	// Google extensions to JSON Schema
	EnumDescriptions []string
	EnumDeprecated   []bool
	Variant          *variant

	RefSchema *schema `json:"-"` // Schema referred to by $ref
	Name      string  `json:"-"` // Schema name, if top level
	Kind      kind    `json:"-"`
}

type variant struct {
	Discriminant string
	Map          []*variantMapItem
}

type variantMapItem struct {
	TypeValue string `json:"type_value"`
	Ref       string `json:"$ref"`
}

func (s *schema) init(topLevelSchemas map[string]*schema) error {
	if s == nil {
		return nil
	}
	var err error
	if s.Ref != "" {
		if s.RefSchema, err = resolveRef(s.Ref, topLevelSchemas); err != nil {
			return err
		}
	}
	s.Kind, err = s.initKind()
	if err != nil {
		return err
	}
	if s.Kind == ArrayKind && s.ItemSchema == nil {
		return fmt.Errorf("schema %+v: array does not have items", s)
	}
	if s.Kind != ArrayKind && s.ItemSchema != nil {
		return fmt.Errorf("schema %+v: non-array has items", s)
	}
	if err := s.AdditionalProperties.init(topLevelSchemas); err != nil {
		return err
	}
	if err := s.ItemSchema.init(topLevelSchemas); err != nil {
		return err
	}
	for _, p := range s.Properties {
		if err := p.Schema.init(topLevelSchemas); err != nil {
			return err
		}
	}
	return nil
}

func resolveRef(ref string, topLevelSchemas map[string]*schema) (*schema, error) {
	rs, ok := topLevelSchemas[ref]
	if !ok {
		return nil, fmt.Errorf("could not resolve schema reference %q", ref)
	}
	return rs, nil
}

func (s *schema) initKind() (kind, error) {
	if s.Ref != "" {
		return ReferenceKind, nil
	}
	switch s.Type {
	case "string", "number", "integer", "boolean", "any":
		return SimpleKind, nil
	case "object":
		if s.AdditionalProperties != nil {
			if s.AdditionalProperties.Type == "any" {
				return AnyStructKind, nil
			}
			return MapKind, nil
		}
		return StructKind, nil
	case "array":
		return ArrayKind, nil
	default:
		return 0, fmt.Errorf("unknown type %q for schema %q", s.Type, s.ID)
	}
}

// kind classifies a Schema.
type kind int

const (
	// SimpleKind is the category for any JSON Schema that maps to a
	// primitive Go type: strings, numbers, booleans, and "any" (since it
	// maps to interface{}).
	SimpleKind kind = iota

	// StructKind is the category for a JSON Schema that declares a JSON
	// object without any additional (arbitrary) properties.
	StructKind

	// MapKind is the category for a JSON Schema that declares a JSON
	// object with additional (arbitrary) properties that have a non-"any"
	// schema type.
	MapKind

	// AnyStructKind is the category for a JSON Schema that declares a
	// JSON object with additional (arbitrary) properties that can be any
	// type.
	AnyStructKind

	// ArrayKind is the category for a JSON Schema that declares an
	// "array" type.
	ArrayKind

	// ReferenceKind is the category for a JSON Schema that is a reference
	// to another JSON Schema.  During code generation, these references
	// are resolved using the API.schemas map.
	// See https://tools.ietf.org/html/draft-zyp-json-schema-03#section-5.28
	// for more details on the format.
	ReferenceKind
)

type property struct {
	Name   string
	Schema *schema
}

type propertyList []*property

// UnmarshalJSON implements the json.Unmarshaler interface.
func (pl *propertyList) UnmarshalJSON(data []byte) error {
	// In the discovery doc, properties are a map. Convert to a list.
	var m map[string]*schema
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for _, k := range sortedKeys(m) {
		*pl = append(*pl, &property{
			Name:   k,
			Schema: m[k],
		})
	}
	return nil
}

type resourceList []*resource

// UnmarshalJSON implements the json.Unmarshaler interface.
func (rl *resourceList) UnmarshalJSON(data []byte) error {
	// In the discovery doc, resources are a map. Convert to a list.
	var m map[string]*resource
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for _, k := range sortedKeys(m) {
		r := m[k]
		r.Name = k
		*rl = append(*rl, r)
	}
	return nil
}

// A resource holds information about a Google API resource.
type resource struct {
	Name       string
	FullName   string // {parent.FullName}.{Name}
	Methods    methodList
	Resources  resourceList
	Deprecated bool
}

func (r *resource) init(parentFullName string, topLevelSchemas map[string]*schema) error {
	r.FullName = fmt.Sprintf("%s.%s", parentFullName, r.Name)
	for _, m := range r.Methods {
		if err := m.init(topLevelSchemas); err != nil {
			return err
		}
	}
	for _, r2 := range r.Resources {
		if err := r2.init(r.FullName, topLevelSchemas); err != nil {
			return err
		}
	}
	return nil
}

type methodList []*method

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ml *methodList) UnmarshalJSON(data []byte) error {
	// In the discovery doc, resources are a map. Convert to a list.
	var m map[string]*method
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for _, k := range sortedKeys(m) {
		meth := m[k]
		meth.Name = k
		*ml = append(*ml, meth)
	}
	return nil
}

// A method holds information about a resource method.
type method struct {
	Name                  string
	ID                    string
	Path                  string
	HTTPMethod            string
	Description           string
	Parameters            parameterList
	ParameterOrder        []string
	Request               *schema
	Response              *schema
	Scopes                []string
	MediaUpload           *mediaUpload
	SupportsMediaDownload bool
	APIVersion            string
	Deprecated            bool
}

type mediaUpload struct {
	Accept    []string
	MaxSize   string
	Protocols map[string]protocol
}

type protocol struct {
	Multipart bool
	Path      string
}

func (m *method) init(topLevelSchemas map[string]*schema) error {
	if err := m.Request.init(topLevelSchemas); err != nil {
		return err
	}
	if err := m.Response.init(topLevelSchemas); err != nil {
		return err
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (m *method) UnmarshalJSON(data []byte) error {
	type T method // avoid a recursive call to UnmarshalJSON
	if err := json.Unmarshal(data, (*T)(m)); err != nil {
		return err
	}
	return nil
}

type parameterList []*parameter

// UnmarshalJSON implements the json.Unmarshaler interface.
func (pl *parameterList) UnmarshalJSON(data []byte) error {
	// In the discovery doc, resources are a map. Convert to a list.
	var m map[string]*parameter
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for _, k := range sortedKeys(m) {
		p := m[k]
		p.Name = k
		*pl = append(*pl, p)
	}
	return nil
}

// A parameter holds information about a method parameter.
type parameter struct {
	Name string
	schema
	Required bool
	Repeated bool
	Location string
}

// sortedKeys returns the keys of m, which must be a map[string]T, in sorted order.
func sortedKeys[Map ~map[string]V, V any](m Map) []string {
	keys := slices.Collect(maps.Keys(m))
	sort.Strings(keys)
	return keys
}
