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
	"github.com/iancoleman/strcase"
)

// ArgumentBuilder encapsulates the state required to generate a single
// argument for a gcloud command.
type ArgumentBuilder struct {
	method    *api.Method
	overrides *Config
	model     *api.API
	service   *api.Service
	field     *api.Field
	apiField  string
}

// NewArgumentBuilder constructs a new ArgumentBuilder.
func NewArgumentBuilder(method *api.Method, overrides *Config, model *api.API, service *api.Service, field *api.Field, apiField string) *ArgumentBuilder {
	return &ArgumentBuilder{
		method:    method,
		overrides: overrides,
		model:     model,
		service:   service,
		field:     field,
		apiField:  apiField,
	}
}

// isIgnored determines if a field should be excluded from the generated command arguments.
// These are fields that are either implicit in the command context or handled
// automatically by the gcloud framework.
func isIgnored(field *api.Field, method *api.Method) bool {
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
	if isList(method) {
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
	if isUpdate(method) && slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_IMMUTABLE) {
		return true
	}

	return false
}

// Build creates a single command-line argument (a `Argument` struct) from the builder's state.
func (b *ArgumentBuilder) Build() (Argument, error) {
	spec, params, err := b.resourceSpecAndParams()
	if err != nil {
		return Argument{}, err
	}

	// TODO(https://github.com/googleapis/librarian/issues/3414): Abstract away casing logic in the model.
	arg := Argument{
		ArgName:              strcase.ToKebab(b.field.Name),
		APIField:             b.apiField,
		Required:             b.field.DocumentAsRequired(),
		Repeated:             b.isRepeated(),
		Clearable:            b.isClearable(),
		HelpText:             b.helpText(),
		Type:                 b.argType(),
		Choices:              b.choices(),
		ResourceSpec:         spec,
		ResourceMethodParams: params,
		Spec:                 b.mapSpec(),
	}
	return arg, nil
}

func (b *ArgumentBuilder) isRepeated() bool {
	return b.field.Repeated || b.field.Map
}

func (b *ArgumentBuilder) isClearable() bool {
	return isUpdate(b.method) && b.isRepeated()
}

func (b *ArgumentBuilder) helpText() string {
	if rule := findFieldHelpTextRule(b.field, b.overrides); rule != nil {
		return rule.HelpText.Brief
	}
	// TODO(https://github.com/googleapis/librarian/issues/3033): improve default help text inference
	return fmt.Sprintf("Value for the `%s` field.", strcase.ToKebab(b.field.Name))
}

func (b *ArgumentBuilder) argType() string {
	if b.field.ResourceReference == nil && !b.field.Map && b.field.EnumType == nil {
		return getGcloudType(b.field.Typez)
	}
	return ""
}

func (b *ArgumentBuilder) choices() []Choice {
	if b.field.EnumType == nil {
		return nil
	}
	var choices []Choice
	for _, v := range b.field.EnumType.Values {
		// Skip the default "UNSPECIFIED" value.
		if !strings.HasSuffix(v.Name, "_UNSPECIFIED") {
			choices = append(choices, Choice{
				ArgValue:  strcase.ToKebab(v.Name),
				EnumValue: v.Name,
			})
		}
	}
	return choices
}

func (b *ArgumentBuilder) mapSpec() []ArgSpec {
	if b.field.Map {
		return []ArgSpec{{APIField: "key"}, {APIField: "value"}}
	}
	return nil
}

func (b *ArgumentBuilder) resourceSpecAndParams() (*ResourceSpec, map[string]string, error) {
	if b.field.ResourceReference == nil {
		return nil, nil, nil
	}
	spec, err := b.newResourceReferenceSpec()
	if err != nil {
		return nil, nil, err
	}
	return spec, map[string]string{b.apiField: "{__relative_name__}"}, nil
}

// BuildPrimaryResource creates the main positional resource argument for a command.
// This is the argument that represents the resource being acted upon (e.g., the instance name).
func (b *ArgumentBuilder) BuildPrimaryResource() Argument {
	resource := getResourceForMethod(b.method, b.model)
	var segments []api.PathSegment
	// TODO(https://github.com/googleapis/librarian/issues/3415): Support multiple resource patterns and multitype resources.
	if resource != nil && len(resource.Patterns) > 0 {
		segments = resource.Patterns[0]
	}

	// For List methods, the primary resource is the parent of the method's resource.
	if isList(b.method) {
		segments = getParentFromSegments(segments)
	}

	resourceName := strings.TrimSuffix(b.field.Name, "_id")
	if b.field.Name == "name" || isList(b.method) {
		resourceName = getSingularFromSegments(segments)
	}

	var helpText string
	switch {
	case isCreate(b.method):
		helpText = fmt.Sprintf("The %s to create.", resourceName)
	case isList(b.method):
		helpText = fmt.Sprintf("The project and location for which to retrieve %s information.", getPluralFromSegments(segments))
	default:
		helpText = fmt.Sprintf("The %s to operate on.", resourceName)
	}

	collectionPath := getCollectionPathFromSegments(segments)
	hostParts := strings.Split(b.service.DefaultHost, ".")
	shortServiceName := hostParts[0]

	param := Argument{
		HelpText:          helpText,
		IsPositional:      !isList(b.method),
		IsPrimaryResource: true,
		Required:          true,
		ResourceSpec: &ResourceSpec{
			Name:                  resourceName,
			PluralName:            getPluralFromSegments(segments),
			Collection:            fmt.Sprintf("%s.%s", shortServiceName, collectionPath),
			DisableAutoCompleters: false,
			Attributes:            newAttributesFromSegments(segments),
		},
	}

	if isCreate(b.method) {
		param.RequestIDField = strcase.ToLowerCamel(b.field.Name)
	}

	return param
}

// newResourceReferenceSpec creates a ResourceSpec for a field that references
// another resource type (e.g., a `--network` flag).
func (b *ArgumentBuilder) newResourceReferenceSpec() (*ResourceSpec, error) {
	for _, def := range b.model.ResourceDefinitions {
		if def.Type == b.field.ResourceReference.Type {
			if len(def.Patterns) == 0 {
				return nil, fmt.Errorf("resource definition for %q has no patterns", def.Type)
			}
			// TODO(https://github.com/googleapis/librarian/issues/3415): Support multiple resource patterns and multitype resources.
			segments := def.Patterns[0]

			pluralName := def.Plural
			if pluralName == "" {
				pluralName = getPluralFromSegments(segments)
			}

			name := getSingularFromSegments(segments)

			hostParts := strings.Split(b.service.DefaultHost, ".")
			shortServiceName := hostParts[0]
			baseCollectionPath := getCollectionPathFromSegments(segments)
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
	return nil, fmt.Errorf("resource definition not found for type %q", b.field.ResourceReference.Type)
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
