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
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	typeURLPrefix = "type.googleapis.com/"
)

type messageAnnotations struct {
	Name     string
	DocLines []string
	Model    *modelAnnotations
	TypeURL  string

	HasData             bool
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
	ProtoTypeName     string
	ModulePath        string

	// DiagnoseCodable is true when the `Codable` members need `@diagnose` to
	// suppress a deprecation warning.
	//
	// `init(from:)` and `encode(to:)` must name every field, including the
	// deprecated ones, and the proto conversions do the same. The name avoids
	// colliding with `methodAnnotations.DiagnoseFields`, which means something
	// different: mustache resolves `{{#Codec.X}}` by walking the context
	// stack, so two same-named fields on different annotation types can be
	// confused.
	DiagnoseCodable bool

	// DiagnosePagination is true when the `GoogleGax._PaginatedResponse`
	// members need `@diagnose` to suppress a deprecation warning.
	//
	// `_getPaginatedItems()` names the item type in its return type, so it
	// warns even when the property that holds the items is guarded.
	DiagnosePagination bool
}

// ConvertImports returns the sorted list of dynamic import statements for message conversions.
func (ann *messageAnnotations) ConvertImports() []string {
	importMap := map[string]bool{}
	for _, dep := range ann.DependsOn {
		if dep.Name == "GoogleGax" || dep.Name == ann.ModulePath {
			continue
		}
		if dep.SpiAttribute != "" {
			importMap[fmt.Sprintf("@_spi(%s) import %s", dep.SpiAttribute, dep.Name)] = true
		} else {
			importMap["@_spi(GoogleCloudInternal) import "+dep.Name] = true
		}
		if dep.Name == "GoogleWKT" {
			importMap["internal import GoogleWKTConvert"] = true
		}
	}
	var result []string
	for imp := range importMap {
		result = append(result, imp)
	}
	slices.Sort(result)
	return result
}

// MessageImports returns the list of dependencies for this message.
func (ann *messageAnnotations) MessageImports() []*dependencyImport {
	result := make([]*dependencyImport, 0, len(ann.DependsOn))
	for _, d := range ann.DependsOn {
		dep := &dependencyImport{
			Module: d.Name,
		}
		if d.SpiAttribute != "" {
			dep.Attributes = append(dep.Attributes, fmt.Sprintf("@_spi(%s)", d.SpiAttribute))
		}
		result = append(result, dep)
	}
	slices.SortFunc(result, func(a, b *dependencyImport) int {
		return cmp.Compare(a.Module, b.Module)
	})
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
		Name:              messageName(message),
		DocLines:          docLines,
		Model:             model,
		TypeURL:           typeURLPrefix + strings.TrimPrefix(message.ID, "."),
		DependsOn:         map[string]*Dependency{},
		SampleField:       sampleField,
		ParameterTypeName: parameterTypeName,
		ProtoTypeName:     c.protoMessageTypeName(message),
		ModulePath:        c.ModulePath,
	}
	if message.ServicePlaceholder {
		annotations.PlaceholderName = pascalCase(message.Name + "Client")
	}

	// Ensure the entire package depends on the package this message belongs to.
	dep, err := c.addApiPackageDependency(message.Package)
	if err != nil {
		return err
	}
	if dep != nil {
		annotations.DependsOn[dep.Name] = dep
	}
	// All messages require the well known types for GoogleWKT._AnyPackable.
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
	var hasData bool
	for _, field := range message.Fields {
		fieldCodec, err := c.annotateField(field, model)
		if err != nil {
			return err
		}
		// `message.Fields` includes the oneof variants, so this covers them too.
		if !message.Deprecated && (field.Deprecated || fieldCodec.DiagnoseType) {
			annotations.DiagnoseCodable = true
		}
		// A deprecated message suppresses deprecation diagnostics in its whole
		// scope, so its property declarations need nothing. `annotateField`
		// cannot see the enclosing message, so clear the flag here, after it
		// has been consumed above.
		if message.Deprecated {
			fieldCodec.DiagnoseType = false
		}
		if fieldCodec.PackageName != "" && fieldCodec.PackageName != c.Model.PackageName {
			dep, err := c.addApiPackageDependency(fieldCodec.PackageName)
			if err != nil {
				return err
			}
			if dep != nil {
				annotations.DependsOn[dep.Name] = dep
			}
		}
		if strings.Contains(fieldCodec.FieldType, "Foundation.Data") {
			hasData = true
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

		// `_getPaginatedItems()` returns the item type and reads the item
		// field, `_nextPageToken()` reads the page token field. A deprecated
		// message suppresses the warning for its whole scope.
		itemTypeDeprecated, err := c.fieldTypeDeprecated(itemField)
		if err != nil {
			return err
		}
		tokenDeprecated := message.Pagination.NextPageToken != nil &&
			message.Pagination.NextPageToken.Deprecated
		annotations.DiagnosePagination = !message.Deprecated &&
			(itemField.Deprecated || itemTypeDeprecated || tokenDeprecated)
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
			if nestedCodec.HasData {
				hasData = true
			}
		}
	}
	annotations.HasData = hasData
	for _, enum := range message.Enums {
		if err := c.annotateEnum(enum, model); err != nil {
			return err
		}
	}
	return nil
}
