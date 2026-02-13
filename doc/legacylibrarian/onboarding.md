# Onboarding to Legacy Librarian

Welcome! This guide is intended to help you get started with the Librarian
project and begin contributing effectively.

## Step 1: Setup Environment to Run Legacy Librarian

`librarian` requires:

- Linux
- [Go](https://go.dev/doc/install)
- [sudoless Docker](http://go/installdocker)
- git (if you wish to build it locally)
- [gcloud](https://g3doc.corp.google.com/company/teams/cloud-sdk/cli/index.md?cl=head#installing-and-using-the-cloud-sdk) (to set up Docker access to container images)
- [gh](https://github.com/cli/cli) for GitHub access tokens

While in theory `librarian` should be run from your local remote desktop.

> Note that installing Docker will cause gLinux to warn you that Docker is
> unsupported and discouraged. Within Cloud, support for Docker is a core
> expectation (e.g. for Cloud Run and Cloud Build).

Docker needs to be configured to use gcloud for authentication. The following
command line needs to be run, just once:

```sh
gcloud auth configure-docker us-central1-docker.pkg.dev
```

## Step 2: Set Up Your Editor
Install the Go extension following the
[instructions for your preferred editor](https://github.com/golang/tools/tree/master/gopls#editors)

These extensions provide support for essential tools like
[gofmt](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) (automatic code
formatting) and
[goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) (automatic
import management).

## Step 3: Understand How We Work

Read the
[CONTRIBUTING.md](https://github.com/googleapis/librarian/blob/main/CONTRIBUTING.md)
for information on how we work, how to submit code, and what to expect.

## Step 4: Learn Go

If you are new to Go, complete these tutorials:

- [Tutorial: Get started with Go](https://go.dev/doc/tutorial/getting-started)
- [Tutorial: Create a Go module](https://go.dev/doc/tutorial/create-module)
- [A Tour of Go](https://go.dev/tour/welcome)

These will teach you the foundations for how to write, run, and test Go code.

## Step 5: Understand How We Write Go

Read our guide on
[How We Write Go](https://github.com/googleapis/librarian/blob/main/doc/howwewritego.md), for
[project-specific guidance on writing idiomatic, consistent Go code.

## Step 6: Running Legacy Librarian

Currently running legacy librarian from main is unstable, please use the `latest` tag when running 
locally.

### Using `go run`

```sh
$ go run github.com/googleapis/librarian/cmd/legacylibrarian@latest
```

### Using `go install`

To install a binary locally, and then run it (assuming the `$GOBIN` directory
is in your path):

```sh
$ go install github.com/googleapis/librarian/cmd/legacylibrarian@latest
```


### Obtaining a GitHub access token

`librarian` commands which perform write operations on GitHub require
a GitHub access token to be specified via the `LIBRARIAN_GITHUB_TOKEN`
environment variable. While access tokens can be generated manually
and then stored in environment variables in other ways, it's simplest
to use the [`gh` tool](https://github.com/cli/cli).

Once installed, use `gh auth login` to log into GitHub. After that,
when running `librarian` you can use `gh auth token` to obtain an access
token and set it in the environment variable just for that invocation:

```sh
$ LIBRARIAN_GITHUB_TOKEN=$(gh auth token) legacylibrarian ...
```

The examples below assume include this for convenience; if you have
set the environment variable in a different way, just remove the
`LIBRARIAN_GITHUB_TOKEN=$(gh auth token)` part from the command.

### Repository and library options

`legacylibrarian` can either operate on a local clone of the library repo,
or it can clone the repo itself. Unless you have a particular need to use a
local clone (e.g. to the impact of a local change) we recommend you let
`legacylibrarian` clone the library repo itself, using the `-repo` flag - just specify
the GitHub repository, e.g.
`-repo=https://github.com/googleapis/google-cloud-go`. This avoids any
risk of unwanted local changes accidentally becoming part of a generation/release
pull request.

If you wish to use a local clone, you can specify the directory in the `-repo`
flag, or just run `legacylibrarian` from the root directory of the clone and omit the
`-repo` flag entirely.

The commands in this guide are specifically for generating/releasing a single
library, specified with the `library` flag. This is typically the name of the
package or module, e.g. `bigtable` or `google-cloud-bigtable`. Consult the
state file (`.librarian/state.yaml`) in the library repository to find the
library IDs in use (and ideally record this in a team-specific playbook).

## Helpful Links

Use these links to deepen your understanding as you go:

- **Play with Go** (https://go.dev/play): Playground to run and share Go snippets in your browser.

- **Browse Go Packages** (https://pkg.go.dev): Go's official site for discovering and reading documentation for any Go
  package.

- **Explore the Standard Library** (https://pkg.go.dev/std): Documentation for the Go standard library.
