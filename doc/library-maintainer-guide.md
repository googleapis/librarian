# Library Maintainer Guide

This guide is task-oriented, specifically for core/handwritten/hybrid library
maintainers. See the
[generated CLI documentation](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian)
for a more comprehensive list of commands and flags.

For configuring development environment, please see [onboarding](onboarding.md).

For libraries onboarded to automation, please see [automation section below](#using-automated-releases).

This guide uses the term Librarian (capital L, regular font) for the overall
Librarian system, and `librarian` (lower case L, code font) for the CLI.

## Internal support (Googlers)

If anything in this guide is unclear, please see go/g3doc-cloud-sdk-librarian-support
for appropriate ways of obtaining more support.

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
$ LIBRARIAN_GITHUB_TOKEN=$(gh auth token) librarian release stage -push \
  -repo=https://github.com/googleapis/google-cloud-go -library=bigtable
```

This will use the conventional commits since the last release to populate
any release notes in both the PR and the relevant changelog file in the repo.

If you want to release a version other than the one inferred by the conventional
commits (e.g. for a prerelease or a patch), you can use the `-library-version`
flag:

```sh
$ LIBRARIAN_GITHUB_TOKEN=$(gh auth token) librarian release stage -push \
  -repo=https://github.com/googleapis/google-cloud-go -library=bigtable \
  -library-version=1.2.3
```

Note that if `librarian` doesn't detect any conventional commits that would
trigger a release, you *must* specify the `-library-version` flag.

### Manual release PR creation

If you expect to need to edit the release notes, it's simplest to run
`librarian` *without* the `-push` flag (and using a local repo),
then create the pull request yourself:

1. Make sure your local clone is up-to-date
2. Create a new branch for the release (e.g. `git checkout -b release-bigtable-1.2.3`)
3. Run `librarian`, specifying `-library-version` if you want/need to, as above:
  ```sh
  $ librarian release stage -library=bigtable -library-version=1.2.3
  ```
4. Note the line of the `librarian` output near the end, which tells you where
  it has written a `pr-body.txt` file (split by key below, but all on one line
  in the output):
  ```text
  time=2025-09-26T15:35:23.124Z
  level=INFO
  msg="Wrote body of pull request that might have been created"
  file=/tmp/librarian-837968205/pr-body.txt
  ```
5. Perform any edits to the release notes, and commit the change using a "chore"
   conventional commit, e.g. `git commit -a -m "chore: create release"`
6. Push your change to GitHub *in the main fork* (rather than any personal fork you may use),
   and create a PR from the branch. Use the content of the `pr-body.txt` file as the body
   of the PR, editing it to be consistent with any changes you've made in the release notes.
7. Add the "release:pending" label to the PR.
8. Ask a colleague to review and merge the PR.

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

## Using automated releases

Maintainers *may* configure Librarian for automated releases, but should do so
with a significant amount of care. Using automated releases is convenient, but
cedes control - and once a release has been published, it can't generally be
rolled back in anything like a "clean" way.

### Impact of automated releases

When automated releases are enabled, they will be initiated on a regular
[cadence](https://goto.google.com/g3doc-librarian-automation), in release PRs that contain other libraries needing releasing from
the same repository. These pull requests will be approved and merged by the
Cloud SDK Platform team.

The version number and release notes will be automatically determined by
Librarian from conventional commits. These will *not* be vetted by the
Cloud SDK Platform team before merging.

Using automated releases doesn't *prevent* manual releases - a maintainer team
can always use the process above to create and merge release PRs themselves,
customizing the version number and release notes as they see fit. Creating
a single manual release does not interrupt automated releases - any subsequent
"release-worthy" changes will still cause an automated release to be created.

### Enabling automated releases

If the repository containing the library is not already using automated releases
for other libraries (i.e. if it's a split repo instead of a monorepo),
first [open a ticket](https://buganizer.corp.google.com/issues/new?component=1198207&template=2190445) with the Cloud Platform SDK team to enable automated releases.

Next, edit the `.librarian/config.yaml` file in your repository. You will
see YAML describing additional configuration for the libraries in the
repository, particularly those which have release automation blocked. Find
the library for which you wish to enable release automation, and remove or
comment out the `release_blocked` key. We recommend commenting out rather
than deleting, ideally adding an explanation and potentially caveats, for
future readers.

Create a PR with the configuration change, get it reviewed and merged, and
the next time release automation runs against the repository, it will consider
the library eligible for automatic releases.

### Ownership of PRs once onboarded to automation

Once a library is onboarded to Librarian automation, the Librarian team is
responsible for approving and merging PRs generated by Librarian. Maintainers are
not expected to be involved in this process, unless PR checks fail.  In that
case a ticket will be opened up in the repository and needs to be addressed by
Maintainers.  This can potentially block generation/release until issue has been 
resolved.

