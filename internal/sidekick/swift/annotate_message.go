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

package swift

import (
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	typeURLPrefix = "type.googleapis.com/"
)

type messageAnnotations struct {
	Name                string
	DocLines            []string
	Model               *modelAnnotations
	TypeURL             string
	CustomSerialization bool

	IsPaginatedResponse bool
	PageableItemField   string
	PageableItemType    string
	DependsOn           map[string]*Dependency

	// The name of a field to use in message examples.
	SampleField string

	// GatedBy is the list of package traits that enables this message.
	//
	// Empty unless the package is configured with `per_service_traits` enabled.
	GatedBy []string
	// GatedOp is the operation (&& or ||) to combine all the `GatedBy` traits.
	//
	// For most messages, this is " || ", as messages are enabled when any
	// service that needs them is enabled. Messages that do not map to any
	// service use " && ".
	GatedOp string

	// In discovery-based APIs, the requests messages are nested messages of a
	// message that is not generated, it is just a placeholder to represent the
	// service. This placeholder provides a namespace for the requests.
	//
	// In the generated code, the namespace is implemented by the client struct,
	// this is the name of this struct.
	PlaceholderName string

	// The message type name when it appears as a method parameter name.
	//
	// Most of the time the request types are in the package namespace, or are
	// imported with the mixin, e.g. `import GoogleCloudIamV1` imports the IAM
	// mixin request types.
	//
	// For discovery-based APIs, the request are synthetic and generated within
	// a scope. They need to be fully qualified.
	ParameterTypeName string
}

// MessageImports returns the list of dependencies for this message.
func (ann *messageAnnotations) MessageImports() []string {
	result := make([]string, 0, len(ann.DependsOn))
	for _, dep := range ann.DependsOn {
		result = append(result, dep.Name)
	}
	slices.Sort(result)
	return result
}

// IsGated returns true if this message is gated by some package traits.
func (ann *messageAnnotations) IsGated() bool {
	return len(ann.GatedBy) != 0
}

// GateExpression returns the expression for the `#if` directive.
//
// In the generated code this is used as:
//
// ```
// #if {{{GateExpression}}}
// ... all the normal code ...
// #endif
// ```
//
// Directing the compiler to enable the code only if GateExpression evaluates to
// `true` at compile time.
func (ann *messageAnnotations) GateExpression() string {
	return strings.Join(ann.GatedBy, ann.GatedOp)
}

func (c *codec) annotateMessage(message *api.Message, model *modelAnnotations) error {
	// If the message is already annotated, don't process again
	if message.Codec != nil {
		return nil
	}
	docLines, err := c.formatDocumentation(message.Documentation, message.Scopes())
	if err != nil {
		return err
	}
	sampleField := "<placeholder>"
	if len(message.Fields) != 0 {
		sampleField = camelCase(message.Fields[0].Name)
	}
	parameterTypeName, err := c.messageTypeName(message)
	if err != nil {
		return err
	}
	annotations := &messageAnnotations{
		Name:                pascalCase(message.Name),
		DocLines:            docLines,
		Model:               model,
		TypeURL:             typeURLPrefix + strings.TrimPrefix(message.ID, "."),
		CustomSerialization: len(message.OneOfs) > 0,
		DependsOn:           map[string]*Dependency{},
		SampleField:         sampleField,
		ParameterTypeName:   parameterTypeName,
	}
	if message.ServicePlaceholder {
		annotations.PlaceholderName = pascalCase(message.Name + "Client")
	}

	// Ensure the entire package depends on the package this message belongs to.
	if _, err := c.addApiPackageDependency(message.Package); err != nil {
		return err
	}
	// All messages require the well known types for GoogleCloudWkt._AnyPackable.
	wktDep, err := c.addApiPackageDependency(wellKnownProtobufPackage)
	if err != nil {
		return err
	}
	if wktDep != nil && wktDep.ApiPackage != c.Model.PackageName {
		// Messages generated in the library for WKT library (we have a few)
		// should not import the library.
		annotations.DependsOn[wktDep.Name] = wktDep
	}

	message.Codec = annotations
	for _, oneof := range message.OneOfs {
		if err := c.annotateOneOf(oneof); err != nil {
			return err
		}
	}
	for _, field := range message.Fields {
		fieldCodec, err := c.annotateField(field, model)
		if err != nil {
			return err
		}
		if fieldCodec.Name != field.JSONName || fieldCodec.UrlSafeValue {
			annotations.CustomSerialization = true
		}
		if field.Map && !fieldCodec.IsStringKeyed() {
			// In ProtoJSON map fields with non-string keys need to be
			// serialized as JSON objects with key fields. In the generated
			// Swift code, that requires a custom implementation of the
			// `Decodable` and `Encodable` protocol.
			annotations.CustomSerialization = true
		}
		if fieldCodec.PackageName != "" && fieldCodec.PackageName != c.Model.PackageName {
			dep, err := c.addApiPackageDependency(fieldCodec.PackageName)
			if err != nil {
				return err
			}
			annotations.DependsOn[dep.Name] = dep
		}
	}

	if message.Pagination != nil {
		annotations.IsPaginatedResponse = true
		// If this message is a paginated response, then require the pagination helpers package
		paginationDep, err := c.addPackageDependency(paginationSwiftPackage)
		if err != nil {
			return err
		}
		annotations.DependsOn[paginationDep.Name] = paginationDep

		itemField := message.Pagination.PageableItem
		itemFieldCodec, ok := itemField.Codec.(*fieldAnnotations)
		if !ok {
			return fmt.Errorf("internal error: pageable item field %q is not annotated", itemField.Name)
		}
		annotations.PageableItemField = itemFieldCodec.Name
		switch {
		case itemField.Repeated:
			annotations.PageableItemType = itemFieldCodec.BaseFieldType
		case itemField.Map:
			keyType, valueType, err := c.mapFieldTypeComponents(itemField.MessageType)
			if err != nil {
				return err
			}
			annotations.PageableItemType = fmt.Sprintf("(%s, %s)", keyType, valueType)
		default:
			return fmt.Errorf("pageable item field should be a map or a repeated field: %s", message.ID)
		}
	}

	for _, nested := range message.Messages {
		if err := c.annotateMessage(nested, model); err != nil {
			return err
		}
		if nestedCodec, ok := nested.Codec.(*messageAnnotations); ok {
			// If there are required packages from nested messages, add them to the outer message as well
			for _, dep := range nestedCodec.DependsOn {
				if _, err := c.addDependency(dep); err != nil {
					return err
				}
				annotations.DependsOn[dep.Name] = dep
			}
		}
	}
	for _, enum := range message.Enums {
		if err := c.annotateEnum(enum, model); err != nil {
			return err
		}
	}
	return nil
}
