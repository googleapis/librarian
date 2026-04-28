# gcloud Declarative YAML Schema

This document describes the schema for the gcloud Declarative YAML.

## ArgSpec Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_field` | string | Is the sub-field name within the structured argument value, typically key or value for map arguments. |

## Argument Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `arg_name` | string | Is the name of the argument as it appears to the user, such as instance-id. For flags gcloud prepends --. |
| `api_field` | string | Is the dot-separated path into the API request message that receives this argument's value, for example instance.displayName. |
| `help_text` | string | Is the help text shown for the argument in gcloud help and error messages. |
| `action` | string | Overrides the argparse action used for the argument, for example store_true or store_true_false. Typically used with boolean fields. |
| `is_positional` | bool | Makes the argument positional rather than a flag. |
| `is_primary_resource` | bool | Marks this argument as the primary resource the command operates on. At most one argument per command may set this. |
| `request_id_field` | string | Is the request-message field that receives the user-supplied resource ID on a create command, for example instanceId. |
| `resource_spec` | [ResourceSpec](#resourcespec-configuration) (optional) | Declares this argument as a gcloud concept resource and describes its collection and attributes. |
| `required` | bool | Makes the argument mandatory. gcloud rejects invocations that leave it unset. |
| `repeated` | bool | Accepts more than one value for the argument and maps to a repeated API field. |
| `clearable` | bool | Causes gcloud to also generate companion flags such as<br>--clear-..., --add-..., and --remove-... on update commands. |
| `type` | string | Selects the gcloud argument parser, for example long, double, or a fully-qualified reference to an ArgDict or ArgList parser. |
| `default` | [Default](#default-configuration) | Is the value used when the user does not supply the argument. |
| `choices` | list of [Choice](#choice-configuration) | Is the fixed set of values the user can supply. Used for enum-typed fields; each choice maps an arg_value to an enum_value. |
| `spec` | list of [ArgSpec](#argspec-configuration) | Lists the sub-fields of a structured argument such as a map. For a map field, Spec typically contains entries for key and value. |
| `resource_method_params` | map[string]string | Maps API method parameter names to attribute names on a resource argument. Use it when the API method uses a non-standard parameter name for the resource. |

## Arguments Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `params` | list of [Argument](#argument-configuration) | Is the ordered list of arguments the command accepts. Each entry becomes either a positional argument or a flag. |

## Async Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `collection` | list of string | Is the gcloud API collection of the operation resource, for example service.projects.locations.operations. |
| `extract_resource_result` | bool | Unwraps the target resource from the operation's response field when the operation completes. Set when the LRO response type matches the resource being created or updated. |

## Attribute Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `attribute_name` | string | Is the user-facing name used for the generated flag, for example project. |
| `help` | string | Is the help text shown for the generated flag. |
| `parameter_name` | string | Is the API parameter name that this attribute maps to in the request URL, for example projectsId. |
| `property` | string | Is the gcloud core property consulted when the flag is not supplied, for example core/project. |

## Choice Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `arg_value` | string | Is the value the user types on the command line, typically lowercased and kebab-cased. |
| `enum_value` | string | Is the API enum value that ArgValue maps to. |
| `help_text` | string | Is the per-choice help shown in gcloud help. |

## Command Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `release_tracks` | list of string | Lists the gcloud release tracks the command is registered in. Valid values are "ALPHA", "BETA", and "GA". |
| `auto_generated` | bool | Marks the command as produced by tooling rather than written by hand. It is informational only. |
| `hidden` | bool | Removes the command from gcloud help and the CLI completer. Hidden commands remain runnable for users who know the full path. |
| `help_text` | [HelpText](#helptext-configuration) | Supplies the brief, description, and examples shown for this command in gcloud help. |
| `arguments` | [Arguments](#arguments-configuration) | Declares the positional arguments and flags the command accepts. |
| `request` | [Request](#request-configuration) (optional) | Describes the API request issued when the command runs. It is omitted for commands that do not call an API. |
| `async` | [Async](#async-configuration) (optional) | Describes how gcloud polls the long-running operation returned by the API. Present when the method returns an LRO. |
| `response` | [Response](#response-configuration) (optional) | Describes how gcloud interprets the API response, for example which field holds the resource ID for --uri support on list commands. |
| `update` | [Update](#update-configuration) (optional) | Customizes how update commands build and send patch requests, including field-mask handling. |
| `output` | [Output](#output-configuration) (optional) | Customizes how command output is formatted, for example a table(...) projection for list commands. |

## Default Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `Value` | any (optional) | Holds the default value as it should appear in YAML. A nil pointer means no default was specified. |

## HelpText Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `brief` | string | Is a one-line summary of the command shown in command lists and at the top of gcloud help output. |
| `description` | string | Is the detailed help shown under the DESCRIPTION section of gcloud help. |
| `examples` | string | Are the command examples shown under the EXAMPLES section of gcloud help. The literal {command} is expanded to the full command path. |

## Output Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `format` | string | Is a gcloud resource projection expression applied to the command output, for example table(name, state). |

## Request Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_version` | string | Selects the API version to call, for example v1 or v1beta. |
| `collection` | list of string | Is the list of gcloud API collections the command operates on. Multiple entries support AIP-127 multi-pattern resources. |
| `method` | string | Is the API method name. When empty, gcloud infers the method from the command type (for example, list for a list command). |
| `static_fields` | map[string]string | Sets request fields to fixed values regardless of user input. Keys are dot-separated field paths in the request message. |

## ResourceSpec Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `attributes` | list of [Attribute](#attribute-configuration) | Lists the resource's path components (project, location, resource ID, and so on) in the order they appear in the collection pattern. |
| `collection` | string | Is the fully-qualified gcloud collection identifier, for example parallelstore.projects.locations.instances. |
| `disable_auto_completers` | bool | Turns off automatic tab-completion for this resource. Set for cross-API references where the completer would need to call a different service. |
| `name` | string | Is the singular resource name, for example instance. |
| `plural_name` | string | Is the plural resource name used in help text, for example instances. |

## Response Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `id_field` | string | Is the response field that holds the resource identifier. On list commands it enables the --uri flag. |

## Update Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `disable_auto_field_mask` | bool | Turns off gcloud's automatic field-mask computation from the set of user-specified flags. Set it when the command builds the field mask itself. |
| `read_modify_update` | bool | Makes gcloud fetch the current resource, apply the user's changes, and send the result back. Use it for APIs that do not accept partial updates. |
