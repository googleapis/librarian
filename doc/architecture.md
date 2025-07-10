# Librarian Architecture

Note: this documents the state of the code for Librarian in v0.1.0. It is
not a forward-looking document; proposed changes such as
go/sdk-librarian-state-config and go/librarian:cli-reimagined are not included.
The code links are deliberately to the code in the
[v0.1.0 tag](https://github.com/googleapis/librarian/tree/v0.1.0).

The overall Librarian system is designed to provide simple and consistent
automation for as much of the Google Cloud SDK library ecosystem as possible.

At the heart of Librarian is the Librarian CLI, a command-line interface which
implements language-agnostic requirements once, and delegates to Docker images
for language-specific operations. The CLI is expected to be run in multiple
situations:

- By scheduled jobs running on Google-internal CI systems such as Kokoro
- By automated presubmit checks within Google
- By the Librarian maintainers locally, to test changes to the CLI
- By language teams locally, to test changes to their container
  implementations (including testing the impact of GAPIC generator changes)
- By API producers locally, to experiment with the impact of API changes in
  generated code

## Running the CLI

When running locally, the CLI can be run from a clone of this repository:

```sh
go run ./cmd/librarian <command> <flags>
```

Alternatively, it can be run without cloning:

```sh
go run github.com/googleapis/librarian/cmd/librarian@v0.1.0 <command> <flags>
```

The flag names and descriptions are deliberately shared across all commands, to promote consistency.
(Flags which aren't applicable to a given command are not permitted to be specified.)

The following commands are currently implemented; links are to the source code which
contains more detail:

- Ad hoc generation:
  - [`generate`](https://github.com/googleapis/librarian/blob/v0.1.0/internal/librarian/generate.go):
    generates code for a single API
- Regular language repository maintenance:
  - [`configure`](https://github.com/googleapis/librarian/blob/v0.1.0/internal/librarian/configure.go):
    configures libraries for new APIs
  - [`update-apis`](https://github.com/googleapis/librarian/blob/v0.1.0/internal/librarian/updateapis.go):
    regenerates libraries where the source APIs have been updated
  - Releasing (usually executed as a single automated flow):
    - [`create-release-artifacts`](https://github.com/googleapis/librarian/blob/v0.1.0/internal/librarian/createreleaseartifacts.go):
      creates artifacts for library releases (e.g. package files, documentation)

## Repositories used by Librarian

As well as this repository (https://github.com/googleapis/librarian) which contains the Librarian CLI
source code, there are broadly two classes of repository that Librarian is concerned with:

- The repository of API specifications: https://github.com/googleapis/googleapis
- "Language repositories" each of which is expected to contain libraries for a single language, e.g.
  https://googleapis.com/googleapis/google-cloud-dotnet

Librarian can be used with multiple repositories for a single language,
typically for product-specific repositories such as
https://github.com/googleapis/java-spanner for Java libraries for Spanner.
However, it is (currently) expected that there is a "default" repository for each language, of the form
`https://github.com/googleapis/google-cloud-{language}` which will contain most fully-generated
libraries.

## Files used by Librarian in language repositories

Each language repo used by Librarian must have a `generator-input` root directory.
Within that directory, there must be two files: `pipeline-state.json` and `pipeline-config.json`.
Broadly speaking, `pipeline-state.json` is maintained automatically, but `pipeline-config.json`
is maintained by hand (and is expected to change rarely). The current split of information
is more blurry than we'd like; go/sdk-librarian-state-config proposes a clearer split.

These files are JSON representations of the `PipelineState` and `PipelineConfig` messages
declared in [pipeline.proto](https://github.com/googleapis/librarian/blob/v0.1.0/proto/pipeline.proto).
See the schema for more details, but the most important aspect is the list of libraries within
`PipelineState`.

## The meaning of "library" within Librarian

Librarian manages libraries - but the meaning of "library" is not as intuitive and universal
as we might like or imagine it to be. Within Librarian, the term "library" has a very specific
meaning, namely "an entry within the `pipeline-state.json` file, within a given repository".
Librarian cares about information relating to a library such as:

- The library ID: this is how Librarian instructs language-specific containers to operate
  on a particular library.
- The most recently-released version (if any) of the library.
- Which API paths (in the API specification repo) are used to generate code that's contained
  within the library.
- Which source paths (within the language repo that contains the state file) contribute to
  the library, so that Librarian can determine that a new version of the library should be released.

This is not an exhaustive list; see
[pipeline.proto](https://github.com/googleapis/librarian/blob/v0.1.0/proto/pipeline.proto) for more details.

This approach is designed to accommodate the requirements of different languages:

- A library can include code generated from multiple API paths. Different languages package APIs at
  different granularities. Librarian will regenerate all the code for a whole library when the API
  specifications of any of the API paths are changed.
- A library doesn't have to contain any generated *code* at all. A library which only contains handwritten
  code simply won't have any API paths associated with it. That doesn't mean the library will contain
  *no* generated files; there may be metadata files which are generated based on version numbers etc.
  (In some languages there may genuinely be no generated files, of course.)
- A library doesn't have to correspond to a single package in a package manager. Some languages
  don't have package managers as such, and some libraries might naturally be comprised of multiple
  packages.
- A library isn't absolutely *required* to have any source at all. Currently .NET uses this flexibility
  to publish API reference documentation for dependencies, along with a product-neutral "help" guide.
  This is an unusual use case though.

Fully-generated libraries are typically configured automatically via the `configure` CLI command
(which invokes the [`configure` container command](container-contract.md#configure)).
However, they can also be configured manually - typically for handwritten libraries.

The ID of a library is determined by whatever configures that library initially - either the
language container or the human performing manual configuration. Librarian treats this as an opaque
string, without expecting any particular format. These IDs are included in pull request and commit
messages, so it's useful for them to be human readable, but beyond that different languages have a large
degree of freedom. If a library is published as a single package (e.g. a single NuGet package, Ruby gem or
Go module) we would recommend using the name of the package as the library ID.

Currently, Librarian does not validate the IDs specified in the state file, but it is likely that
we will add constraints later. Those constraints are likely to be primarily around characters (e.g.
only printable characters, potentially only ASCII and prohibiting some characters that could cause
issues when used as a filename, e.g. `/`) - and an empty library ID is likely to be prohibited.

In terms of library ID uniqueness:

- Libraries within a single repository *must* have unique IDs
- There is no explicit requirement for libraries across multiple repositories for the same language
  to have unique IDs (and Librarian would have no way of checking this) but it could easily cause confusion
  for two repositories for the same language to have libraries with the same ID.
- It is absolutely fine for different languages to end up with the same library IDs, and this is fairly likely
  to happen.

As well as being the identity for communication between the CLI and language-specific container operations,
a library is a unit of generation and release:

- If a library includes API paths of "google/test/v1" and "google/test/v2", there's no way of saying "block
  regeneration of google/test/v1, but do generate any changes to google/test/v2".
- The whole library must have a single version, from Librarian's perspective. That doesn't absolutely
  *have* to be the version that's pushed to package managers, but everything will be much more straightforward
  if it is. A library which really consists of multiple artifacts each of which has a different version works
  against the design of Librarian.

## API paths

We expect that many of the libraries managed by Librarian will be generated from API specifications, typically
from https://github.com/googleapis/googleapis/ or an equivalent internal repository. Librarian has built-in
support and expectations for APIs within this repository.

Each [library](#the-meaning-of-library-within-librarian) is associated with a number of *API paths*. (As described
earlier, this number may be zero.) Each API path is a relative directory within the API specification repository,
where the directory contains a service config YAML file. The final segment of the path will usually be a
version (e.g. "v1" or "v2beta") as described in [AIP-191](https://google.aip.dev/191), although there are
some exceptions to this (e.g. "google/longrunning").

The terms "API" and "service" are overloaded in general, but within Librarian this definition of API path is used
consistently.

In particular:

- An API path implicitly includes all nested subdirectories. Librarian does not prevent odd API
  definitions such as "google/outer/v1" and "google/outer/v1/inner/v2" from coexisting, but it
  also doesn't take any action to avoid odd behavior that might stem from that. (For example,
  a change to "google/outer/v1/inner/v2/service.proto" would still count as a change to any
  library which included an API path of "google/outer/v1")
- A single service (e.g. spanner.googleapis.com) may expose multiple API paths using the same
  version e.g. "google/spanner/v1", "google/spanner/admin/instance/v1" and
  "google/spanner/admin/database/v1"; these are deemed as entirely separate as far as Librarian
  is concerned.
- A single service (e.g. cloudfunctions.googleapis.com) may expose multiple API paths which vary
  *only* by version, e.g. "google/cloud/functions/v1", "google/cloud/functions/v2" and
  "google/cloud/functions/v2beta"; these are deemed as entirely separate as far as Librarian
  is concerned.

The last two points do not prevent a single library from including code generated for
"everything in a single service" - it just means that the library would be associated with multiple API paths.
