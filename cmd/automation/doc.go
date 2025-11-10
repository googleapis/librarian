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

//go:generate go run -tags docgen ../doc_generate.go -cmd .

/*
Automation provides logic to trigger Cloud Build jobs that run Librarian commands for
any repository listed in internal/automation/prod/repositories.yaml.

Usage:

	automation <command> [arguments]

The commands are:

# generate

# The generate command runs a Cloud Build job to generate a

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

The publish-release command runs a Cloud Build job to create a tag on a merged release pull
request.

Usage:

	automation publish-release [flags]

Flags:

	-project string
	  	Google Cloud Platform project ID (default "cloud-sdk-librarian-prod")
*/
package main
