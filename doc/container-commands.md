# Generate Command

The `generate` command is used to generate client library code for a single API or for all APIs configured in your repository.

## Usage

```bash
librarian generate [flags]
```

## Flags

| Flag           | Type    | Required | Description |
|----------------|---------|----------|-------------|
| `-api`         | string  | No (Yes for onboarding) | Path to the API to be configured (e.g., `google/cloud/functions/v2`). Only required for onboarding/configure command. |
| `-api-source`  | string  | No       | Location of the API repository. If undefined, googleapis will be cloned to the output. |
| `-build`       | bool    | No       | Whether to build the generated code after generation. |
| `-host-mount`  | string  | No       | A mount point from Docker host and within Docker. Format: `{host-dir}:{local-dir}`. |
| `-image`       | string  | No       | Container image to run for subcommands. Defaults to the image in the pipeline state. |
| `-library`     | string  | No (Yes for onboarding) | The ID of a single library to update or generate. Only required for onboarding/configure command. |
| `-repo`        | string  | No       | Code repository for the generated code. Can be a remote URL (e.g., `https://github.com/{owner}/{repo}`) or a local path. If not specified, will try to detect the current working directory as a language repository. |
| `-output`      | string  | No       | Working directory root. If not specified, a working directory will be created in `/tmp`. |
| `-pr`          | string  | No       | A pull request to operate on. Format: `https://github.com/{owner}/{repo}/pull/{number}`. If not specified, will search for all merged PRs with the label `release:pending` in the last 30 days. |
| `-push`        | bool    | No       | Whether to push the generated code and create a pull request. |

## Example

```bash
librarian generate -api=google/cloud/functions/v2 -api-source=https://github.com/googleapis/googleapis -repo=/path/to/repo -build -push
```

## Behavior

- **Onboarding a new library:** Specify both `-api` and `-library` to configure and generate a new library.
- **Regenerating an existing library:** Specify either `-api` or `-library` to regenerate a single library. If neither is provided, all libraries in `.librarian/state.yaml` are regenerated.
- If `-build` is set, the generated library will be built.
- If `-push` is set, changes will be committed and a pull request will be created.
