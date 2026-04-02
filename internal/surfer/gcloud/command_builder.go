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
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
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

	return &Command{
		Hidden:           b.hidden(),
		HelpText:         b.helpText(),
		ReleaseTracks:    b.releaseTracks(),
		APIVersion:       provider.APIVersion(b.overrides),
		Collection:       b.collectionPath(false),
		Method:           b.requestMethod(),
		Arguments:        args,
		ResponseIDField:  b.responseIDField(),
		OutputFormat:     b.outputFormat(),
		ReadModifyUpdate: provider.IsUpdate(b.method),
		Async:            b.async(),
	}, nil
}

func (b *commandBuilder) responseIDField() string {
	if provider.IsList(b.method) {
		// List commands should have an id_field to enable the --uri flag.
		return "name"
	}
	return ""
}

// outputFormat generates the string output format for List commands.
func (b *commandBuilder) outputFormat() string {
	if !provider.IsList(b.method) {
		return ""
	}

	resourceMsg := provider.FindResourceMessage(b.method.OutputType)
	if resourceMsg == nil {
		return ""
	}

	return tableFormat(resourceMsg)
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
	if len(b.overrides.APIs) > 0 {
		return b.overrides.APIs[0].RootIsHidden
	}
	// Default to hidden if no API overrides are provided.
	return true
}

func (b *commandBuilder) helpText() HelpText {
	rule := provider.FindHelpTextRule(b.overrides, strings.TrimPrefix(b.method.ID, "."))
	if rule != nil {
		return HelpText{
			Brief:       rule.HelpText.Brief,
			Description: rule.HelpText.Description,
			Examples:    strings.Join(rule.HelpText.Examples, "\n\n"),
		}
	}
	return HelpText{}
}

func (b *commandBuilder) releaseTracks() []string {
	// Infer default release track from proto package.
	// TODO(https://github.com/googleapis/librarian/issues/3289): Allow gcloud config to overwrite the track for this command.
	inferredTrack := provider.InferTrackFromPackage(b.method.Service.Package)
	return []string{strings.ToUpper(inferredTrack)}
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

// newArguments generates the set of arguments for a command by parsing the
// fields of the method's request message.
func (b *commandBuilder) newArguments() ([]Argument, error) {
	var args []Argument
	if b.method.InputType == nil {
		return args, nil
	}

	for _, field := range b.method.InputType.Fields {
		fieldArgs, err := b.argumentsFromField(field, field.JSONName)
		if err != nil {
			return nil, err
		}
		args = append(args, fieldArgs...)
	}
	return args, nil
}

// argumentsFromField recursively processes a field and its sub-fields to generate
// command-line flags. It uses a dispatch pattern to classify each field:
//  1. Primary resource arguments (positional resource identifiers).
//  2. Ignored fields (implicit or framework-handled).
//  3. Nested messages (flattened into top-level flags).
//  4. Standard arguments (scalars, maps, enums, resource references).
//
// TODO(https://github.com/googleapis/librarian/issues/3413): Improve error
// handling strategy (Error vs Skip) and messaging.
func (b *commandBuilder) argumentsFromField(field *api.Field, prefix string) ([]Argument, error) {
	// Primary resource args are checked first because fields like "parent"
	// and "name" are primary resources in certain method types (e.g., List
	// and Get/Delete/Update respectively) and must not be ignored.
	if provider.IsPrimaryResource(field, b.method) {
		arg := newArgumentBuilder(b.method, b.overrides, b.model, b.service, field, prefix).buildPrimaryResource()
		return []Argument{arg}, nil
	}

	if isIgnored(field, b.method) {
		return nil, nil
	}

	// Nested messages are flattened into top-level flags.
	// TODO(https://github.com/googleapis/librarian/issues/3287): Support arg_groups.
	if field.MessageType != nil && !field.Map {
		var args []Argument
		for _, f := range field.MessageType.Fields {
			fieldArgs, err := b.argumentsFromField(f, fmt.Sprintf("%s.%s", prefix, f.JSONName))
			if err != nil {
				return nil, err
			}
			args = append(args, fieldArgs...)
		}
		return args, nil
	}

	// Standard arguments: scalars, maps, enums, and resource references.
	param, err := newArgumentBuilder(b.method, b.overrides, b.model, b.service, field, prefix).build()
	if err != nil {
		return nil, err
	}
	return []Argument{param}, nil
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
		isScalar := f.Typez == api.STRING_TYPE ||
			f.Typez == api.INT32_TYPE || f.Typez == api.INT64_TYPE ||
			f.Typez == api.BOOL_TYPE || f.Typez == api.ENUM_TYPE ||
			f.Typez == api.DOUBLE_TYPE || f.Typez == api.FLOAT_TYPE

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

// isIgnored determines if a field should be excluded from the generated command arguments.
func isIgnored(field *api.Field, method *api.Method) bool {
	if field.Name == "parent" || field.Name == "name" || field.Name == "update_mask" {
		return true
	}
	if provider.IsList(method) {
		switch field.Name {
		case "page_size", "page_token", "filter", "order_by":
			return true
		}
	}
	if slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_OUTPUT_ONLY) {
		return true
	}
	if provider.IsUpdate(method) && slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_IMMUTABLE) {
		return true
	}
	return false
}
