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
	"log/slog"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type enumAnnotation struct {
	Name        string
	ModuleName  string
	DocLines    []string
	UniqueNames []*api.EnumValue
	// The fully qualified name, including the `codec.modulePath`
	// (typically `crate::model::`) prefix. For external enums this is prefixed
	// by the external crate name.
	QualifiedName string
	// The fully qualified name, relative to `codec.modulePath`. Typically this
	// is the `QualifiedName` with the `crate::model::` prefix removed.
	RelativeName string
	// The fully qualified name for examples. For messages in external packages
	// this is basically `QualifiedName`. For messages in the current package
	// this includes `modelAnnotations.PackageName`.
	NameInExamples string
	// There's a mismatch between the sidekick model representation of wkt::NullValue
	// and the representation in Rust. We us this for sample generation.
	IsWktNullValue bool
	// If set, this enum is only enabled when some features are enabled
	FeatureGates   []string
	FeatureGatesOp string
}

// MultiFeatureGates returns true if there are multiple feature gates.
func (a *enumAnnotation) MultiFeatureGates() bool {
	return len(a.FeatureGates) > 1
}

// SingleFeatureGate returns true if there is a single feature gate.
func (a *enumAnnotation) SingleFeatureGate() bool {
	return len(a.FeatureGates) == 1
}

func (c *codec) annotateEnum(e *api.Enum, model *api.API, full bool) error {
	for _, ev := range e.Values {
		if err := c.annotateEnumValue(ev, model, full); err != nil {
			return err
		}
	}

	qualifiedName, err := c.fullyQualifiedEnumName(e, model.PackageName)
	if err != nil {
		return err
	}
	relativeName := strings.TrimPrefix(qualifiedName, c.modulePath+"::")
	nameInExamples := c.nameInExamplesFromQualifiedName(qualifiedName, model)

	// For BigQuery (and so far only BigQuery), the enum values conflict when
	// converted to the Rust style [1]. Basically, there are several enum values
	// in this service that differ only in case, such as `FULL` vs. `full`.
	//
	// We create a list with the duplicates removed to avoid conflicts in the
	// generated code.
	//
	// [1]: Both Rust and Protobuf use `SCREAMING_SNAKE_CASE` for these, but
	//      some services do not follow the Protobuf convention.
	seen := map[string]*api.EnumValue{}
	var unique []*api.EnumValue
	for _, ev := range e.Values {
		name := enumValueVariantName(ev)
		if existing, ok := seen[name]; ok {
			if existing.Number != ev.Number {
				slog.Warn("conflicting names for enum values", "enum.ID", e.ID)
			}
		} else {
			unique = append(unique, ev)
			seen[name] = ev
		}
	}

	annotations := &enumAnnotation{
		Name:           enumName(e),
		ModuleName:     toSnake(enumName(e)),
		QualifiedName:  qualifiedName,
		RelativeName:   relativeName,
		NameInExamples: nameInExamples,
		IsWktNullValue: nameInExamples == "wkt::NullValue",
	}
	e.Codec = annotations

	if !full {
		// We have basic annotations, we are done.
		return nil
	}

	lines, err := c.formatDocComments(e.Documentation, e.ID, model, e.Scopes())
	if err != nil {
		return err
	}
	annotations.DocLines = lines
	annotations.UniqueNames = unique
	return nil
}
