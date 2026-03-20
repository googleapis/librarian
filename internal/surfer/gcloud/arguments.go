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

package gcloud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/provider"
)

// isIgnored determines if a field should be excluded from the generated command arguments.
// These are fields that are either implicit in the command context or handled
// automatically by the gcloud framework.
func (b *commandBuilder) isIgnored(field *api.Field) bool {
	// The "parent" field is usually implicit in the command context (handled by the primary resource or hierarchy).
	if field.Name == "parent" {
		return true
	}

	// The "name" field is usually the primary resource identifier, handled separately.
	if field.Name == "name" {
		return true
	}

	// The "update_mask" field is handled automatically by the gcloud framework.
	if field.Name == "update_mask" {
		return true
	}

	// For List methods, standard pagination/filtering arguments are handled by gcloud.
	if b.method.Type() == provider.MethodTypeList {
		switch field.Name {
		case "page_size", "page_token", "filter", "order_by":
			return true
		}
	}

	// Output-only fields are read-only and should not be settable via CLI flags.
	if slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_OUTPUT_ONLY) {
		return true
	}

	// For Update commands, fields marked as IMMUTABLE cannot be changed and should be hidden.
	if (b.method.Type() == provider.MethodTypeUpdate) && slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_IMMUTABLE) {
		return true
	}

	return false
}

// addFlattenedParams recursively processes a field and its sub-fields to generate
// command-line flags. It uses a dispatch pattern to classify each field:
//  1. Primary resource arguments (positional resource identifiers).
//  2. Ignored fields (implicit or framework-handled).
//  3. Nested messages (flattened into top-level flags).
//  4. Standard arguments (scalars, maps, enums, resource references).
//
// TODO(https://github.com/googleapis/librarian/issues/3413): Improve error
// handling strategy (Error vs Skip) and messaging.
func (b *commandBuilder) flattenField(field *api.Field, prefix string) ([]Param, error) {
	// Primary resource args are checked first because fields like "parent"
	// and "name" are primary resources in certain method types (e.g., List
	// and Get/Delete/Update respectively) and must not be ignored.
	if b.method.IsPrimaryResource(field) {
		return []Param{b.newPrimaryResourceParam(field)}, nil
	}

	if b.isIgnored(field) {
		return nil, nil
	}

	// Nested messages are flattened into top-level flags.
	// TODO(https://github.com/googleapis/librarian/issues/3287): Support arg_groups.
	if field.MessageType != nil && !field.Map {
		var params []Param
		for _, f := range field.MessageType.Fields {
			nestedParams, err := b.flattenField(f, fmt.Sprintf("%s.%s", prefix, f.JSONName))
			if err != nil {
				return nil, err
			}
			params = append(params, nestedParams...)
		}
		return params, nil
	}

	// Standard arguments: scalars, maps, enums, and resource references.
	param, err := b.newParam(field, prefix)
	if err != nil {
		return nil, err
	}
	return []Param{param}, nil
}

// newParam creates a single command-line argument (a `Param` struct) from a proto field.
func (b *commandBuilder) newParam(field *api.Field, apiField string) (Param, error) {
	// TODO(https://github.com/googleapis/librarian/issues/3414): Abstract away casing logic in the model.
	param := Param{
		ArgName:  field.Name,
		APIField: apiField,
		Required: field.DocumentAsRequired(),
		Repeated: field.Repeated,
	}

	if field.ResourceReference != nil {
		spec, err := b.newResourceReferenceSpec(field)
		if err != nil {
			return Param{}, err
		}
		param.ResourceSpec = spec
		param.ResourceMethodParams = map[string]string{
			apiField: "{__relative_name__}",
		}
	} else if field.Map {
		param.Repeated = true
		param.Spec = []ArgSpec{
			{APIField: "key"},
			{APIField: "value"},
		}
	} else if field.EnumType != nil {
		for _, v := range field.EnumType.Values {
			// Skip the default "UNSPECIFIED" value.
			if strings.HasSuffix(v.Name, "_UNSPECIFIED") {
				continue
			}
			param.Choices = append(param.Choices, Choice{
				ArgValue:  v.Name,
				EnumValue: v.Name,
			})
		}
	} else {
		param.Type = provider.GetGcloudType(field.Typez)
	}

	if (b.method.Type() == provider.MethodTypeUpdate) && param.Repeated {
		param.Clearable = true
	}

	if rule := b.findFieldHelpTextRule(field); rule != nil {
		param.HelpText = rule.HelpText.Brief
	} else {
		// TODO(https://github.com/googleapis/librarian/issues/3033): improve default help text inference
		param.HelpText = fmt.Sprintf("Value for the `%s` field.", field.Name)
	}
	return param, nil
}

// newPrimaryResourceParam creates the main positional resource argument for a command.
// This is the argument that represents the resource being acted upon (e.g., the instance name).
func (b *commandBuilder) newPrimaryResourceParam(field *api.Field) Param {
	resource := b.method.GetResource(b.model)
	var segments []api.PathSegment
	// TODO(https://github.com/googleapis/librarian/issues/3415): Support multiple resource patterns and multitype resources.
	if resource != nil && len(resource.Patterns) > 0 {
		segments = resource.Patterns[0]
	}

	// For List methods, the primary resource is the parent of the method's resource.
	if b.method.Type() == provider.MethodTypeList {
		segments = provider.GetParentFromSegments(segments)
	}

	resourceName := strings.TrimSuffix(field.Name, "_id")
	if field.Name == "name" || (b.method.Type() == provider.MethodTypeList) {
		resourceName = provider.GetSingularFromSegments(segments)
	}

	var helpText string
	switch {
	case (b.method.Type() == provider.MethodTypeCreate):
		helpText = fmt.Sprintf("The %s to create.", resourceName)
	case (b.method.Type() == provider.MethodTypeList):
		helpText = fmt.Sprintf("The project and location for which to retrieve %s information.", provider.GetPluralFromSegments(segments))
	default:
		helpText = fmt.Sprintf("The %s to operate on.", resourceName)
	}

	collectionPath := provider.GetCollectionPathFromSegments(segments)
	hostParts := strings.Split(b.service.DefaultHost, ".")
	shortServiceName := hostParts[0]

	param := Param{
		HelpText:          helpText,
		IsPositional:      b.method.Type() != provider.MethodTypeList,
		IsPrimaryResource: true,
		Required:          true,
		ResourceSpec: &ResourceSpec{
			Name:                  resourceName,
			PluralName:            provider.GetPluralFromSegments(segments),
			Collection:            fmt.Sprintf("%s.%s", shortServiceName, collectionPath),
			DisableAutoCompleters: false,
			Attributes:            newAttributesFromSegments(segments),
		},
	}

	if b.method.Type() == provider.MethodTypeCreate {
		param.RequestIDField = field.Name
	}

	return param
}

// newResourceReferenceSpec creates a ResourceSpec for a field that references
// another resource type (e.g., a `--network` flag).
func (b *commandBuilder) newResourceReferenceSpec(field *api.Field) (*ResourceSpec, error) {
	for _, def := range b.model.ResourceDefinitions {
		if def.Type == field.ResourceReference.Type {
			if len(def.Patterns) == 0 {
				return nil, fmt.Errorf("resource definition for %q has no patterns", def.Type)
			}
			// TODO(https://github.com/googleapis/librarian/issues/3415): Support multiple resource patterns and multitype resources.
			segments := def.Patterns[0]

			pluralName := def.Plural
			if pluralName == "" {
				pluralName = provider.GetPluralFromSegments(segments)
			}

			name := provider.GetSingularFromSegments(segments)

			hostParts := strings.Split(b.service.DefaultHost, ".")
			shortServiceName := hostParts[0]
			baseCollectionPath := provider.GetCollectionPathFromSegments(segments)
			fullCollectionPath := fmt.Sprintf("%s.%s", shortServiceName, baseCollectionPath)

			return &ResourceSpec{
				Name:       name,
				PluralName: pluralName,
				Collection: fullCollectionPath,
				// TODO(https://github.com/googleapis/librarian/issues/3416): Investigate and enable auto-completers for referenced resources.
				DisableAutoCompleters: true,
				Attributes:            newAttributesFromSegments(segments),
			}, nil
		}
	}
	return nil, fmt.Errorf("resource definition not found for type %q", field.ResourceReference.Type)
}

// newAttributesFromSegments parses a structured resource pattern and extracts the attributes
// that make up the resource's name.
func newAttributesFromSegments(segments []api.PathSegment) []Attribute {
	var attributes []Attribute

	for i, part := range segments {
		if part.Variable == nil {
			continue
		}

		if len(part.Variable.FieldPath) == 0 {
			continue
		}
		name := part.Variable.FieldPath[len(part.Variable.FieldPath)-1]
		var parameterName string

		// The `parameter_name` is derived from the preceding literal segment
		// (e.g., "projects" -> "projectsId"). This is a gcloud convention.
		if i > 0 && segments[i-1].Literal != nil {
			parameterName = *segments[i-1].Literal + "Id"
		} else {
			parameterName = name + "sId"
		}

		attr := Attribute{
			AttributeName: name,
			ParameterName: parameterName,
			Help:          fmt.Sprintf("The %s id of the {resource} resource.", name),
		}

		// Standard gcloud property fallback so users don't need to specify --project
		// if it's already configured.
		if name == "project" {
			attr.Property = "core/project"
		}
		attributes = append(attributes, attr)
	}
	return attributes
}
