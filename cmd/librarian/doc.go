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

# Add a new client library

Usage:

	librarian add <apis...>

add registers one or more APIs as a new client library in librarian.yaml.

Each <api> is a path within the configured googleapis source, such as
"google/cloud/secretmanager/v1". The library name and other defaults are
derived from the first API path using language-specific rules.

Multiple API paths may be passed to bundle them into a single library. To
add a preview client of an existing library, prefix every API path with
"preview/"; preview and non-preview APIs cannot be mixed in one invocation.

Examples:

	librarian add google/cloud/secretmanager/v1
	librarian add google/cloud/foo/v1 google/cloud/foo/v1beta
	librarian add preview/google/cloud/secretmanager/v1beta

A typical librarian workflow for adding a new client library is:

	librarian add <api>            # onboard a new API into librarian.yaml
	librarian generate <library>   # generate the client library

# Update sources to the latest version

Usage:

	librarian update <sources...>

update refreshes the upstream source repositories declared in
librarian.yaml to their latest commits and updates the recorded commit
SHAs in librarian.yaml accordingly.

Each <source> names an upstream repository that librarian consumes:

  - conformance: protocolbuffers/protobuf conformance tests
  - discovery: googleapis/discovery-artifact-manager
  - googleapis: googleapis/googleapis (the API definitions)
  - protobuf: protocolbuffers/protobuf
  - showcase: googleapis/gapic-showcase

At least one source must be specified.

Examples:

	librarian update googleapis
	librarian update googleapis protobuf

A typical librarian workflow for regenerating every library against the
latest API definitions is:

	librarian update googleapis
	librarian generate --all

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

# Bump version numbers and prepare release artifacts

Usage:

	librarian bump <library>

bump updates version numbers and prepares the files needed for a new release.

If a library name is given, only that library is updated. The --all flag updates every
library in the workspace. When a library is specified explicitly, the --version flag can
be used to override the new version.

Examples:

	librarian bump <library>           # update version for one library
	librarian bump --all               # update versions for all libraries

Flags:

	--all             update all libraries in the workspace
	--version string  specific version to update to; not valid with --all

# Publish client libraries

Usage:

	librarian publish

publish releases the libraries that were updated in a release commit
prepared by librarian bump.

By default, publish performs a dry run that prints the actions it would
take. Pass --execute to actually publish. By default, the most recent
release commit reachable from HEAD is used; --release-commit overrides
this with a specific commit.

The --dry-run, --dry-run-keep-going, and --skip-semver-checks flags are
only honored when the workspace language is Rust; they are retained for
backwards compatibility with the legacy Rust release jobs and will be
removed once Rust migrates to the unified flow.

Examples:

	librarian publish                          # dry run
	librarian publish --execute                # publish for real
	librarian publish --release-commit=<sha>   # publish a specific commit

Flags:

	--execute                fully publish (default is to only perform a dry run)
	--release-commit string  the release commit to publish; default finds latest release commit
	--dry-run                print commands without executing (legacy Rust-only flag)
	--dry-run-keep-going     print commands without executing, don't stop on error (legacy Rust-only flag)
	--skip-semver-checks     skip semantic versioning checks (legacy Rust-only flag)

# Tag a release commit based on the libraries published

Usage:

	librarian tag

tag creates git tags on a release commit, one tag per library that the
commit released, using the tag_format declared for each library in
librarian.yaml.

Run tag after librarian publish has succeeded. By default, the most
recent release commit reachable from HEAD is used; --release-commit
overrides this with a specific commit.

The --create-release-tag flag additionally creates a tag of the form
release-<PR number>; this is used by the legacy release jobs and will be
removed once those jobs are retired.

Examples:

	librarian tag
	librarian tag --release-commit=<sha>
	librarian tag --create-release-tag

Flags:

	--release-commit string  the release commit to tag; default finds latest release commit
	--create-release-tag     whether to create a tag of the form release-{PR number}

# Print the binary version

Usage:

	librarian version

version prints the librarian binary version and exits. The version is
embedded at build time and follows the conventions described at
https://go.dev/ref/mod#versions.
*/
package main
