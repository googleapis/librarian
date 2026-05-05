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
)

type oneOfAnnotation struct {
	// In Rust, `oneof` fields are fields inside a struct. These must be
	// `snake_case`. Possibly mangled with `r#` if the name is a Rust reserved
	// word.
	FieldName string
	// In Rust, each field gets a `set_{{FieldName}}` setter. These must be
	// `snake_case`, but are never mangled with a `r#` prefix.
	SetterName string
	// The `oneof` is represented by a Rust `enum`, these need to be `PascalCase`.
	EnumName string
	// The Rust `enum` may be in a deeply nested scope. This is a shortcut.
	QualifiedName string
	// The fully qualified name, relative to `codec.modulePath`. Typically this
	// is the `QualifiedName` with the `crate::model::` prefix removed.
	RelativeName string
	// The Rust `struct` that contains this oneof, fully qualified
	StructQualifiedName string
	// The fully qualified name for examples. For messages in external packages
	// this is basically `QualifiedName`. For messages in the current package
	// this includes `modelAnnotations.PackageName`.
	NameInExamples string
	// The unqualified oneof name may be the same as the unqualified name of the
	// containing type. If that happens we need to alias one of them.
	AliasInExamples string
	// This is AliasInExamples if there's one, EnumName otherwise.
	EnumNameInExamples string
	FieldType          string
	DocLines           []string
	// If set, this enum is only enabled when some features are enabled.
	FeatureGates   []string
	FeatureGatesOp string
}

// MultiFeatureGates returns true if there are multiple feature gates.
func (a *oneOfAnnotation) MultiFeatureGates() bool {
	return len(a.FeatureGates) > 1
}

// SingleFeatureGate returns true if there is a single feature gate.
func (a *oneOfAnnotation) SingleFeatureGate() bool {
	return len(a.FeatureGates) == 1
}

func (c *codec) annotateOneOf(oneof *api.OneOf, message *api.Message, model *api.API) (*oneOfAnnotation, error) {
	scope, err := c.messageScopeName(message, "", model.PackageName)
	if err != nil {
		return nil, err
	}
	enumName := c.OneOfEnumName(oneof)
	qualifiedName := fmt.Sprintf("%s::%s", scope, enumName)
	relativeEnumName := strings.TrimPrefix(qualifiedName, c.modulePath+"::")
	structQualifiedName, err := c.fullyQualifiedMessageName(message, model.PackageName)
	if err != nil {
		return nil, err
	}
	nameInExamples := c.nameInExamplesFromQualifiedName(qualifiedName, model)
	docLines, err := c.formatDocComments(oneof.Documentation, oneof.ID, model, message.Scopes())
	if err != nil {
		return nil, err
	}

	ann := &oneOfAnnotation{
		FieldName:           toSnake(oneof.Name),
		SetterName:          toSnakeNoMangling(oneof.Name),
		EnumName:            enumName,
		QualifiedName:       qualifiedName,
		RelativeName:        relativeEnumName,
		StructQualifiedName: structQualifiedName,
		NameInExamples:      nameInExamples,
		FieldType:           fmt.Sprintf("%s::%s", scope, enumName),
		DocLines:            docLines,
	}
	// Note that this is different from OneOf name-overrides
	// as those solve for fully qualified name clashes where a oneof
	// and a child message have the same name.
	// This is solving for unqualified name clashes that affect samples
	// because we show usings for all types involved.
	if ann.EnumName == message.Name {
		ann.AliasInExamples = fmt.Sprintf("%sOneOf", ann.EnumName)
	}
	if ann.AliasInExamples == "" {
		ann.EnumNameInExamples = ann.EnumName
	} else {
		ann.EnumNameInExamples = ann.AliasInExamples
	}

	oneof.Codec = ann
	return ann, nil
}
