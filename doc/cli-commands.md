# Generate Command

The `generate` command is used to generate client library code for a repository.

## Usage

```bash
librarian generate [flags]
```

## Flags

| Flag           | Type    | Required | Description |
|----------------|---------|----------|-------------|
| `-api`         | string  | No (Yes for onboarding) | Path to the API to be configured (e.g., `google/cloud/functions/v2`). |
| `-api-source`  | string  | No       | Location of the API repository. If undefined, googleapis will be cloned to the output. |
| `-build`       | bool    | No       | Whether to build the generated code after generation. |
| `-host-mount`  | string  | No       | A mount point from Docker host and within Docker. Format: `{host-dir}:{local-dir}`. |
| `-image`       | string  | No       | Language specific container image. Defaults to the image in the pipeline state. |
| `-library`     | string  | No (Yes for onboarding) | The ID of a single library to update or onboard.  If updating this should match the library ID in the state.yaml file. |
| `-repo`        | string  | No       | Code repository for the generated code. Can be a remote URL (e.g., `https://github.com/{owner}/{repo}`) or a local path. If not specified, will try to detect the current working directory as a language repository. |
| `-output`      | string  | No       | Working directory root. If not specified, a working directory will be created in `/tmp`. |
| `-commit`      | bool    | No       | Whether to commit the generated code change locally. |
| `-push`        | bool    | No       | Whether to push the change and create a pull request in GitHub. |

## Example

```bash
librarian generate -repo=https://github.com/googleapis/your-repo -library=your-ilbrary-id -build -push
```

## Behavior

- **Onboarding a new library:** Specify both `-api` and `-library` to configure and generate a new library.
- **Regenerating an existing library:** Specify `-library` to regenerate a single library. If this flag is not provided, all libraries in `.librarian/state.yaml` are regenerated.