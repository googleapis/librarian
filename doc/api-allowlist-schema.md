# API Allowlist Schema

This document describes the schema for the API Allowlist.

## API Configuration

| Field | Type | Description |
| :--- | :--- | :--- |
| `description` | string | provides the information for describing an API. |
| `discovery` | string | is the file path to a discovery document in github.com/googleapis/discovery-artifact-manager. Used by sidekick languages (Rust, Dart) as an alternative to proto files. |
| `documentation_uri` | string | overrides the product documentation URI from the service config's publishing section. |
| `languages` | list of string | restricts which languages can generate client libraries for this API. Empty means all languages can use this API.<br><br>Restrictions exist for several reasons:<br>- Newer languages (Rust, Dart) skip older beta versions when stable versions exist<br>- Python has historical legacy APIs not available to other languages<br>- Some APIs (like DIREGAPIC protos) are only used by specific languages |
| `new_issue_uri` | string | overrides the new issue URI from the service config's publishing section. |
| `open_api` | string | is the file path to an OpenAPI spec, currently in internal/testdata. This is not an official spec yet and exists only for Rust to validate OpenAPI support. |
| `path` | string | is the proto directory path in github.com/googleapis/googleapis. If ServiceConfig is empty, the service config is assumed to live at this path. |
| `short_name` | string | overrides the API short name from the service config's publishing section. |
| `service_config` | string | is the service config file path override. If empty, the service config is discovered in the directory specified by Path. |
| `service_name` | string | is a DNS-like logical identifier for the service, such as `calendar.googleapis.com`. |
| `title` | string | overrides the API title from the service config. |
| `transports` | map[string]Transport | defines the supported transports per language. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, all languages use GRPCRest by default. |
