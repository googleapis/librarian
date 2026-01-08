# state.yaml Schema

This document describes the schema for the `state.yaml` file, which is used by Librarian to track the status of managed files. This file should not be edited manually.

For more details, see the Go implementation in [state.go](../internal/config/state.go).

## Top-Level Fields

| Field       | Type   | Description                                         | Required | Validation Constraints |
|-------------|--------|-----------------------------------------------------|----------|------------------------------------------------------------------------------------|
| `image`     | string | The name and tag of the generator image to use.     | Yes      | Must be a container image reference that includes a tag and contains no whitespace. |
| `libraries` | list   | A list of [library configurations](#libraries-object). | Yes      | Must not be empty.     |

## `libraries` Object

Each object in the `libraries` list represents a single library and has the following fields:

| Field                   | Type   | Description                                                                                                                                                           | Required | Validation Constraints |
|-------------------------|--------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|------------------------|
| `id`                    | string | A unique identifier for the library, in a language-specific format. It should not be empty and only contains alphanumeric characters, slashes, periods, underscores, and hyphens.                                                                                                  | Yes      | Must be a valid library ID. |
| `version`               | string | The last released version of the library.                                                                                                                             | No       | Must be a valid semantic version, "v" prefix is optional. |
| `last_generated_commit` | string | The commit hash from the API definition repository at which the library was last generated.                                                                         | No       | Must be a 40-character hexadecimal string. |
| `apis`                  | list   | A list of [APIs](#apis-object) that are part of this library.                                                                                                             | Yes      | Must not be empty.     |
| `source_roots`          | list   | A list of directories in the language repository where Librarian contributes code.                                                                                    | Yes      | Must not be empty, and each path must be a valid directory path. |
| `preserve_regex`        | list   | A list of regular expressions for files and directories to preserve during the copy and remove process.                                                                    | No       | Each entry must be a valid regular expression. |
| `remove_regex`          | list   | A list of regular expressions for files and directories to remove before copying generated code. If not set, this defaults to the `source_roots`. A more specific `preserve_regex` takes precedence. | No       | Each entry must be a valid regular expression. |
| `release_exclude_paths` | list   | A list of paths to exclude from the release. Files matching these paths will not be considered part of a commit for this library.                                                                                                                                   | No       | Each entry must be a valid directory file/path.     |
| `tag_format`            | string | A format string for the release tag. The supported placeholders are `{id}` and `{version}`.                                                                           | No       | Must contain `{version}` and may optionally contain `{id}`. No other placeholders are allowed. |

## `apis` Object

Each object in the `apis` list represents a single API and has the following fields:

| Field            | Type   | Description                                                                                             | Required | Validation Constraints |
|------------------|--------|---------------------------------------------------------------------------------------------------------|----------|------------------------|
| `path`           | string | The path to the API, relative to the root of the API definition repository (e.g., `google/storage/v1`).      | Yes      | Must be a valid directory path. |
| `service_config` | string | The name of the service config file, relative to the API `path`.                                        | No       | None.                  |

## Example

```yaml
image: "gcr.io/my-special-project/language-generator:v1.2.5"
libraries:
  - id: "secretmanager"
    version: "1.15.0"
    last_generated_commit: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
    apis:
      - path: "google/cloud/secretmanager/v1"
        service_config: "secretmanager_v1.yaml"
    source_roots:
      - "src/google/cloud/secretmanager"
      - "test/google/cloud/secretmanager"
    preserve_regex:
      - "src/google/cloud/secretmanager/generated-dir/HandWrittenFile.java"
    remove_regex:
      - "src/google/cloud/secretmanager/generated-dir"
```