// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run -tags docgen doc_generate.go

/*
Librarian manages Google API client libraries by automating onboarding,
regeneration, and release. It runs language-agnostic workflows while
delegating language-specific tasks—such as code generation, building, and
testing—to Docker images.

Usage:

	librarian <command> [arguments]

The commands are:

# generate

The generate command is the primary tool for all code generation
tasks. It handles both the initial setup of a new library (onboarding) and the
regeneration of existing ones. Librarian works by delegating language-specific
tasks to a container, which is configured in the .librarian/state.yaml file.
Librarian is environment aware and will check if the current directory is the
root of a librarian repository. If you are not executing in such a directory the
'--repo' flag must be provided.

# Onboarding a new library

To configure and generate a new library for the first time, you must specify the
API to be generated and the library it will belong to. Librarian will invoke the
'configure' command in the language container to set up the repository, add the
new library's configuration to the '.librarian/state.yaml' file, and then
proceed with generation.

Example:

	librarian generate --library=secretmanager --api=google/cloud/secretmanager/v1

# Regenerating existing libraries

You can regenerate a single, existing library by specifying either the library
ID or the API path. If no specific library or API is provided, Librarian will
regenerate all libraries listed in '.librarian/state.yaml'. If '--library' or
'--api' is specified the whole library will be regenerated.

Examples:

	# Regenerate a single library by its ID
	librarian generate --library=secretmanager

	# Regenerate a single library by its API path
	librarian generate --api=google/cloud/secretmanager/v1

	# Regenerate all libraries in the repository
	librarian generate

# Workflow and Options:

The generation process involves delegating to the language container's
'generate' command. After the code is generated, the tool cleans the destination
directories and copies the new files into place, according to the configuration
in '.librarian/state.yaml'.

  - If the '--build' flag is specified, the 'build' command is also executed in
    the container to compile and validate the generated code.
  - If the '--push' flag is provided, the changes are committed to a new branch,
    and a pull request is created on GitHub. Otherwise, the changes are left in
    your local working directory for inspection.

Example with build and push:

	LIBRARIAN_GITHUB_TOKEN=xxx librarian generate --push --build

Usage:

	librarian generate [flags]

Flags:

	-api string
	  	Relative path to the API to be configured/generated (e.g., google/cloud/functions/v2).
	  	Must be specified when generating a new library.
	-api-source string
	  	The location of an API specification repository.
	  	Can be a remote URL or a local file path. (default "https://github.com/googleapis/googleapis")
	-branch string
	  	The branch to use with remote code repositories. This is used to specify
	  	which branch to clone and which branch to use as the base for a pull
	  	request. (default "main")
	-build
	  	If true, Librarian will build each generated library by invoking the
	  	language-specific container.
	-host-mount string
	  	For use when librarian is running in a container. A mapping of a
	  	directory from the host to the container, in the format
	  	<host-mount>:<local-mount>.
	-image string
	  	Language specific image used to invoke code generation and releasing.
	  	If not specified, the image configured in the state.yaml is used.
	-library string
	  	The library ID to generate or release (e.g. google-cloud-secretmanager-v1).
	  	This corresponds to a releasable language unit.
	-output string
	  	Working directory root. When this is not specified, a working directory
	  	will be created in /tmp.
	-push
	  	If true, Librarian will create a commit and a pull request for the changes.
	  	A GitHub token with push access must be provided via the
	  	LIBRARIAN_GITHUB_TOKEN environment variable.
	-repo string
	  	Code repository where the generated code will reside. Can be a remote
	  	in the format of a remote URL such as https://github.com/{owner}/{repo} or a
	  	local file path like /path/to/repo. Both absolute and relative paths are
	  	supported. If not specified, will try to detect if the current working directory
	  	is configured as a language repository.
	-v	enables verbose logging

# release

Manages releases of libraries.

Usage:

	librarian release <command> [arguments]

Commands:

	init                       initiates a release by creating a release pull request.
	tag-and-release            tags and creates a GitHub release for a merged pull request.

# release init

The 'release init' command is the primary entry point for initiating
a new release. It automates the creation of a release pull request by parsing
conventional commits, determining the next semantic version for each library,
and generating a changelog. Librarian is environment aware and will check if the
current directory is the root of a librarian repository. If you are not
executing in such a directory the '--repo' flag must be provided.

This command scans the git history since the last release, identifies changes
(feat, fix, BREAKING CHANGE), and calculates the appropriate version bump
according to semver rules. It then delegates all language-specific file
modifications, such as updating a CHANGELOG.md or bumping the version in a pom.xml,
to the configured language-specific container.

If a specific library is configured for release via the '--library' flag, a single
releasable change is needed to automatically calculate a version bump. If there are
no releasable changes since the last release, the '--version' flag should be included
to set a new version for the library. The new version must be "SemVer" greater than the
current version.

By default, 'release init' leaves the changes in your local working directory
for inspection. Use the '--push' flag to automatically commit the changes to
a new branch and create a pull request on GitHub. The '--commit' flag may be
used to create a local commit without creating a pull request; this flag is
ignored if '--push' is also specified.

Examples:

	# Create a release PR for all libraries with pending changes.
	librarian release init --push

	# Create a release PR for a single library.
	librarian release init --library=secretmanager --push

	# Manually specify a version for a single library, overriding the calculation.
	librarian release init --library=secretmanager --library-version=2.0.0 --push

Usage:

	librarian release init [flags]

Flags:

	-branch string
	  	The branch to use with remote code repositories. This is used to specify
	  	which branch to clone and which branch to use as the base for a pull
	  	request. (default "main")
	-commit
	  	If true, librarian will create a commit for the release but not create
	  	a pull request. This flag is ignored if push is set to true.
	-image string
	  	Language specific image used to invoke code generation and releasing.
	  	If not specified, the image configured in the state.yaml is used.
	-library string
	  	The library ID to generate or release (e.g. google-cloud-secretmanager-v1).
	  	This corresponds to a releasable language unit.
	-library-version string
	  	Overrides the automatic semantic version calculation and forces a specific
	  	version for a library. Requires the --library flag to be specified.
	-output string
	  	Working directory root. When this is not specified, a working directory
	  	will be created in /tmp.
	-push
	  	If true, Librarian will create a commit and a pull request for the changes.
	  	A GitHub token with push access must be provided via the
	  	LIBRARIAN_GITHUB_TOKEN environment variable.
	-repo string
	  	Code repository where the generated code will reside. Can be a remote
	  	in the format of a remote URL such as https://github.com/{owner}/{repo} or a
	  	local file path like /path/to/repo. Both absolute and relative paths are
	  	supported. If not specified, will try to detect if the current working directory
	  	is configured as a language repository.
	-v	enables verbose logging

# release tag-and-release

The 'tag-and-release' command is the final step in the release
process. It is designed to be run after a release pull request, created by
'release init', has been merged.

This command's primary responsibilities are to:

  - Create a Git tag for each library version included in the merged pull request.
  - Create a corresponding GitHub Release for each tag, using the release notes
    from the pull request body.
  - Update the pull request's label from 'release:pending' to 'release:done' to
    mark the process as complete.

You can target a specific merged pull request using the '--pr' flag. If no pull
request is specified, the command will automatically search for and process all
merged pull requests with the 'release:pending' label from the last 30 days.

Examples:

	# Tag and create a GitHub release for a specific merged PR.
	librarian release tag-and-release --repo=https://github.com/googleapis/google-cloud-go --pr=https://github.com/googleapis/google-cloud-go/pull/123

	# Find and process all pending merged release PRs in a repository.
	librarian release tag-and-release --repo=https://github.com/googleapis/google-cloud-go

Usage:

	librarian release tag-and-release [arguments]

Flags:

	-github-api-endpoint string
	  	The GitHub API endpoint to use for all GitHub API operations.
	  	This is intended for testing and should not be used in production.
	-pr string
	  	The URL of a pull request to operate on.
	  	It should be in the format of https://github.com/{owner}/{repo}/pull/{number}.
	  	If not specified, will search for all merged pull requests with the label
	  	"release:pending" in the last 30 days.
	-repo string
	  	Code repository where the generated code will reside. Can be a remote
	  	in the format of a remote URL such as https://github.com/{owner}/{repo} or a
	  	local file path like /path/to/repo. Both absolute and relative paths are
	  	supported. If not specified, will try to detect if the current working directory
	  	is configured as a language repository.
	-v	enables verbose logging

# version

Version prints version information for the librarian binary.

Usage:

	librarian version
*/
package main
