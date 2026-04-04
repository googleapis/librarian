# API Allowlist Schema

This document describes the schema for the API Allowlist.

## API Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | string | Is the proto directory path in github.com/googleapis/googleapis. If ServiceConfig is empty, the service config is assumed to live at this path. |
| `description` | string | Provides the information for describing an API. |
| `discovery` | string | Is the file path to a discovery document in github.com/googleapis/discovery-artifact-manager. Used by sidekick languages (Rust, Dart) as an alternative to proto files. |
| `documentation_uri` | string | Overrides the product documentation URI from the service config's publishing section. |
| `languages` | list of string | Restricts which languages can generate client libraries for this API. Empty means all languages can use this API. We should be explicit about supported languages when adding entries.<br><br>Restrictions exist for several reasons:<br>- Newer languages (Rust, Dart) skip older beta versions when stable versions exist<br>- Python has historical legacy APIs not available to other languages<br>- Some APIs (like DIREGAPIC protos) are only used by specific languages |
| `new_issue_uri` | string | Overrides the new issue URI from the service config's publishing section. |
| `no_rest_numeric_enums` | map[string]bool | Determines whether to use numeric enums in REST requests. The "No" prefix is used because the default behavior (when this field is `false` or omitted) is to generate numeric enums. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, the generator default is used. |
| `open_api` | string | Is the file path to an OpenAPI spec, currently in internal/testdata. This is not an official spec yet and exists only for Rust to validate OpenAPI support. |
| `release_level` | map[string]string | Is the release level per language. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, the generator default is used.<br><br>TODO(https://github.com/googleapis/librarian/issues/4834): Go uses "alpha", "beta", and "ga" instead of "preview" and "stable". We should standardize release level vocabulary across lanaguages. |
| `short_name` | string | Overrides the API short name from the service config's publishing section. |
| `service_config` | string | Is the service config file path override. If empty, the service config is discovered in the directory specified by Path. |
| `service_name` | string | Is a DNS-like logical identifier for the service, such as `calendar.googleapis.com`. |
| `title` | string | Overrides the API title from the service config. |
| `transports` | map[string]Transport | Defines the supported transports per language. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, all languages use GRPCRest by default. |
| `gapic_yaml` | [GAPICYamlConfig](#gapicyamlconfig-configuration) (optional) | Contains configuration imported from *_gapic.yaml files in googleapis. |

## GAPICBatchDescriptor Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `batched_field` | string | Is the request field whose values are combined when batching (e.g. "entries" in a logging API). |
| `discriminator_fields` | list of string | Are request fields that must have identical values for requests to be batched together. |
| `subresponse_field` | string | Is the response field that contains per-element results, used to split a batched response back to individual callers. |

## GAPICBatching Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `thresholds` | [GAPICBatchingThresholds](#gapicbatchingthresholds-configuration) (optional) | Defines the conditions that trigger a batch to be sent. |
| `batch_descriptor` | [GAPICBatchDescriptor](#gapicbatchdescriptor-configuration) (optional) | Describes how individual requests are combined into a batch. |

## GAPICBatchingThresholds Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `element_count_threshold` | int | Is the number of elements that triggers a batch to be sent. |
| `request_byte_threshold` | int | Is the total byte size of elements that triggers a batch to be sent. |
| `delay_threshold_millis` | int | Is the maximum time to wait before sending a batch, in milliseconds. |
| `flow_control_element_limit` | int | Is the maximum number of elements that may be outstanding at once. |
| `flow_control_byte_limit` | int | Is the maximum total byte size of elements that may be outstanding at once. |
| `flow_control_limit_exceeded_behavior` | string | Controls what happens when the flow control limit is exceeded (e.g. "ThrowException", "Block"). |

## GAPICInterface Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the fully-qualified name of the service interface (e.g. "google.cloud.speech.v1.Speech"). |
| `methods` | list of [GAPICMethod](#gapicmethod-configuration) | Contains per-method configuration such as long-running operation polling and batching settings. |

## GAPICLongRunning Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `initial_poll_delay_millis` | int64 | Is the delay before the first poll, in milliseconds. |
| `poll_delay_multiplier` | float64 | Is the multiplier applied to the poll delay after each attempt. |
| `max_poll_delay_millis` | int64 | Is the maximum poll delay, in milliseconds. |
| `total_poll_timeout_millis` | int64 | Is the total time allowed for polling before the operation is considered timed out, in milliseconds. |

## GAPICMethod Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the simple method name (e.g. "LongRunningRecognize"). |
| `long_running` | [GAPICLongRunning](#gapiclongrunning-configuration) (optional) | Contains polling configuration for long-running operations. Nil when the method is not long-running. |
| `batching` | [GAPICBatching](#gapicbatching-configuration) (optional) | Contains request batching configuration. Nil when the method does not support batching. |

## GAPICYamlConfig Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `java_package_name` | string | Is the Java package name. TODO(https://github.com/googleapis/librarian/issues/3041): move to librarian.yaml. |
| `java_interface_names` | map[string]string | Maps fully-qualified interface names to short Java class names. TODO(https://github.com/googleapis/librarian/issues/3041): move to librarian.yaml. |
| `interfaces` | list of [GAPICInterface](#gapicinterface-configuration) | Contains language-neutral per-interface, per-method configuration including long-running operation polling settings and batching config. |
