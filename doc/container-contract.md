# The Librarian Container Contract

Note: this documents the state of the container contract for Librarian in v0.1.0. It is
not a forward-looking document; proposed changes such as
go/sdk-librarian-state-config and go/librarian:cli-reimagined are not included.

When the Librarian CLI needs to perform a language-specific operation
(such as generating or building a library) it uses Docker to start a container
from an image. This document describes the contract between the Librarian CLI
and the container.

A single image is used to implement the *container commands* listed below. The
terms "container command" and "CLI command" are used to differentiate between the
command specified when running the CLI, and the command specified *by* the CLI
when starting a container. For example, `update-apis` is a CLI command which invokes
container commands of `generate-library`, `clean` and `build-library`.

Executing one CLI command will often start multiple containers, including running
the same container command multiple times. The CLI always waits for a container to
complete before proceeding - implementations do not need to worry about
multiple containers running concurrently, or files being changed by the CLI during
container execution.

## Startup

The container is started using the image's default entry point, with command
line arguments of:

- The container command to run (e.g. `generate-library`)
- Any flags, where each flag is of the form `--name=value`; `value` is always
  present even for Boolean flags, for simplicity of parsing.

Any flag which represents a file or directory will be in a location which the CLI
has mounted within the container. The value of the flag will always be an absolute
path unless indicated otherwise in the documentation below.

The CLI is able to pass additional information via environment variables. This is
configured on a per-container-command basis in `pipeline-config.json`. Each
environment variable listed for a container command can be populated from:

- An environment variable of the same name in the host (CLI) environment
- A secret from [Secret Manager](https://cloud.google.com/security/products/secret-manager),
  if `-secrets-project` has been specified as a CLI argument.
- A default value specified in the configuration file

This is used for language-specific information such as the GCP project to use
for integration testing, or an API key for pushing to package managers.

(Note: one of the proposed changes in go/sdk-librarian-state-config is that the location
of the configuration for these environment variables should be specified separately from
the general Librarian config. For automation tasks, this would usually be a file in Piper.
This change is designed to reduce the risk of secret exfiltration via malicious configuration
changes.)

## Diagnostic output and errors

The exit code of the process must be used to indicate success (0) or
failure (any non-zero value). The CLI does not differentiate between failure exit codes,
and these should generally not be used to convey any useful diagnostic information - text
output is much clearer.

Implementations should use stdout and stderr to report progress and failure information.
The CLI includes the container output within its own output, but never includes this
information in any pull request or commit messages. In practice, this means:

- The implementation should never log highly sensitive information which even
  Google Cloud SDK engineers don't usually have access to, such as package manager API keys.
- It's fine (and often very useful) to log information which is Google-confidential but
  which a Google Cloud SDK engineer would have access to (such as the GCP project in
  which integration tests are running, or an error response).

The CLI does not parse the output of containers to determine nuances around success or
failure; any result information beyond the exit code of the process is always conveyed
using files.

## Container command details

For each container command, the sections below specify:

- The general purpose of the command
- Which CLI commands it is called from
- The flags provided to the command
- Additional requirements

Flags are described as "required" or "optional" - from the implementation perspective,
this means "the implementation can assume it is provided" or "the implementation should
expect that it may be omitted in some cases". For the purposes of making manual testing
easier, implementations are encouraged to validate that all required flags have been
provided, but there is no absolute requirement to do so.

### build-library

`build-library` builds and optionally runs unit tests for either a single library or all libraries
configured within the repository.

Called from CLI commands: `configure`, `generate`, `update-apis`, `update-image-tag`,
`create-release-pr`, `create-release-artifacts`

Flags:

- `--repo-root`: the root of the language repo; required.
- `--library-id`: the library to build; optional.
  When omitted, all configured libraries within the repository should be built.
- `--test`: `true` or `false` based on whether or not unit tests should be run; optional.
  When omitted, implementations should default to not running tests

This command should not run integration tests. The container has network access, but this
should only be used for pulling dependencies - in the long term, we may restrict this network
access to only known hosts.

In most cases, the `--library-id` flag is specified, as the CLI performs most operations
on a library-by-library basis. However, the `update-image-tag` CLI command needs to build
all libraries as a validation step, and the current implementation calls the `build-library`
container command once without specifying `--library-id` as this is significantly more
efficient than calling it separately for each library.

This command must not modify, add or delete any files within the repo root which are
considered relevant to the repo. In other words, running `git status` on the repo after
the command has executed successfully must not show any changes. (It's fine to write files
which are ignored via `.gitignore`, or to perform file system operations outside the repo.)

## build-raw

`build-raw` builds the result of a previous `generate-raw` command, to build the code generated
for an API without any of the context of the repo in which a fully-configured library would
normally live.

Called from CLI commands: `generate`

Flags:

- `--generator-output`: the directory containing the result of `generate-raw`; required.
- `--api-path`: the relative API path that was generated by `generate-raw`; required.

Compared with `build-library`:

- An API path is provided rather than a library ID, as we don't *have* a library ID. This is
  the same API path that was passed to `generate-raw`, and implementations may ignore it if
  it's not useful, or they may use it to determine where they'd expect files to be generated.
- There's no requirement that building the library avoids creating new files etc. The command
  operates in the context of a simple directory structure rather than a git repository, so
  there's no `.gitignore` which could configure output directories etc.
- There's no `--test` option as we do not expect unit tests to be created for generated code.
  For languages without a separate "compile" step, if unit tests are used to validate that
  the generated code is reasonable, those tests should be run.

Note: this command may be combined with `generate-raw` in the future, as there isn't really
a good reason to keep the two commands separate.

## clean

`clean` removes all generated files related to a given configured library from within a repository.

Called from CLI commands: `configure`, `generate`, `update-apis`, `update-image-tag`

Flags:

- `--repo-root`: the root of the language repo; required.
- `--library-id`: the library whose generated files should be removed; required.

The `clean` command must remove all files that would be generated by `generate-library`.
This includes any files which are regenerated for *all* libraries, such as a root README.md file.
The implementation must *not* remove handwritten files.

The intention is that generating or regenerating a library goes through steps of:

- Generating the library into an empty directory (`generate-library`)
- Cleaning previously-generated files (`clean`) from the repository
- Copying the generation result into the repository, failing if any newly-generated files
  already exist (because they should have been cleaned)
- Building the result (`build-library`)

This process ensures that any obsolete files are removed, and also that we never end up
with a mixture of "old and new" generated files.

## configure

`configure` modifies the repository state to include a specified API path in either a new library
or a new part of an existing library.

Called from CLI commands: `configure`

Flags:

- `--api-root`: the root of the API specifications (googleapis) repository; required.
- `--api-path`: the relative path to the API within the API root; required.
- `--generator-input`: the path to the `generator-input` directory within the language repository; required.

See [the Architecture documentation](architecture.md#api-paths) for more details on API paths.

The `configure` command must *only* perform changes within the `generator-input` directory to effectively
provide the information required for a later `generate-library` call. It must not actually perform
generation (or if it wants to do that for some reason, it must discard the results).

The changes in `generator-input` after `configure` has completed are expected to be:

- Changes to `pipeline-state.json` such that the specified API path is now represented in a library
- Any changes to language-specific generator input files which are required for later generation

## generate-library

`generate-library` generates code for a single library, when provided with the full API specifications and the
language repository's `generator-input` directory.

Called from CLI commands: `configure`, `generate`, `update-apis`, `update-image-tag`

Flags:

- `--api-root`: the root of the API specifications (googleapis) repository; required.
- `--output`: the path to an empty directory in which to generate the library; required.
- `--generator-input`: the path to the `generator-input` directory of the language repository; required.
- `--library-id`: the ID of the library to generate; required.

In order to provide repeatable, isolated generation, `generate-library` is not provided with the results
of previous generation operations. For example, it can't modify "sometimes hand-edited, sometimes generated" files:
it must be able to generate everything with only the API specifications and the `generator-input` directories.

The files must be generated in locations corresponding to the eventual desired location in the repository root,
using the `--output` flag as the "logical" root. For example, if the eventual location of a file should be
`apis/Google.Cloud.Test.V1/Google.Cloud.Test.V1/TestClient.g.cs`, and if the command is executed with a flag of
`--output=/genout`, then the file should be generated as `/genout/apis/Google.Cloud.Test.V1/Google.Cloud.Test.V1/TestClient.g.cs`.

Generating source for a whole library *may* require running generators multiple times, if the library includes
multiple API paths. For example, a `google.cloud.functions` library might include the generated code for
`google/cloud/functions/v1`, `google/cloud/functions/v2` and `google/cloud/functions/v2beta`.
Librarian makes no assumptions about the internal details of generation - only that when the command has completed,
all the relevant generated files have been regenerated.

The `generate-library` command *may* modify the API specification repository, e.g. to modify protos to
work around known issues that can't be fixed upstream. This sort of preprocessing is undesirable and should be fixed
in the API specifications wherever possible, but it is permitted as a matter of practicality.

The output of library generation is usually not ready to build in a standalone fashion, because it depends on
the expected context of the repository, e.g. for common dependency files, tooling versions etc which are not generated.
Additionally, the full library may include handwritten files. Librarian never attempts to build the results of
`generate-library` in a standalone fashion. Instead, a sequence of `generate-library`, `clean`, copy files, `build-library`
is executed. This contrasts with [`generate-raw`](#generate-raw).

## generate-raw

`generate-raw` generates code for a single API path, without any additional information from a language repository.
This is expected to be used for experimentation by API producers, for API validation in presubmits, and for generator
experimentation by language teams.

Called from CLI commands: `generate`

Flags:

- `--api-root`: the root of the API specifications (googleapis) repository; required.
- `--api-path`: the relative path to the API within the API root; required.
- `--output`: the path to an empty directory in which to generate the code; required.

See [the Architecture documentation](architecture.md#api-paths) for more details on API paths.

Unlike [`generate-library`](#generate-library), `generate-raw` is expected to generate code which can be built
with no other context. It is not expected to be ready to publish to package managers - it will have no useful
concept of a package version, for example. However, it should be ready to build and use, both as validation
that the API specification is reasonable and for local experimentation purposes.

There are no specific requirements as to the layout of generated files, so long as `build-raw` is able to
build them when provided with the populated output directory and the same API path.

Note: this command may be combined with `build-raw` in the future, as there isn't really
a good reason to keep the two commands separate.

## integration-test-library

`integration-test-library` runs integration tests for a single library. Currently this is only used as
part of the release process.

Called from CLI commands: `create-release-pr`, `create-release-artifacts`

Flags:

- `--repo-root`: the root of the language repo; required.
- `--library-id`: the library to test; required.

If the specified library has no integration tests, the container must exit successfully.
(In other words, the absence of integration tests for a library does not constitute an error.)

The `integration-test-library` command is only ever run after `build-library` has completed successfully,
so containers may assume that the code has already been built, and any output from `build-library` within the
repo root (but ignored by `.gitignore`, as `build-library` mustn't leave the repository dirty) can be reused.

The Librarian CLI does not implement any retry strategy; containers may wish to implement a retry strategy
themselves to avoid unnecessary failures when integrating with external dependencies which aren't 100% reliable.

Like `build-library`, `integration-test-library` must not modify, add or delete any files within the repo root which are
considered relevant to the repo. In other words, running `git status` on the repo after
the command has executed successfully must not show any changes.

## package-library

`package-library` creates any releasable artifacts - typically package binaries ready to publish to a package manager,
and bundles of documentation files ready to publish to documentation sites.

Called from CLI commands: `create-release-artifacts`

Flags:

- `--repo-root`: the root of the language repo; required.
- `--library-id`: the library whose artifacts should be created; required.
- `--output`: the empty directory in which to create artifacts; required.

If a language does not publish packages as part of its intended release process, the container should simply create
any documentation bundles (if any). If nothing at all needs to be published for the given library, the container
should simply exit successfully without creating any files.

The `package-library` command is only ever run after `build-library` has completed successfully,
so containers may assume that the code has already been built, and any output from `build-library` within the
repo root (but ignored by `.gitignore`, as `build-library` mustn't leave the repository dirty) can be reused.

The Librarian CLI creates a folder structure for all releasable artifacts and provides
the output directory corresponding with the single library, so implementations should not feel
any need to create a nested structure within the output directory for disambiguation purposes. (If
a folder structure is needed for any other reason, that's fine.)

The `package-library` command must not perform any actual publication; that is performed by `publish-library`,
using the output of `package-library` once all libraries have been successfully packaged.

## publish-library

`publish-library` publishes releaseable artifacts for a library (such as packages and documentation),
as created by `package-library`.

Called from CLI commands: `publish-release-artifacts`

Flags:

- `--package-output`: the output directory that was previously written to by `package-library`; required.
- `--library-id`: the library whose artifacts should be published; required.
- `--version`: the version of the library being published; required.

Note that `publish-library` is provided with the library version, whereas `package-library` is not. This is
because `package-library` operates in an environment where the full repository information is available,
whereas `publish-library` only has the files created by `package-library`. As the version string is potentially
useful information which the Librarian CLI has to hand, it is provided to `publish-library` to avoid implementations
which need that information performing toil of creating a file just to record the version. If an implementation
*doesn't* need the version, it can ignore the flag entirely.

Where possible, `publish-library` should be retriable: if it's safe to *attempt* to publish a library that may
already have been published, this allows the whole `publish-release-artifacts` command to be retried in the case
of an error (e.g. if a package manager is flaky, and we manage to publish 9 out of 10 libraries in the first attempt,
but the 10th fails). The implementation must ensure if it is run multiple times for the same artifacts, it does
not create redundant copies which would cause user confusion. If the operation cannot be made safely retriable, it
must detect retries and fail with a clear error message.

If a language does not publish packages as part of its intended release process, so `package-library` creates
no artifacts, then the `publish-library` container command is still invoked, and will be provided with the empty directory.
An implementation should just exit successfully in that case.
