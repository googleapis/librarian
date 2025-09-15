# config.yaml Schema

This document describes the schema for the `config.yaml` file, which is used by
Librarian to specify repository-level and library-level configuration. This file
is maintained by the repository owner.

For more details, see the Go implementation in
[librarian_config.go](../internal/config/librarian_config.go).

## Top-Level Fields

Field                    | Type | Description                                            | Required | Validation Constraints
------------------------ | ---- | ------------------------------------------------------ | -------- | ----------------------
`global_files_allowlist` | list | A list of [global files](#global-files-object)         | No       | See details below.
`libraries`              | list | A list of [library configurations](#libraries-object). | No       | See details below.

## `global-files` Object

Each object in the `global_files_allowlist` list represents a global file that
Librarian is able to modify.

| Field         | Type   | Description        | Required | Validation    |
:               :        :                    :          : Constraints   :
| ------------- | ------ | ------------------ | -------- | ------------- |
| `path`        | string | A path from the    | Yes      | Cannot be     |
:               :        : repository root.   :          : empty. May    :
:               :        :                    :          : include       :
:               :        :                    :          : relative      :
:               :        :                    :          : paths, but    :
:               :        :                    :          : cannot escape :
:               :        :                    :          : the           :
:               :        :                    :          : repository    :
:               :        :                    :          : root.         :
| `permissions` | string | Permissions of the | Yes      | One of        |
:               :        : mounted file.      :          : `read-only`,  :
:               :        :                    :          : `write-only`, :
:               :        :                    :          : `read-write`. :

## `libraries` Object

Each object in the `libraries` list represents a single library and has the
following fields:

| Field          | Type   | Description       | Required | Validation        |
:                :        :                   :          : Constraints       :
| -------------- | ------ | ----------------- | -------- | ----------------- |
| `id`           | string | A unique          | Yes      | Must be a valid   |
:                :        : identifier for    :          : library ID.       :
:                :        : the library, in a :          :                   :
:                :        : language-specific :          :                   :
:                :        : format. It should :          :                   :
:                :        : not be empty and  :          :                   :
:                :        : only contains     :          :                   :
:                :        : alphanumeric      :          :                   :
:                :        : characters,       :          :                   :
:                :        : slashes, periods, :          :                   :
:                :        : underscores, and  :          :                   :
:                :        : hyphens.          :          :                   :
| `next_version` | string | The next released | No       | Must be a valid   |
:                :        : version of the    :          : semantic version, :
:                :        : library. Ignored  :          : "v" prefix is     :
:                :        : unless it would   :          : optional.         :
:                :        : increase the      :          :                   :
:                :        : release version.  :          :                   :

## Example

```yaml
# .librarian/config.yaml

# A list of files that will be provided to the 'configure' and 'release-init'
# container invocations.
global_files_allowlist:
  # Allow the container to read and write the root go.work file during the
  # 'configure' step to add new modules.
  - path: "go.work"
    permissions: "read-write"

  # Allow the container to read a template.
  - path: "internal/README.md.template"
    permissions: "read-only"

  # Allow publishing the updated root README.md.
  - path: "README.md"
    permissions: "write-only"
# A list of library overrides
libraries:
  - id: "example-library"
    next_version: "2.3.4"
```
