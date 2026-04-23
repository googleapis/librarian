# librarian.yaml Schema

This document describes the schema for the librarian.yaml.

## Root Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `language` | string | Is the language for this workspace (go, python, rust). |
| `version` | string | Is the librarian tool version to use. |
| `repo` | string | Is the repository name, such as "googleapis/google-cloud-python". It is used for:<br>- Providing to the Java GAPIC generator for observability features.<br>- Generating the .repo-metadata.json. |
| `sources` | [Sources](#sources-configuration) (optional) | References external source repositories. |
| `tools` | [Tools](#tools-configuration) (optional) | Defines required tools. |
| `release` | [Release](#release-configuration) (optional) | Holds the configuration parameter for publishing and release subcommands. |
| `default` | [Default](#default-configuration) (optional) | Contains default settings for all libraries. They apply to all libraries unless overridden. |
| `libraries` | list of [Library](#library-configuration) (optional) | Contains configuration overrides for libraries that need special handling, and differ from default settings. |

## Release Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `ignored_changes` | list of string | Defines globs that are ignored in change analysis. |
| `preinstalled` | map[string]string | Tools defines the list of tools that must be preinstalled.<br><br>This is indexed by the well-known name of the tool vs. its path, e.g. [preinstalled] cargo = /usr/bin/cargo |
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

## Tools Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `cargo` | list of [CargoTool](#cargotool-configuration) (optional) | Defines tools to install via cargo. |
| `npm` | list of [NPMTool](#npmtool-configuration) (optional) | Defines tools to install via npm. |
| `pip` | list of [PipTool](#piptool-configuration) (optional) | Defines tools to install via pip. |
| `go` | list of [GoTool](#gotool-configuration) (optional) | Defines tools to install via go. |

## CargoTool Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the cargo package name. |
| `version` | string | Is the version to install. |

## NPMTool Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the npm package name. |
| `version` | string | Is the version to install. |
| `package` | string | Is the URL or path of the package to install. |
| `checksum` | string | Is the SHA256 checksum of the package. |
| `build` | list of string | Defines the commands to run to build the tool after installation. |

## PipTool Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the pip package name. |
| `version` | string | Is the version to install. |
| `package` | string | Is the pip install specifier (e.g., "pkg@git+https://..."). |

## GoTool Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the go module name. |
| `version` | string | Is the version to install. |

## Default Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `keep` | list of string | Lists files and directories to preserve during regeneration. |
| `output` | string | Is the directory where code is written. For example, for Rust this is src/generated. |
| `tag_format` | string | Is the template for git tags, such as "{name}/v{version}". |
| `dotnet` | [DotnetPackage](#dotnetpackage-configuration) (optional) | Contains .NET-specific default configuration. |
| `dart` | [DartPackage](#dartpackage-configuration) (optional) | Contains Dart-specific default configuration. |
| `java` | [JavaModule](#javamodule-configuration) (optional) | Contains Java-specific default configuration. |
| `nodejs` | [NodejsPackage](#nodejspackage-configuration) (optional) | Contains Node.js-specific default configuration. |
| `rust` | [RustDefault](#rustdefault-configuration) (optional) | Contains Rust-specific default configuration. |
| `python` | [PythonDefault](#pythondefault-configuration) (optional) | Contains Python-specific default configuration. |
| `swift` | [SwiftDefault](#swiftdefault-configuration) (optional) | Contains Swift-specific default configuration. |

## Library Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is the library name, such as "secretmanager" or "storage". |
| `version` | string | Is the library version. |
| `preview` | [Library](#library-configuration) (optional) | Signifies that this API has a preview variant, and it contains overrides specific to the preview API variant. This is merged with the containing [Library], preferring those [Library.Preview] values that are set over their counterpart in the containing configuration.<br><br>The most common overrides are [Library.Version] and [Library.APIs], with the former containing a pre-release version based on the containing version of the stable client, and the latter being a subset of APIs, typically omitting alpha and beta paths.<br><br>The [Library.Output] may be a different location and derived on a per-language basis, but will not be serialized in the configuration.<br><br>Important: The boolean fields [Library.SkipRelease] and [Library.SkipGenerate] set in the containing config will always be applied to the Preview library as well, because previews are related to the stable library and should be managed identically. |
| `apis` | list of [API](#api-configuration) (optional) | API specifies which googleapis API to generate from (for generated libraries). |
| `copyright_year` | string | Is the copyright year for the library. |
| `description_override` | string | Overrides the library description. |
| `title_override` | string | Overrides the title used in README generation. |
| `keep` | list of string | Lists files and directories to preserve during regeneration. |
| `output` | string | Is the directory where code is written. This overrides Default.Output. |
| `roots` | list of string | Specifies the source roots to use for generation. Defaults to googleapis. |
| `skip_generate` | bool | Disables code generation for this library. |
| `skip_release` | bool | Disables release for this library. |
| `specification_format` | string | Specifies the API specification format. Valid values are "protobuf" (default) or "discovery". |
| `dotnet` | [DotnetPackage](#dotnetpackage-configuration) (optional) | Contains .NET-specific library configuration. |
| `dart` | [DartPackage](#dartpackage-configuration) (optional) | Contains Dart-specific library configuration. |
| `go` | [GoModule](#gomodule-configuration) (optional) | Contains Go-specific library configuration. |
| `java` | [JavaModule](#javamodule-configuration) (optional) | Contains Java-specific library configuration. |
| `nodejs` | [NodejsPackage](#nodejspackage-configuration) (optional) | Contains Node.js-specific library configuration. |
| `python` | [PythonPackage](#pythonpackage-configuration) (optional) | Contains Python-specific library configuration. |
| `rust` | [RustCrate](#rustcrate-configuration) (optional) | Contains Rust-specific library configuration. |
| `swift` | [SwiftPackage](#swiftpackage-configuration) (optional) | Contains Swift-specific library configuration. |

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

## DotnetCsproj Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `snippets` | [DotnetCsprojSnippets](#dotnetcsprojsnippets-configuration) (optional) | Contains XML fragments for .csproj files. |
| `integration_tests` | [DotnetCsprojSnippets](#dotnetcsprojsnippets-configuration) (optional) | Contains configuration for integration test projects. |

## DotnetCsprojSnippets Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `embedded_resources` | list of string | Is a list of glob patterns for embedded resources. |

## DotnetPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `additional_service_descriptors` | list of string | Is a list of extra service descriptors to include. |
| `csproj` | [DotnetCsproj](#dotnetcsproj-configuration) (optional) | Contains configuration for .csproj file generation and overrides. |
| `dependencies` | map[string]string | Maps NuGet package IDs to version strings. |
| `generator` | string | Overrides the default generator (e.g., "proto"). |
| `package_group` | list of string | Lists packages that must be released together. |
| `postgeneration` | list of [DotnetPostgeneration](#dotnetpostgeneration-configuration) (optional) | Contains post-generation shell commands or extra protos. |
| `pregeneration` | list of [DotnetPregeneration](#dotnetpregeneration-configuration) (optional) | Contains declarative proto mutations. |

## DotnetPostgeneration Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `run` | string | Is a shell command to execute. |
| `extra_proto` | string | Is an extra proto file to compile. |

## DotnetPregeneration Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `rename_message` | [DotnetRenameMessage](#dotnetrenamemessage-configuration) (optional) | Renames a message. |
| `remove_field` | [DotnetRemoveField](#dotnetremovefield-configuration) (optional) | Removes a field from a message. |
| `rename_rpc` | [DotnetRenameRPC](#dotnetrenamerpc-configuration) (optional) | Renames an RPC. |

## DotnetRemoveField Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `message` | string |  |
| `field` | string |  |

## DotnetRenameMessage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `from` | string |  |
| `to` | string |  |

## DotnetRenameRPC Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `from` | string |  |
| `to` | string |  |
| `wire_name` | string |  |

## GoAPI Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `client_package` | string | Is the package name of the generated client. |
| `diregapic` | bool | Indicates whether generation uses DIREGAPIC (Discovery REST GAPICs). This is typically false. Used for the GCE (compute) client. |
| `enabled_generator_features` | list of string | Provides a mechanism for enabling generator features at the API level. |
| `import_path` | string | Is the Go import path for the API. |
| `nested_protos` | list of string | Is a list of nested proto files. |
| `no_metadata` | bool | Indicates whether to skip generating gapic_metadata.json. This is typically false. |
| `no_snippets` | bool | Indicates whether to skip generating snippets. This is typically false. |
| `path` | string | Is the source path. |
| `proto_only` | bool | Determines whether to generate a Proto-only client. A proto-only client does not define a service in the proto files. |
| `proto_package` | string | Is the proto package name. |

## GoModule Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `delete_generation_output_paths` | list of string | Is a list of paths to delete before generation. |
| `go_apis` | list of [GoAPI](#goapi-configuration) (optional) | Is a list of Go-specific API configurations. |
| `module_path_version` | string | Is the version of the Go module path. |
| `nested_module` | string | Is the name of a nested module directory. |

## JavaAPI Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | string | Is the source path. |
| `additional_protos` | list of string | Is a list of additional proto files to include in generation. |
| `excluded_protos` | list of string | Is a list of proto files to exclude from generation. It expects the full path starting from the root of the googleapis directory (e.g., "google/cloud/aiplatform/v1/schema/io_format.proto"). |
| `gapic_artifact_id_override` | string | Overrides the artifact ID for the GAPIC module. It determines the module's directory name and is used to derive proto and gRPC artifact IDs if they are not explicitly overridden. |
| `grpc_artifact_id_override` | string | Overrides the artifact ID for the gRPC module. The artifact ID is also used as the name for the module's directory. |
| `proto_artifact_id_override` | string | Overrides the artifact ID for the proto module. The artifact ID is also used as the name for the module's directory. |
| `proto_only` | bool | Determines whether to generate a Proto-only client. A proto-only client does not define a service in the proto files. |
| `copy_files` | list of [JavaFileCopy](#javafilecopy-configuration) (optional) | Is a list of file copies to perform after generation. It applies to files in the GAPIC module. |
| `samples` | bool (optional) | Determines whether to generate samples for the API, default is true when omitted. |

## JavaFileCopy Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `source` | string | Is the source path relative to the generated GAPIC module directory (e.g., "src/main/java/com/google/storage/v2/gapic_metadata.json"). These paths are used before restructuring the output into Maven modules. |
| `destination` | string | Is the destination path relative to the generated GAPIC module directory. These paths are used before restructuring the output into Maven modules. |

## JavaModule Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_id_override` | string | Is the ID of the API (e.g., "pubsub.googleapis.com"), allows the "api_id" field in .repo-metadata.json to be overridden. Defaults to "{library.api_shortname}.googleapis.com". |
| `api_reference` | string | Is the URL for the API reference documentation. |
| `api_description_override` | string | Allows the "api_description" field in .repo-metadata.json to be overridden. |
| `api_shortname_override` | string | Allows the "api_shortname" field in .repo-metadata.json to be overridden. |
| `client_documentation_override` | string | Allows the "client_documentation" field in .repo-metadata.json to be overridden. |
| `non_cloud_api` | bool | Indicates whether the API is NOT a Google Cloud API. Defaults to false. |
| `codeowner_team` | string | Is the GitHub team that owns the code. |
| `distribution_name_override` | string | Allows the "distribution_name" field in .repo-metadata.json to be overridden. |
| `excluded_dependencies` | string | Is a list of dependencies to exclude. |
| `excluded_poms` | string | Is a list of POM files to exclude. |
| `extra_versioned_modules` | string | Is a list of extra versioned modules. |
| `group_id` | string | Is the Maven group ID, defaults to "com.google.cloud". |
| `issue_tracker_override` | string | Allows the "issue_tracker" field in .repo-metadata.json to be overridden. |
| `libraries_bom_version` | string | Is the version of the libraries-bom to use for Java. |
| `library_type_override` | string | Allows the "library_type" field in .repo-metadata.json to be overridden. |
| `min_java_version` | int | Is the minimum Java version required. |
| `name_pretty_override` | string | Allows the "name_pretty" field in .repo-metadata.json to be overridden. |
| `java_apis` | list of [JavaAPI](#javaapi-configuration) (optional) | Is a list of Java-specific API configurations. |
| `product_documentation_override` | string | Allows the "product_documentation" field in .repo-metadata.json to be overridden. |
| `recommended_package` | string | Is the recommended package name. |
| `billing_not_required` | bool | Indicates whether the API does NOT require billing. This is typically false. |
| `rest_documentation` | string | Is the URL for the REST documentation. |
| `rpc_documentation` | string | Is the URL for the RPC documentation. |
| `transport_override` | string | Allows the "transport" field in .repo-metadata.json to be overridden. |

## NodejsAPI Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `additional_protos` | list of string | Is a list of additional proto files to include in generation. |
| `diregapic` | bool | Indicates whether generation uses DIREGAPIC (Discovery REST GAPICs). This is typically false. Used for the GCE (compute) client. |
| `path` | string | Is the source path. |

## NodejsPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `additional_protos` | list of string | Is a list of additional proto files to include in generation. This can be overridden at the API level. |
| `bundle_config` | string | Is the path to a GAPIC bundle config file. |
| `dependencies` | map[string]string | Maps npm package names to version constraints. |
| `extra_protoc_parameters` | list of string | Is a list of extra parameters to pass to protoc. |
| `handwritten_layer` | bool | Indicates the library has a handwritten layer on top of the generated code. |
| `main_service` | string | Is the name of the main service for libraries with a handwritten layer. |
| `mixins` | string | Controls mixin behavior (e.g., "none" to disable). |
| `nodejs_apis` | list of [NodejsAPI](#nodejsapi-configuration) (optional) | Is a list of Node.js-specific API configurations. |
| `package_name` | string | Is the npm package name (e.g., "@google-cloud/access-approval"). |

## PythonDefault Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `common_gapic_paths` | list of string | Contains paths which are generated for any package containing a GAPIC API. These are relative to the package's output directory, and the string "{neutral-source}" is replaced with the path to the version-neutral source code (e.g. "google/cloud/run"). If a library defines its own common_gapic_paths, they will be appended to the defaults. |
| `library_type` | string | Is the type to emit in .repo-metadata.json. |

## PythonPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| (embedded) | [PythonDefault](#pythondefault-configuration) |  |
| `opt_args_by_api` | map[string][]string | Contains additional options passed to the generator. In each entry, the key is the API path and the value is the list of options to pass when generating that API. Example: {"google/cloud/secrets/v1beta": ["python-gapic-name=secretmanager"]} |
| `proto_only_apis` | list of string | Contains the list of API paths which are proto-only, so should use regular protoc Python generation instead of GAPIC. |
| `name_pretty_override` | string | Allows the "name_pretty" field in .repo-metadata.json to be overridden, to reduce diffs while migrating. TODO(https://github.com/googleapis/librarian/issues/4175): remove this field. |
| `product_documentation_override` | string | Allows the "product_documentation" field in .repo-metadata.json to be overridden, to reduce diffs while migrating. TODO(https://github.com/googleapis/librarian/issues/4175): remove this field. |
| `api_shortname_override` | string | Allows the "api_shortname" field in .repo-metadata.json to be overridden, to reduce diffs while migrating. TODO(https://github.com/googleapis/librarian/issues/4175): remove this field. |
| `api_id_override` | string | Allows the "api_id" field in .repo-metadata.json to be overridden, to reduce diffs while migrating. TODO(https://github.com/googleapis/librarian/issues/4175): remove this field. |
| `client_documentation_override` | string | Allows the client_documentation field in .repo-metadata.json to be overridden from the default that's inferred. TODO(https://github.com/googleapis/librarian/issues/4175): reduce uses of this field to only cases where it's really needed. |
| `issue_tracker_override` | string | Allows the issue_tracker field in .repo-metadata.json to be overridden, to reduce diffs while migrating. TODO(https://github.com/googleapis/librarian/issues/4175): remove this field. |
| `metadata_name_override` | string | Allows the name in .repo-metadata.json (which is also used as part of the client documentation URI) to be overridden. By default, it's the package name, but older packages use the API short name instead. |
| `default_version` | string | Is the default version of the API to use. When omitted, the version in the first API path is used. |
| `skip_readme_copy` | bool | Prevents generation from copying README.rst from the root directory to the docs directory. TODO(https://github.com/googleapis/librarian/issues/4738): revisit whether or not this field should exist after migration. |

## RustCrate Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| (embedded) | [RustDefault](#rustdefault-configuration) |  |
| `modules` | list of [RustModule](#rustmodule-configuration) (optional) | Specifies generation targets for veneer crates. Each module defines a source proto path, output location, and template to use. |
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
| `include_streaming_methods` | bool | Indicates whether to include gRPC streaming methods. |
| `post_process_protos` | string | Indicates whether to post-process protos. |
| `documentation_overrides` | list of [RustDocumentationOverride](#rustdocumentationoverride-configuration) | Contains overrides for element documentation. |
| `pagination_overrides` | list of [RustPaginationOverride](#rustpaginationoverride-configuration) | Contains overrides for pagination configuration. |
| `name_overrides` | string | Contains codec-level overrides for type and service names. |
| `discovery` | [RustDiscovery](#rustdiscovery-configuration) (optional) | Contains discovery-specific configuration for LRO polling. |
| `quickstart_service_override` | string | Overrides the default heuristically selected service for the package-level quickstart. |

## RustDefault Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `package_dependencies` | list of [RustPackageDependency](#rustpackagedependency-configuration) (optional) | Is a list of default package dependencies. These are inherited by all libraries. If a library defines its own package_dependencies, the library-specific ones take precedence over these defaults for dependencies with the same name. |
| `disabled_rustdoc_warnings` | list of string | Is a list of rustdoc warnings to disable. |
| `generate_setter_samples` | string | Indicates whether to generate setter samples. |
| `generate_rpc_samples` | string | Indicates whether to generate RPC samples. |
| `detailed_tracing_attributes` | bool (optional) | Indicates whether to include detailed tracing attributes. |
| `resource_name_heuristic` | bool (optional) | Indicates whether to apply heuristics to identify and generate resource names. |

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
| `detailed_tracing_attributes` | bool (optional) | Indicates whether to include detailed tracing attributes. This overrides the crate-level setting. |
| `documentation_overrides` | list of [RustDocumentationOverride](#rustdocumentationoverride-configuration) | Contains overrides for element documentation. |
| `extend_grpc_transport` | bool | Indicates whether the transport stub can be extended (in order to support streams). |
| `generate_setter_samples` | string | Indicates whether to generate setter samples. |
| `generate_rpc_samples` | string | Indicates whether to generate RPC samples. |
| `has_veneer` | bool | Indicates whether this module has a handwritten wrapper. |
| `included_ids` | list of string | Is a list of proto IDs to include in generation. |
| `include_grpc_only_methods` | bool | Indicates whether to include gRPC-only methods. |
| `include_list` | yaml.StringSlice | Is a list of proto files to include (e.g., "date.proto", "expr.proto"). |
| `include_streaming_methods` | bool | Indicates whether to include gRPC streaming methods. |
| `internal_builders` | bool | Indicates whether generated builders should be internal to the crate. |
| `module_path` | string | Is the Rust module path for converters (e.g., "crate::generated::gapic::model"). |
| `module_roots` | map[string]string |  |
| `name_overrides` | string | Contains codec-level overrides for type and service names. |
| `output` | string | Is the directory where generated code is written (e.g., "src/storage/src/generated/gapic"). |
| `post_process_protos` | string | Contains code to post-process generated protos. |
| `resource_name_heuristic` | bool (optional) | Indicates whether to apply heuristics to identify and generate resource names. This overrides the crate-level setting. |
| `root_name` | string | Is the key for the root directory in the source map. It overrides the default root, googleapis, used by the rust+prost generator. |
| `routing_required` | bool | Indicates whether routing is required. |
| `service_config` | string | Is the path to the service config file. |
| `skipped_ids` | list of string | Is a list of proto IDs to skip in generation. |
| `specification_format` | string | Overrides the library-level specification format. |
| `api_path` | string | Is the proto path to generate from (e.g., "google/storage/v2"). |
| `template` | string | Specifies which generator template to use. Valid values: "grpc-client", "http-client", "prost", "convert-prost", "mod", "storage". |

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

## SwiftDefault Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `dependencies` | list of [SwiftDependency](#swiftdependency-configuration) | Is a list of package dependencies. |

## SwiftDependency Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Is an identifier for the package within the project.<br><br>For example, `swift-protobuf`. This is usually the last component of the path or the URL. |
| `path` | string | Configures the path for local (to the monorepo) packages.<br><br>For example, the authentication package definition will set this to `packages/auth`, which would generate the following snippet in the `Package.swift` files:<br><br>``` .package(path: "../../packages/auth") ``` |
| `url` | string | Configures the `url:` parameter in the package definition.<br><br>For example, `https://github.com/apple/swift-protobuf` would generate the following snippet in the `Package.swift` files:<br><br>``` .package(url: "https://github.com/apple/swift-protobuf") ``` |
| `version` | string | Configures the minimum version for exaternal package definitions.<br><br>For example, if the `swift-protobuf` package used `1.36.1`, then the codec would generate the following snippet in the `Package.swift` files:<br><br>``` .package(url: "https://github.com/apple/swift-protobuf", from: "1.36.1") ``` |
| `required_by_services` | bool | Is true if this dependency is required by packages with services.<br><br>This will be set for the `gax` library and the `auth` library. Maybe more if we split the HTTP and gRPC clients into separate libraries. |
| `api_package` | string | Is the name of the API package provided by this library.<br><br>In Swift a package contains at most one channel for one API. For packages that implement an API, this field contains the name of the package in the specification language of that API. At the moment this is only used by Protobuf-based APIs, as OpenAPI and discovery doc APIs are self-contained.<br><br>Note that some packages, for example `auth` and `gax`, do not implement APIs. This field is empty for such libraries.<br><br>Examples:<br>- The `GoogleCloudWkt` package will set this to `google.cloud.protobuf`.<br>- The `GoogleCloudLocation` package will set this to `google.cloud.location`. |

## SwiftPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| (embedded) | [SwiftDefault](#swiftdefault-configuration) |  |
| `include_list` | list of string | Is a subset of proto files under the target API path to include (e.g., ["date.proto", "expr.proto"]). |
