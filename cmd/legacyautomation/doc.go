// Copyright 2025 Google LLC
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
Librarian manages Google API client libraries by automating onboarding,
regeneration, and release. It runs language-agnostic workflows while
delegating language-specific tasks—such as code generation, building, and
testing—to Docker images.

Usage:

	librarian <command> [arguments]

The commands are:

# generate

The generate command triggers a Cloud Build job that runs librarian generate command for every
repository onboarded to Librarian generate automation.

Usage:

	automation generate [flags]

Flags:

	-build
	  	The _BUILD flag (true/false) to Librarian CLI's -build option
	-project string
	  	Google Cloud Platform project ID (default "cloud-sdk-librarian-prod")
	-push
	  	The _PUSH flag (true/false) to Librarian CLI's -push option

# publish-release

The publish-release command triggers a Cloud Build job that runs librarian release tag command
for every repository onboarded to Librarian publish-release automation.

Usage:

	automation publish-release [flags]

Flags:

	-project string
	  	Google Cloud Platform project ID (default "cloud-sdk-librarian-prod")

# stage-release

The stage-release command triggers a Cloud Build job that runs librarian release stage command for
every repository onboarded to Librarian stage-release automation.

Usage:

	automation stage-release [flags]

Flags:

	-project string
	  	Google Cloud Platform project ID (default "cloud-sdk-librarian-prod")
	-push
	  	The _PUSH flag (true/false) to Librarian CLI's -push option

# version

Version prints version information for the automation binary.

Usage:

	automation version
*/
package main
