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
when starting a container. For example, `generate` is a CLI command which invokes
container commands of `configure`, `generate` and `build`.

Executing one CLI command will often start multiple containers, including running
the same container command multiple times. The CLI always waits for a container to
complete before proceeding - implementations do not need to worry about
multiple containers running concurrently, or files being changed by the CLI during
container execution.

## Startup

The container is started using the image's default entry point, with command
line arguments of:

- The container command to run (e.g. `generate`)
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

## configure container command

The configure container command is defined by the following contract:

|   context    |     type       |   description |
|:--------------|:---------------|:---------------|
| /librarian | mount(read/write) |This mount will contain exactly one file named `configure-request.json`. The container is expected to process this request and write a file back to this mount named `configure-response.json`. Both of these files use the schema of a library defined above in the state file. The container may wish to add more context to the library configuration which it expresses back to librarian via this message passing. It will then be librarians responsibility to commit these changes to the state.yaml file in the language repository.|
| /input       | mount(read/write) | The exact contents of the generator-input folder, e.g. google-cloud-go/.librarian/generator-input. This folder has read/write access to allow the container to add any new language specific configuration required. |
| /source      | mount(read) | This folder is mounted into the container. It contains, for example, the whole contents of [googleapis](https://github.com/googleapis/googleapis). This will be needed in order to read the service config files and likely also the BUILD.bazel files that hold a lot of configuration today. |
| command      | Positional Argument | The value will always be `configure` for this invocation. |


## generate container command

In order for the the container to have enough context on how and what to generate, librarian will provide the container the following context:

|   context    |     type       |   description |
|:--------------|:---------------|:---------------|
| /librarian/generate-request.json | mount(read) |A JSON file that describes which library to generate. |
| /input       | mount(read/write) | The exact contents of the generator-input folder, e.g. google-cloud-go/.librarian/generator-input. This folder has read/write access to allow the container to add any new language specific configuration required. |
| /output     | mount(write) | This folder is mounted into the container. It is meant to be the destination for any code generated by the container. Its output structure should match that of where the code should land in the resulting repository. For example if we are generating the [secretmanger v1](https://github.com/googleapis/google-cloud-go/tree/main/secretmanager/apiv1) client for Go, we would write files to `/output/secretmanager`. |
| /source     | mount(read) | This folder is mounted into the container. It contains, for example, the whole contents of [googleapis](https://github.com/googleapis/googleapis). This will be needed in order to read the service config files and likely also the BUILD.bazel files that hold a lot of configuration today. |
| command      | Positional Argument | The value will always be `generate` for this invocation. |

## build container command

In addition to the “generate” container command, if the `build` flag is specified during generation librarian will invoke the container image again in “build/test” mode. During execution, the container is expected to try to compile/unit-test/etc to make sure that the generated code is functional.

|   context    |     type       |   description |
|:--------------|:---------------|:---------------|
| /librarian | mount(read/write) | The exact contents of the `.librarian` folder in the language repository. Additionally this will contain a file name `build-request.json` describing the library being processed. |
| /repo       | mount(read/write) | The whole language repo. The mount is read/write to make diff-testing easier. Any changes made to this directory will have no-effect on the generated code, it is a deep-copy. |
| command      | Positional Argument | The value will always be `build` for this invocation. |
