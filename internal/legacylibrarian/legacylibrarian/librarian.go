// Copyright 2024 Google LLC
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

// Package legacylibrarian provides the core implementation for the Librarian CLI tool.
package legacylibrarian

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacycli"
)

// Run executes the Librarian CLI with the given command line arguments.
func Run(ctx context.Context, arg ...string) error {
	cmd := newLibrarianCommand()
	return cmd.Run(ctx, arg)
}

func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(os.Stderr, opts)
	slog.SetDefault(slog.New(handler))
}

func newLibrarianCommand() *legacycli.Command {
	commands := []*legacycli.Command{
		newCmdGenerate(),
		newCmdRelease(),
		newCmdUpdateImage(),
	}

	return legacycli.NewCommandSet(
		commands,
		"legacylibrarian manages client libraries for Google APIs",
		"legacylibrarian <command> [arguments]",
		librarianLongHelp)
}

func newCmdGenerate() *legacycli.Command {
	var verbose bool
	cmdGenerate := &legacycli.Command{
		Short:     "generate onboards and generates client library code",
		UsageLine: "legacylibrarian generate [flags]",
		Long:      generateLongHelp,
		Action: func(ctx context.Context, cmd *legacycli.Command) error {
			setupLogger(verbose)
			slog.Debug("generate command verbose logging")
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
	addFlagGenerateUnchanged(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagHostMount(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagImage(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagLibrary(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagRepo(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagBranch(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagWorkRoot(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagPush(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagVerbose(cmdGenerate.Flags, &verbose)
	return cmdGenerate
}

func newCmdRelease() *legacycli.Command {
	cmdRelease := &legacycli.Command{
		Short:     "release manages releases of libraries.",
		UsageLine: "legacylibrarian release <command> [arguments]",
		Long:      releaseLongHelp,
		Commands: []*legacycli.Command{
			newCmdStage(),
			newCmdTag(),
		},
	}
	cmdRelease.Init()
	return cmdRelease
}

func newCmdTag() *legacycli.Command {
	var verbose bool
	cmdTag := &legacycli.Command{
		Short:     "tag tags and creates a GitHub release for a merged pull request.",
		UsageLine: "legacylibrarian release tag [arguments]",
		Long:      tagLongHelp,
		Action: func(ctx context.Context, cmd *legacycli.Command) error {
			setupLogger(verbose)
			slog.Debug("tag command verbose logging")
			if err := cmd.Config.SetDefaults(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			if _, err := cmd.Config.IsValid(); err != nil {
				return fmt.Errorf("failed to validate config: %s", err)
			}
			runner, err := newTagRunner(cmd.Config)
			if err != nil {
				return err
			}
			return runner.run(ctx)
		},
	}
	cmdTag.Init()
	addFlagRepo(cmdTag.Flags, cmdTag.Config)
	addFlagPR(cmdTag.Flags, cmdTag.Config)
	addFlagGitHubAPIEndpoint(cmdTag.Flags, cmdTag.Config)
	addFlagVerbose(cmdTag.Flags, &verbose)
	return cmdTag
}

func newCmdStage() *legacycli.Command {
	var verbose bool
	cmdStage := &legacycli.Command{
		Short:     "stage stages a release by creating a release pull request.",
		UsageLine: "legacylibrarian release stage [flags]",
		Long:      releaseStageLongHelp,
		Action: func(ctx context.Context, cmd *legacycli.Command) error {
			setupLogger(verbose)
			slog.Debug("stage command verbose logging")
			if err := cmd.Config.SetDefaults(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			if _, err := cmd.Config.IsValid(); err != nil {
				return fmt.Errorf("failed to validate config: %s", err)
			}
			runner, err := newStageRunner(cmd.Config)
			if err != nil {
				return err
			}
			return runner.run(ctx)
		},
	}
	cmdStage.Init()
	addFlagCommit(cmdStage.Flags, cmdStage.Config)
	addFlagPush(cmdStage.Flags, cmdStage.Config)
	addFlagImage(cmdStage.Flags, cmdStage.Config)
	addFlagLibrary(cmdStage.Flags, cmdStage.Config)
	addFlagLibraryVersion(cmdStage.Flags, cmdStage.Config)
	addFlagRepo(cmdStage.Flags, cmdStage.Config)
	addFlagBranch(cmdStage.Flags, cmdStage.Config)
	addFlagWorkRoot(cmdStage.Flags, cmdStage.Config)
	addFlagVerbose(cmdStage.Flags, &verbose)
	return cmdStage
}

func newCmdUpdateImage() *legacycli.Command {
	var verbose bool
	cmdUpdateImage := &legacycli.Command{
		Short:     "update-image updates configured language image container",
		UsageLine: "legacylibrarian update-image [flags]",
		Long:      updateImageLongHelp,
		Action: func(ctx context.Context, cmd *legacycli.Command) error {
			setupLogger(verbose)
			slog.Debug("update image command verbose logging")
			if err := cmd.Config.SetDefaults(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			if _, err := cmd.Config.IsValid(); err != nil {
				return fmt.Errorf("failed to validate config: %s", err)
			}
			runner, err := newUpdateImageRunner(cmd.Config)
			if err != nil {
				return err
			}
			return runner.run(ctx)
		},
	}
	cmdUpdateImage.Init()
	addFlagAPISource(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagBuild(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagCommit(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagHostMount(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagImage(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagRepo(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagBranch(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagWorkRoot(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagPush(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagTest(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagLibraryToTest(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagCheckUnexpectedChanges(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagVerbose(cmdUpdateImage.Flags, &verbose)
	return cmdUpdateImage
}
