# test-gcr

The `test-gcr` tool automates the process of testing local changes to the `librarian/sidekick` generator against the `google-cloud-rust` repository.

It orchestrates the following steps:

1. Validates paths and dependencies.
2. Prepares the target `google-cloud-rust` repository.
3. Runs the local `sidekick` to regenerate the `showcase` crate.
4. Checks that changes were generated.
5. Formats the generated code.
6. Runs the tests for the `showcase` crate and relevant integration tests.
7. Cleans up the `google-cloud-rust` repository.

## Usage

```bash
go run ./contrib/test-gcr
```

### Options

*   `--gcr-path <path>`: (Required) Absolute path to the local `google-cloud-rust` repository clone.
*   `--librarian-path <path>`: (Required) Absolute path to the local `librarian` repository clone (this directory).
*   `--gcr-branch <branch>`: (Optional) The branch, tag, or commit in `google-cloud-rust` to check out and reset to before testing. Defaults to `upstream/main`.
*   `--cargo-args "<args>"`: (Optional) Additional space-separated arguments to pass to `cargo test`. Enclose multiple arguments in quotes.
*   `--sidekick-args "<args>"`: (Optional) Additional space-separated arguments to pass to the `sidekick refresh` command. Enclose multiple arguments in quotes.
*   `--dry-run`: (Optional) Print commands that would be executed instead of running them. Read-only commands for validation will still be executed.

### Prerequisites

*   Go >= 1.25.0
*   Rustc >= 1.85.0
*   The `google-cloud-rust` and `librarian` repositories must have an `upstream` remote pointing to the main `googleapis` repositories.
    *   `git remote add upstream https://github.com/googleapis/google-cloud-rust.git`
    *   `git remote add upstream https://github.com/googleapis/librarian.git`

## Example

```bash
# Run tests using the current directory as librarian path
go run ./cmd/test-gcr \
  --gcr-path /path/to/your/google-cloud-rust \
  --librarian-path $(pwd)

# Run tests on a specific branch in google-cloud-rust
go run ./cmd/test-gcr \
  --gcr-path /path/to/your/google-cloud-rust \
  --librarian-path $(pwd) \
  --gcr-branch my-feature-branch

# Pass extra arguments to cargo test
go run ./cmd/test-gcr \
  --gcr-path /path/to/your/google-cloud-rust \
  --librarian-path $(pwd) \
  --cargo-args "--release --nocapture"
```
