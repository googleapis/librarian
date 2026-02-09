# API Allowlist Schema

This document describes the schema for the API Allowlist.

## API Configuration

[Link to code](../internal/serviceconfig/api.go#L48)
| Field | Type | Description |
| :--- | :--- | :--- |
| `Discovery` | string | Discovery is the file path to a discovery document in github.com/googleapis/discovery-artifact-manager. Used by sidekick languages (Rust, Dart) as an alternative to proto files. |
| `Languages` | list of string | Languages restricts which languages can generate client libraries for this API. Empty means all languages can use this API.<br><br>Restrictions exist for several reasons:<br>- Newer languages (Rust, Dart) skip older beta versions when stable versions exist<br>- Python has historical legacy APIs not available to other languages<br>- Some APIs (like DIREGAPIC protos) are only used by specific languages |
| `OpenAPI` | string | OpenAPI is the file path to an OpenAPI spec, currently in internal/testdata. This is not an official spec yet and exists only for Rust to validate OpenAPI support. |
| `Path` | string | Path is the proto directory path in github.com/googleapis/googleapis. If ServiceConfig is empty, the service config is assumed to live at this path. |
| `ReleaseLevels` | map[string]string |  |
| `ServiceConfig` | string | ServiceConfig is the service config file path override. If empty, the service config is discovered in the directory specified by Path. |
| `Title` | string | Title overrides the API title from the service config. |
| `Transports` | map[string]Transport | Transports defines the supported transports per language. Map key is the language name (e.g., "python", "rust"). Optional. If omitted, all languages use GRPCRest by default. |
