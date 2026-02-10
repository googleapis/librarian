// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run -tags docgen ../doc_generate.go -cmd .

/*
Librarian CLI runs local workflow that

	adds, generates, updates and publishes client libraries.

Usage:

	librarian <command> [arguments]

The commands are:

# add

NAME:

	librarian add - add a new client library to librarian.yaml

USAGE:

	librarian add <apis...> [flags]

OPTIONS:

	--help, -h  show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging

# generate

NAME:

	librarian generate - generate a client library

USAGE:

	librarian generate [library] [--all]

OPTIONS:

	--all       generate all libraries
	--help, -h  show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging

# bump

NAME:

	librarian bump - update versions and prepare release artifacts

USAGE:

	librarian bump [library] [--all] [--version=<version>]

DESCRIPTION:

	bump updates version numbers and prepares the files needed for a new release.

	If a library name is given, only that library is updated. The --all flag updates every
	library in the workspace. When a library is specified explicitly, the --version flag can
	be used to override the new version.

	Examples:
	  librarian bump <library>           # update version for one library
	  librarian bump --all               # update versions for all libraries

OPTIONS:

	--all             update all libraries in the workspace
	--version string  specific version to update to; not valid with --all
	--help, -h        show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging

# tidy

NAME:

	librarian tidy - format and validate librarian.yaml

USAGE:

	librarian tidy [path]

OPTIONS:

	--help, -h  show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging

# update

NAME:

	librarian update - update sources to the latest version

USAGE:

	librarian update [--all | source]

DESCRIPTION:

	Supported sources are:
	  - conformance
	  - discovery
	  - googleapis
	  - protobuf
	  - showcase

OPTIONS:

	--all       update discovery and googleapis sources
	--help, -h  show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging

# version

NAME:

	librarian version - print the version

USAGE:

	librarian version

OPTIONS:

	--help, -h  show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging

# publish

NAME:

	librarian publish - publishes client libraries

USAGE:

	librarian publish

OPTIONS:

	--execute             fully publish (default is to only perform a dry run)
	--library string      library to find a release commit for; default finds latest release commit for any library
	--dry-run             print commands without executing (legacy Rust-only flag)
	--dry-run-keep-going  print commands without executing, don't stop on error (legacy Rust-only flag)
	--skip-semver-checks  skip semantic versioning checks (legacy Rust-only flag)
	--help, -h            show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging

# tag

NAME:

	librarian tag - tags a release commit based on the libraries published

USAGE:

	librarian tag

OPTIONS:

	--library string  library to find a release commit for; default finds latest release commit for any library
	--help, -h        show help

GLOBAL OPTIONS:

	--force, -f    skip binary version check
	--verbose, -v  enable verbose logging
*/
package main
