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

package rust

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type fieldAnnotations struct {
	// In Rust, message fields are fields inside a struct. These must be
	// `snake_case`. Possibly mangled with `r#` if the name is a Rust reserved
	// word.
	FieldName string
	// In Rust, each fields gets a `set_{{FieldName}}` setter. These must be
	// `snake_case`, but are never mangled with a `r#` prefix.
	SetterName string
	// In Rust, fields that appear in a OneOf also appear as a enum branch.
	// These must be in `PascalCase`.
	BranchName string
	// The fully qualified name of the containing message.
	FQMessageName      string
	DocLines           []string
	FieldType          string
	PrimitiveFieldType string
	AddQueryParameter  string
	// For fields that are singular mesaage or list of messages, this is the
	// message type.
	MessageType *api.Message
	// For fields that are maps, these are the type of the key and value,
	// respectively.
	KeyType    string
	KeyField   *api.Field
	ValueType  string
	ValueField *api.Field
	// The templates need to generate different code for boxed fields.
	IsBoxed bool
	// If true, it requires a serde_with::serde_as() transformation.
	SerdeAs string
	// If true, the field is boxed in the prost generated type.
	MapToBoxed bool
	// If true, use `wkt::internal::is_default()` to skip the field
	SkipIfIsDefault bool
	// If true, this is a `wkt::Value` field, and requires super-extra custom
	// deserialization.
	IsWktValue bool
	// If true, this is a `wkt::NullValue` field, and also requires super-extra
	// custom deserialization.
	IsWktNullValue bool
	// Some fields may be the type of the message they are defined in.
	// We need to know this in sample generation to avoid importing
	// the parent type twice.
	// This applies to single value, repeated and map fields.
	FieldTypeIsParentType bool
	// In some cases, for instance, for OpenApi and Discovery synthetic requests,
	// types in different namespaces have the same unqualified name. When the field type and the
	// containing type have the same unqualified name, we need to alias one of those.
	AliasInExamples string
	// If this field is part of a oneof group, this will contain the other fields
	// in the group.
	OtherFieldsInGroup []*api.Field
	// FormattedResource contains information on how to format the resource name.
	FormattedResource *FormattedResource
}

// FormattedResource contain the format string and the format arguments of a resource name.
type FormattedResource struct {
	// The Rust format string, e.g., "projects/{}/secrets/{}"
	FormatString string
	// The variables used in the format string.
	FormatArgs []string
}

// SkipIfIsEmpty returns true if the field should be skipped if it is empty.
func (a *fieldAnnotations) SkipIfIsEmpty() bool {
	return !a.SkipIfIsDefault
}

// RequiresSerdeAs returns true if the field requires a serde_as annotation.
func (a *fieldAnnotations) RequiresSerdeAs() bool {
	return a.SerdeAs != ""
}

// MessageNameInExamples is the type name as used in examples.
// This will be AliasInExamples if there's an alias,
// otherwise it will be the message type or value type name.
func (a *fieldAnnotations) MessageNameInExamples() string {
	if a.AliasInExamples != "" {
		return a.AliasInExamples
	}
	if a.MessageType != nil {
		ma, _ := a.MessageType.Codec.(*messageAnnotation)
		return ma.Name
	}
	if a.ValueField != nil && a.ValueField.MessageType != nil {
		ma, _ := a.ValueField.MessageType.Codec.(*messageAnnotation)
		return ma.Name
	}
	return ""
}

func (c *codec) primitiveSerdeAs(field *api.Field) string {
	switch field.Typez {
	case api.TypezInt32, api.TypezSfixed32, api.TypezSint32:
		return "wkt::internal::I32"
	case api.TypezInt64, api.TypezSfixed64, api.TypezSint64:
		return "wkt::internal::I64"
	case api.TypezUint32, api.TypezFixed32:
		return "wkt::internal::U32"
	case api.TypezUint64, api.TypezFixed64:
		return "wkt::internal::U64"
	case api.TypezFloat:
		return "wkt::internal::F32"
	case api.TypezDouble:
		return "wkt::internal::F64"
	case api.TypezBytes:
		if c.bytesUseUrlSafeAlphabet {
			return "serde_with::base64::Base64<serde_with::base64::UrlSafe>"
		}
		return "serde_with::base64::Base64"
	default:
		return ""
	}
}

func (c *codec) mapKeySerdeAs(field *api.Field) string {
	if field.Typez == api.TypezBool {
		return "serde_with::DisplayFromStr"
	}
	return c.primitiveSerdeAs(field)
}

func (c *codec) mapValueSerdeAs(field *api.Field) string {
	if field.Typez == api.TypezMessage {
		return c.messageFieldSerdeAs(field)
	}
	return c.primitiveSerdeAs(field)
}

func (c *codec) messageFieldSerdeAs(field *api.Field) string {
	switch field.TypezID {
	case ".google.protobuf.BytesValue":
		if c.bytesUseUrlSafeAlphabet {
			return "serde_with::base64::Base64<serde_with::base64::UrlSafe>"
		}
		return "serde_with::base64::Base64"
	case ".google.protobuf.UInt64Value":
		return "wkt::internal::U64"
	case ".google.protobuf.Int64Value":
		return "wkt::internal::I64"
	case ".google.protobuf.UInt32Value":
		return "wkt::internal::U32"
	case ".google.protobuf.Int32Value":
		return "wkt::internal::I32"
	case ".google.protobuf.FloatValue":
		return "wkt::internal::F32"
	case ".google.protobuf.DoubleValue":
		return "wkt::internal::F64"
	case ".google.protobuf.BoolValue":
		return ""
	default:
		return ""
	}
}

func (c *codec) annotateField(field *api.Field, message *api.Message, model *api.API) (*fieldAnnotations, error) {
	fqMessageName, err := c.fullyQualifiedMessageName(message, model.PackageName)
	if err != nil {
		return nil, err
	}
	docLines, err := c.formatDocComments(field.Documentation, field.ID, model, message.Scopes())
	if err != nil {
		return nil, err
	}
	fieldType, err := c.fieldType(field, model, false, model.PackageName)
	if err != nil {
		return nil, err
	}
	primitiveFieldType, err := c.fieldType(field, model, true, model.PackageName)
	if err != nil {
		return nil, err
	}
	ann := &fieldAnnotations{
		FieldName:          toSnake(field.Name),
		SetterName:         toSnakeNoMangling(field.Name),
		FQMessageName:      fqMessageName,
		BranchName:         toPascal(field.Name),
		DocLines:           docLines,
		FieldType:          fieldType,
		PrimitiveFieldType: primitiveFieldType,
		AddQueryParameter:  addQueryParameter(field),
		SerdeAs:            c.primitiveSerdeAs(field),
		SkipIfIsDefault:    field.Typez != api.TypezString && field.Typez != api.TypezBytes,
		IsWktValue:         field.Typez == api.TypezMessage && field.TypezID == ".google.protobuf.Value",
		IsWktNullValue:     field.Typez == api.TypezEnum && field.TypezID == ".google.protobuf.NullValue",
	}
	if field.Recursive || (field.Typez == api.TypezMessage && field.IsOneOf) {
		ann.IsBoxed = true
	}
	ann.MapToBoxed = mapToBoxed(field, message, model)
	field.Codec = ann
	if field.Typez == api.TypezMessage {
		if msg := model.Message(field.TypezID); msg != nil && msg.IsMap {
			if len(msg.Fields) != 2 {
				return nil, fmt.Errorf("expected exactly two fields for field's map message (%q), fieldId=%s", field.TypezID, field.ID)
			}
			keyType, err := c.mapType(msg.Fields[0], model, model.PackageName)
			if err != nil {
				return nil, err
			}
			valueType, err := c.mapType(msg.Fields[1], model, model.PackageName)
			if err != nil {
				return nil, err
			}
			ann.KeyField = msg.Fields[0]
			ann.KeyType = keyType
			ann.ValueField = msg.Fields[1]
			ann.ValueType = valueType
			key := c.mapKeySerdeAs(msg.Fields[0])
			value := c.mapValueSerdeAs(msg.Fields[1])
			if key != "" || value != "" {
				if key == "" {
					key = "serde_with::Same"
				}
				if value == "" {
					value = "serde_with::Same"
				}
				ann.SerdeAs = fmt.Sprintf("std::collections::HashMap<%s, %s>", key, value)
			}
		} else {
			ann.SerdeAs = c.messageFieldSerdeAs(field)
			ann.MessageType = field.MessageType
		}
	}
	if field.Group != nil {
		ann.OtherFieldsInGroup = language.FilterSlice(field.Group.Fields, func(f *api.Field) bool { return field != f })
	}
	ann.FieldTypeIsParentType = (field.MessageType == message || // Single or repeated field whose type is the same as the containing type.
		// Map field whose value type is the same as the containing type.
		(ann.ValueField != nil && ann.ValueField.MessageType == message))
	if !ann.FieldTypeIsParentType && // When the type of the field is the same as the containing type we don't import twice. No alias needed.
		// Single or repeated field whose type's unqualified name is the same as the containing message's.
		((field.MessageType != nil && field.MessageType.Name == message.Name) ||
			// Map field whose type's unqualified name is the same as the containing message's.
			(ann.ValueField != nil && ann.ValueField.MessageType != nil && ann.ValueField.MessageType.Name == message.Name)) {
		ann.AliasInExamples = toPascal(field.Name)
		if ann.AliasInExamples == toPascal(message.Name) {
			// The field name was the same as the type name so we still have to disambiguate.
			ann.AliasInExamples = fmt.Sprintf("%sField", ann.AliasInExamples)
		}
	}

	if field.ResourceNamePattern != nil && len(field.ResourceNamePattern.Segments) > 0 {
		fr := &FormattedResource{}
		var formatString []string
		hasLiterals := false
		for _, s := range field.ResourceNamePattern.Segments {
			if s.Literal != "" {
				hasLiterals = true
				formatString = append(formatString, s.Literal)
			}
			if s.Variable != "" {
				// Convert variable to snake_case and add _id suffix if needed
				arg := toSnakeNoMangling(s.Variable)
				if !strings.HasSuffix(arg, "_id") && !strings.HasSuffix(arg, "_name") {
					arg += "_id"
				}
				formatString = append(formatString, "{"+arg+"}")
				fr.FormatArgs = append(fr.FormatArgs, arg)
			}
		}
		if hasLiterals && len(fr.FormatArgs) > 0 {
			fr.FormatString = strings.Join(formatString, "/")
			ann.FormattedResource = fr
		}
	}

	return ann, nil
}

func mapToBoxed(field *api.Field, message *api.Message, model *api.API) bool {
	if field.Typez != api.TypezMessage || field.Repeated || field.Map {
		return false
	}

	var check func(typezID string, targetID string, visited map[string]bool) bool
	check = func(typezID string, targetID string, visited map[string]bool) bool {
		if typezID == targetID {
			return true
		}
		if visited[typezID] {
			return false
		}
		visited[typezID] = true
		msg := model.Message(typezID)
		if msg == nil {
			return false
		}
		for _, f := range msg.Fields {
			if f.Typez != api.TypezMessage || f.Repeated || f.Map {
				continue
			}
			if check(f.TypezID, targetID, visited) {
				return true
			}
		}
		return false
	}

	visited := make(map[string]bool)
	return check(field.TypezID, message.ID, visited)
}
