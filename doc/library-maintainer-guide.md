# Library Maintainer Guide

This guide is aimed at maintainers of libraries using Librarian, where the
library contains hand-written code. (Fully-generated libraries are automated
for both generation and release.)

This guide is task-oriented, specifically for core/handwritten/hybrid library
maintainers. See the
[generated CLI documentation](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian)
for a more comprehensive list of commands and flags.

This guide uses the term Librarian (capital L, regular font) for the overall
Librarian system, and `librarian` (lower case L, code font) for the CLI.

## Prerequisites

`librarian` requires:

- Linux
- Go (or a prebuilt binary)
- sudoless Docker
- git (if you wish to build it locally)
- gcloud (to set up Docker access to conatiner images)
- [gh](https://github.com/cli/cli) for GitHub access tokens

While in theory `librarian` can be run in non-Linux environments that support
Linux Docker containers, Google policies make this at least somewhat infeasible
(while staying conformant), so `librarian` is not tested other than on Linux.

See go/docker for instructions on how to install Docker, ensuring that you
follow the sudoless part.

> Note that installing Docker will cause gLinux to warn you that Docker is
> unsupported and discouraged. Within Cloud, support for Docker is a core
> expectation (e.g. for Cloud Run and Cloud Build). Using Docker is the most
> practical way of abstracting away language details. We are confident that
> there are enough Googlers who require Docker to work on gLinux that it won't
> actually go away any time soon. We may investigate using podman instead if
> necessary.

## Running `librarian`

There are various options for running `librarian`. We recommend using `go run`
(the first option) unless you're developing `librarian`. You may wish to use
a bash alias for simplicity. For example, using the first option below you might
use:

```sh
$ alias librarian='go run github.com/googleapis/librarian/cmd/librarian@latest'
```

In this guide, we just assume that `librarian` is either a binary in your path,
or a suitable alias.

### Using `go run`

The latest released version of `librarian` can be run directly without cloning
using:

```sh
$ go run github.com/googleapis/librarian/cmd/librarian@latest
```

### Using `go install`

To install a binary locally, and then run it (assuming the `$GOBIN` directory
is in your path):

```sh
$ go install github.com/googleapis/librarian/cmd/librarian@latest
```

Note that while this makes it easier to run `librarian`, you'll need to know
to install a new version when it's released.

### Building locally

Clone the source code, then run it:

```sh
$ git clone https://github.com/googleapis/librarian
$ cd librarian
$ go run ./cmd/librarian
```

## Obtaining a GitHub access token

`librarian` commands which perform write operations on GitHub require
a GitHub access token to be specified via the `LIBRARIAN_GITHUB_TOKEN`
environment variable. While access tokens can be generated manually
and then stored in environment variables in other ways, it's simplest
to use the [`gh` tool](https://github.com/cli/cli).

Once installed, use `gh auth login` to log into GitHub. After that,
when running `librarian` you can use `gh auth token` to obtain an access
token and set it in the environment variable just for that invocation:

```sh
$ LIBRARIAN_GITHUB_TOKEN=$(gh auth token) librarian ...
```

The examples below assume include this for convenience; if you have
set the environment variable in a different way, just remove the
`LIBRARIAN_GITHUB_TOKEN=$(gh auth token)` part from the command.

## Repository and library options

`librarian` can either operate on a local clone of the library repo,
or it can clone the repo itself. Unless you have a particular need to use a
local clone (e.g. to the impact of a local change) we recommend you let
`librarian` clone the library repo itself, using the `-repo` flag - just specify
the GitHub repository, e.g.
`-repo=https://github.com/googleapis/google-cloud-go`. This avoids any
risk of unwanted local changes accidentally becoming part of a generation/release
PR.

If you wish to use a local clone, you can specify the directory in the `-repo`
flag, or just run `librarian` from the root directory of the clone and omit the
`-repo` flag entirely.

The commands in this guide are specifically for generating/releasing a single
library, specified with the `library` flag. This is typically the name of the
package or module, e.g. `bigtable` or `google-cloud-bigtable`. Consult the
state file (`.librarian/state.yaml`) in the library repository to find the
library IDs in use (and ideally record this in a team-specific playbook).

The remainder of this guide uses
`https://github.com/googleapis/google-cloud-go` as the repository and `bigtable`
as the library ID, in order to provide concrete examples.

## Initiating a release

The release process consists of four steps (at a high level; there are more
details which aren't relevant to this guide):

1. Creating a release PR using `librarian`
2. Reviewing/merging the release PR
3. Creating a tag for the commit created by the merged release PR
4. Running a release job (to build, test, publish) on the tagged commit

Step 1 is described in this section.

Step 2 is simply the normal process of reviewing and merging a PR.

Steps 3 and 4 are automated; step 3 should occur within about 5 minutes of the
release PR being merged, and this will trigger step 4. The instructions for the
remainder of this section are about step 1.

### Hands-off release PR creation

If you are reasonably confident that the release notes won't need editing, the
simplest way to initiate a release is to ask `librarian` to create the release
PR for you:

```sh
$ LIBRARIAN_GITHUB_TOKEN=$(gh auth token) librarian release init -push \
  -repo=https://github.com/googleapis/google-cloud-go -library=bigtable
```

This will use the conventional commits since the last release to populate
any release notes in both the PR and the relevant changelog file in the repo.

If you want to release a version other than the one inferred by the conventional
commits (e.g. for a prerelease or a patch), you can use the `-library-version`
flag:

```sh
$ LIBRARIAN_GITHUB_TOKEN=$(gh auth token) librarian release init -push \
  -repo=https://github.com/googleapis/google-cloud-go -library=bigtable \
  -library-version=1.2.3
```

Note that if `librarian` doesn't detect any conventional commits that would
trigger a release, you *must* specify the `-library-version` flag.

### Manual release PR creation

TODO(https://github.com/googleapis/librarian/issues/2389): Improve this

Currently, the simplest way of creating a release PR with customized release
notes is to:

1. Use the above "hands-off" flow to create a PR
2. Pull the branch into a local clone
3. Modify the release notes and create a new "chore" commit with the changes
4. Push the branch back to GitHub
5. Modify the release PR description to be consistent with the release notes

We aim to simplify this flow in the future.

## Updating generated code

Librarian automation updates GAPIC/proto-generated code on a weekly basis on a
Wednesday, but generation can be run manually if an API change urgently needs
to be included in client libraries.

### Create a PR with updated code

In the common case where you just need to expedite generation, `librarian` can
create the generation PR for you:

```sh
$ LIBRARIAN_GITHUB_TOKEN=$(gh auth token) librarian generate -push \
  -repo=https://github.com/googleapis/google-cloud-go -library=bigtable
```

Ask a colleague to review and merge the PR.

### Test a local API change

If you need to test the potential impact of an API change which isn't yet in the
`googleapis` repository, you can use the `-api-source` flag to specify a local
clone of `googleapis`. You should *not* push this to GitHub other than for
sharing purposes.

Typical flow, assuming a directory structure of `~/github/googleapis` containing
clones of `googleapis` and `google-cloud-go` (and assuming they're up-to-date):

```sh
$ cd ~/github/googleapis/googleapis
$ git checkout -b test-api-changes
# Make changes here
$ git commit -a -m "Test API changes"
$ cd ../google-cloud-go
$ git checkout -b test-generated-api-changes
$ librarian generate -api-source=../googleapis -library=bigtable
```
