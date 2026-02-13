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

	librarianops generate [<repo> | -C <dir>]

DESCRIPTION:

	Examples:
	  librarianops generate google-cloud-rust
	  librarianops generate -C ~/workspace/google-cloud-rust

	Specify a repository name to clone and process, or use -C to work in a specific
	directory (repo name is inferred from the directory basename).

	For each repository, librarianops will:
	  1. Clone the repository to a temporary directory (or use existing directory with -C)
	  2. Create a branch: librarianops-generateall-YYYY-MM-DD
	  3. Run librarian tidy
	  4. Run librarian update --all
	  5. Run librarian generate --all
	  6. Run cargo update --workspace (google-cloud-rust only)
	  7. Commit changes
	  8. Create a pull request

OPTIONS:

	-C directory  work in directory (repo name inferred from basename)
	-v            run librarian with verbose output
	--help, -h    show help
*/
package main
