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

// commandBuilder encapsulates the state required to build a gcloud command
// definition from an API method.
type commandBuilder struct {
	method    *api.Method
	overrides *provider.Config
	model     *api.API
	service   *api.Service
}

// newCommandBuilder constructs a new commandBuilder for a specific method execution.
func newCommandBuilder(method *api.Method, overrides *provider.Config, model *api.API, service *api.Service) *commandBuilder {
	return &commandBuilder{
		method:    method,
		overrides: overrides,
		model:     model,
		service:   service,
	}
}

// Build constructs a single gcloud command definition from an API method.
// This function assembles all the necessary pieces: help text, arguments,
// request details, and async configuration. It takes no parameters, relying
// on the commandBuilder's state, and returns the constructed Command and
// any error encountered during assembly.
func (b *commandBuilder) build() (*Command, error) {
	args, err := b.newArguments()
	if err != nil {
		return nil, err
	}

	useUpdateMask := b.updateMask()

	return &Command{
		Name:                 b.name(),
		Hidden:               b.hidden(),
		HelpText:             b.helpText(),
		APIVersion:           provider.APIVersion(b.overrides),
		Collection:           b.collectionPath(false),
		Method:               b.requestMethod(),
		Arguments:            args,
		ResponseIDField:      b.responseIDField(),
		OutputFormat:         b.outputFormat(),
		ReadModifyUpdate:     provider.IsUpdate(b.method),
		StarUpdateMask:       useUpdateMask,
		DisableAutoFieldMask: useUpdateMask,
		Async:                b.async(),
	}, nil
}

func (b *commandBuilder) name() string {
	name, err := provider.GetCommandName(b.method)
	if err != nil {
		return ""
	}
	return name
}

func (b *commandBuilder) responseIDField() string {
	if provider.IsList(b.method) {
		// List commands should have an id_field to enable the --uri flag.
		return "name"
	}
	return ""
}

// outputFormat generates the string output format for List commands.
// TODO(https://github.com/googleapis/librarian/issues/5231): Make this default configurable by gcloud.yaml.
// Use tableFormat if specified.
func (b *commandBuilder) outputFormat() string {
	return ""
}

// async creates the `Async` part of the command definition for long-running operations.
func (b *commandBuilder) async() *Async {
	if b.method.OperationInfo == nil {
		return nil
	}

	async := &Async{
		Collection: b.collectionPath(true),
	}

	// Extract the resource result if the LRO response type matches the
	// method's resource type.
	resource := provider.GetResourceForMethod(b.method, b.model)
	if resource == nil {
		return async
	}

	// Heuristic: Check if response type ID (e.g. ".google.cloud.parallelstore.v1.Instance")
	// matches the resource singular name or type.
	responseTypeID := b.method.OperationInfo.ResponseTypeID
	// Extract short name from FQN (last element after dot)
	responseTypeName := responseTypeID
	if idx := strings.LastIndex(responseTypeID, "."); idx != -1 {
		responseTypeName = responseTypeID[idx+1:]
	}

	singular := provider.GetSingularResourceNameForMethod(b.method, b.model)
	if strings.EqualFold(responseTypeName, singular) || strings.HasSuffix(resource.Type, "/"+responseTypeName) {
		async.ExtractResourceResult = true
	}

	return async
}

func (b *commandBuilder) hidden() bool {
	if b.overrides != nil && len(b.overrides.APIs) > 0 {
		return b.overrides.APIs[0].RootIsHidden
	}
	// Default to hidden if no API overrides are provided.
	return true
}

func (b *commandBuilder) helpText() HelpText {
	h := provider.GetMethodHelpText(b.overrides, b.method, b.model)
	return HelpText{
		Brief:       h.Brief,
		Description: h.Description,
		Examples:    h.Examples,
	}
}

// requestMethod determines the API method name for the command execution.
func (b *commandBuilder) requestMethod() string {
	// For custom methods (AIP-136), the `method` field in the request configuration
	// MUST match the custom verb defined in the HTTP binding (e.g., ":exportData" -> "exportData").
	if b.method.PathInfo != nil && len(b.method.PathInfo.Bindings) > 0 && b.method.PathInfo.Bindings[0].PathTemplate.Verb != nil {
		return *b.method.PathInfo.Bindings[0].PathTemplate.Verb
	} else if !provider.IsStandardMethod(b.method) {
		commandName, _ := provider.GetCommandName(b.method)
		// GetCommandName returns snake_case (e.g. "export_data"), but request.method expects camelCase (e.g. "exportData").
		return strcase.ToLowerCamel(commandName)
	}

	return ""
}

type fieldWithPrefix struct {
	field  *api.Field
	prefix string
}

type classifiedFields struct {
	primaryField    *fieldWithPrefix
	resourceIdField *fieldWithPrefix
	other           []fieldWithPrefix
}

// newArguments generates the set of arguments for a command by parsing the
// fields of the method's request message.
func (b *commandBuilder) newArguments() ([]Argument, error) {
	var args []Argument
	if b.method.InputType == nil {
		return args, nil
	}

	cf, err := b.categorizeFields()
	if err != nil {
		return nil, err
	}

	if cf.primaryField != nil {
		var idField *api.Field
		if cf.resourceIdField != nil {
			idField = cf.resourceIdField.field
		}
		arg := newArgumentBuilder(b.method, b.overrides, b.model, b.service, cf.primaryField.field, cf.primaryField.prefix).buildPrimaryResource(idField)
		args = append(args, arg)
	}

	for _, fwp := range cf.other {
		arg, err := newArgumentBuilder(b.method, b.overrides, b.model, b.service, fwp.field, fwp.prefix).build()
		if err != nil {
			return nil, err
		}
		if arg == nil {
			continue
		}
		args = append(args, *arg)
	}

	return args, nil
}

// categorizeFields gathers fields from the method input type, expanding the body field
// if present, and separates them into at most one primary resource field, at most one
// resource ID field, and other fields. Returns an error if multiple are found.
func (b *commandBuilder) categorizeFields() (classifiedFields, error) {
	var cf classifiedFields
	bodyFieldPath := ""
	if b.method.PathInfo != nil {
		bodyFieldPath = b.method.PathInfo.BodyFieldPath
	}

	var collected []fieldWithPrefix
	for _, field := range b.method.InputType.Fields {
		isExpandableMessage := field.MessageType != nil && !field.Map
		isBodyField := bodyFieldPath != "" && (bodyFieldPath == field.Name || bodyFieldPath == "*")

		if isExpandableMessage && isBodyField {
			for _, f := range field.MessageType.Fields {
				collected = append(collected, fieldWithPrefix{
					field:  f,
					prefix: fmt.Sprintf("%s.%s", field.JSONName, f.JSONName),
				})
			}
			continue
		}

		collected = append(collected, fieldWithPrefix{
			field:  field,
			prefix: field.JSONName,
		})
	}

	for _, fwp := range collected {
		switch {
		case provider.IsPrimaryResourceField(fwp.field, b.method):
			if cf.primaryField != nil {
				return cf, fmt.Errorf("method %q has multiple primary resource fields: %q and %q", b.method.Name, cf.primaryField.field.Name, fwp.field.Name)
			}
			cf.primaryField = &fieldWithPrefix{field: fwp.field, prefix: fwp.prefix}
		case provider.IsResourceIdField(fwp.field, b.method, b.model):
			if cf.resourceIdField != nil {
				return cf, fmt.Errorf("method %q has multiple resource ID fields: %q and %q", b.method.Name, cf.resourceIdField.field.Name, fwp.field.Name)
			}
			cf.resourceIdField = &fieldWithPrefix{field: fwp.field, prefix: fwp.prefix}
		case provider.IsCreate(b.method) && fwp.field.Name == "name":
			// Ignore name field in Create methods as it's redundant with resource_id
		default:
			cf.other = append(cf.other, fwp)
		}
	}

	return cf, nil
}

// collectionPath constructs the gcloud collection path(s) for a request or async operation.
// It follows AIP-127 and AIP-132 by extracting the collection structure directly from
// the method's HTTP annotation (PathInfo).
func (b *commandBuilder) collectionPath(isAsync bool) []string {
	var collections []string
	hostParts := strings.Split(b.service.DefaultHost, ".")
	shortServiceName := hostParts[0]

	// Iterate over all bindings (primary + additional) to support multitype resources (AIP-127).
	for _, binding := range b.method.PathInfo.Bindings {
		if binding.PathTemplate == nil {
			continue
		}

		basePath := provider.ExtractPathFromSegments(binding.PathTemplate.Segments)

		if basePath == "" {
			continue
		}

		if isAsync {
			// For Async operations (AIP-151), the operations resource usually resides in the
			// parent collection of the primary resource. We replace the last segment (the resource collection)
			// with "operations".
			// Example: projects.locations.instances -> projects.locations.operations
			if idx := strings.LastIndex(basePath, "."); idx != -1 {
				basePath = basePath[:idx] + ".operations"
			} else {
				basePath = "operations"
			}
		}

		fullPath := fmt.Sprintf("%s.%s", shortServiceName, basePath)
		collections = append(collections, fullPath)
	}

	// Remove duplicates if any.
	slices.Sort(collections)
	return slices.Compact(collections)
}

func (b *commandBuilder) updateMask() bool {
	if !provider.IsUpdate(b.method) || b.method.InputType == nil {
		return false
	}
	for _, f := range b.method.InputType.Fields {
		if f.Name == "update_mask" {
			return true
		}
	}
	return false
}

// tableFormat generates a gcloud table format string from a message definition.
func tableFormat(message *api.Message) string {
	var sb strings.Builder
	first := true

	for _, f := range message.Fields {
		// Sanitize field name to prevent DSL injection.
		if !provider.IsSafeName(f.JSONName) {
			continue
		}

		// Include scalars and enums.
		isScalar := f.Typez == api.TypezString ||
			f.Typez == api.TypezInt32 || f.Typez == api.TypezInt64 ||
			f.Typez == api.TypezBool || f.Typez == api.TypezEnum ||
			f.Typez == api.TypezDouble || f.Typez == api.TypezFloat

		if isScalar {
			if !first {
				sb.WriteString(",\n")
			}
			if f.Repeated {
				// Format repeated scalars with .join(',').
				sb.WriteString(f.JSONName)
				sb.WriteString(".join(',')")
			} else {
				sb.WriteString(f.JSONName)
			}
			first = false
			continue
		}

		// Include timestamps (usually messages like google.protobuf.Timestamp).
		if f.MessageType != nil && strings.HasSuffix(f.TypezID, ".Timestamp") {
			if !first {
				sb.WriteString(",\n")
			}
			sb.WriteString(f.JSONName)
			first = false
		}
	}

	if sb.Len() == 0 {
		return ""
	}
	return fmt.Sprintf("table(\n%s)", sb.String())
}
