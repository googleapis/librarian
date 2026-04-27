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
	"github.com/googleapis/librarian/internal/sidekick/gcloud/provider"
	"github.com/iancoleman/strcase"
)

// argumentBuilder encapsulates the state required to generate the set of
// arguments for a gcloud command.
type argumentBuilder struct {
	method    *api.Method
	overrides *provider.Config
	model     *api.API
	service   *api.Service
	field     *api.Field
	apiField  string
}

// newArgumentBuilder constructs a new argumentBuilder.
func newArgumentBuilder(method *api.Method, overrides *provider.Config, model *api.API, service *api.Service, field *api.Field, apiField string) *argumentBuilder {
	return &argumentBuilder{
		method:    method,
		overrides: overrides,
		model:     model,
		service:   service,
		field:     field,
		apiField:  apiField,
	}
}

// build creates a single command-line argument (a `Argument` struct) from the builder's state.
// It returns nil if the field should be ignored.
func (b *argumentBuilder) build() (*Argument, error) {
	if b.isIgnored() {
		return nil, nil
	}

	// TODO(https://github.com/googleapis/librarian/issues/3414): Abstract away casing logic in the model.
	arg := &Argument{
		ArgName:   strcase.ToKebab(b.field.Name),
		APIField:  b.apiField,
		Required:  b.field.DocumentAsRequired(),
		Repeated:  b.repeated(),
		Clearable: b.clearable(),
		HelpText:  b.helpText(),
	}

	if b.field.ResourceReference != nil {
		spec, err := b.resourceReferenceSpec()
		if err != nil {
			return nil, err
		}
		arg.ResourceSpec = spec
		arg.ResourceMethodParams = map[string]string{
			b.apiField: "{__relative_name__}",
		}
	} else if b.field.Map {
		arg.Spec = b.mapSpec()
	} else if b.field.EnumType != nil {
		arg.Choices = b.choices()
	} else {
		arg.Type = provider.GetGcloudType(b.field.Typez)
		if b.field.Typez == api.TypezBool {
			if provider.IsUpdate(b.method) {
				arg.Action = "store_true_false"
			} else {
				arg.Action = "store_true"
			}
		}
	}

	return arg, nil
}

func (b *argumentBuilder) isIgnored() bool {
	if b.field.Name == "update_mask" {
		return true
	}
	if provider.IsList(b.method) {
		switch b.field.Name {
		case "page_size", "page_token", "filter", "order_by":
			return true
		}
	}
	if slices.Contains(b.field.Behavior, api.FieldBehaviorOutputOnly) {
		return true
	}
	if provider.IsUpdate(b.method) && slices.Contains(b.field.Behavior, api.FieldBehaviorImmutable) {
		return true
	}
	return false
}

func (b *argumentBuilder) repeated() bool {
	return b.field.Repeated || b.field.Map
}

func (b *argumentBuilder) clearable() bool {
	return provider.IsUpdate(b.method) && b.repeated()
}

func (b *argumentBuilder) helpText() string {
	return provider.GetFieldHelpText(b.overrides, b.field)
}

func (b *argumentBuilder) choices() []Choice {
	var choices []Choice
	for _, v := range b.field.EnumType.Values {
		// Skip the default "UNSPECIFIED" value.
		if !strings.HasSuffix(v.Name, "_UNSPECIFIED") {
			choices = append(choices, Choice{
				ArgValue:  strcase.ToKebab(v.Name),
				EnumValue: v.Name,
				HelpText:  fmt.Sprintf("Value for the `%s` field.", strcase.ToKebab(v.Name)),
			})
		}
	}
	return choices
}

func (b *argumentBuilder) mapSpec() []ArgSpec {
	return []ArgSpec{{APIField: "key"}, {APIField: "value"}}
}

// BuildPrimaryResource creates the main positional resource argument for a command.
// This is the argument that represents the resource being acted upon (e.g., the instance name).
func (b *argumentBuilder) buildPrimaryResource(idField *api.Field) Argument {
	resource := provider.GetResourceForMethod(b.method, b.model)
	var segments []api.PathSegment
	// TODO(https://github.com/googleapis/librarian/issues/3415): Support multiple resource patterns and multitype resources.
	if resource != nil && len(resource.Patterns) > 0 {
		segments = resource.Patterns[0]
	}

	// Grab the parent if it is collection based method unless you have a resource id field.
	if provider.IsCollectionMethod(b.method) && idField == nil {
		segments = provider.GetParentFromSegments(segments)
	}

	// resourceName should always be GetSingularFromSegments.
	resourceName := provider.GetSingularFromSegments(segments)

	// Help text should be documentation of builder.field name.
	// However, if you have resource id, then you actually want resource.name field.
	fieldHelpText := b.field.Documentation
	if nameField := provider.FindNameField(resource); idField != nil && nameField != nil {
		fieldHelpText = nameField.Documentation
	}

	collectionPath := provider.GetCollectionPathFromSegments(segments)
	hostParts := strings.Split(b.service.DefaultHost, ".")
	shortServiceName := hostParts[0]

	param := Argument{
		HelpText:          provider.CleanDocumentation(fieldHelpText),
		IsPositional:      !provider.IsList(b.method),
		IsPrimaryResource: true,
		Required:          true,
		ResourceSpec: &ResourceSpec{
			Name:                  resourceName,
			PluralName:            provider.GetPluralFromSegments(segments),
			Collection:            fmt.Sprintf("%s.%s", shortServiceName, collectionPath),
			DisableAutoCompleters: provider.IsList(b.method),
			Attributes:            newAttributesFromSegments(segments),
		},
	}

	if idField != nil {
		param.RequestIDField = strcase.ToLowerCamel(idField.Name)
	}

	return param
}

// resourceReferenceSpec creates a ResourceSpec for a field that references
// another resource type (e.g., a `--network` flag).
func (b *argumentBuilder) resourceReferenceSpec() (*ResourceSpec, error) {
	for _, def := range b.model.ResourceDefinitions {
		if def.Type == b.field.ResourceReference.Type {
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
