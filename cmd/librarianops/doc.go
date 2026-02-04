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
Librarianops orchestrates librarian operations across multiple repositories.

Usage:

	librarianops <command> [arguments]

The commands are:

# generate

NAME:

	librarianops generate - generate libraries across repositories

USAGE:

	librarianops generate [<repo> | --all]

DESCRIPTION:

	Examples:
	  librarianops generate google-cloud-rust
	  librarianops generate --all
	  librarianops generate -C ~/workspace/google-cloud-rust google-cloud-rust

	Specify a repository name (e.g., google-cloud-rust) to process a single repository,
	or use --all to process all repositories.

	Use -C to work in a specific directory (assumes repository already exists there).

	For each repository, librarianops will:
	  1. Clone the repository to a temporary directory
	  2. Create a branch: librarianops-generateall-YYYY-MM-DD
	  3. Resolve librarian version from @main and update version field in librarian.yaml
	  4. Run librarian tidy
	  5. Run librarian update --all
	  6. Run librarian generate --all
	  7. Run cargo update --workspace (google-cloud-rust only)
	  8. Commit changes
	  9. Create a pull request

OPTIONS:

	--all         process all repositories
	-C directory  work in directory (assumes repo exists)
	-v            run librarian with verbose output
	--help, -h    show help
*/
package main
