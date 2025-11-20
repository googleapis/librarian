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
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Generate generates gcloud commands for a service.
func Generate(ctx context.Context, googleapis, gcloudconfig, output string) error {
	wd, _ := os.Getwd()
	log.Printf("current working directory: %s", wd)
	log.Printf("gcloud.Generate called with googleapis: %s, gcloudconfig: %s, output: %s", googleapis, gcloudconfig, output)
	cfg, err := readGcloudConfig(gcloudconfig)
	if err != nil {
		return err
	}

	//TODO(santi): move this function outside of this function
	//TODO(santi): pass googleapisPath, and the includ_list to function
	createAPIModel := func() (*api.API, error) {

		parserConfig := &config.Config{

			General: config.GeneralConfig{

				SpecificationFormat: "protobuf",
			},

			Source: map[string]string{

				// TODO(santi): Use the googleapis path as the root for the parser.
				"local-root": ".",

				// TODO(santi): Add a way to specify the target surface instead of hardcoding it.

				"include-list": "google/cloud/parallelstore/v1/parallelstore.proto",
			},
		}

		// We use `parser.CreateModel` instead of calling the individual parsing and processing
		// functions directly because CreateModel is the designated entry point that ensures
		// the API model is not only parsed but also fully linked (cross-referenced), validated,
		// and processed with all necessary configuration overrides. This guarantees a complete
		// and consistent model for the generator without code duplication. It's worth noting that
		// we don't use all the functionality of post-processing of CreateModel, so depending
		// on our needs, if we don't find ourselves needing the additional post-processing
		// functionality, we could write our own simpler `CreateModel` function

		model, err := parser.CreateModel(parserConfig)

		if err != nil {

			return nil, fmt.Errorf("failed to create API model: %w", err)

		}

		log.Printf("****** LIBRARIAN PARSED MODEL ******\nName: %s\nPackageName: %s\nServices: %d\nMessages: %d\nEnums: %d\n", model.Name, model.PackageName, len(model.Services), len(model.Messages), len(model.Enums))

		return model, nil

	}

	model, err := createAPIModel()

	if err != nil {

		return err

	}

	// We need the short service name (e.g., "parallelstore") to use as the root
	// directory for the generated command surface. We derive this from the `name`
	// of the first API definition in the `gcloud.yaml` file.
	// Example: "Parallelstore" -> "parallelstore"
	if len(cfg.APIs) == 0 {
		return fmt.Errorf("no APIs defined in gcloud.yaml")
	}
	shortServiceName := strings.ToLower(cfg.APIs[0].Name)

	// The final output will be placed in a directory structure like:
	// `{outdir}/{shortServiceName}/surface/`
	surfaceDir := filepath.Join(output, shortServiceName, "surface")

	// gcloud commands are resource-centric commands (e.g., `gcloud parallelstore instances create`),
	// so we first need to group all the API methods by the resource they operate on.
	// We'll create a map where the key is the resource's collection ID (e.g., "instances")
	// and the value is a list of methods that act on that resource.
	methodsByResource := make(map[string][]*api.Method)
	log.Printf("methodsByResource initialized: %+v", methodsByResource)

	// We iterate through all services and their methods defined in the API model.
	for _, service := range model.Services {
		log.Printf("Processing service: %s", service.Name)
		for _, method := range service.Methods {
			log.Printf("  Processing method: %s", method.Name)
			// For each method, we determine the plural name of the resource it operates on.
			// This plural name (e.g., "instances") will serve as our collection ID.
			// Example: For the `CreateInstance` method, this will return "instances".
			collectionID := getPluralName(method, model)
			log.Printf("    Determined collectionID for method %s: %s", method.Name, collectionID)

			// If a collection ID is found, we add the method to our map.
			if collectionID != "" {
				log.Printf("    Adding method %s to collectionID %s", method.Name, collectionID)
				methodsByResource[collectionID] = append(methodsByResource[collectionID], method)
			}
		}
	}
	log.Printf("Finished processing all services and methods. Final methodsByResource: %+v", methodsByResource)
	// Now that we have grouped the methods by resource, we can generate the
	// command files for each resource.
	log.Printf("Iterating through methodsByResource")
	for collectionID, methods := range methodsByResource {
		log.Printf("collectionID: %s, methods: %s", collectionID, methods)

		// The `generateResourceCommands` function will handle the creation of the
		// directory structure and YAML files for this specific resource.
		err := generateResourceCommands(collectionID, methods, surfaceDir, cfg, model)
		if err != nil {
			return err
		}
	}
	return nil
}

// readGcloudConfig loads the gcloud configuration from a gcloud.yaml file.
func readGcloudConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read gcloud config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse gcloud config YAML: %w", err)
	}
	return &cfg, nil
}

// generateResourceCommands creates the directory structure and YAML files for a
// single resource's commands (e.g., create, delete, list).
//
// For a given collectionID like "instances", this function will create a directory
// `instances/` and populate it with `create.yaml`, `delete.yaml`, etc.
func generateResourceCommands(collectionID string, methods []*api.Method, baseDir string, cfg *Config, model *api.API) error {
	// The main directory for the resource is named after its collection ID.
	// Example: `{baseDir}/instances`
	resourceDir := filepath.Join(baseDir, collectionID)

	// Gcloud commands are defined in a `_partials` directory. This allows
	// for sharing command definitions across different release tracks (GA, Beta, Alpha).
	partialsDir := filepath.Join(resourceDir, "_partials")
	if err := os.MkdirAll(partialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create partials directory for %q: %w", collectionID, err)
	}

	// We iterate through each method associated with this resource.
	for _, method := range methods {
		// We map the API method name to a standard gcloud command verb.
		// Example: `CreateInstance` -> "create"
		verb := getVerb(method.Name)

		// We construct the complete command definition from the API method.
		// This involves generating all the arguments, help text, and request details.
		cmd := newCommand(method, cfg, model)

		// in gcloud convention, the final YAML file must contain a list of commands,
		// even if there is only one.
		cmdList := []*Command{cmd}

		// We create the main command file (e.g., `create.yaml`). This file doesn't
		// contain the command definition itself, but rather a directive that tells
		// gcloud to look in the `_partials` directory.
		mainCmdPath := filepath.Join(resourceDir, fmt.Sprintf("%s.yaml", verb))
		if err := os.WriteFile(mainCmdPath, []byte("_PARTIALS_: true\n"), 0644); err != nil {
			return fmt.Errorf("failed to write main command file for %q: %w", method.Name, err)
		}

		// We define the name of the partial file, which includes the verb and the
		// release track. For now, we are hardcoding to the "ga" track.
		// Example: "_create_ga.yaml"
		// TODO(santi): generate multiple tracks
		track := "ga"
		partialFileName := fmt.Sprintf("_%s_%s.yaml", verb, track)
		partialCmdPath := filepath.Join(partialsDir, partialFileName)

		// We marshal the command definition struct into YAML format.
		b, err := yaml.Marshal(cmdList)
		if err != nil {
			return fmt.Errorf("failed to marshal partial command for %q: %w", method.Name, err)
		}

		// Finally, we write the generated YAML to the partial file.
		if err := os.WriteFile(partialCmdPath, b, 0644); err != nil {
			return fmt.Errorf("failed to write partial command file for %q: %w", method.Name, err)
		}
	}
	return nil
}

// newCommand constructs a single gcloud command definition from an API method.
// This function assembles all the necessary pieces: help text, arguments,
// request details, and async configuration.
func newCommand(method *api.Method, cfg *Config, model *api.API) *Command {
	// We look up the help text and API definition for this specific method in the
	// `gcloud.yaml` configuration file.
	rule := findHelpTextRule(method, cfg)
	apiDef := findAPI(method, cfg)
	//TODO: parse exaples from `gcloud.yaml`

	// We initialize the command with some default values.
	cmd := &Command{
		AutoGenerated: true,
		Hidden:        true,
	}

	// If a help text rule was found in the config, we apply it to the command.
	if rule != nil {
		cmd.HelpText = HelpText{
			Brief:       rule.HelpText.Brief,
			Description: rule.HelpText.Description,
			Examples:    rule.HelpText.Examples[0], //TODO(santi): add all examples
		}
	}

	// If an API definition was found, we apply the specified release tracks.
	if apiDef != nil {
		for _, track := range apiDef.ReleaseTracks {
			cmd.ReleaseTracks = append(cmd.ReleaseTracks, string(track))
		}
	}

	// The core of the command generation happens here: we generate the arguments,
	// request details, and async configuration.
	cmd.Arguments = newArguments(method, cfg, model)
	cmd.Request = newRequest(method, cfg, model)
	if method.OperationInfo != nil {
		cmd.Async = newAsync(method, cfg)
	}

	return cmd
}

// newArguments generates the set of arguments for a command by parsing the
// fields of the method's request message.
func newArguments(method *api.Method, cfg *Config, model *api.API) Arguments {
	args := Arguments{}
	if method.InputType == nil {
		return args
	}

	// We iterate over each field in the method's request message (e.g., `CreateInstanceRequest`).
	for _, field := range method.InputType.Fields {
		// The "parent" field is a special case. Its information is captured by the
		// primary resource argument, so we skip it here to avoid creating a redundant flag.
		if field.Name == "parent" {
			continue
		}

		// We check if the current field represents the primary resource of the command.
		// For example, in a `CreateInstance` method, this would be the `instance_id` field.
		if isPrimaryResource(field, method) {
			// If it is the primary resource, we generate a special positional argument for it.
			param := newPrimaryResourceParam(field, method, model, cfg)
			args.Params = append(args.Params, param)
			continue
		}

		// For all other fields, we generate a standard flag argument. If the field
		// is a nested message, its fields will be "flattened" into top-level flags.
		// For example, a field `instance.description` becomes the `--description` flag.
		addFlattenedParams(field, field.JSONName, &args, cfg, model)
	}
	return args
}

// addFlattenedParams recursively processes a field and its sub-fields to generate
// a flat list of command-line flags. This is necessary for nested messages in
// the request proto.
func addFlattenedParams(field *api.Field, prefix string, args *Arguments, cfg *Config, model *api.API) {
	// We skip fields that are marked as `OUTPUT_ONLY` in the proto, as these are
	// not meant to be provided by the user. We also skip the "name" field, as it's
	// handled by the primary resource argument.
	if isOutputOnly(field) || field.Name == "name" {
		return
	}

	// If the field is a nested message (and not a map, which is handled differently),
	// we need to recurse into its fields. This is the "flattening" process.
	// For example, in the Parallelstore API, the `CreateInstanceRequest` message
	// has a field named `instance` which is of type `Instance`. The `Instance`
	// message itself has fields like `description` and `capacity_gib`.
	// This block will recurse into the `Instance` message's fields.
	if field.MessageType != nil && !field.Map {
		for _, f := range field.MessageType.Fields {
			// The prefix is updated to create a dot-separated path for the `api_field`.
			// Continuing the example: when processing the `capacity_gib` field inside the
			// `Instance` message, the prefix will become "instance.capacityGib". This
			// results in a `--capacity-gib` flag that maps to the correct nested field.
			addFlattenedParams(f, fmt.Sprintf("%s.%s", prefix, f.JSONName), args, cfg, model)
		}
		return
	}

	// If the field is a scalar, map, or enum, we generate a parameter for it.
	param := newParam(field, prefix, cfg, model)
	args.Params = append(args.Params, param)
}

// newParam creates a single command-line argument (a `Param` struct) from a proto field.
func newParam(field *api.Field, apiField string, cfg *Config, model *api.API) Param {
	// We initialize the Param with the basic information derived from the field.
	param := Param{
		// The command-line flag name is the kebab-case version of the field name.
		// Example: `requestId` -> `request-id`
		ArgName: ToKebabCase(field.Name),
		// The `api_field` is the dot-separated path to the field in the request message.
		APIField: apiField,
		// We determine if the field is required based on the `(google.api.field_behavior)` annotation.
		Required: field.DocumentAsRequired(),
		// We check if the field is repeated in the proto.
		Repeated: field.Repeated,
	}

	// Now we handle the different types of fields.
	if field.ResourceReference != nil {
		// If the field is a resource reference (e.g., a field for a network), we
		// generate a `ResourceSpec` for it. This tells gcloud how to parse the
		// resource name provided by the user.
		param.ResourceSpec = newResourceReferenceSpec(field, model, cfg)
		param.ResourceMethodParams = map[string]string{
			apiField: "{__relative_name__}",
		}
	} else if field.Map {
		// If the field is a map, we generate a spec for its key-value pairs.
		param.Repeated = true
		param.Spec = []ArgSpec{
			{APIField: "key"},
			{APIField: "value"},
		}
	} else if field.EnumType != nil {
		// If the field is an enum, we generate a list of choices for the flag.
		for _, v := range field.EnumType.Values {
			// We skip the default "UNSPECIFIED" value, as it's not a valid choice for the user.
			if strings.HasSuffix(v.Name, "_UNSPECIFIED") {
				continue
			}
			param.Choices = append(param.Choices, Choice{
				ArgValue:  ToKebabCase(v.Name),
				EnumValue: v.Name,
			})
		}
	} else {
		// If it's a scalar type (string, int, bool, etc.), we map its proto type
		// to the corresponding gcloud type.
		param.Type = getGcloudType(field.Typez)
	}

	// We try to find help text for this field in the `gcloud.yaml` config.
	// If none is found, we generate a default help text.
	if rule := findFieldHelpTextRule(field, cfg); rule != nil {
		param.HelpText = rule.HelpText.Brief
	} else {
		// TODO: improve default help text inference
		param.HelpText = fmt.Sprintf("Value for the `%s` field.", ToKebabCase(field.Name))
	}
	return param
}

// newPrimaryResourceParam creates the main positional resource argument for a command.
// This is the argument that represents the resource being acted upon (e.g., the instance name).
func newPrimaryResourceParam(field *api.Field, method *api.Method, model *api.API, cfg *Config) Param {
	// We first need to get the full resource definition for the method.
	resource := getResourceForMethod(method, model)
	pattern := ""
	if resource != nil && len(resource.Pattern) > 0 {
		pattern = resource.Pattern[0]
	}

	// We construct the gcloud collection path from the resource's pattern string.
	// Example: `projects/{project}/locations/{location}/instances/{instance}` -> `projects.locations.instances`
	collectionPath := getCollectionPathFromPattern(pattern)
	shortServiceName := strings.Split(cfg.ServiceName, ".")[0]

	// We determine the singular name of the resource.
	// For `Create` methods, this comes from the `_id` field. For others, it's the `name` field.
	resourceName := toSnakeCase(strings.TrimSuffix(field.Name, "_id"))
	if field.Name == "name" {
		resourceName = getSingularFromPattern(pattern)
	}

	// We generate a helpful help text based on whether the command is a `Create` command or not.
	helpText := fmt.Sprintf("The %s to create.", resourceName)
	if !strings.HasPrefix(method.Name, "Create") {
		helpText = fmt.Sprintf("The %s to operate on.", resourceName)
	}

	// We assemble the final `Param` struct with all the necessary information for a primary resource.
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
// another resource type (e.g., a `--network` flag).
func newResourceReferenceSpec(field *api.Field, model *api.API, cfg *Config) *ResourceSpec {
	// We iterate through all the resource definitions in the API model to find the
	// one that matches the type of our resource reference.
	for _, def := range model.ResourceDefinitions {
		if def.Type == field.ResourceReference.Type {
			if len(def.Pattern) == 0 {
				return nil // We cannot proceed without a pattern.
			}
			pattern := def.Pattern[0]

			// We determine the plural name, using the explicit `plural` field if available,
			// and falling back to parsing the pattern otherwise.
			pluralName := def.Plural
			if pluralName == "" {
				pluralName = getPluralFromPattern(pattern)
			}

			// We determine the singular name from the pattern.
			name := getSingularFromPattern(pattern)

			// We construct the full gcloud collection path for the referenced resource.
			shortServiceName := strings.Split(cfg.ServiceName, ".")[0]
			baseCollectionPath := getCollectionPathFromPattern(pattern)
			fullCollectionPath := fmt.Sprintf("%s.%s", shortServiceName, baseCollectionPath)

			// We assemble and return the `ResourceSpec`.
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
// "projects/{project}/locations/{location}") and extracts the attributes
// that make up the resource's name.
func newAttributesFromPattern(pattern string) []Attribute {
	var attributes []Attribute
	parts := strings.Split(pattern, "/")

	// We iterate over the segments of the pattern.
	for i, part := range parts {
		// A variable segment is enclosed in curly braces.
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			// The `attribute_name` is the name of the variable (e.g., "project").
			name := strings.Trim(part, "{}")
			var parameterName string

			// The `parameter_name` is derived from the preceding literal segment
			// (e.g., "projects" -> "projectsId"). This is a gcloud convention.
			if i > 0 {
				parameterName = parts[i-1] + "Id"
			} else {
				// This is a fallback for the unlikely case that a pattern starts with a variable.
				parameterName = name + "sId"
			}

			attr := Attribute{
				AttributeName: name,
				ParameterName: parameterName,
				Help:          fmt.Sprintf("The %s id of the {resource} resource.", name),
			}

			// If the attribute is a project, we add the standard gcloud property fallback,
			// so users don't have to specify `--project` if it's already configured.
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
	// For `Create` methods, the primary resource is identified by a field named
	// in the format "{resource}_id" (e.g., "instance_id").
	if strings.HasPrefix(method.Name, "Create") {
		resourceName := getResourceName(method)
		if resourceName != "" && field.Name == toSnakeCase(resourceName)+"_id" {
			return true
		}
	}
	// For `Get`, `Delete`, and `Update` methods, the primary resource is identified
	// by a field named "name", which holds the full resource name.
	if (strings.HasPrefix(method.Name, "Get") || strings.HasPrefix(method.Name, "Delete") || strings.HasPrefix(method.Name, "Update")) && field.Name == "name" {
		return true
	}
	return false
}

// getResourceName extracts the name of the resource from a method's input message.
// For example, for `CreateInstanceRequest`, it would return "Instance".
func getResourceName(method *api.Method) string {
	for _, f := range method.InputType.Fields {
		if msg := f.MessageType; msg != nil && msg.Resource != nil {
			return msg.Name
		}
	}
	return ""
}

// getResourceForMethod finds the `api.Resource` definition associated with a method.
// This is a crucial function for linking a method to the resource it operates on.
func getResourceForMethod(method *api.Method, model *api.API) *api.Resource {
	log.Printf("****** LIBRARIAN GET RESOURCE FOR METHOD ******\n%+v\n", method)
	// Strategy 1: For `Create` and `Update` methods, the request message usually
	// contains a field that is the resource message itself. We look for that first.
	if method.InputType != nil {
		for _, f := range method.InputType.Fields {
			if msg := f.MessageType; msg != nil && msg.Resource != nil {
				log.Printf("  Strategy 1: Found resource %s from input type field %s", msg.Resource.Type, f.Name)
				return msg.Resource
			}
		}
	} else {
		log.Printf("  Strategy 1: Method input type is nil for method: %s", method.Name)
	}

	// Strategy 2: For `Get`, `Delete`, and `List` methods, the request message
	// usually contains a "name" or "parent" field with a `resource_reference`.
	var resourceType string
	if method.InputType != nil {
		for _, field := range method.InputType.Fields {
			if (field.Name == "name" || field.Name == "parent") && field.ResourceReference != nil {
				resourceType = field.ResourceReference.Type
				if resourceType == "" {
					resourceType = field.ResourceReference.ChildType
				}
				log.Printf("  Strategy 2: Found resource reference type %s from field %s", resourceType, field.Name)
				break
			}
		}
	}

	// If we found a resource type, we now need to look up its full definition
	// in the API model.
	if resourceType != "" {
		for _, msg := range model.Messages {
			if msg.Resource != nil && msg.Resource.Type == resourceType {
				log.Printf("  Strategy 2: Found resource %s in model messages", msg.Resource.Type)
				return msg.Resource
			}
		}
		for _, def := range model.ResourceDefinitions {
			if def.Type == resourceType {
				log.Printf("  Strategy 2: Found resource definition %s in model resource definitions", def.Type)
				return def
			}
		}
	}

	log.Printf("getResourceForMethod: No resource found for method %s", method.Name)
	return nil
}

// newRequest creates the `Request` part of the command definition.
func newRequest(method *api.Method, cfg *Config, model *api.API) *Request {
	return &Request{
		APIVersion: apiVersion(cfg),
		Collection: []string{fmt.Sprintf("parallelstore.projects.locations.%s", getPluralName(method, model))},
	}
}

// newAsync creates the `Async` part of the command definition for long-running operations.
func newAsync(method *api.Method, cfg *Config) *Async {
	return &Async{
		Collection: []string{"parallelstore.projects.locations.operations"},
	}
}

// apiVersion extracts the API version from the configuration.
func apiVersion(cfg *Config) string {
	if len(cfg.APIs) > 0 {
		return cfg.APIs[0].APIVersion
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

// getPluralName determines the plural name of a resource. It follows a clear
// hierarchy of truth: first, the explicit `plural` field in the resource
// definition, and second, inference from the resource pattern.
func getPluralName(method *api.Method, model *api.API) string {
	resource := getResourceForMethod(method, model)
	log.Printf("getPluralName: resource for method %s: %+v", method.Name, resource)
	if resource != nil {
		// The `plural` field in the `(google.api.resource)` annotation is the
		// most authoritative source.
		if resource.Plural != "" {
			log.Printf("getPluralName: using explicit plural from resource: %s", resource.Plural)
			return resource.Plural
		}
		// If the `plural` field is not present, we fall back to inferring the
		// plural name from the resource's pattern string, as per AIP-122.
		if len(resource.Pattern) > 0 {
			log.Printf("getPluralName: inferring plural from pattern: %s", resource.Pattern[0])
			return getPluralFromPattern(resource.Pattern[0])
		}
	}
	log.Printf("getPluralName: no plural name found for method %s", method.Name)
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
		// For non-standard methods, we just use the snake_case version of the method name.
		return toSnakeCase(methodName)
	}
}

// findAPI finds the API definition from the config that applies to the current method.
func findAPI(method *api.Method, cfg *Config) *API {
	if cfg.APIs == nil {
		return nil
	}
	// This implementation currently assumes a single API definition in the config.
	if len(cfg.APIs) > 0 {
		return &cfg.APIs[0]
	}
	return nil
}

// findHelpTextRule finds the help text rule from the config that applies to the current method.
func findHelpTextRule(method *api.Method, cfg *Config) *HelpTextRule {
	if cfg.APIs == nil {
		return nil
	}
	for _, api := range cfg.APIs {
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
func findFieldHelpTextRule(field *api.Field, cfg *Config) *HelpTextRule {
	if cfg.APIs == nil {
		return nil
	}
	for _, api := range cfg.APIs {
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

// isOutputOnly checks if a field is marked as output-only in the proto.
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
// Per AIP-122, the plural is the literal segment before the final variable segment.
// Example: `.../instances/{instance}` -> "instances"
func getPluralFromPattern(pattern string) string {
	parts := strings.Split(pattern, "/")
	if len(parts) >= 2 {
		if strings.HasPrefix(parts[len(parts)-1], "{") {
			return parts[len(parts)-2]
		}
	}
	return ""
}

// getSingularFromPattern infers the singular name of a resource from its pattern string.
// The singular is the name of the final variable segment.
// Example: `.../instances/{instance}` -> "instance"
func getSingularFromPattern(pattern string) string {
	parts := strings.Split(pattern, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if strings.HasPrefix(last, "{") && strings.HasSuffix(last, "}") {
			return strings.Trim(last, "{}")
		}
	}
	return ""
}

// getCollectionPathFromPattern constructs the base gcloud collection path from a
// resource pattern string, according to AIP-122 conventions.
// It joins the literal collection identifiers with dots.
// Example: `projects/{project}/locations/{location}/instances/{instance}` -> `projects.locations.instances`
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
