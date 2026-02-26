# librarian.yaml Schema

This document describes the schema for the librarian.yaml.

## Root Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `language` | string | Is the language for this workspace (go, python, rust). |
| `version` | string | Is the librarian tool version to use. |
| `repo` | string | Is the repository name, such as "googleapis/google-cloud-python".<br><br>TODO(https://github.com/googleapis/librarian/issues/3003): Remove this field when .repo-metadata.json generation is removed. |
| `sources` | [Sources](#sources-configuration) (optional) | References external source repositories. |
| `release` | [Release](#release-configuration) (optional) | Holds the configuration parameter for publishing and release subcommands. |
| `default` | [Default](#default-configuration) (optional) | Contains default settings for all libraries. They apply to all libraries unless overridden. |
| `libraries` | list of [Library](#library-configuration) (optional) | Contains configuration overrides for libraries that need special handling, and differ from default settings. |

## Release Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `branch` | string | Sets the name of the release branch, typically `main` |
| `ignored_changes` | list of string | Defines globs that are ignored in change analysis. |
| `preinstalled` | map[string]string | Tools defines the list of tools that must be preinstalled.<br><br>This is indexed by the well-known name of the tool vs. its path, e.g. [preinstalled] cargo = /usr/bin/cargo |
| `remote` | string | Sets the name of the source-of-truth remote for releases, typically `upstream`. |
| `roots_pem` | string | An alternative location for the `roots.pem` file. If empty it has no effect. |
| `tools` | map[string][]Tool | Defines the list of tools to install, indexed by installer. |

## Tool Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the name of the tool e.g. nox. |
| `version` | string | Is the version of the tool e.g. 1.2.4. |

## Sources Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `conformance` | [Source](#source-configuration) (optional) | Is the path to the `conformance-tests` repository, used as include directory for `protoc`. |
| `discovery` | [Source](#source-configuration) (optional) | Is the discovery-artifact-manager repository configuration. |
| `googleapis` | [Source](#source-configuration) (optional) | Is the googleapis repository configuration. |
| `protobuf` | [Source](#source-configuration) (optional) | Is the path to the `protobuf` repository, used as include directory for `protoc`. |
| `showcase` | [Source](#source-configuration) (optional) | Is the showcase repository configuration. |

## Source Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `branch` | string | Is the source's git branch to pull updates from. Unset should be interpreted as the repository default branch. |
| `commit` | string | Is the git commit hash or tag to use. |
| `dir` | string | Is a local directory path to use instead of fetching. If set, Commit and SHA256 are ignored. |
| `sha256` | string | Is the expected hash of the tarball for this commit. |
| `subpath` | string | Is a directory inside the fetched archive that should be treated as the root for operations. |

## Default Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `keep` | list of string | Lists files and directories to preserve during regeneration. |
| `output` | string | Is the directory where code is written. For example, for Rust this is src/generated. |
| `release_level` | string | Is either "stable" or "preview". |
| `tag_format` | string | Is the template for git tags, such as "{name}/v{version}". |
| `transport` | string | Is the transport protocol, such as "grpc+rest" or "grpc". |
| `dart` | [DartPackage](#dartpackage-configuration) (optional) | Contains Dart-specific default configuration. |
| `rust` | [RustDefault](#rustdefault-configuration) (optional) | Contains Rust-specific default configuration. |
| `python` | [PythonDefault](#pythondefault-configuration) (optional) | Contains Python-specific default configuration. |

## Library Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the library name, such as "secretmanager" or "storage". |
| `version` | string | Is the library version. |
| `apis` | list of [API](#api-configuration) (optional) | API specifies which googleapis API to generate from (for generated libraries). |
| `copyright_year` | string | Is the copyright year for the library. |
| `description_override` | string | Overrides the library description. |
| `keep` | list of string | Lists files and directories to preserve during regeneration. |
| `output` | string | Is the directory where code is written. This overrides Default.Output. |
| `release_level` | string | Is the release level, such as "stable" or "preview". This overrides Default.ReleaseLevel. |
| `roots` | list of string | Specifies the source roots to use for generation. Defaults to googleapis. |
| `skip_generate` | bool | Disables code generation for this library. |
| `skip_release` | bool | Disables release for this library. |
| `specification_format` | string | Specifies the API specification format. Valid values are "protobuf" (default) or "discovery". |
| `transport` | string | Is the transport protocol, such as "grpc+rest" or "grpc". This overrides Default.Transport. |
| `veneer` | bool | Indicates this library has handwritten code. A veneer may contain generated libraries. |
| `dart` | [DartPackage](#dartpackage-configuration) (optional) | Contains Dart-specific library configuration. |
| `go` | [GoModule](#gomodule-configuration) (optional) | Contains Go-specific library configuration. |
| `python` | [PythonPackage](#pythonpackage-configuration) (optional) | Contains Python-specific library configuration. |
| `rust` | [RustCrate](#rustcrate-configuration) (optional) | Contains Rust-specific library configuration. |

## API Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | string | Specifies which googleapis Path to generate from (for generated libraries). |

## DartPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_keys_environment_variables` | string | Is a comma-separated list of environment variable names that can contain API keys (e.g., "GOOGLE_API_KEY,GEMINI_API_KEY"). |
| `dependencies` | string | Is a comma-separated list of dependencies. |
| `dev_dependencies` | string | Is a comma-separated list of development dependencies. |
| `extra_imports` | string | Is additional imports to include in the generated library. |
| `include_list` | list of string | Is a list of proto files to include (e.g., "date.proto", "expr.proto"). |
| `issue_tracker_url` | string | Is the URL for the issue tracker. |
| `library_path_override` | string | Overrides the library path. |
| `name_override` | string | Overrides the package name |
| `packages` | map[string]string | Maps Dart package names to version constraints. Keys are in the format "package:googleapis_auth" and values are version strings like "^2.0.0". These are merged with default settings, with library settings taking precedence. |
| `part_file` | string | Is the path to a part file to include in the generated library. |
| `prefixes` | map[string]string | Maps protobuf package names to Dart import prefixes. Keys are in the format "prefix:google.protobuf" and values are the prefix names. These are merged with default settings, with library settings taking precedence. |
| `protos` | map[string]string | Maps protobuf package names to Dart import paths. Keys are in the format "proto:google.api" and values are import paths like "package:google_cloud_api/api.dart". These are merged with default settings, with library settings taking precedence. |
| `readme_after_title_text` | string | Is text to insert in the README after the title. |
| `readme_quickstart_text` | string | Is text to use for the quickstart section in the README. |
| `repository_url` | string | Is the URL to the repository for this package. |
| `title_override` | string | Overrides the API title. |
| `version` | string | Is the version of the dart package. |

## GoAPI Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `client_directory` | string | Is the directory where the client is generated, relative to Library.Output. |
| `disable_gapic` | bool | Determines whether to generate the GAPIC client. Also known as proto-only client, which does not define a service in the proto files. |
| `enabled_generator_features` | list of string | Provides a mechanism for enabling generator features at the API level. |
| `has_diregapic` | bool | Indicates whether generation uses DIREGAPIC (Discovery REST GAPICs). This is typically false. Used for the GCE (compute) client. |
| `import_path` | string | Is the Go import path for the API. |
| `nested_protos` | list of string | Is a list of nested proto files. |
| `no_metadata` | bool | Indicates whether to skip generating gapic_metadata.json. This is typically false. |
| `no_rest_numeric_enums` | bool | Determines whether to use numeric enums in REST requests. The "No" prefix is used because the default behavior (when this field is `false` or omitted) is to generate numeric enums |
| `path` | string | Is the source path. |
| `proto_package` | string | Is the proto package name. |
| `version_suffix` | string | Is an optional suffix appended to the version part of the GAPIC import path. |

## GoModule Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `delete_generation_output_paths` | list of string | Is a list of paths to delete before generation. |
| `go_apis` | list of [GoAPI](#goapi-configuration) (optional) | Is a list of Go-specific API configurations. |
| `module_path_version` | string | Is the version of the Go module path. |
| `nested_module` | string | Is the name of a nested module directory. |

## PythonDefault Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `common_gapic_paths` | list of string | Contains paths which are generated for any package containing a GAPIC API. These are relative to the package's output directory, and the string "{neutral-source}" is replaced with the path to the version-neutral source code (e.g. "google/cloud/run"). If a library defines its own common_gapic_paths, they will be appended to the defaults. |

## PythonPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| (embedded) | [PythonDefault](#pythondefault-configuration) |  |
| `opt_args_by_api` | map[string][]string | Contains additional options passed to the generator. In each entry, the key is the API path and the value is the list of options to pass when generating that API. Example: {"google/cloud/secrets/v1beta": ["python-gapic-name=secretmanager"]} |
| `proto_only_apis` | list of string | Contains the list of API paths which are proto-only, so should use regular protoc Python generation instead of GAPIC. |
| `name_pretty_override` | string | Allows the "name_pretty" field in .repo-metadata.json to be overridden, to reduce diffs while migrating. TODO(https://github.com/googleapis/librarian/issues/4175): remove this field. |
| `product_documentation_override` | string | Allows the "product_documentation" field in .repo-metadata.json to be overridden, to reduce diffs while migrating. TODO(https://github.com/googleapis/librarian/issues/4175): remove this field. |
| `metadata_name_override` | string | Allows the name in .repo-metadata.json (which is also used as part of the client documentation URI) to be overridden. By default it's the package name, but older packages use the API short name instead. |

## RustCrate Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| (embedded) | [RustDefault](#rustdefault-configuration) |  |
| `modules` | list of [RustModule](#rustmodule-configuration) (optional) | Specifies generation targets for veneer crates. Each module defines a source proto path, output location, and template to use. This is only used when the library has veneer: true. |
| `per_service_features` | bool | Enables per-service feature flags. |
| `module_path` | string | Is the module path for the crate. |
| `template_override` | string | Overrides the default template. |
| `package_name_override` | string | Overrides the package name. |
| `root_name` | string | Is the root name for the crate. |
| `default_features` | list of string | Is a list of default features to enable. |
| `include_list` | list of string | Is a list of proto files to include (e.g., "date.proto", "expr.proto"). |
| `included_ids` | list of string | Is a list of IDs to include. |
| `skipped_ids` | list of string | Is a list of IDs to skip. |
| `disabled_clippy_warnings` | list of string | Is a list of clippy warnings to disable. |
| `has_veneer` | bool | Indicates whether the crate has a veneer. |
| `routing_required` | bool | Indicates whether routing is required. |
| `include_grpc_only_methods` | bool | Indicates whether to include gRPC-only methods. |
| `post_process_protos` | string | Indicates whether to post-process protos. |
| `detailed_tracing_attributes` | bool | Indicates whether to include detailed tracing attributes. |
| `documentation_overrides` | list of [RustDocumentationOverride](#rustdocumentationoverride-configuration) | Contains overrides for element documentation. |
| `pagination_overrides` | list of [RustPaginationOverride](#rustpaginationoverride-configuration) | Contains overrides for pagination configuration. |
| `name_overrides` | string | Contains codec-level overrides for type and service names. |
| `discovery` | [RustDiscovery](#rustdiscovery-configuration) (optional) | Contains discovery-specific configuration for LRO polling. |

## RustDefault Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `package_dependencies` | list of [RustPackageDependency](#rustpackagedependency-configuration) (optional) | Is a list of default package dependencies. These are inherited by all libraries. If a library defines its own package_dependencies, the library-specific ones take precedence over these defaults for dependencies with the same name. |
| `disabled_rustdoc_warnings` | list of string | Is a list of rustdoc warnings to disable. |
| `generate_setter_samples` | string | Indicates whether to generate setter samples. |
| `generate_rpc_samples` | string | Indicates whether to generate RPC samples. |

## RustDiscovery Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `operation_id` | string | Is the ID of the LRO operation type (e.g., ".google.cloud.compute.v1.Operation"). |
| `pollers` | list of [RustPoller](#rustpoller-configuration) | Is a list of LRO polling configurations. |

## RustDocumentationOverride Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | string | Is the fully qualified element ID (e.g., .google.cloud.dialogflow.v2.Message.field). |
| `match` | string | Is the text to match in the documentation. |
| `replace` | string | Is the replacement text. |

## RustModule Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `disabled_rustdoc_warnings` | yaml.StringSlice | Specifies rustdoc lints to disable. An empty slice explicitly enables all warnings. |
| `documentation_overrides` | list of [RustDocumentationOverride](#rustdocumentationoverride-configuration) | Contains overrides for element documentation. |
| `extend_grpc_transport` | bool | Indicates whether the transport stub can be extended (in order to support streams). |
| `generate_setter_samples` | string | Indicates whether to generate setter samples. |
| `generate_rpc_samples` | string | Indicates whether to generate RPC samples. |
| `has_veneer` | bool | Indicates whether this module has a handwritten wrapper. |
| `included_ids` | list of string | Is a list of proto IDs to include in generation. |
| `include_grpc_only_methods` | bool | Indicates whether to include gRPC-only methods. |
| `include_list` | string | Is a list of proto files to include (e.g., "date.proto,expr.proto"). |
| `internal_builders` | bool | Indicates whether generated builders should be internal to the crate. |
| `language` | string | Can be used to select a variation of the Rust generator. For example, `rust_storage` enables special handling for the storage client. |
| `module_path` | string | Is the Rust module path for converters (e.g., "crate::generated::gapic::model"). |
| `module_roots` | map[string]string |  |
| `name_overrides` | string | Contains codec-level overrides for type and service names. |
| `output` | string | Is the directory where generated code is written (e.g., "src/storage/src/generated/gapic"). |
| `post_process_protos` | string | Contains code to post-process generated protos. |
| `root_name` | string | Is the key for the root directory in the source map. It overrides the default root, googleapis-root, used by the rust+prost generator. |
| `routing_required` | bool | Indicates whether routing is required. |
| `service_config` | string | Is the path to the service config file. |
| `skipped_ids` | list of string | Is a list of proto IDs to skip in generation. |
| `specification_format` | string | Overrides the library-level specification format. |
| `api_path` | string | Is the proto path to generate from (e.g., "google/storage/v2"). |
| `template` | string | Specifies which generator template to use. Valid values: "grpc-client", "http-client", "prost", "convert-prost", "mod". |

## RustPackageDependency Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the dependency name. It is listed first so it appears at the top of each dependency entry in YAML. |
| `ignore` | bool | Prevents this package from being mapped to an external crate. When true, references to this package stay as `crate::` instead of being mapped to the external crate name. This is used for self-referencing packages like location and longrunning. |
| `package` | string | Is the package name. |
| `source` | string | Is the dependency source. |
| `feature` | string | Is the feature name for the dependency. |
| `force_used` | bool | Forces the dependency to be used even if not referenced. |
| `used_if` | string | Specifies a condition for when the dependency is used. |

## RustPaginationOverride Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | string | Is the fully qualified method ID (e.g., .google.cloud.sql.v1.Service.Method). |
| `item_field` | string | Is the name of the field used for items. |

## RustPoller Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `prefix` | string | Is an acceptable prefix for the URL path (e.g., "compute/v1/projects/{project}/zones/{zone}"). |
| `method_id` | string | Is the corresponding method ID (e.g., ".google.cloud.compute.v1.zoneOperations.get"). |
