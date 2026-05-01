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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type messageAnnotation struct {
	Name       string
	ModuleName string
	// The fully qualified name, including the `codec.modulePath` prefix. For
	// messages in external packages this includes the package name.
	QualifiedName string
	// The fully qualified name, relative to `codec.modulePath`. Typically this
	// is the `QualifiedName` with the `crate::model::` prefix removed.
	RelativeName string
	// The fully qualified name for examples. For messages in external packages
	// this is basically `QualifiedName`. For messages in the current package
	// this includes `modelAnnotations.PackageName`.
	NameInExamples string
	// The package name mapped to Rust modules. That is, `google.service.v1`
	// becomes `google::service::v1`.
	PackageModuleName string
	// The FQN is the source specification
	SourceFQN      string
	DocLines       []string
	HasNestedTypes bool
	// All the fields except OneOfs.
	BasicFields []*api.Field
	// If set, this message is only enabled when some features are enabled.
	FeatureGates   []string
	FeatureGatesOp string
	// If true, this message's visibility should only be `pub(crate)`
	Internal bool
}

// MultiFeatureGates returns true if there are multiple feature gates.
func (a *messageAnnotation) MultiFeatureGates() bool {
	return len(a.FeatureGates) > 1
}

// SingleFeatureGate returns true if there is a single feature gate.
func (a *messageAnnotation) SingleFeatureGate() bool {
	return len(a.FeatureGates) == 1
}

// annotateMessage annotates the message with basic or full annotations.
// When fully annotating a message, its fields, its nested messages, and its nested enums
// are also annotated.
// Basic annotations are useful for annotating external messages with information used in samples.
func (c *codec) annotateMessage(m *api.Message, model *api.API, full bool) error {
	qualifiedName, err := c.fullyQualifiedMessageName(m, model.PackageName)
	if err != nil {
		return err
	}
	relativeName := strings.TrimPrefix(qualifiedName, c.modulePath+"::")
	nameInExamples := c.nameInExamplesFromQualifiedName(qualifiedName, model)
	annotations := &messageAnnotation{
		Name:              toPascal(m.Name),
		ModuleName:        toSnake(m.Name),
		QualifiedName:     qualifiedName,
		RelativeName:      relativeName,
		NameInExamples:    nameInExamples,
		PackageModuleName: packageToModuleName(m.Package),
		SourceFQN:         strings.TrimPrefix(m.ID, "."),
	}
	m.Codec = annotations

	if !full {
		// We have basic annotations, we are done.
		return nil
	}

	for _, f := range m.Fields {
		if _, err := c.annotateField(f, m, model); err != nil {
			return err
		}
	}
	for _, o := range m.OneOfs {
		if _, err := c.annotateOneOf(o, m, model); err != nil {
			return err
		}
	}
	for _, e := range m.Enums {
		if err := c.annotateEnum(e, model, true); err != nil {
			return err
		}
	}
	for _, child := range m.Messages {
		if err := c.annotateMessage(child, model, true); err != nil {
			return err
		}
	}
	basicFields := language.FilterSlice(m.Fields, func(f *api.Field) bool {
		return !f.IsOneOf
	})

	docLines, err := c.formatDocComments(m.Documentation, m.ID, model, m.Scopes())
	if err != nil {
		return err
	}
	annotations.DocLines = docLines
	annotations.HasNestedTypes = language.HasNestedTypes(m)
	annotations.BasicFields = basicFields
	annotations.Internal = slices.Contains(c.internalTypes, m.ID)
	return nil
}
