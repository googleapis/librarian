# gcloud Declarative YAML Schema

This document describes the schema for the gcloud Declarative YAML.

## ArgSpec Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_field` | string |  |

## Argument Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `arg_name` | string |  |
| `api_field` | string |  |
| `help_text` | string |  |
| `action` | string |  |
| `is_positional` | bool |  |
| `is_primary_resource` | bool |  |
| `request_id_field` | string |  |
| `resource_spec` | [ResourceSpec](#resourcespec-configuration) (optional) |  |
| `required` | bool |  |
| `repeated` | bool |  |
| `clearable` | bool |  |
| `type` | string |  |
| `default` | [Default](#default-configuration) |  |
| `choices` | list of [Choice](#choice-configuration) |  |
| `spec` | list of [ArgSpec](#argspec-configuration) |  |
| `resource_method_params` | map[string]string |  |

## Arguments Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `params` | list of [Argument](#argument-configuration) |  |

## Async Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `collection` | list of string |  |
| `extract_resource_result` | bool |  |

## Attribute Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `parameter_name` | string |  |
| `attribute_name` | string |  |
| `help` | string |  |
| `property` | string |  |

## Choice Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `arg_value` | string |  |
| `enum_value` | string |  |
| `help_text` | string |  |

## Command Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `release_tracks` | list of string |  |
| `auto_generated` | bool |  |
| `hidden` | bool |  |
| `help_text` | [HelpText](#helptext-configuration) |  |
| `arguments` | [Arguments](#arguments-configuration) |  |
| `request` | [Request](#request-configuration) (optional) |  |
| `async` | [Async](#async-configuration) (optional) |  |
| `response` | [Response](#response-configuration) (optional) |  |
| `update` | [UpdateConfig](#updateconfig-configuration) (optional) |  |
| `output` | [OutputConfig](#outputconfig-configuration) (optional) |  |

## Default Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `Value` | any (optional) |  |

## HelpText Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `brief` | string |  |
| `description` | string |  |
| `examples` | string |  |

## OutputConfig Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `format` | string |  |

## Request Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_version` | string |  |
| `collection` | list of string |  |
| `method` | string |  |
| `static_fields` | map[string]string |  |

## ResourceSpec Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string |  |
| `plural_name` | string |  |
| `collection` | string |  |
| `attributes` | list of [Attribute](#attribute-configuration) |  |
| `disable_auto_completers` | bool |  |

## Response Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `id_field` | string |  |

## UpdateConfig Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `read_modify_update` | bool |  |
| `disable_auto_field_mask` | bool |  |
