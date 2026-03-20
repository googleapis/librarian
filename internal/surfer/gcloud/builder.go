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
	"github.com/iancoleman/strcase"
)

type commandBuilder struct {
	method  *provider.MethodAdapter
	config  *provider.Config
	model   *api.API
	service *api.Service
	cmd     *Command
	err     error
}

// NewCommand constructs a single gcloud command definition from an API method.
// This function assembles all the necessary pieces: help text, arguments,
// request details, and async configuration.
func NewCommand(method *provider.MethodAdapter, overrides *provider.Config, model *api.API, service *api.Service) (*Command, error) {
	b := &commandBuilder{
		method:  method,
		config:  overrides,
		model:   model,
		service: service,
		cmd:     &Command{},
	}
	return b.WithHidden().
		WithHelpText().
		WithReleaseTracks().
		WithArguments().
		WithRequest().
		WithOutputConfig().
		WithUpdateConfig().
		WithAsync().
		Build()
}

func (b *commandBuilder) WithHidden() *commandBuilder {
	if b.err != nil {
		return b
	}
	if len(b.config.APIs) > 0 {
		b.cmd.Hidden = b.config.APIs[0].RootIsHidden
	} else {
		// Default to hidden if no API overrides are provided.
		b.cmd.Hidden = true
	}
	return b
}

func (b *commandBuilder) WithHelpText() *commandBuilder {
	if b.err != nil {
		return b
	}
	rule := b.findHelpTextRule()
	if rule != nil {
		b.cmd.HelpText = HelpText{
			Brief:       rule.HelpText.Brief,
			Description: rule.HelpText.Description,
			Examples:    strings.Join(rule.HelpText.Examples, "\n\n"),
		}
	}
	return b
}

func (b *commandBuilder) WithReleaseTracks() *commandBuilder {
	if b.err != nil {
		return b
	}
	// Infer default release track from proto package.
	// TODO(https://github.com/googleapis/librarian/issues/3289): Allow gcloud config to overwrite the track for this command.
	inferredTrack := provider.InferTrackFromPackage(b.method.Method.Service.Package)
	b.cmd.ReleaseTracks = []string{strings.ToUpper(inferredTrack)}
	return b
}

func (b *commandBuilder) WithArguments() *commandBuilder {
	if b.err != nil {
		return b
	}

	args := Arguments{}
	if b.method.Method.InputType == nil {
		b.cmd.Arguments = args
		return b
	}

	for _, field := range b.method.Method.InputType.Fields {
		params, err := b.flattenField(field, field.JSONName)
		if err != nil {
			b.err = err
			return b
		}
		args.Params = append(args.Params, params...)
	}

	b.cmd.Arguments = args
	return b
}

func (b *commandBuilder) WithRequest() *commandBuilder {
	if b.err != nil {
		return b
	}
	req := &Request{
		APIVersion: b.apiVersion(),
		Collection: b.newCollectionPath(false),
	}

	// For custom methods (AIP-136), the `method` field in the request configuration
	// MUST match the custom verb defined in the HTTP binding (e.g., ":exportData" -> "exportData").
	if len(b.method.Method.PathInfo.Bindings) > 0 && b.method.Method.PathInfo.Bindings[0].PathTemplate.Verb != nil {
		req.Method = *b.method.Method.PathInfo.Bindings[0].PathTemplate.Verb
	} else if !b.method.IsStandardMethod() {
		commandName, _ := b.method.GetCommandName()
		// GetCommandName returns snake_case (e.g. "export_data"), but request.method expects camelCase (e.g. "exportData").
		req.Method = strcase.ToLowerCamel(commandName)
	}

	b.cmd.Request = req
	// Special logic historically bound here for List requests
	if b.method.Type() == provider.MethodTypeList {
		// List commands should have an id_field to enable the --uri flag.
		b.cmd.Response = &Response{
			IDField: "name",
		}
	}
	return b
}

func (b *commandBuilder) WithOutputConfig() *commandBuilder {
	if b.err != nil {
		return b
	}
	if b.method.Type() != provider.MethodTypeList {
		return b
	}

	resourceMsg := provider.FindResourceMessage(b.method.Method.OutputType)
	if resourceMsg != nil {
		if format := newFormat(resourceMsg); format != "" {
			b.cmd.Output = &OutputConfig{
				Format: format,
			}
		}
	}

	return b
}

func (b *commandBuilder) WithUpdateConfig() *commandBuilder {
	if b.err != nil {
		return b
	}
	if b.method.Type() == provider.MethodTypeUpdate {
		// Standard Update methods in gcloud use the Read-Modify-Update pattern.
		b.cmd.Update = &UpdateConfig{
			ReadModifyUpdate: true,
		}
	}
	return b
}

func (b *commandBuilder) WithAsync() *commandBuilder {
	if b.err != nil {
		return b
	}

	if b.method.Method.OperationInfo == nil {
		return b
	}

	async := &Async{
		Collection: b.newCollectionPath(true),
	}

	// Extract the resource result if the LRO response type matches the
	// method's resource type.
	resource := b.method.GetResource(b.model)
	if resource == nil {
		b.cmd.Async = async
		return b
	}

	// Heuristic: Check if response type ID (e.g. ".google.cloud.parallelstore.v1.Instance")
	// matches the resource singular name or type.
	responseTypeID := b.method.Method.OperationInfo.ResponseTypeID
	// Extract short name from FQN (last element after dot)
	responseTypeName := responseTypeID
	if idx := strings.LastIndex(responseTypeID, "."); idx != -1 {
		responseTypeName = responseTypeID[idx+1:]
	}

	singular := b.method.GetSingularResourceName(b.model)
	if strings.EqualFold(responseTypeName, singular) || strings.HasSuffix(resource.Type, "/"+responseTypeName) {
		async.ExtractResourceResult = true
	} else {
		async.ExtractResourceResult = false
	}

	b.cmd.Async = async
	return b
}

func (b *commandBuilder) Build() (*Command, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.cmd, nil
}

// newCollectionPath constructs the gcloud collection path(s) for a request or async operation.
// It follows AIP-127 and AIP-132 by extracting the collection structure directly from
// the method's HTTP annotation (PathInfo).
func (b *commandBuilder) newCollectionPath(isAsync bool) []string {
	var collections []string
	hostParts := strings.Split(b.service.DefaultHost, ".")
	shortServiceName := hostParts[0]

	// Iterate over all bindings (primary + additional) to support multitype resources (AIP-127).
	for _, binding := range b.method.Method.PathInfo.Bindings {
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

// newFormat generates a gcloud table format string from a message definition.
func newFormat(message *api.Message) string {
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

// findHelpTextRule finds the help text rule from the config that applies to the current method.
func (b *commandBuilder) findHelpTextRule() *provider.HelpTextRule {
	if b.config.APIs == nil {
		return nil
	}
	for _, api := range b.config.APIs {
		if api.HelpText == nil {
			continue
		}
		for _, rule := range api.HelpText.MethodRules {
			if rule.Selector == strings.TrimPrefix(b.method.Method.ID, ".") {
				return rule
			}
		}
	}
	return nil
}

// findFieldHelpTextRule finds the help text rule from the config that applies to the current field.
func (b *commandBuilder) findFieldHelpTextRule(field *api.Field) *provider.HelpTextRule {
	if b.config.APIs == nil {
		return nil
	}
	for _, api := range b.config.APIs {
		if api.HelpText == nil {
			continue
		}
		for _, rule := range api.HelpText.FieldRules {
			if rule.Selector == field.ID {
				return rule
			}
		}
	}
	return nil
}

// apiVersion extracts the API version from the configuration.
func (b *commandBuilder) apiVersion() string {
	if len(b.config.APIs) > 0 {
		return b.config.APIs[0].APIVersion
	}
	return ""
}
