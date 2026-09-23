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

type enumAnnotations struct {
	Model             *modelAnnotations
	Name              string
	DocLines          []string
	DefaultCaseName   string
	UnknownIntName    string
	UnknownStringName string
	UseStringEnums    bool

	// GatedBy is the list of package traits that enables this enum.
	//
	// Empty unless the package is configured with `per_service_traits` enabled.
	GatedBy []string

	// GatedOp is the operation (&& or ||) to combine all the `GatedBy` traits.
	//
	// For most enums, this is " || ", as enums are enabled when any service
	// that needs them is enabled. Messages that do not map to any service use
	// " && ".
	GatedOp string

	// The target import module containing raw stubs.
	ModulePath string

	// The full name of the raw proto enum under the private module target.
	ProtoTypeName string

	// FullyQualifiedName is the parent-qualified Swift name (e.g. `FindingSummary.SummaryDetails.ResourceType` or `Color`).
	//
	// In Swift, extensions cannot be nested inside structs or other extensions. For nested enums,
	// top-level extension declarations use FullyQualifiedName: `extension FindingSummary.SummaryDetails.ResourceType { ... }`.
	FullyQualifiedName string

	// DiagnoseValues is true when the initializers that assign enum cases need
	// `@diagnose` to suppress a deprecation warning.
	//
	// Only assignment warns. The members that merely match a case in a
	// `switch`, such as the `intValue` and `stringValue` computed properties
	// and `encode(to:)`, need nothing.
	//
	// This is a slight over-approximation for `init(intValue:)` and the proto
	// converting `init(proto:)`, which enumerate `UniqueNumberValues` while
	// the flag scans every value: a deprecated alias sharing a number with a
	// live value guards them unnecessarily. `init(stringValue:)` does
	// enumerate every value, so one flag cannot be exact for all three.
	DiagnoseValues bool

	// DiagnoseDefault is true when `init()`, which assigns the default case,
	// needs `@diagnose` to suppress a deprecation warning.
	DiagnoseDefault bool
}

// HasDocLines returns true if this enum has proto documentation lines.
func (ann *enumAnnotations) HasDocLines() bool {
	return len(ann.DocLines) > 0
}

// IsGated returns true if this message is gated by some package traits.
func (ann *enumAnnotations) IsGated() bool {
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
func (ann *enumAnnotations) GateExpression() string {
	return strings.Join(ann.GatedBy, ann.GatedOp)
}

func (c *codec) annotateEnum(enum *api.Enum, model *modelAnnotations) error {
	// We need to find non-clashing names for the `unknownIntValue` and
	// `unknownStringvalue` cases. In practice, no enum uses those names, but
	// if one ever this, this will add enough trailing `_` to make the name
	// unique.
	type u struct{}
	caseNames := make(map[string]u)
	uniqueCaseName := func(seed string) string {
		_, ok := caseNames[seed]
		for ok {
			seed = seed + "_"
			_, ok = caseNames[seed]
		}
		return seed
	}

	existing := map[int32]*enumValueAnnotations{}
	var defaultCaseName string
	var defaultValue *api.EnumValue
	for _, ev := range enum.UniqueNumberValues {
		if err := c.annotateUniqueEnumValue(ev); err != nil {
			return err
		}
		ann := ev.Codec.(*enumValueAnnotations)
		if ann == nil {
			return fmt.Errorf("unknown annotation format for enum value: %s", ev.ID)
		}
		caseNames[ann.CaseName] = u{}
		existing[ev.Number] = ann
		if ev.Number == 0 {
			defaultCaseName = ann.CaseName
			defaultValue = ev
		}
	}
	// Fallback to first case if no 0 value found (should not happen in proto3)
	if defaultCaseName == "" {
		if len(enum.UniqueNumberValues) != 0 {
			ann := enum.UniqueNumberValues[0].Codec.(*enumValueAnnotations)
			if ann == nil {
				panic("mismatched annotation, previously checked, must be a bug")
			}
			defaultCaseName = ann.CaseName
			defaultValue = enum.UniqueNumberValues[0]
		} else {
			return fmt.Errorf("cannot determine a default value for enum: %s", enum.ID)
		}
	}
	for _, ev := range enum.Values {
		if err := c.annotateEnumValue(ev, existing); err != nil {
			return err
		}
		existing[ev.Number] = ev.Codec.(*enumValueAnnotations)
	}

	docLines, err := c.formatDocumentation(enum.Documentation, enum.Scopes())
	if err != nil {
		return err
	}
	extensionTypeName, err := c.enumTypeName(enum)
	if err != nil {
		return err
	}
	annotations := &enumAnnotations{
		Model:              model,
		Name:               pascalCase(enum.Name),
		FullyQualifiedName: extensionTypeName,
		DocLines:           docLines,
		DefaultCaseName:    defaultCaseName,
		UnknownIntName:     uniqueCaseName("unknownIntValue"),
		UnknownStringName:  uniqueCaseName("unknownStringValue"),
		ModulePath:         c.ModulePath,
		ProtoTypeName:      c.protoEnumTypeName(enum),
		UseStringEnums:     c.UseStringEnums,
	}
	if !enum.Deprecated {
		annotations.DiagnoseDefault = defaultValue.Deprecated
		annotations.DiagnoseValues = slices.ContainsFunc(enum.Values, func(ev *api.EnumValue) bool {
			return ev.Deprecated
		})
	}

	enum.Codec = annotations
	return nil
}
