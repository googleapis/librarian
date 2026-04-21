# gcloud.yaml Schema

This document describes the schema for the gcloud.yaml.

## Root Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `service_name` | string | Is the name of a service. Each gcloud.yaml file should correlate to a single service config with one or more APIs defined. |
| `generate_operations` | bool (optional) | Indicates whether to generate top-level operations commands. |
| `apis` | list of [API](#api-configuration) | Describes the APIs for which to generate a gcloud surface. |
| `resource_patterns` | list of [ResourcePattern](#resourcepattern-configuration) | Describes resource patterns not included in descriptors, providing additional patterns that might be used for resource identification or command generation. |

## API Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the name of the API. This should be the API name as it appears in the normalized service config (e.g., "compute.googleapis.com"). |
| `api_version` | string | Is the API version of the API (e.g., "v1", "v2beta1"). |
| `supports_star_update_masks` | bool | Indicates that this API supports '*' updateMasks in accordance with https://google.aip.dev/134#request-message. The default is assumed to be true for AIP compliant APIs. |
| `root_is_hidden` | bool | Applies the gcloud 'hidden' flag to the root command group of the generated surface. When true, the top-level command group for this API will not appear in `--help` output by default. |
| `release_tracks` | list of ReleaseTrack | Are the gcloud release tracks this surface should appear in. This determines the visibility and stability level of the generated commands and resources. |
| `help_text` | [HelpTextRules](#helptextrules-configuration) (optional) | Contains all help text configurations for the surfaces including groups, commands, resources, and flags/arguments related to this API. |
| `output_formatting` | list of [OutputFormatting](#outputformatting-configuration) (optional) | Contains all output formatting rules for commands within this API. These rules dictate how the results of commands are displayed to the user. |
| `command_operations_config` | list of [CommandOperationsConfig](#commandoperationsconfig-configuration) (optional) | Contains long running operations config for methods within this API. This allows customization of how asynchronous operations are handled and displayed. |

## HelpTextRules Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `service_rules` | list of [HelpTextRule](#helptextrule-configuration) (optional) | Defines help text rules specifically for services. |
| `message_rules` | list of [HelpTextRule](#helptextrule-configuration) (optional) | Defines help text rules specifically for messages (resource command groups). |
| `method_rules` | list of [HelpTextRule](#helptextrule-configuration) (optional) | Defines help text rules specifically for API methods (commands). |
| `field_rules` | list of [HelpTextRule](#helptextrule-configuration) (optional) | Defines help text rules specifically for individual fields (flags/arguments). |

## HelpTextRule Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `selector` | string | Is a comma-separated list of patterns for any element such as a method, a field, an enum value. Each pattern is a qualified name of the element which may end in "*", indicating a wildcard. Wildcards are only allowed at the end and for a whole component of the qualified name, i.e. "foo.*" is ok, but not "foo.b*" or "foo.*.bar".<br><br>Wildcard may not be applicable for some elements, in those cases an 'InvalidSelectorWildcardError' error will be thrown. Additionally, some gcloud data elements expect a singular selector, if a comma separated selector string is passed, a 'InvalidSelectorList' error will be thrown.<br><br>See the API documentation for API selector details. |
| `help_text` | [HelpTextElement](#helptextelement-configuration) (optional) | Contains the detailed help text content for the selected element. |

## HelpTextElement Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `brief` | string | Is a concise, single-line summary of the help text for the CLI element. |
| `description` | string | Provides a detailed, multi-line description for the CLI element. |
| `examples` | list of string | Provides a list of string examples illustrating how to use the CLI element. |

## OutputFormatting Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `selector` | string | Is a comma-separated list of patterns for any element such as a method, a field, an enum value. Each pattern is a qualified name of the element which may end in "*", indicating a wildcard. Wildcards are only allowed at the end and for a whole component of the qualified name, i.e. "foo.*" is ok, but not "foo.b*" or "foo.*.bar".<br><br>Wildcard may not be applicable for some elements, in those cases an 'InvalidSelectorWildcardError' error will be thrown. Additionally, some gcloud data elements expect a singular selector, if a comma separated selector string is passed, a 'InvalidSelectorList' error will be thrown.<br><br>See the API documentation for API selector details. Must point to a single RPC/command. Wildcards ('*') not allowed for output formatting. |
| `format` | string | Is the output formatting string to apply. This string typically follows the `gcloud topic formats` specification (e.g., "table(name, createTime)", "json"). |

## CommandOperationsConfig Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `selector` | string | Is a comma-separated list of patterns for any element such as a method, a field, an enum value. Each pattern is a qualified name of the element which may end in "*", indicating a wildcard. Wildcards are only allowed at the end and for a whole component of the qualified name, i.e. "foo.*" is ok, but not "foo.b*" or "foo.*.bar".<br><br>Wildcard may not be applicable for some elements, in those cases an 'InvalidSelectorWildcardError' error will be thrown. Additionally, some gcloud data elements expect a singular selector, if a comma separated selector string is passed, a 'InvalidSelectorList' error will be thrown.<br><br>See the API documentation for API selector details. |
| `display_operation_result` | bool | Determines whether to display the resource result in the output of the command by default. Set to `true` to display the operation result instead of the final resource. See the gcloud documentation on async operations for more details. |

## ResourcePattern Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `type` | string | Is the resource type (e.g., "example.googleapis.com/Service"). |
| `patterns` | list of string | Is a list of resource patterns (e.g., "projects/{project}/locations/{location}/services/{service}"). These define the structure of resource names. |
| `api_version` | string | Is the API version associated with this resource pattern (e.g., "v1"). |

## HelpText Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `Brief` | string |  |
| `Description` | string |  |
| `Examples` | string |  |
