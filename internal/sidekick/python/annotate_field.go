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
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

// fieldAnnotations decorates api.Field with Python-specific metadata.
type fieldAnnotations struct {
	Message        *messageAnnotations
	Field          *api.Field
	Name           string
	DocLines       []string
	HasDocLines    bool
	TypeName       string
	IsRepeated     bool
	IsMap          bool
	ProtoType      string
	Number         int32
	TypeHint       string
	SphinxType     string
	TypeRef        string
	IsPrimitive    bool
	IsMessage      bool
	IsEnum         bool
	IsOptional     bool
	IsOneOf        bool
	OneOfName      string
	DocIsOneOf     bool
	DocOneOfName   string
	KeyProtoType   string
	ValueProtoType string
	KeyTypeHint    string
	ValueTypeHint  string
	ValueTypeRef   string
}

func (c *codec) annotateField(field *api.Field, message *messageAnnotations) error {
	docLines := formatFieldDocLines(field.Documentation)
	ann := &fieldAnnotations{
		Message:     message,
		Field:       field,
		Name:        pythonIdentifier(field.Name),
		DocLines:    docLines,
		HasDocLines: len(docLines) > 0,
		TypeName:    field.TypezID,
		IsRepeated:  field.Repeated,
		IsMap:       field.Map,
		ProtoType:   field.Typez.String(),
		Number:      field.Number,
	}

	if field.Group != nil {
		ann.IsOneOf = true
		ann.OneOfName = pythonIdentifier(field.Group.Name)
		ann.DocIsOneOf = true
		ann.DocOneOfName = ann.OneOfName
	} else if field.IsOneOf {
		ann.IsOneOf = true
	}

	if field.Typez == api.TypezMessage {
		ann.IsOptional = c.isComputeField(field) && !field.DocumentAsRequired() && !field.Map && !field.Repeated && !ann.IsOneOf
	} else {
		ann.IsOptional = field.Optional && !field.Map && !field.Repeated && !ann.IsOneOf
	}
	if ann.IsOptional {
		ann.DocIsOneOf = true
		ann.DocOneOfName = "_" + field.Name
	}
	ann.IsPrimitive = !field.Map && field.Typez != api.TypezMessage && field.Typez != api.TypezEnum

	if field.Map {
		mapMsg := field.MessageType
		if mapMsg == nil && field.TypezID != "" && c.Model != nil {
			mapMsg = c.Model.Message(field.TypezID)
		}
		var keyField, valField *api.Field
		if mapMsg != nil && len(mapMsg.Fields) >= 2 {
			keyField = mapMsg.Fields[0]
			valField = mapMsg.Fields[1]
		}
		if keyField != nil {
			ann.KeyProtoType = keyField.Typez.String()
			kHint, _, _ := c.resolveType(message, keyField)
			ann.KeyTypeHint = kHint
		} else {
			ann.KeyProtoType = "STRING"
			ann.KeyTypeHint = "str"
		}
		var valSphinx string
		if valField != nil {
			ann.ValueProtoType = valField.Typez.String()
			vHint, vSphinx, vRef := c.resolveType(message, valField)
			ann.ValueTypeHint = vHint
			valSphinx = vSphinx
			ann.ValueTypeRef = vRef
			ann.IsMessage = valField.Typez == api.TypezMessage
			ann.IsEnum = valField.Typez == api.TypezEnum
		} else {
			ann.ValueProtoType = "STRING"
			ann.ValueTypeHint = "str"
			valSphinx = "str"
			ann.ValueTypeRef = "str"
		}
		ann.TypeHint = fmt.Sprintf("MutableMapping[%s, %s]", ann.KeyTypeHint, ann.ValueTypeHint)
		ann.SphinxType = fmt.Sprintf("MutableMapping[%s, %s]", ann.KeyTypeHint, valSphinx)
	} else {
		tHint, sType, tRef := c.resolveType(message, field)
		ann.TypeRef = tRef
		ann.IsMessage = field.Typez == api.TypezMessage
		ann.IsEnum = field.Typez == api.TypezEnum
		if field.Repeated {
			ann.TypeHint = fmt.Sprintf("MutableSequence[%s]", tHint)
			ann.SphinxType = fmt.Sprintf("MutableSequence[%s]", sType)
		} else {
			ann.TypeHint = tHint
			ann.SphinxType = sType
		}
	}

	field.Codec = ann
	return nil
}

func primitiveTypeHint(t api.Typez) string {
	switch t {
	case api.TypezString:
		return "str"
	case api.TypezBytes:
		return "bytes"
	case api.TypezBool:
		return "bool"
	case api.TypezDouble, api.TypezFloat:
		return "float"
	case api.TypezInt32, api.TypezInt64, api.TypezUint32, api.TypezUint64,
		api.TypezSint32, api.TypezSint64, api.TypezFixed32, api.TypezFixed64,
		api.TypezSfixed32, api.TypezSfixed64:
		return "int"
	default:
		return ""
	}
}

func relativeTypeName(target any) string {
	switch t := target.(type) {
	case *api.Message:
		if t == nil {
			return ""
		}
		if t.Parent == nil {
			return pascalCase(t.Name)
		}
		return relativeTypeName(t.Parent) + "." + pascalCase(t.Name)
	case *api.Enum:
		if t == nil {
			return ""
		}
		if t.Parent == nil {
			return pascalCase(t.Name)
		}
		return relativeTypeName(t.Parent) + "." + pascalCase(t.Name)
	default:
		return ""
	}
}

func (c *codec) resolveType(currentMessage *messageAnnotations, field *api.Field) (typeHint, sphinxType, typeRef string) {
	switch field.Typez {
	case api.TypezMessage:
		targetMsg := field.MessageType
		if targetMsg == nil && field.TypezID != "" && c.Model != nil {
			targetMsg = c.Model.Message(field.TypezID)
		}
		if targetMsg != nil {
			return c.resolveMessageTarget(currentMessage, targetMsg)
		}
		if field.TypezID != "" {
			return field.TypezID, field.TypezID, field.TypezID
		}
	case api.TypezEnum:
		targetEnum := field.EnumType
		if targetEnum == nil && field.TypezID != "" && c.Model != nil {
			targetEnum = c.Model.Enum(field.TypezID)
		}
		if targetEnum != nil {
			return c.resolveEnumTarget(currentMessage, targetEnum)
		}
		if field.TypezID != "" {
			return field.TypezID, field.TypezID, field.TypezID
		}
	}

	prim := primitiveTypeHint(field.Typez)
	return prim, prim, ""
}

func (c *codec) resolveTarget(currentMessage *messageAnnotations, targetPackage, targetFile, targetRelName, targetName string, isDirectChild bool) (string, string, string) {
	currentFile := ""
	if currentMessage != nil && currentMessage.Message != nil && currentMessage.Message.SourceLocation != nil {
		currentFile = currentMessage.Message.SourceLocation.File
	}

	currentStem := strings.TrimSuffix(filepath.Base(currentFile), ".proto")
	targetStem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
	if targetFile == "" {
		targetStem = currentStem
	}

	modelPkg := ""
	if c.Model != nil {
		modelPkg = c.Model.PackageName
	}

	pyPkg := c.pythonPackage()

	if modelPkg != "" && targetPackage != modelPkg {
		module := pythonModuleFromProto(targetFile)
		stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
		importAlias := stem + "_pb2"
		typeHint := importAlias + "." + targetRelName
		typeRef := importAlias + "." + targetRelName
		sphinxType := module + "." + targetRelName
		return typeHint, sphinxType, typeRef
	}

	if targetStem != currentStem {
		importAlias := targetStem
		if c.fileNeedsAlias(currentFile, targetStem) {
			initials := packageInitials(pyPkg)
			if initials != "" {
				importAlias = initials + "_" + targetStem
			}
		}
		typeHint := importAlias + "." + targetRelName
		typeRef := importAlias + "." + targetRelName
		sphinxType := targetRelName
		if pyPkg != "" {
			sphinxType = pyPkg + ".types." + targetRelName
		}
		return typeHint, sphinxType, typeRef
	}

	// Same file
	if isDirectChild {
		typeHint := pascalCase(targetName)
		typeRef := pascalCase(targetName)
		sphinxType := targetRelName
		if pyPkg != "" {
			sphinxType = pyPkg + ".types." + targetRelName
		}
		return typeHint, sphinxType, typeRef
	}

	typeHint := "'" + targetRelName + "'"
	typeRef := "'" + targetRelName + "'"
	sphinxType := targetRelName
	if pyPkg != "" {
		sphinxType = pyPkg + ".types." + targetRelName
	}
	return typeHint, sphinxType, typeRef
}

func (c *codec) resolveMessageTarget(currentMessage *messageAnnotations, target *api.Message) (string, string, string) {
	targetPackage := target.Package
	if targetPackage == "" && c.Model != nil {
		targetPackage = c.Model.PackageName
	}
	targetFile := resolveProtoFile(target.SourceLocation, targetPackage, target.Name)
	isDirectChild := currentMessage != nil && currentMessage.Message != nil && currentMessage.Message.Parent == nil && target.Parent == currentMessage.Message
	return c.resolveTarget(currentMessage, targetPackage, targetFile, relativeTypeName(target), target.Name, isDirectChild)
}

func (c *codec) resolveEnumTarget(currentMessage *messageAnnotations, target *api.Enum) (string, string, string) {
	targetPackage := target.Package
	if targetPackage == "" && c.Model != nil {
		targetPackage = c.Model.PackageName
	}
	targetFile := resolveProtoFile(target.SourceLocation, targetPackage, target.Name)
	isDirectChild := currentMessage != nil && currentMessage.Message != nil && currentMessage.Message.Parent == nil && target.Parent == currentMessage.Message
	return c.resolveTarget(currentMessage, targetPackage, targetFile, relativeTypeName(target), target.Name, isDirectChild)
}

func (c *codec) isComputeField(field *api.Field) bool {
	if c.Model != nil && strings.Contains(c.Model.PackageName, "compute") {
		return true
	}
	if c.Library != nil && strings.Contains(c.Library.Name, "compute") {
		return true
	}
	if field != nil && field.MessageType != nil && strings.Contains(field.MessageType.Package, "compute") {
		return true
	}
	if field != nil && field.Parent != nil && strings.Contains(field.Parent.Package, "compute") {
		return true
	}
	return false
}
