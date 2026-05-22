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
*/
package main
