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

package librarian

import "github.com/googleapis/librarian/internal/cli"

// newLibrarianCommand creates the complete command tree declaratively.
// This replaces the init function pattern with explicit registration.
func newLibrarianCommand() *cli.Command {
	// Create version command
	version := &cli.Command{
		Short:     "version prints the version information",
		UsageLine: "librarian version",
		Long:      "Version prints version information for the librarian binary.",
		Action:    cmdVersion.Action,
	}
	version.Init()

	// Create generate command
	generate := &cli.Command{
		Short:     cmdGenerate.Short,
		UsageLine: cmdGenerate.UsageLine,
		Long:      cmdGenerate.Long,
		Action:    cmdGenerate.Action,
	}
	generate.Init()
	addFlagAPI(generate.Flags, generate.Config)
	addFlagAPISource(generate.Flags, generate.Config)
	addFlagBuild(generate.Flags, generate.Config)
	addFlagHostMount(generate.Flags, generate.Config)
	addFlagImage(generate.Flags, generate.Config)
	addFlagLibrary(generate.Flags, generate.Config)
	addFlagRepo(generate.Flags, generate.Config)
	addFlagBranch(generate.Flags, generate.Config)
	addFlagWorkRoot(generate.Flags, generate.Config)
	addFlagPush(generate.Flags, generate.Config)

	// Create release init subcommand
	releaseInit := &cli.Command{
		Short:     cmdInit.Short,
		UsageLine: cmdInit.UsageLine,
		Long:      cmdInit.Long,
		Action:    cmdInit.Action,
	}
	releaseInit.Init()
	addFlagCommit(releaseInit.Flags, releaseInit.Config)
	addFlagPush(releaseInit.Flags, releaseInit.Config)
	addFlagImage(releaseInit.Flags, releaseInit.Config)
	addFlagLibrary(releaseInit.Flags, releaseInit.Config)
	addFlagLibraryVersion(releaseInit.Flags, releaseInit.Config)
	addFlagRepo(releaseInit.Flags, releaseInit.Config)
	addFlagBranch(releaseInit.Flags, releaseInit.Config)
	addFlagWorkRoot(releaseInit.Flags, releaseInit.Config)

	// Create tag-and-release subcommand
	tagAndRelease := &cli.Command{
		Short:     cmdTagAndRelease.Short,
		UsageLine: cmdTagAndRelease.UsageLine,
		Long:      cmdTagAndRelease.Long,
		Action:    cmdTagAndRelease.Action,
	}
	tagAndRelease.Init()
	addFlagRepo(tagAndRelease.Flags, tagAndRelease.Config)
	addFlagPR(tagAndRelease.Flags, tagAndRelease.Config)

	// Create release command and subcommands
	release := &cli.Command{
		Short:     "release manages releases of libraries.",
		UsageLine: "librarian release <command> [arguments]",
		Long:      "Manages releases of libraries.",
		Commands:  []*cli.Command{releaseInit, tagAndRelease},
	}
	release.Init()

	// Create and initialize the root command
	root := &cli.Command{
		Short:     "librarian manages client libraries for Google APIs",
		UsageLine: "librarian <command> [arguments]",
		Long:      "Librarian manages client libraries for Google APIs.",
		Commands:  []*cli.Command{generate, release, version},
	}
	root.Init()
	return root
}
