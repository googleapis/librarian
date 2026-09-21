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
	"slices"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// pagerTypeImport defines a type import for pagers.py.
type pagerTypeImport struct {
	Package string // e.g. "google.cloud.asset_v1.types"
	Module  string // e.g. "assets"
}

// pagerAnnotations holds template metadata for a single paged method's pagers.
type pagerAnnotations struct {
	MethodNameSnake    string
	MethodNamePascal   string
	RequestType        string
	RequestDocType     string
	ResponseType       string
	ResponseDocType    string
	ItemType           string
	PageTokenField     string
	NextPageTokenField string
	PageableItemField  string
}

// pagersAnnotation aggregates all pagers and required type imports for a service.
type pagersAnnotation struct {
	Pagers      []*pagerAnnotations
	TypeImports []*pagerTypeImport
	HasPagers   bool
}

// annotatePagers detects and annotates all paged methods in a service.
func (c *codec) annotatePagers(service *api.Service) *pagersAnnotation {
	if service == nil {
		return &pagersAnnotation{}
	}

	typesPkg := c.pythonPackage() + ".types"
	typeModules := make(map[string]bool)
	var pagers []*pagerAnnotations

	for _, method := range service.Methods {
		if method == nil || isMixin(method, service) {
			continue
		}
		if method.Pagination == nil || method.OutputType == nil || method.OutputType.Pagination == nil || method.OutputType.Pagination.PageableItem == nil {
			continue
		}

		methodNameSnake := pythonIdentifier(snakeCase(method.Name))
		methodNamePascal := pascalCase(method.Name)

		pageTokenField := "page_token"
		if method.Pagination != nil && method.Pagination.Name != "" {
			pageTokenField = pythonIdentifier(snakeCase(method.Pagination.Name))
		}

		nextPageTokenField := "next_page_token"
		if method.OutputType.Pagination.NextPageToken != nil && method.OutputType.Pagination.NextPageToken.Name != "" {
			nextPageTokenField = pythonIdentifier(snakeCase(method.OutputType.Pagination.NextPageToken.Name))
		}

		pageableItemField := pythonIdentifier(snakeCase(method.OutputType.Pagination.PageableItem.Name))

		reqModule, reqRelName := c.resolveMessageTypeName(method.InputType, method.InputTypeID, service)
		if reqModule != "" {
			typeModules[reqModule] = true
		}

		respModule, respRelName := c.resolveMessageTypeName(method.OutputType, method.OutputTypeID, service)
		if respModule != "" {
			typeModules[respModule] = true
		}

		itemType := c.resolveItemType(method.OutputType.Pagination.PageableItem, service, typeModules)

		pager := &pagerAnnotations{
			MethodNameSnake:    methodNameSnake,
			MethodNamePascal:   methodNamePascal,
			RequestType:        qualifyType(reqModule, reqRelName),
			RequestDocType:     qualifyType(typesPkg, reqRelName),
			ResponseType:       qualifyType(respModule, respRelName),
			ResponseDocType:    qualifyType(typesPkg, respRelName),
			ItemType:           itemType,
			PageTokenField:     pageTokenField,
			NextPageTokenField: nextPageTokenField,
			PageableItemField:  pageableItemField,
		}
		pagers = append(pagers, pager)
		if mAnn, ok := method.Codec.(*methodAnnotations); ok {
			mAnn.Pager = pager
			mAnn.IsPaged = true
		}
	}

	var modules []string
	for mod := range typeModules {
		modules = append(modules, mod)
	}
	slices.Sort(modules)

	var typeImports []*pagerTypeImport
	for _, mod := range modules {
		typeImports = append(typeImports, &pagerTypeImport{
			Package: typesPkg,
			Module:  mod,
		})
	}

	return &pagersAnnotation{
		Pagers:      pagers,
		TypeImports: typeImports,
		HasPagers:   len(pagers) > 0,
	}
}

func (c *codec) resolveMessageTypeName(msg *api.Message, typeID string, service *api.Service) (module, relName string) {
	if typeID == "" && msg != nil {
		typeID = msg.ID
	}
	if msg == nil && c.Model != nil && typeID != "" {
		msg = c.Model.Message(typeID)
	}
	module = c.resolveTypeModule(typeID, service)
	if msg != nil {
		relName = relativeTypeName(msg)
	} else if typeID != "" {
		relName = typeNameFromID(typeID)
	}
	return module, relName
}

func qualifyType(mod, relName string) string {
	if relName == "" {
		return ""
	}
	if mod != "" {
		return mod + "." + relName
	}
	return relName
}

// resolveItemType resolves the Python type expression for a pageable item field.
func (c *codec) resolveItemType(field *api.Field, service *api.Service, typeModules map[string]bool) string {
	if field == nil {
		return ""
	}
	var target any
	typeID := field.TypezID
	switch {
	case field.MessageType != nil || field.Typez == api.TypezMessage:
		msg := field.MessageType
		if msg == nil && c.Model != nil && typeID != "" {
			msg = c.Model.Message(typeID)
		}
		if typeID == "" && msg != nil {
			typeID = msg.ID
		}
		target = msg
	case field.EnumType != nil || field.Typez == api.TypezEnum:
		enum := field.EnumType
		if enum == nil && c.Model != nil && typeID != "" {
			enum = c.Model.Enum(typeID)
		}
		if typeID == "" && enum != nil {
			typeID = enum.ID
		}
		target = enum
	default:
		return primitiveTypeHint(field.Typez)
	}
	mod := c.resolveTypeModule(typeID, service)
	if mod != "" && typeModules != nil {
		typeModules[mod] = true
	}
	relName := ""
	if target != nil {
		relName = relativeTypeName(target)
	} else if typeID != "" {
		relName = typeNameFromID(typeID)
	}
	return qualifyType(mod, relName)
}
