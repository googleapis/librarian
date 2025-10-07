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

// Package bootstrap provides the main function LanguageContainerMain.
package bootstrap

import (
	"context"
)

// GenerateFlags is the flags in Librarian's container contract for the
// generate command. Each value represents the path of the context, such as
// "/librarian" in which the CLI and the language container exchange
// generate-request.json and generate-response.json.
// https://github.com/googleapis/librarian/blob/main/doc/language-onboarding.md#generate
type GenerateFlags struct {
	Librarian string
	Input     string
	Output    string
	Source    string
}

// LanguageContainerFunctions lists the functions a language container
// defines to accept requests from Librarian CLI.
type LanguageContainerFunctions struct {
	GenerateFunc func(ctx context.Context, generateFlags *GenerateFlags)

	// TODO: Create "configure" and "release-init"
}

// Entry point for a language container. This parses the arguments and
// calls the corresponding function in the functions passed by the caller
// with the Flags object, for example, GenerateFlags object for the
// generate command).
//
// A language container defines the following main function:
//
// ```
//
//	func main() {
//		bootstrap.LanguageContainerMain(os.Args,
//			bootstrap.LanguageContainerFunctions{
//				GenerateFunc: generateFunc,
//	            ... (omit) ...
//			})
//	}
//
// ```
func LanguageContainerMain(args []string, functions LanguageContainerFunctions) {
	// TODO: Call the generateFunc only when it's "generate" command.
	// TODO: Parse the arguments correctly.
	if args[0] == "generate" {
		generateFlags := GenerateFlags{
			Librarian: "/dummy/librarian",
			Input:     "/dummy/input",
			Output:    "/dummy/output",
			Source:    "/dummy/source",
		}
		functions.GenerateFunc(nil, &generateFlags)
	}
}
