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

package surfer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
	"github.com/iancoleman/strcase"
)

func newCommand(method *api.Method, overrides *provider.Config, model *api.API, service *api.Service) (*Command, error) {
	args, err := newArguments(method, overrides, model, service)
	if err != nil {
		return nil, err
	}

	apiVersion, err := provider.APIVersionFromMethod(method)
	if err != nil {
		return nil, err
	}

	useUpdateMask := updateMask(model, method, overrides)
	return &Command{
		Name:                 name(method),
		Hidden:               hidden(overrides, method),
		HelpText:             helpText(overrides, method, model),
		APIVersion:           apiVersion,
		Collection:           collectionPath(method, service, false),
		Method:               requestMethod(method),
		Arguments:            args,
		ResponseIDField:      responseIDField(method),
		OutputFormat:         outputFormat(overrides, method),
		ReadModifyUpdate:     provider.IsUpdate(method),
		StarUpdateMask:       useUpdateMask,
		DisableAutoFieldMask: useUpdateMask,
		Async:                async(method, model, service),
	}, nil
}

// newWaitCommand synthesizes a 'wait' command for operations based on GetOperation method.
func newWaitCommand(getMethod *api.Method, overrides *provider.Config, model *api.API, service *api.Service) (*Command, error) {
	arg, err := positionalResourceArg(getMethod, overrides, model, service)
	if err != nil {
		return nil, err
	}

	if arg == nil {
		return nil, fmt.Errorf("missing positional resource argument for wait command")
	}

	apiVersion, err := provider.APIVersionFromMethod(getMethod)
	if err != nil {
		return nil, err
	}

	arg.HelpText = "The name of the operation resource to wait on."
	waitArgs := []Argument{*arg}

	return &Command{
		Name:   "wait",
		Hidden: hidden(overrides, getMethod),
		HelpText: HelpText{

			Brief:       "Wait operations",
			Description: "Wait an operation",
			Examples:    "To wait the operation, run:\n\n    $ {command}",
		},
		APIVersion: apiVersion,
		Collection: collectionPath(getMethod, service, false),
		Arguments:  waitArgs,
		Async: &Async{
			Collection:            collectionPath(getMethod, service, true),
			ExtractResourceResult: false,
		},
	}, nil
}

func name(method *api.Method) string {
	name, err := provider.GetCommandName(method)
	if err != nil {
		return ""
	}
	return name
}

func responseIDField(method *api.Method) string {
	if provider.IsList(method) {
		// List commands should have an id_field to enable the --uri flag.
		return "name"
	}
	return ""
}

// TODO(https://github.com/googleapis/librarian/issues/5231): add default table output if not specified in the config.
func outputFormat(overrides *provider.Config, method *api.Method) string {
	return provider.OutputFormat(overrides, method.ID)
}

// async creates the `Async` part of the command definition for long-running operations.
func async(method *api.Method, model *api.API, service *api.Service) *Async {
	if method.OperationInfo == nil {
		return nil
	}

	async := &Async{
		Collection: collectionPath(method, service, true),
	}

	// Extract the resource result if the LRO response type matches the
	// method's resource type.
	resource := provider.GetResourceForMethod(method, model)
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

	singular := provider.GetSingularResourceNameForMethod(method, model)
	if strings.EqualFold(responseTypeName, singular) || strings.HasSuffix(resource.Type, "/"+responseTypeName) {
		async.ExtractResourceResult = true
	}

	return async
}

func hidden(overrides *provider.Config, method *api.Method) bool {
	apiCfg := provider.FindAPIConfig(overrides, method)
	if apiCfg != nil {
		return apiCfg.RootIsHidden
	}
	return false
}

func helpText(overrides *provider.Config, method *api.Method, model *api.API) HelpText {
	h := provider.GetMethodHelpText(overrides, method, model)
	return HelpText{
		Brief:       h.Brief,
		Description: h.Description,
		Examples:    h.Examples,
	}
}

// requestMethod determines the API method name for the command execution.
func requestMethod(method *api.Method) string {
	// For custom methods (AIP-136), the `method` field in the request configuration
	// MUST match the custom verb defined in the HTTP binding (e.g., ":exportData" -> "exportData").
	if method.PathInfo != nil && len(method.PathInfo.Bindings) > 0 && method.PathInfo.Bindings[0].PathTemplate.Verb != nil {
		return *method.PathInfo.Bindings[0].PathTemplate.Verb
	} else if !provider.IsStandardMethod(method) {
		commandName, _ := provider.GetCommandName(method)
		// GetCommandName returns snake_case (e.g. "export_data"), but request.method expects camelCase (e.g. "exportData").
		return strcase.ToLowerCamel(commandName)
	}

	return ""
}

type classifiedFields struct {
	primaryField    []*api.Field
	resourceIdField []*api.Field
	other           [][]*api.Field
}

// newArguments generates the set of arguments for a command by parsing the
// fields of the method's request message.
func newArguments(method *api.Method, overrides *provider.Config, model *api.API, service *api.Service) ([]Argument, error) {
	var args []Argument
	if method.InputType == nil {
		return args, nil
	}

	cf, err := categorizeFields(method, model)
	if err != nil {
		return nil, err
	}

	if cf.primaryField != nil {
		var idField *api.Field
		if cf.resourceIdField != nil {
			idField = cf.resourceIdField[len(cf.resourceIdField)-1]
		}
		arg := newPrimaryResourceArgument(&argumentParams{
			method:    method,
			overrides: overrides,
			model:     model,
			service:   service,
			field:     cf.primaryField[len(cf.primaryField)-1],
			fieldPath: cf.primaryField,
		}, idField)
		args = append(args, arg)
	}

	for _, path := range cf.other {
		field := path[len(path)-1]
		arg, err := newArgument(&argumentParams{
			method:    method,
			overrides: overrides,
			model:     model,
			service:   service,
			field:     field,
			fieldPath: path,
		})
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

func positionalResourceArg(method *api.Method, overrides *provider.Config, model *api.API, service *api.Service) (*Argument, error) {
	cf, err := categorizeFields(method, model)
	if err != nil {
		return nil, err
	}

	if cf.primaryField == nil {
		return nil, nil
	}

	var idField *api.Field
	if cf.resourceIdField != nil {
		idField = cf.resourceIdField[len(cf.resourceIdField)-1]
	}

	arg := newPrimaryResourceArgument(&argumentParams{
		method:    method,
		overrides: overrides,
		model:     model,
		service:   service,
		field:     cf.primaryField[len(cf.primaryField)-1],
		fieldPath: cf.primaryField,
	}, idField)
	return &arg, nil
}

// categorizeFields gathers fields from the method input type, expanding the body field
// if present, and separates them into at most one primary resource field, at most one
// resource ID field, and other fields. Returns an error if multiple are found.
func categorizeFields(method *api.Method, model *api.API) (classifiedFields, error) {
	var cf classifiedFields
	bodyFieldPath := ""
	if method.PathInfo != nil {
		bodyFieldPath = method.PathInfo.BodyFieldPath
	}

	var collected [][]*api.Field
	for _, field := range method.InputType.Fields {
		isExpandableMessage := field.MessageType != nil && !field.Map
		isBodyWildcard := bodyFieldPath == "*"
		isBodyField := bodyFieldPath == field.Name || isBodyWildcard

		var path []*api.Field

		if isExpandableMessage && isBodyField {
			path = append(path, field)
			for _, f := range field.MessageType.Fields {
				collected = append(collected, append(slices.Clone(path), f))
			}
			continue
		}

		collected = append(collected, append(path, field))
	}

	for _, path := range collected {
		field := path[len(path)-1]
		switch {
		case provider.IsPrimaryResourceField(field, method):
			if cf.primaryField != nil {
				return cf, fmt.Errorf("method %q has multiple primary resource fields: %q and %q", method.Name, cf.primaryField[len(cf.primaryField)-1].Name, field.Name)
			}
			cf.primaryField = path
		case provider.IsResourceIdField(field, method, model):
			if cf.resourceIdField != nil {
				return cf, fmt.Errorf("method %q has multiple resource ID fields: %q and %q", method.Name, cf.resourceIdField[len(cf.resourceIdField)-1].Name, field.Name)
			}
			cf.resourceIdField = path
		case provider.IsCreate(method) && field.Name == "name":
			// Ignore name field in Create methods as it's redundant with resource_id
		default:
			cf.other = append(cf.other, path)
		}
	}

	return cf, nil
}

// collectionPath constructs the gcloud collection path(s) for a request or async operation.
// It follows AIP-127 and AIP-132 by extracting the collection structure directly from
// the method's HTTP annotation (PathInfo).
func collectionPath(method *api.Method, service *api.Service, isAsync bool) []string {
	var collections []string
	hostParts := strings.Split(service.DefaultHost, ".")
	shortServiceName := hostParts[0]

	// Iterate over all bindings (primary + additional) to support multitype resources (AIP-127).
	for _, binding := range method.PathInfo.Bindings {
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

func updateMask(model *api.API, method *api.Method, overrides *provider.Config) bool {
	if !provider.IsUpdate(method) || method.InputType == nil {
		return false
	}
	version := provider.APIVersionFromModel(model)
	if !provider.SupportsStarUpdateMasks(overrides, method.Service.Name, version) {
		return false
	}
	for _, f := range method.InputType.Fields {
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
