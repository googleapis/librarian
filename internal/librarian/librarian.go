// Copyright 2024 Google LLC
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

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/googleapis/librarian/internal/cli"
)

// Run executes the Librarian CLI with the given command line arguments.
func Run(ctx context.Context, arg ...string) error {
	cmd := newLibrarianCommand()
	slog.Info("librarian", "arguments", arg)
	return cmd.Run(ctx, arg)
}

func newLibrarianCommand() *cli.Command {
	cmdVersion := &cli.Command{
		Short:     "version prints the version information",
		UsageLine: "librarian version",
		Long:      versionLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(cli.Version())
			return nil
		},
	}
	cmdVersion.Init()

	cmdRelease := &cli.Command{
		Short:     "release manages releases of libraries.",
		UsageLine: "librarian release <command> [arguments]",
		Long:      releaseLongHelp,
		Commands: []*cli.Command{
			newCmdInit(),
			newCmdTagAndRelease(),
		},
	}
	cmdRelease.Init()

	cmd := &cli.Command{
		Short:     "librarian manages client libraries for Google APIs",
		UsageLine: "librarian <command> [arguments]",
		Long:      librarianLongHelp,
		Commands: []*cli.Command{
			newCmdGenerate(),
			cmdRelease,
			cmdVersion,
		},
	}
	cmd.Init()
	return cmd
}

func newCmdGenerate() *cli.Command {
	cmdGenerate := &cli.Command{
		Short:     "generate onboards and generates client library code",
		UsageLine: "librarian generate [flags]",
		Long:      generateLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := cmd.Config.SetDefaults(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			if _, err := cmd.Config.IsValid(); err != nil {
				return fmt.Errorf("failed to validate config: %s", err)
			}
			runner, err := newGenerateRunner(cmd.Config)
			if err != nil {
				return err
			}
			return runner.run(ctx)
		},
	}
	cmdGenerate.Init()
	addFlagAPI(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagAPISource(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagBuild(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagHostMount(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagImage(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagLibrary(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagRepo(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagBranch(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagWorkRoot(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagPush(cmdGenerate.Flags, cmdGenerate.Config)
	return cmdGenerate
}

func newCmdTagAndRelease() *cli.Command {
	cmdTagAndRelease := &cli.Command{
		Short:     "tag-and-release tags and creates a GitHub release for a merged pull request.",
		UsageLine: "librarian release tag-and-release [arguments]",
		Long:      tagAndReleaseLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := cmd.Config.SetDefaults(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			if _, err := cmd.Config.IsValid(); err != nil {
				return fmt.Errorf("failed to validate config: %s", err)
			}
			runner, err := newTagAndReleaseRunner(cmd.Config)
			if err != nil {
				return err
			}
			return runner.run(ctx)
		},
	}
	cmdTagAndRelease.Init()
	addFlagRepo(cmdTagAndRelease.Flags, cmdTagAndRelease.Config)
	addFlagPR(cmdTagAndRelease.Flags, cmdTagAndRelease.Config)
	return cmdTagAndRelease
}

func newCmdInit() *cli.Command {
	cmdInit := &cli.Command{
		Short:     "init initiates a release by creating a release pull request.",
		UsageLine: "librarian release init [flags]",
		Long:      releaseInitLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := cmd.Config.SetDefaults(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			if _, err := cmd.Config.IsValid(); err != nil {
				return fmt.Errorf("failed to validate config: %s", err)
			}
			runner, err := newInitRunner(cmd.Config)
			if err != nil {
				return err
			}
			return runner.run(ctx)
		},
	}
	cmdInit.Init()
	addFlagCommit(cmdInit.Flags, cmdInit.Config)
	addFlagPush(cmdInit.Flags, cmdInit.Config)
	addFlagImage(cmdInit.Flags, cmdInit.Config)
	addFlagLibrary(cmdInit.Flags, cmdInit.Config)
	addFlagLibraryVersion(cmdInit.Flags, cmdInit.Config)
	addFlagRepo(cmdInit.Flags, cmdInit.Config)
	addFlagBranch(cmdInit.Flags, cmdInit.Config)
	addFlagWorkRoot(cmdInit.Flags, cmdInit.Config)
	return cmdInit
}
