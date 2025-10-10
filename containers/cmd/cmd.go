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

// Package cmd provides the main function LanguageContainerMain that
// handles command line argument parsing and invocation of the corresponding methods.
// A Librarian language container implementation will call the function
// from its main function.
package cmd

import (
	"context"
)

// GenerateFlags is the flags in Librarian's container contract for the
// generate command. Each value represents the path of the context, such as
// "/librarian", in which the CLI and the language container exchange
// generate-request.json and generate-response.json.
// https://github.com/googleapis/librarian/blob/main/doc/language-onboarding.md#generate
type GenerateFlags struct {
	Librarian string
	Input     string
	Output    string
	Source    string
}

// ConfigureFlags holds flags for the `configure` command.
// https://github.com/googleapis/librarian/blob/main/doc/language-onboarding.md#configure
type ConfigureFlags struct {
	Librarian string
	Input     string
	Repo      string
	Source    string
	Output    string
}

// ReleaseInitFlags holds flags for the `release-init` command.
// https://github.com/googleapis/librarian/blob/main/doc/language-onboarding.md#release-init
type ReleaseInitFlags struct {
	Librarian string
	Repo      string
	Output    string
}

// LanguageContainer defines the contract for a language-specific container.
type LanguageContainer interface {
	Generate(ctx context.Context, flags *GenerateFlags) error
	Configure(ctx context.Context, flags *ConfigureFlags) error
	ReleaseInit(ctx context.Context, flags *ReleaseInitFlags) error
}

// Entry point for a language container. This parses the arguments and
// calls the corresponding method on the container.
//
// A language container defines the following main function:
//
// ```
//
//	func main() {
//		cmd.LanguageContainerMain(os.Args, &myContainer{})
//	}
//
// ```
func LanguageContainerMain(args []string, container LanguageContainer) {
	// TODO: Call the generateFunc only when it's "generate" command.
	// TODO: Parse the arguments correctly.
	if args[0] == "generate" {
		generateFlags := GenerateFlags{
			Librarian: "/librarian",
			Input:     "/input",
			Output:    "/output",
			Source:    "/source",
		}
		if err := container.Generate(context.Background(), &generateFlags); err != nil {
			// TODO: handle error
		}
	}
}
