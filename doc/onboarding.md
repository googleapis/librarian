# Onboarding to Librarian

Welcome! This guide is intended to help you get started with the Librarian
project and begin contributing effectively.

## Prerequisites

`librarian` requires:

- Linux
- [Go](https://go.dev/doc/install)
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

Docker needs to be configured to use gcloud for authentication. The following
command line needs to be run, just once:

```sh
gcloud auth configure-docker us-central1-docker.pkg.dev
```

## Step 1: Set Up Your Editor
Install the Go extension following the
[instructions for your preferred editor](https://github.com/golang/tools/tree/master/gopls#editors)

These extensions provide support for essential tools like
[gofmt](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) (automatic code
formatting) and
[goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) (automatic
import management).

## Step 2: Understand How We Work

Read the
[CONTRIBUTING.md](https://github.com/googleapis/librarian/blob/main/CONTRIBUTING.md)
for information on how we work, how to submit code, and what to expect.

## Step 3: Learn Go

If you are new to Go, complete these tutorials:

- [Tutorial: Get started with Go](https://go.dev/doc/tutorial/getting-started)
- [Tutorial: Create a Go module](https://go.dev/doc/tutorial/create-module)
- [A Tour of Go](https://go.dev/tour/welcome)

These will teach you the foundations for how to write, run, and test Go code.

## Step 4: Understand How We Write Go

Read our guide on
[How We Write Go](https://github.com/googleapis/librarian/blob/main/doc/howwewritego.md), for
[project-specific guidance on writing idiomatic, consistent Go code.

## Helpful Links

Use these links to deepen your understanding as you go:

- **Play with Go** (https://go.dev/play): Playground to run and share Go snippets in your browser.

- **Browse Go Packages** (https://pkg.go.dev): Go's official site for discovering and reading documentation for any Go
  package.

- **Explore the Standard Library** (https://pkg.go.dev/std): Documentation for the Go standard library.
