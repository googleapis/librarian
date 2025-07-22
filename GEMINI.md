# Gemini CLI Context for the Librarian Project

This document provides context for the Gemini CLI to effectively assist with development on the Librarian project.

## Project Overview

The Librarian project is a command-line tool written in Go that automates the management of Google Cloud SDK client
libraries. It handles tasks such as configuration, generation, and releasing of these libraries. The tool is designed
to be language-agnostic, using Docker containers to perform language-specific operations. The core logic resides in
this repository, while language-specific implementations are defined in separate Docker images.

## Key Technologies & Libraries

- **Go:** The primary language for the CLI.
- **Docker:** Used for language-specific tasks, isolating them from the main CLI.
- **GitHub:** Used for version control and managing pull requests.
- **[go-github](https://github.com/google/go-github):** Go library for interacting with the GitHub API.
- **[go-git](https://github.com/go-git/go-git):** A pure Go implementation of Git.
- **YAML:** Used for the `state.yaml` file.

## Project Structure

- `cmd/librarian/main.go`: The entrypoint for the CLI application.
- `internal/`: Contains the core logic of the application, organized by domain.
  - `cli/`: A lightweight framework for building the CLI commands.
  - `config/`: Defines the data structures for configuration and state.
  - `docker/`: Handles interaction with Docker containers.
  - `github/`: A client for interacting with the GitHub API.
  - `gitrepo/`: A client for interacting with local Git repositories.
  - `librarian/`: The main business logic for the CLI commands.
  - `secrets/`: A client for Google Secret Manager.
- `doc/`: Contains project documentation, including architecture and contribution guidelines.
- `testdata/`: Contains data used for testing.

## Core Commands & Entrypoints

The main entrypoint is `cmd/librarian/main.go`. The core commands are:

- `generate`: Generates client library code for a single API. It can operate in two modes:
  - **Regeneration:** For existing, configured libraries. This uses the configuration in
      `.librarian/state.yaml`.
  - **Onboarding:** For new or unconfigured APIs, to get a baseline implementation.
- `version`: Prints the version of the Librarian tool.

## Important Files & Configuration

- `.librarian/state.yaml`: The main state file for the pipeline, tracking the status of managed libraries. It is
  automatically managed and should not be edited manually. See the schema in `doc/state-schema.md`.

## Development & Testing Workflow

- **Running tests:** Use `go test -race ./...` to run all tests.
- **Running tests and generate coverage**: Use `go test -race -coverprofile=coverage.out -covermode=atomic ./...`.
- **Analyze coverage report**: Use `go tool cover -func=coverage.out` to check more details about coverage.
- **Building code:** Use `go build ./...` to build the project and check for compilation errors.
- **Formatting:** Use `gofmt` to format the code. The CI checks for unformatted files.

## Contribution Guidelines

- **Commits:** Commit messages should follow the [Conventional Commits](https://www.conventionalcommits.org/)
  specification. The format is `<type>(<package>): <description>`. The type should be one of the following: fix, feat,
  build, chore, docs, test, or refactor. The package should refer to the relative path the Go package where the change
  is being made.
- **Issues:** All significant changes should start with a GitHub issue.
- **Pull Requests:** All code changes must be submitted via a pull request and require review.
- **Code Style:** Follow the guidelines in `doc/howwewritego.md`.
- **Testing:** All new logic should be accompanied by tests. Use table-driven tests and `cmp.Diff` for comparisons.
- For more details, see `CONTRIBUTING.md`.
