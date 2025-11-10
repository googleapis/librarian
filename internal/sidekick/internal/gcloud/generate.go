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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/config/gcloudyaml"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Generate is the main entrypoint for the gcloud command generator. It orchestrates
// the process of parsing the API model and generating the corresponding gcloud
// command surface.
func Generate(model *api.API, outdir string, cfg *config.Config) error {
	// Ensure that the gcloud configuration is present.
	if cfg.Gcloud == nil {
		return fmt.Errorf("gcloud config is missing")
	}
	// Extract the short service name (e.g., "parallelstore") from the full
	// service name (e.g., "parallelstore.googleapis.com").
	serviceNameParts := strings.Split(cfg.Gcloud.ServiceName, ".")
	if len(serviceNameParts) == 0 {
		return fmt.Errorf("invalid service name in gcloud.yaml: %s", cfg.Gcloud.ServiceName)
	}
	shortServiceName := serviceNameParts[0]
	// Define the output directory for the generated command surface.
	surfaceDir := filepath.Join(outdir, shortServiceName, "surface")

	// Group all API methods by the resource they operate on. The resource is
	// identified by its collection ID (e.g., "instances").
	methodsByResource := make(map[string][]*api.Method)
	for _, service := range model.Services {
		for _, method := range service.Methods {
			// Determine the plural name of the resource, which serves as the collection ID.
			collectionID := getPluralName(method, model)
			if collectionID != "" {
				// Add the method to the list of methods for this resource.
				methodsByResource[collectionID] = append(methodsByResource[collectionID], method)
			}
		}
	}

	// Iterate over each resource and generate the corresponding command files.
	for collectionID, methods := range methodsByResource {
		err := generateResourceCommands(collectionID, methods, surfaceDir, cfg, model)
		if err != nil {
			return err
		}
	}
	return nil
}

// generateResourceCommands creates the directory structure and YAML files for a
// single resource's commands (e.g., create, delete, list).
func generateResourceCommands(collectionID string, methods []*api.Method, baseDir string, cfg *config.Config, model *api.API) error {
	// Create the main directory for the resource (e.g., "instances").
	resourceDir := filepath.Join(baseDir, collectionID)
	// Create the "_partials" directory where the actual command definitions will live.
	partialsDir := filepath.Join(resourceDir, "_partials")
	if err := os.MkdirAll(partialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create partials directory for %q: %w", collectionID, err)
	}

	// Iterate over each method associated with the resource and generate a command file.
	for _, method := range methods {
		// Determine the gcloud verb (e.g., "create", "describe") from the method name.
		verb := getVerb(method.Name)
		// Construct the complete command definition from the API method.
		cmd := newCommand(method, cfg, model)
		// wrap the command definition in a list (current gcloud convention).
		cmdList := []*Command{cmd}

		// Create the main command file (e.g., "create.yaml") which simply points
		// to the partials directory.
		mainCmdPath := filepath.Join(resourceDir, fmt.Sprintf("%s.yaml", verb))
		if err := os.WriteFile(mainCmdPath, []byte("_PARTIALS_: true\n"), 0644); err != nil {
			return fmt.Errorf("failed to write main command file for %q: %w", method.Name, err)
		}

		// Define the name of the partial file, including the release track (e.g., "_create_ga.yaml").
		track := "ga"
		partialFileName := fmt.Sprintf("_%s_%s.yaml", verb, track)
		partialCmdPath := filepath.Join(partialsDir, partialFileName)

		// Marshal the command definition into YAML.
		b, err := yaml.Marshal(cmdList)
		if err != nil {
			return fmt.Errorf("failed to marshal partial command for %q: %w", method.Name, err)
		}

		// Write the YAML to the partial file.
		if err := os.WriteFile(partialCmdPath, b, 0644); err != nil {
			return fmt.Errorf("failed to write partial command file for %q: %w", method.Name, err)
		}
	}
	return nil
}

// newCommand constructs a single gcloud command definition from an API method.
func newCommand(method *api.Method, cfg *config.Config, model *api.API) *Command {
	// Find the help text and API definition for this method from the config.
	rule := findHelpTextRule(method, cfg)
	apiDef := findAPI(method, cfg)
	// Initialize the command with default values.
	cmd := &Command{
		AutoGenerated: true,
		Hidden:        true,
	}
	// If a help text rule is found, apply it.
	if rule != nil {
		cmd.HelpText = HelpText{
			Brief:       rule.HelpText.Brief,
			Description: rule.HelpText.Description,
			Examples:    "TODO: Add examples from " + rule.Selector,
		}
	}
	// If an API definition is found, apply the release tracks.
	if apiDef != nil {
		for _, track := range apiDef.ReleaseTracks {
			cmd.ReleaseTracks = append(cmd.ReleaseTracks, string(track))
		}
	}
	// Generate the arguments for the command.
	cmd.Arguments = newArguments(method, cfg, model)
	// Generate the request details for the command.
	cmd.Request = newRequest(method, cfg, model)
	// If the method is a long-running operation, generate the async details.
	if method.OperationInfo != nil {
		cmd.Async = newAsync(method, cfg)
	}
	return cmd
}

// newArguments generates the set of arguments for a command by parsing the
// fields of the method's request message.
func newArguments(method *api.Method, cfg *config.Config, model *api.API) Arguments {
	args := Arguments{}
	if method.InputType == nil {
		return args
	}

	// Iterate over each field in the request message.
	for _, field := range method.InputType.Fields {
		// The "parent" field is handled by the primary resource argument, so we skip it here.
		if field.Name == "parent" {
			continue
		}

		// If the field represents the primary resource, generate a special
		// positional resource argument for it.
		if isPrimaryResource(field, method) {
			param := newPrimaryResourceParam(field, method, model, cfg)
			args.Params = append(args.Params, param)
			continue
		}

		// For all other fields, generate a standard flag argument. If the field
		// is a nested message, its fields will be "flattened" into top-level flags.
		addFlattenedParams(field, field.JSONName, &args, cfg, model)
	}
	return args
}

// addFlattenedParams recursively processes a field and its sub-fields to generate
// a flat list of command-line flags.
func addFlattenedParams(field *api.Field, prefix string, args *Arguments, cfg *config.Config, model *api.API) {
	// Skip fields that are output-only or are the resource's name (which is handled
	// by the primary resource argument).
	if isOutputOnly(field) || field.Name == "name" {
		return
	}

	// If the field is a nested message (and not a map), recurse into its fields.
	if field.MessageType != nil && !field.Map {
		for _, f := range field.MessageType.Fields {
			// The prefix is updated to create a dot-separated path for the API field.
			addFlattenedParams(f, fmt.Sprintf("%s.%s", prefix, f.JSONName), args, cfg, model)
		}
		return
	}

	// If the field is a scalar, map, or enum, generate a parameter for it.
	param := newParam(field, prefix, cfg, model)
	args.Params = append(args.Params, param)
}

// newParam creates a single command-line argument from a proto field.
func newParam(field *api.Field, apiField string, cfg *config.Config, model *api.API) Param {
	param := Param{
		ArgName:  ToKebabCase(field.Name),
		APIField: apiField,
		Required: field.DocumentAsRequired(),
		Repeated: field.Repeated,
	}

	// If the field is a resource reference, generate a resource spec for it.
	if field.ResourceReference != nil {
		param.ResourceSpec = newResourceReferenceSpec(field, model, cfg)
		param.ResourceMethodParams = map[string]string{
			apiField: "{__relative_name__}",
		}
		// If the field is a map, generate a spec for its key-value pairs.
	} else if field.Map {
		param.Repeated = true
		param.Spec = []ArgSpec{
			{APIField: "key"},
			{APIField: "value"},
		}
		// If the field is an enum, generate choices for its possible values.
	} else if field.EnumType != nil {
		for _, v := range field.EnumType.Values {
			// Skip the default "UNSPECIFIED" value.
			if strings.HasSuffix(v.Name, "_UNSPECIFIED") {
				continue
			}
			param.Choices = append(param.Choices, Choice{
				ArgValue:  ToKebabCase(v.Name),
				EnumValue: v.Name,
			})
		}
		// Otherwise, it's a scalar type.
	} else {
		param.Type = getGcloudType(field.Typez)
	}

	// Find the help text for this field from the config, or generate a default.
	if rule := findFieldHelpTextRule(field, cfg); rule != nil {
		param.HelpText = rule.HelpText.Brief
	} else {
		param.HelpText = fmt.Sprintf("Value for the `%s` field.", ToKebabCase(field.Name))
	}
	return param
}

// newPrimaryResourceParam creates the main positional resource argument for a command.
func newPrimaryResourceParam(field *api.Field, method *api.Method, model *api.API, cfg *config.Config) Param {
	// Get the resource definition for this method.
	resource := getResourceForMethod(method, model)
	pattern := ""
	if resource != nil && len(resource.Pattern) > 0 {
		pattern = resource.Pattern[0]
	}

	// Construct the gcloud collection path from the resource pattern.
	collectionPath := getCollectionPathFromPattern(pattern)
	shortServiceName := strings.Split(cfg.Gcloud.ServiceName, ".")[0]

	// Determine the singular name of the resource.
	resourceName := toSnakeCase(strings.TrimSuffix(field.Name, "_id"))
	if field.Name == "name" {
		resourceName = getSingularFromPattern(pattern)
	}

	// Generate appropriate help text based on the command verb.
	helpText := fmt.Sprintf("The %s to create.", resourceName)
	if !strings.HasPrefix(method.Name, "Create") {
		helpText = fmt.Sprintf("The %s to operate on.", resourceName)
	}

	// Construct and return the Param struct.
	return Param{
		HelpText:          helpText,
		IsPositional:      true,
		IsPrimaryResource: true,
		Required:          true,
		RequestIDField:    toLowerCamelCase(field.Name),
		ResourceSpec: &ResourceSpec{
			Name:                  resourceName,
			PluralName:            getPluralName(method, model),
			Collection:            fmt.Sprintf("%s.%s", shortServiceName, collectionPath),
			DisableAutoCompleters: false,
			Attributes:            newAttributesFromPattern(pattern),
		},
	}
}

// newResourceReferenceSpec creates a ResourceSpec for a field that references
// another resource type.
func newResourceReferenceSpec(field *api.Field, model *api.API, cfg *config.Config) *ResourceSpec {
	// Find the definition of the referenced resource in the API model.
	for _, def := range model.ResourceDefinitions {
		if def.Type == field.ResourceReference.Type {
			if len(def.Pattern) == 0 {
				return nil // Cannot proceed without a pattern
			}
			pattern := def.Pattern[0]

			// Determine the plural name, falling back to parsing the resource pattern if not explicit.
			pluralName := def.Plural
			if pluralName == "" {
				pluralName = getPluralFromPattern(pattern)
			}

			// Determine the singular name from the pattern.
			name := getSingularFromPattern(pattern)

			// Construct the full gcloud collection path.
			shortServiceName := strings.Split(cfg.Gcloud.ServiceName, ".")[0]
			baseCollectionPath := getCollectionPathFromPattern(pattern)
			fullCollectionPath := fmt.Sprintf("%s.%s", shortServiceName, baseCollectionPath)

			// Construct and return the ResourceSpec.
			return &ResourceSpec{
				Name:                  name,
				PluralName:            pluralName,
				Collection:            fullCollectionPath,
				DisableAutoCompleters: true,
				Attributes:            newAttributesFromPattern(pattern),
			}
		}
	}
	return nil
}

// newAttributesFromPattern parses a resource pattern string (e.g.,
// "projects/{project}/locations/{location}") and extracts the attributes.
func newAttributesFromPattern(pattern string) []Attribute {
	var attributes []Attribute
	parts := strings.Split(pattern, "/")
	// Iterate over the segments of the pattern.
	for i, part := range parts {
		// A variable segment is enclosed in curly braces.
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := strings.Trim(part, "{}")
			var parameterName string
			// The parameter name is derived from the preceding literal segment
			// (e.g., "projects" -> "projectsId").
			if i > 0 {
				parameterName = parts[i-1] + "Id"
			} else {
				// Fallback for patterns that start with a variable (unlikely).
				parameterName = name + "sId"
			}
			attr := Attribute{
				AttributeName: name,
				ParameterName: parameterName,
				Help:          fmt.Sprintf("The %s id of the {resource} resource.", name),
			}
			// If the attribute is a project, add the standard gcloud property fallback.
			if name == "project" {
				attr.Property = "core/project"
			}
			attributes = append(attributes, attr)
		}
	}
	return attributes
}

// isPrimaryResource determines if a field represents the primary resource of a method.
func isPrimaryResource(field *api.Field, method *api.Method) bool {
	if method.InputType == nil {
		return false
	}
	// For Create methods, the primary resource is identified by a field named
	// "{resource}_id".
	if strings.HasPrefix(method.Name, "Create") {
		resourceName := getResourceName(method)
		if resourceName != "" && field.Name == toSnakeCase(resourceName)+"_id" {
			return true
		}
	}
	// For other methods, the primary resource is identified by a field named "name".
	if (strings.HasPrefix(method.Name, "Get") || strings.HasPrefix(method.Name, "Delete") || strings.HasPrefix(method.Name, "Update")) && field.Name == "name" {
		return true
	}
	return false
}

// getResourceName extracts the name of the resource from a method's input message.
func getResourceName(method *api.Method) string {
	for _, f := range method.InputType.Fields {
		if msg := f.MessageType; msg != nil && msg.Resource != nil {
			return msg.Name
		}
	}
	return ""
}

// getResourceForMethod finds the `api.Resource` definition associated with a method.
func getResourceForMethod(method *api.Method, model *api.API) *api.Resource {
	// Strategy 1: Find a field that is the resource message itself (for Create/Update).
	for _, f := range method.InputType.Fields {
		if msg := f.MessageType; msg != nil && msg.Resource != nil {
			return msg.Resource
		}
	}

	// Strategy 2: Find a 'name' or 'parent' field with a resource_reference (for Get/Delete/List).
	var resourceType string
	if method.InputType != nil {
		for _, field := range method.InputType.Fields {
			if (field.Name == "name" || field.Name == "parent") && field.ResourceReference != nil {
				resourceType = field.ResourceReference.Type
				if resourceType == "" {
					resourceType = field.ResourceReference.ChildType
				}
				break
			}
		}
	}

	// If a resource type was found, look up its full definition in the API model.
	if resourceType != "" {
		for _, msg := range model.Messages {
			if msg.Resource != nil && msg.Resource.Type == resourceType {
				return msg.Resource
			}
		}
		for _, def := range model.ResourceDefinitions {
			if def.Type == resourceType {
				return def
			}
		}
	}

	return nil
}

// newRequest creates the `Request` part of the command definition.
func newRequest(method *api.Method, cfg *config.Config, model *api.API) *Request {
	return &Request{
		APIVersion: apiVersion(cfg),
		Collection: []string{fmt.Sprintf("parallelstore.projects.locations.%s", getPluralName(method, model))},
	}
}

// newAsync creates the `Async` part of the command definition for long-running operations.
func newAsync(method *api.Method, cfg *config.Config) *Async {
	return &Async{
		Collection: []string{"parallelstore.projects.locations.operations"},
	}
}

// apiVersion extracts the API version from the configuration.
func apiVersion(cfg *config.Config) string {
	if cfg.Gcloud != nil && len(cfg.Gcloud.APIs) > 0 {
		return cfg.Gcloud.APIs[0].APIVersion
	}
	return ""
}

// getGcloudType maps a proto data type to its corresponding gcloud type.
func getGcloudType(t api.Typez) string {
	switch t {
	case api.STRING_TYPE:
		return "" // Default is string
	case api.INT32_TYPE, api.INT64_TYPE, api.UINT32_TYPE, api.UINT64_TYPE:
		return "long"
	case api.BOOL_TYPE:
		return "boolean"
	case api.FLOAT_TYPE, api.DOUBLE_TYPE:
		return "float"
	default:
		return ""
	}
}

// getPluralName determines the plural name of a resource, using the explicit
// `plural` field if available, and falling back to parsing the resource pattern.
func getPluralName(method *api.Method, model *api.API) string {
	resource := getResourceForMethod(method, model)
	if resource != nil {
		// Prefer the explicit plural name from the resource definition.
		if resource.Plural != "" {
			return resource.Plural
		}
		// Fall back to inferring it from the resource pattern.
		if len(resource.Pattern) > 0 {
			return getPluralFromPattern(resource.Pattern[0])
		}
	}
	return ""
}

// getVerb maps an API method name to a standard gcloud command verb.
func getVerb(methodName string) string {
	switch {
	case strings.HasPrefix(methodName, "Get"):
		return "describe"
	case strings.HasPrefix(methodName, "List"):
		return "list"
	case strings.HasPrefix(methodName, "Create"):
		return "create"
	case strings.HasPrefix(methodName, "Update"):
		return "update"
	case strings.HasPrefix(methodName, "Delete"):
		return "delete"
	default:
		return toSnakeCase(methodName)
	}
}

// findAPI finds the API definition from the config that applies to the current method.
func findAPI(method *api.Method, cfg *config.Config) *gcloudyaml.API {
	if cfg.Gcloud == nil || cfg.Gcloud.APIs == nil {
		return nil
	}
	// This implementation currently assumes a single API definition.
	if len(cfg.Gcloud.APIs) > 0 {
		return &cfg.Gcloud.APIs[0]
	}
	return nil
}

// findHelpTextRule finds the help text rule from the config that applies to the current method.
func findHelpTextRule(method *api.Method, cfg *config.Config) *gcloudyaml.HelpTextRule {
	if cfg.Gcloud == nil || cfg.Gcloud.APIs == nil {
		return nil
	}
	for _, api := range cfg.Gcloud.APIs {
		if api.HelpText == nil {
			continue
		}
		for _, rule := range api.HelpText.MethodRules {
			if rule.Selector == method.MethodFullName() {
				return rule
			}
		}
	}
	return nil
}

// findFieldHelpTextRule finds the help text rule from the config that applies to the current field.
func findFieldHelpTextRule(field *api.Field, cfg *config.Config) *gcloudyaml.HelpTextRule {
	//TODO(santi): fix helptext
	if cfg.Gcloud == nil || cfg.Gcloud.APIs == nil {
		return nil
	}
	for _, api := range cfg.Gcloud.APIs {
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

// isOutputOnly checks if a field is marked as output-only.
func isOutputOnly(field *api.Field) bool {
	return slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_OUTPUT_ONLY)
}

// toLowerCamelCase converts a snake_case string to lowerCamelCase.
func toLowerCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		caser := cases.Title(language.AmericanEnglish)
		parts[i] = caser.String(parts[i])
	}
	return strings.Join(parts, "")
}

// getPluralFromPattern infers the plural name of a resource from its pattern string.
func getPluralFromPattern(pattern string) string {
	parts := strings.Split(pattern, "/")
	if len(parts) >= 2 {
		// The plural is the literal segment before the final variable segment.
		if strings.HasPrefix(parts[len(parts)-1], "{") {
			return parts[len(parts)-2]
		}
	}
	return ""
}

// getSingularFromPattern infers the singular name of a resource from its pattern string.
func getSingularFromPattern(pattern string) string {
	parts := strings.Split(pattern, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		// The singular is the name of the final variable segment.
		if strings.HasPrefix(last, "{") && strings.HasSuffix(last, "}") {
			return strings.Trim(last, "{}")
		}
	}
	return ""
}

// getCollectionPathFromPattern constructs the base gcloud collection path from a
// resource pattern string, according to AIP-122 conventions.
func getCollectionPathFromPattern(pattern string) string {
	parts := strings.Split(pattern, "/")
	var collectionParts []string
	for i := 0; i < len(parts)-1; i++ {
		// A collection identifier is a literal segment followed by a variable segment.
		if !strings.HasPrefix(parts[i], "{") && strings.HasPrefix(parts[i+1], "{") {
			collectionParts = append(collectionParts, parts[i])
		}
	}
	return strings.Join(collectionParts, ".")
}

