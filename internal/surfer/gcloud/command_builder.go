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

// NewCommand constructs a single gcloud command definition from an API method.
// This function assembles all the necessary pieces: help text, arguments,
// request details, and async configuration.
func NewCommand(method *api.Method, overrides *Config, model *api.API, service *api.Service) (*Command, error) {
	rule := findHelpTextRule(method, overrides)

	cmd := &Command{}

	if len(overrides.APIs) > 0 {
		cmd.Hidden = overrides.APIs[0].RootIsHidden
	} else {
		// Default to hidden if no API overrides are provided.
		cmd.Hidden = true
	}

	if rule != nil {
		cmd.HelpText = HelpText{
			Brief:       rule.HelpText.Brief,
			Description: rule.HelpText.Description,
			Examples:    strings.Join(rule.HelpText.Examples, "\n\n"),
		}
	}

	// Infer default release track from proto package.
	// TODO(https://github.com/googleapis/librarian/issues/3289): Allow gcloud config to overwrite the track for this command.
	inferredTrack := inferTrackFromPackage(method.Service.Package)
	cmd.ReleaseTracks = []string{strings.ToUpper(inferredTrack)}

	args, err := newArguments(method, overrides, model, service)
	if err != nil {
		return nil, err
	}
	cmd.Arguments = args
	cmd.Request = newRequest(method, overrides, service)

	if isList(method) {
		// List commands should have an id_field to enable the --uri flag.
		cmd.ResponseIDField = "name"
		cmd.OutputFormat = newOutputFormat(method)
	}

	if isUpdate(method) {
		// Standard Update methods in gcloud use the Read-Modify-Update pattern.
		cmd.ReadModifyUpdate = true
	}

	if method.OperationInfo != nil {
		cmd.Async = newAsync(method, model, service)
	}

	return cmd, nil
}

// newRequest creates the `Request` part of the command definition.
func newRequest(method *api.Method, overrides *Config, service *api.Service) *Request {
	req := &Request{
		APIVersion: apiVersion(overrides),
		Collection: newCollectionPath(method, service, false),
	}

	// For custom methods (AIP-136), the `method` field in the request configuration
	// MUST match the custom verb defined in the HTTP binding (e.g., ":exportData" -> "exportData").
	if len(method.PathInfo.Bindings) > 0 && method.PathInfo.Bindings[0].PathTemplate.Verb != nil {
		req.Method = *method.PathInfo.Bindings[0].PathTemplate.Verb
	} else if !isStandardMethod(method) {
		commandName, _ := getCommandName(method)
		// GetCommandName returns snake_case (e.g. "export_data"), but request.method expects camelCase (e.g. "exportData").
		req.Method = strcase.ToLowerCamel(commandName)
	}

	return req
}

// newAsync creates the `Async` part of the command definition for long-running operations.
func newAsync(method *api.Method, model *api.API, service *api.Service) *Async {
	async := &Async{
		Collection: newCollectionPath(method, service, true),
	}

	// Extract the resource result if the LRO response type matches the
	// method's resource type.
	resource := getResourceForMethod(method, model)
	if resource == nil {
		return async
	}

	// Heuristic: Check if response type ID (e.g. ".google.cloud.parallelstore.v1.Instance")
	// matches the resource singular name or type.
	responseTypeID := method.OperationInfo.ResponseTypeID
	// Extract short name from FQN (last element after dot)
	responseTypeName := responseTypeID
	if idx := strings.LastIndex(responseTypeID, "."); idx != -1 {
		responseTypeName = responseTypeID[idx+1:]
	}

	singular := getSingularResourceNameForMethod(method, model)
	if strings.EqualFold(responseTypeName, singular) || strings.HasSuffix(resource.Type, "/"+responseTypeName) {
		async.ExtractResourceResult = true
	} else {
		async.ExtractResourceResult = false
	}

	return async
}

// newCollectionPath constructs the gcloud collection path(s) for a request or async operation.
// It follows AIP-127 and AIP-132 by extracting the collection structure directly from
// the method's HTTP annotation (PathInfo).
func newCollectionPath(method *api.Method, service *api.Service, isAsync bool) []string {
	var collections []string
	hostParts := strings.Split(service.DefaultHost, ".")
	shortServiceName := hostParts[0]

	// Iterate over all bindings (primary + additional) to support multitype resources (AIP-127).
	for _, binding := range method.PathInfo.Bindings {
		if binding.PathTemplate == nil {
			continue
		}

		basePath := extractPathFromSegments(binding.PathTemplate.Segments)

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

// newOutputFormat generates the string output format for List commands.
func newOutputFormat(method *api.Method) string {
	if !isList(method) {
		return ""
	}

	resourceMsg := findResourceMessage(method.OutputType)
	if resourceMsg == nil {
		return ""
	}

	return newFormat(resourceMsg)
}

// newFormat generates a gcloud table format string from a message definition.
func newFormat(message *api.Message) string {
	var sb strings.Builder
	first := true

	for _, f := range message.Fields {
		// Sanitize field name to prevent DSL injection.
		if !isSafeName(f.JSONName) {
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
func findHelpTextRule(method *api.Method, overrides *Config) *HelpTextRule {
	if overrides.APIs == nil {
		return nil
	}
	for _, api := range overrides.APIs {
		if api.HelpText == nil {
			continue
		}
		for _, rule := range api.HelpText.MethodRules {
			if rule.Selector == strings.TrimPrefix(method.ID, ".") {
				return rule
			}
		}
	}
	return nil
}

// findFieldHelpTextRule finds the help text rule from the config that applies to the current field.
func findFieldHelpTextRule(field *api.Field, overrides *Config) *HelpTextRule {
	if overrides.APIs == nil {
		return nil
	}
	for _, api := range overrides.APIs {
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
func apiVersion(overrides *Config) string {
	if len(overrides.APIs) > 0 {
		return overrides.APIs[0].APIVersion
	}
	return ""
}
