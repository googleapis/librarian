# librarian.yaml Schema

This document describes the schema for the `librarian.yaml` file.

## Root Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `language` | string | Language is the language for this workspace (go, python, rust). |
| `version` | string | Version is the librarian tool version to use. |
| `repo` | string | Repo is the repository name, such as "googleapis/google-cloud-python". TODO(https://github.com/googleapis/librarian/issues/3003): Remove this field when .repo-metadata.json generation is removed. |
| `sources` | [Sources](#sources-configuration) (optional) | Sources references external source repositories. |
| `release` | [Release](#release-configuration) (optional) | Release holds the configuration parameter for publishing and release subcommands. |
| `default` | [Default](#default-configuration) (optional) | Default contains default settings for all libraries. They apply to all libraries unless overridden. |
| `libraries` | list of [Library](#library-configuration) (optional) | Libraries contains configuration overrides for libraries that need special handling, and differ from default settings. |

## Release Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `branch` | string | Branch sets the name of the release branch, typically `main` |
| `ignored_changes` | list of string | IgnoredChanges defines globs that are ignored in change analysis. |
| `preinstalled` | map[string]string | Preinstalled tools defines the list of tools that must be preinstalled. This is indexed by the well-known name of the tool vs. its path, e.g. [preinstalled] cargo = /usr/bin/cargo |
| `remote` | string | Remote sets the name of the source-of-truth remote for releases, typically `upstream`. |
| `roots_pem` | string | An alternative location for the `roots.pem` file. If empty it has no effect. |
| `tools` | map[string][]Tool | Tools defines the list of tools to install, indexed by installer. |

## Tool Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Name is the name of the tool e.g. nox. |
| `version` | string | Version is the version of the tool e.g. 1.2.4. |

## Sources Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `conformance` | [Source](#source-configuration) (optional) | Conformance is the path to the `conformance-tests` repository, used as include directory for `protoc`. |
| `discovery` | [Source](#source-configuration) (optional) | Discovery is the discovery-artifact-manager repository configuration. |
| `googleapis` | [Source](#source-configuration) (optional) | Googleapis is the googleapis repository configuration. |
| `protobuf` | [Source](#source-configuration) (optional) | ProtobufSrc is the path to the `protobuf` repository, used as include directory for `protoc`. |
| `showcase` | [Source](#source-configuration) (optional) | Showcase is the showcase repository configuration. |

## Source Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `branch` | string | Branch is the source's git branch to pull updates from. Unset should be interpreted as the repository default branch. |
| `commit` | string | Commit is the git commit hash or tag to use. |
| `dir` | string | Dir is a local directory path to use instead of fetching. If set, Commit and SHA256 are ignored. |
| `sha256` | string | SHA256 is the expected hash of the tarball for this commit. |
| `subpath` | string | Subpath is a directory inside the fetched archive that should be treated as the root for operations. |

## Default Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `output` | string | Output is the directory where code is written. For example, for Rust this is src/generated. |
| `release_level` | string | ReleaseLevel is either "stable" or "preview". |
| `tag_format` | string | TagFormat is the template for git tags, such as "{name}/v{version}". |
| `transport` | string | Transport is the transport protocol, such as "grpc+rest" or "grpc". |
| `dart` | [DartPackage](#dartpackage-configuration) (optional) | Dart contains Dart-specific default configuration. |
| `rust` | [RustDefault](#rustdefault-configuration) (optional) | Rust contains Rust-specific default configuration. |

## Library Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Name is the library name, such as "secretmanager" or "storage". |
| `version` | string | Version is the library version. |
| `channels` | list of [API](#api-configuration) (optional) | API specifies which googleapis API to generate from (for generated libraries). |
| `copyright_year` | string | CopyrightYear is the copyright year for the library. |
| `description_override` | string | DescriptionOverride overrides the library description. |
| `keep` | list of string | Keep lists files and directories to preserve during regeneration. |
| `output` | string | Output is the directory where code is written. This overrides Default.Output. |
| `release_level` | string | ReleaseLevel is the release level, such as "stable" or "preview". This overrides Default.ReleaseLevel. |
| `skip_generate` | bool | SkipGenerate disables code generation for this library. |
| `skip_publish` | bool | SkipPublish disables publishing for this library. |
| `skip_release` | bool | SkipRelease disables releasing for this library. |
| `specification_format` | string | SpecificationFormat specifies the API specification format. Valid values are "protobuf" (default) or "discovery". |
| `transport` | string | Transport is the transport protocol, such as "grpc+rest" or "grpc". This overrides Default.Transport. |
| `veneer` | bool | Veneer indicates this library has hand-written code. A veneer may contain generated libraries. |
| `dart` | [DartPackage](#dartpackage-configuration) (optional) | Dart contains Dart-specific library configuration. |
| `go` | [GoModule](#gomodule-configuration) (optional) | Go contains Go-specific library configuration. |
| `python` | [PythonPackage](#pythonpackage-configuration) (optional) | Python contains Python-specific library configuration. |
| `rust` | [RustCrate](#rustcrate-configuration) (optional) | Rust contains Rust-specific library configuration. |

## API Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | string | Path specifies which googleapis Path to generate from (for generated libraries). |

## DartPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `api_keys_environment_variables` | string | APIKeysEnvironmentVariables is a comma-separated list of environment variable names that can contain API keys (e.g., "GOOGLE_API_KEY,GEMINI_API_KEY"). |
| `dependencies` | string | Dependencies is a comma-separated list of dependencies. |
| `dev_dependencies` | string | DevDependencies is a comma-separated list of development dependencies. |
| `extra_imports` | string | ExtraImports is additional imports to include in the generated library. |
| `issue_tracker_url` | string | IssueTrackerURL is the URL for the issue tracker. |
| `library_path_override` | string | LibraryPathOverride overrides the library path. |
| `not_for_publication` | string | NotForPublication indicates whether this package should not be published. |
| `packages` | map[string]string | Packages maps Dart package names to version constraints. Keys are in the format "package:googleapis_auth" and values are version strings like "^2.0.0". |
| `part_file` | string | PartFile is the path to a part file to include in the generated library. |
| `prefixes` | map[string]string | Prefixes maps protobuf package names to Dart import prefixes. Keys are in the format "prefix:google.protobuf" and values are the prefix names. |
| `protos` | map[string]string | Protos maps protobuf package names to Dart import paths. Keys are in the format "proto:google.api" and values are import paths like "package:google_cloud_api/api.dart". |
| `readme_after_title_text` | string | ReadmeAfterTitleText is text to insert in the README after the title. |
| `readme_quickstart_text` | string | ReadmeQuickstartText is text to use for the quickstart section in the README. |
| `repository_url` | string | RepositoryURL is the URL to the repository for this package. |

## GoAPI Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `path` | string |  |
| `client_directory` | string |  |
| `disable_gapic` | bool |  |
| `nested_protos` | list of string |  |
| `proto_package` | string |  |

## GoModule Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `delete_generation_output_paths` | list of string |  |
| `go_apis` | list of [GoAPI](#goapi-configuration) (optional) |  |
| `module_path_version` | string |  |

## PythonPackage Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `opt_args` | list of string | OptArgs contains additional options passed to the generator, where the options are common to all channels. Example: ["warehouse-package-name=google-cloud-batch"] |
| `opt_args_by_api` | map[string][]string | OptArgsByAPI contains additional options passed to the generator, where the options vary by channel. In each entry, the key is the channel (API path) and the value is the list of options to pass when generating that API channel. Example: {"google/cloud/secrets/v1beta": ["python-gapic-name=secretmanager"]} |

## RustCrate Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| (embedded) | [RustDefault](#rustdefault-configuration) | |
| `modules` | list of [RustModule](#rustmodule-configuration) (optional) | Modules specifies generation targets for veneer crates. Each module defines a source proto path, output location, and template to use. This is only used when the library has veneer: true. |
| `per_service_features` | bool | PerServiceFeatures enables per-service feature flags. |
| `module_path` | string | ModulePath is the module path for the crate. |
| `template_override` | string | TemplateOverride overrides the default template. |
| `package_name_override` | string | PackageNameOverride overrides the package name. |
| `root_name` | string | RootName is the root name for the crate. |
| `roots` | list of string | Roots specifies the source roots to use for generation. Defaults to googleapis. |
| `default_features` | list of string | DefaultFeatures is a list of default features to enable. |
| `include_list` | list of string | IncludeList is a list of items to include. |
| `included_ids` | list of string | IncludedIds is a list of IDs to include. |
| `skipped_ids` | list of string | SkippedIds is a list of IDs to skip. |
| `disabled_clippy_warnings` | list of string | DisabledClippyWarnings is a list of clippy warnings to disable. |
| `has_veneer` | bool | HasVeneer indicates whether the crate has a veneer. |
| `routing_required` | bool | RoutingRequired indicates whether routing is required. |
| `include_grpc_only_methods` | bool | IncludeGrpcOnlyMethods indicates whether to include gRPC-only methods. |
| `post_process_protos` | string | PostProcessProtos indicates whether to post-process protos. |
| `detailed_tracing_attributes` | bool | DetailedTracingAttributes indicates whether to include detailed tracing attributes. |
| `documentation_overrides` | list of [RustDocumentationOverride](#rustdocumentationoverride-configuration) | DocumentationOverrides contains overrides for element documentation. |
| `pagination_overrides` | list of [RustPaginationOverride](#rustpaginationoverride-configuration) | PaginationOverrides contains overrides for pagination configuration. |
| `name_overrides` | string | NameOverrides contains codec-level overrides for type and service names. |
| `discovery` | [RustDiscovery](#rustdiscovery-configuration) (optional) | Discovery contains discovery-specific configuration for LRO polling. |

## RustDefault Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `package_dependencies` | list of [RustPackageDependency](#rustpackagedependency-configuration) (optional) | PackageDependencies is a list of default package dependencies. |
| `disabled_rustdoc_warnings` | list of string | DisabledRustdocWarnings is a list of rustdoc warnings to disable. |
| `generate_setter_samples` | string | GenerateSetterSamples indicates whether to generate setter samples. |
| `generate_rpc_samples` | string | GenerateRpcSamples indicates whether to generate RPC samples. |

## RustDiscovery Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `operation_id` | string | OperationID is the ID of the LRO operation type (e.g., ".google.cloud.compute.v1.Operation"). |
| `pollers` | list of [RustPoller](#rustpoller-configuration) | Pollers is a list of LRO polling configurations. |

## RustDocumentationOverride Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | string | ID is the fully qualified element ID (e.g., .google.cloud.dialogflow.v2.Message.field). |
| `match` | string | Match is the text to match in the documentation. |
| `replace` | string | Replace is the replacement text. |

## RustModule Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `disabled_rustdoc_warnings` | yaml.StringSlice | DisabledRustdocWarnings specifies rustdoc lints to disable. An empty slice explicitly enables all warnings. |
| `documentation_overrides` | list of [RustDocumentationOverride](#rustdocumentationoverride-configuration) | DocumentationOverrides contains overrides for element documentation. |
| `extend_grpc_transport` | bool | ExtendGrpcTransport indicates whether the transport stub can be extended (in order to support streams). |
| `generate_setter_samples` | string | GenerateSetterSamples indicates whether to generate setter samples. |
| `generate_rpc_samples` | string | GenerateRpcSamples indicates whether to generate RPC samples. |
| `has_veneer` | bool | HasVeneer indicates whether this module has a handwritten wrapper. |
| `included_ids` | list of string | IncludedIds is a list of proto IDs to include in generation. |
| `include_grpc_only_methods` | bool | IncludeGrpcOnlyMethods indicates whether to include gRPC-only methods. |
| `include_list` | string | IncludeList is a list of proto files to include (e.g., "date.proto,expr.proto"). |
| `language` | string | Language can be used to select a variation of the Rust generator. For example, `rust_storage` enables special handling for the storage client. |
| `module_path` | string | ModulePath is the Rust module path for converters (e.g., "crate::generated::gapic::model"). |
| `module_roots` | map[string]string |  |
| `name_overrides` | string | NameOverrides contains codec-level overrides for type and service names. |
| `output` | string | Output is the directory where generated code is written (e.g., "src/storage/src/generated/gapic"). |
| `post_process_protos` | string | PostProcessProtos contains code to post-process generated protos. |
| `root_name` | string | RootName is the key for the root directory in the source map. It overrides the default root, googleapis-root, used by the rust+prost generator. |
| `routing_required` | bool | RoutingRequired indicates whether routing is required. |
| `service_config` | string | ServiceConfig is the path to the service config file. |
| `skipped_ids` | list of string | SkippedIds is a list of proto IDs to skip in generation. |
| `specification_format` | string | SpecificationFormat overrides the library-level specification format. |
| `source` | string | Source is the proto path to generate from (e.g., "google/storage/v2"). |
| `template` | string | Template specifies which generator template to use. Valid values: "grpc-client", "http-client", "prost", "convert-prost", "mod". |

## RustPackageDependency Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `name` | string | Name is the dependency name. It is listed first so it appears at the top of each dependency entry in YAML. |
| `ignore` | bool | Ignore prevents this package from being mapped to an external crate. When true, references to this package stay as `crate::` instead of being mapped to the external crate name. This is used for self-referencing packages like location and longrunning. |
| `package` | string | Package is the package name. |
| `source` | string | Source is the dependency source. |
| `feature` | string | Feature is the feature name for the dependency. |
| `force_used` | bool | ForceUsed forces the dependency to be used even if not referenced. |
| `used_if` | string | UsedIf specifies a condition for when the dependency is used. |

## RustPaginationOverride Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | string | ID is the fully qualified method ID (e.g., .google.cloud.sql.v1.Service.Method). |
| `item_field` | string | ItemField is the name of the field used for items. |

## RustPoller Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `prefix` | string | Prefix is an acceptable prefix for the URL path (e.g., "compute/v1/projects/{project}/zones/{zone}"). |
| `method_id` | string | MethodID is the corresponding method ID (e.g., ".google.cloud.compute.v1.zoneOperations.get"). |
