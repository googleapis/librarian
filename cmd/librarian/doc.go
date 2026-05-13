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

//go:generate go run -tags docgen ../../tool/cmd/docgen -cmd .

/*
Librarian manages Google Cloud client libraries. It runs a local workflow
that onboards new APIs, generates client code, bumps versions, publishes
releases, and tags release commits. Language-specific work, such as code
generation, building, and testing, is delegated to per-language tooling.

All behavior is driven by librarian.yaml at the root of the repository,
whose schema is documented at
https://github.com/googleapis/librarian/blob/main/doc/config-schema.md.

Usage:

	librarian <command> [arguments]

Global flags:

	--verbose, -v    enable verbose logging

# Read and write librarian.yaml configuration

Usage:

	librarian config [get|set] [path] [value]

# Get a configuration value

Usage:

	librarian config get [path]

# Set a configuration value

Usage:

	librarian config set [path] [value]

# Add a new client library

Usage:

	librarian add <api>

add registers a single API in librarian.yaml.

The <api> is a path within the configured googleapis source, such as
"google/cloud/secretmanager/v1". The library name and other defaults are
derived from the first API path using language-specific rules.

If the API path should naturally be included in an existing library, and if the
language supports doing so, that library is modified. Otherwise, a new library
is created.

To add a preview client of an existing library, prefix the API path with
"preview/".

Examples:

	librarian add google/cloud/secretmanager/v1
	librarian add preview/google/cloud/secretmanager/v1beta

A typical librarian workflow for adding a new client library is:

	librarian add <api>            # onboard a new API into librarian.yaml
	librarian generate <library>   # generate the client library

# Generate a client library

Usage:

	librarian generate <library>

generate produces client library code from the APIs configured in
librarian.yaml.

The library name argument selects a single library to regenerate. Use the
--all flag to regenerate every library in the workspace instead. Exactly
one of <library> or --all must be provided.

Generation is delegated to the language-specific tooling configured in
librarian.yaml. Libraries marked with skip_generate are skipped.

Examples:

	librarian generate <library>   # regenerate one library
	librarian generate --all       # regenerate every library

Flags:

	--all       generate all libraries

A typical librarian workflow for regenerating every library against the
latest API definitions is:

	librarian update googleapis
	librarian generate --all

# Install tool dependencies for a language

Usage:

	librarian install [language]

install installs the language-specific tools that librarian uses to
generate and build client libraries (for example, language SDKs and code
generators).

If [language] is omitted, the language is read from librarian.yaml in the
current directory.

Examples:

	librarian install              # use language from librarian.yaml
	librarian install go           # install Go-specific tools

# Tidy and validate librarian.yaml

Usage:

	librarian tidy

tidy reads librarian.yaml, validates its contents, applies any
language-specific defaults and normalization, and writes the file back
with a canonical formatting.

Run tidy after editing librarian.yaml by hand, or as a quick check that
the configuration is well-formed.

# Update sources or version to the latest version

Usage:

	librarian update <version | source>...

update refreshes the upstream source repositories declared in
librarian.yaml to their latest commits and updates the recorded commit
SHAs in librarian.yaml accordingly. It also supports updating the librarian version.

Supported targets:

  - sources.conformance: protocolbuffers/protobuf conformance tests
  - sources.discovery: googleapis/discovery-artifact-manager
  - sources.googleapis: googleapis/googleapis (the API definitions)
  - sources.protobuf: protocolbuffers/protobuf
  - sources.showcase: googleapis/gapic-showcase
  - version: the librarian tool version

At least one target must be specified.

Examples:

	librarian update sources.googleapis
	librarian update sources.googleapis sources.protobuf
	librarian update version

A typical librarian workflow for regenerating every library against the
latest API definitions is:

	librarian update sources.googleapis
	librarian generate --all

# Print the binary version

Usage:

	librarian version

version prints the librarian binary version and exits. The version is
embedded at build time and follows the conventions described at
https://go.dev/ref/mod#versions.
*/
package main
