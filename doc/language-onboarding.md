# Language Onboarding Guide

This document provides a comprehensive guide for language teams to onboard their projects to the Librarian platform. It
details the necessary steps and configurations required to develop a language-specific container that Librarian can
delegate tasks to.

## Core Concepts

Before diving into the specifics, it's important to understand the key components of the Librarian ecosystem:

* **Librarian:** The core orchestration tool that automates the generation, release, and maintenance of client
  libraries.
* **Language-Specific Container:** A Docker container, built by each language team, that encapsulates the logic for
  generating, building, and releasing libraries for that specific language. Librarian interacts with this container by
  invoking different commands.
* **`state.yaml`:** A manifest file within each language repository that defines the libraries managed by Librarian,
  their versions, and other essential metadata.
* **`config.yaml`:** A configuration file that allows for repository-level customization of Librarian's behavior, such
  as specifying which files the container can access.

## Configuration Files

Librarian relies on two key configuration files to manage its operations: `state.yaml` and `config.yaml`. These files
must be present in the `.librarian` directory at the root of the language repository.

### `state.yaml`

The `state.yaml` file is the primary manifest that informs Librarian about the libraries it is responsible for managing.
It contains a comprehensive list of all libraries within the repository, along with their current state and
configuration.

For a detailed breakdown of all of the fields in the `state.yaml` file, please refer to [state-schema.md].

### `config.yaml`

The `config.yaml` file is a handwritten configuration file that allows you to customize Librarian's behavior at the
repository level. Its primary use is to define which files the language-specific container is allowed to access.

Here is an example of a `config.yaml` file:

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
```

## Container Contracts

Librarian orchestrates its workflows by making a series of invocations to a language-specific container. Each invocation
corresponds to a specific command and is designed to perform a distinct task. For the container to function correctly,
it must have a binary entrypoint that can accept the arguments passed by Librarian.

A successful container invocation is expected to exit with a code of `0`. Any non-zero exit code will be treated as an
error and will halt the current workflow. If a container would like to send an error message back to librarian it can do
so by including a field in the various response files outlined below.

The following sections detail the contracts for each container command.

### `configure`

The `configure` command is invoked only during the onboarding of a new API. Its primary responsibility is to process
the new API information and generate the necessary configuration for the library.

The container is expected to produce up to two artifacts:

* A `configure-response.json` file, which is derived from the `configure-request.json` and contains language-specific
  details. This response will be committed back to the `state.yaml` file by Librarian.
* Any "side-configuration" files that the language may need for its libraries. These should be written to the `/input` mount, which corresponds to the `.librarian/generator-input` directory in the language repository.

TODO: Global file edits

**Contract:**

| Context      | Type                | Description                                                                     |
| :----------- | :------------------ |  :----------------------------------------------------------------------------- |
| `/librarian` | Mount (Read/Write)  | Contains `configure-request.json`. The container must process this and write back `configure-response.json`. |
| `/input`     | Mount (Read/Write)  | The contents of the `.librarian/generator-input` directory. The container can add new language-specific configuration here. |
| `/repo`      | Mount (Read)        | Contains only the files specified in the `global_files_allowlist` from `config.yaml`. |
| `/source`    | Mount (Read).       | Contains the complete contents of the API definition repository (e.g., [googleapis/googleapis](https://github.com/googleapis/googleapis)). |
| `/output`    | Mount (Read/Write)  | An output directory for writing any global file edits allowed by `global_files_allowlist`. |
| `command`    | Positional Argument | The value will always be `configure`. |
| flags        | Flags               | Flags indicating the locations of the mounts: `--librarian`, `--input`, `--source`, `--repo`, `--output` |

**Example `configure-request.json`:**

*Note: There will be only one API with a `status` of `new`.*

```json
{
  "libraries": [
    {
      "id": "google-cloud-secretmanager",
      "apis": [
        {
          "path": "google/cloud/secretmanager/v1",
          "service_config": "secretmanager_v1.yaml",
          "status": "new"
        }
      ],
    },
    {
      "id": "google-cloud-pubsub-v1",
      "apis": [
        {
          "path": "google/cloud/pubsub/v1",
          "service_config": "pubsub_v1.yaml",
          "status": "existing"
        }
      ],
      "source_roots": [ "pubsub" ]
    }
  ]
}
```

**Example `configure-response.json`:**

*Note: Only the library with a `status` of `new` should be returned.*

```json
{
  "id": "google-cloud-secretmanager",
  "apis": [
    {
      "path": "google/cloud/secretmanager/v1",
    }
  ],
  "source_roots": [ "secretmanager" ],
  "preserve_regex": [
    "secretmanager/subdir/handwritten-file.go"
  ],
  "remove_regex": [
    "secretmanager/generated-dir"
  ],
  "version": "0.0.0",
  "tag_format": "{id}/v{version}",
  "error": "An optional field to share error context back to Librarian."
}
```

### `generate`

The `generate` command is where the core work of code generation happens. The container is expected to generate the library code and write it to the `/output` mount, preserving the directory structure of the language repository.

**Contract:**

| Context      | Type                | Description                                                                     |
| :----------- | :------------------ | :------------------------------------------------------------------------------ |
| `/librarian` | Mount (Read/Write)  | Contains `generate-request.json`. Container can optionally write back a `generate-response.json`. |
| `/input`     | Mount (Read/Write)  | The contents of the `.librarian/generator-input` directory. |
| `/output`    | Mount (Write)       | The destination for the generated code. The output structure should match the target repository. |
| `/source`    | Mount (Read)        | The complete contents of the API definition repository. (e.g. googlapis/googleapis) |
| `command`    | Positional Argument | The value will always be `generate`. |
| flags        | Flags               | Flags indicating the locations of the mounts: `--librarian`, `--input`, `--output`, `--source` |

**Example `generate-request.json`:**

```json
{
  "id": "google-cloud-secretmanager",
  "apis": [
    {
      "path": "google/cloud/secretmanager/v1",
      "service_config": "secretmanager_v1.yaml"
    }
  ],
  "source_paths": [
    "secretmanager"
  ],
  "preserve_regex": [
    "secretmanager/subdir/handwritten-file.go"
  ],
  "remove_regex": [
    "secretmanager/generated-dir"
  ],
  "version": "0.0.0",
  "tag_format": "{id}/v{version}"
}
```

**Example `generate-response.json`:**

```json
{
  "error": "An optional field to share error context back to Librarian."
}
```

After the `generate` container finishes, Librarian is responsible for copying the generated code to the language
repository and handling any merging or deleting actions as defined in the library's state.

### `build`

The `build` command is responsible for building and testing the newly generated library to ensure its integrity.

**Contract:**

| Context      | Type                | Description                                                                     |
| :----------- | :------------------ | :------------------------------------------------------------------------------ |
| `/librarian` | Mount (Read/Write)  | Contains `build-request.json`. Container can optionally write back a `build-response.json`. |
| `/repo`      | Mount (Read/Write)  | The entire language repository. This is a deep copy, so any changes made here will not affect the final generated code. |
| `command`    | Positional Argument | The value will always be `build`. |
| flags.       | Flags               | Flags indicating the locations of the mounts: `--librarian`, `--repo` |

**Example `build-request.json`:**

```json
{
  "id": "google-cloud-secretmanager",
  "apis": [
    {
      "path": "google/cloud/secretmanager/v1",
      "service_config": "secretmanager_v1.yaml"
    }
  ],
  "source_paths": [
    "secretmanager"
  ],
  "preserve_regex": [
    "secretmanager/subdir/handwritten-file.go"
  ],
  "remove_regex": [
    "secretmanager/generated-dir"
  ],
  "version": "0.0.0",
  "tag_format": "{id}/v{version}"
}
```

**Example `build-response.json`:**

```json
{
  "error": "An optional field to share error context back to Librarian."
}
```

### `release-init`

The `release-init` command is the core of the release workflow. After Librarian determines the new version and collates
the commits for a release, it invokes this container command to apply the necessary changes to the repository.

The container command's primary responsibility is to update all required files with the new version and commit
information. This includes, but is not limited to, updating `CHANGELOG.md` files, bumping version numbers in metadata
files (e.g., `pom.xml`, `package.json`), and updating any global files that reference the libraries being released.

**Contract:**

| Context      | Type                | Description                                                                     |
| :----------- | :------------------ | :------------------------------------------------------------------------------ |
| `/librarian` | Mount (Read/Write)  | Contains `release-init-request.json`. Container writes back a `release-init-response.json`. |
| `/repo`      | Mount (Read/Write)  | The entire language repository, allowing the container to make any necessary global edits. |
| `/output`    | Mount (Write)       | Any files updated during the release phase should be moved to this directory, preserving their original paths. |
| `command`    | Positional Argument | The value will always be `release-init`. |
| flags.       | Flags               | Flags indicating the locations of the mounts: `--librarian`, `--repo`, `--output` |

**Example `release-init-request.json`:**

```json
{
  "libraries": [
    {
      "id": "google-cloud-secretmanager-v1",
      "version": "1.3.0",
      "changes": [
        {
          "type": "feat",
          "subject": "add new UpdateRepository API",
          "body": "This adds the ability to update a repository's properties.",
          "piper_cl_number": "786353207",
          "source_commit_hash": "9461532e7d19c8d71709ec3b502e5d81340fb661"
        },
        {
          "type": "docs",
          "subject": "fix typo in BranchRule comment",
          "body": "",
          "piper_cl_number": "786353207",
          "source_commit_hash": "9461532e7d19c8d71709ec3b502e5d81340fb661"
        }
      ],
      "apis": [
        {
          "path": "google/cloud/secretmanager/v1"
        },
        {
          "path": "google/cloud/secretmanager/v1beta"
        }
      ],
      "source_roots": [
        "secretmanager",
        "other/location/secretmanager"
      ]
    }
  ]
}
```

**Example `release-init-response.json`:**

```json
{
  "error": "An optional field to share error context back to Librarian."
}
```

[state-schema.md]: state-schema.md
