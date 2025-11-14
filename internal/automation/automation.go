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

// Package automation implements the command-line interface and core logic
// for Librarian's automated workflows.
package automation

import (
	"context"

	"github.com/googleapis/librarian/internal/cli"
)

var runCommandFn = RunCommand

// Run executes the Librarian CLI with the given command line arguments.
func Run(ctx context.Context, arg []string) error {
	cmd := newAutomationCommand()
	return cmd.Run(ctx, arg)
}

func newAutomationCommand() *cli.Command {
	commands := []*cli.Command{
		newCmdGenerate(),
		newCmdPublishRelease(),
		newCmdStageRelease(),
		newCmdUpdateImage(),
	}

	return cli.NewCommandSet(
		commands,
		"automation manages Cloud Build resources to run Librarian CLI.",
		"automation <command> [arguments]",
		automationLongHelp)
}

func newCmdGenerate() *cli.Command {
	cmdGenerate := &cli.Command{
		Short:     "generate",
		UsageLine: "automation generate [flags]",
		Long:      generateLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			runner := newGenerateRunner(cmd.Config)
			return runner.run(ctx)
		},
	}

	cmdGenerate.Init()
	addFlagBuild(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagProject(cmdGenerate.Flags, cmdGenerate.Config)
	addFlagPush(cmdGenerate.Flags, cmdGenerate.Config)

	return cmdGenerate
}

func newCmdPublishRelease() *cli.Command {
	cmdPublishRelease := &cli.Command{
		Short:     "publish-release",
		UsageLine: "automation publish-release [flags]",
		Long:      publishLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			runner := newPublishRunner(cmd.Config)
			return runner.run(ctx)
		},
	}

	cmdPublishRelease.Init()
	addFlagProject(cmdPublishRelease.Flags, cmdPublishRelease.Config)

	return cmdPublishRelease
}

func newCmdStageRelease() *cli.Command {
	cmdStageRelease := &cli.Command{
		Short:     "stage-release",
		UsageLine: "automation stage-release [flags]",
		Long:      stageLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			runner := newStageRunner(cmd.Config)
			return runner.run(ctx)
		},
	}

	cmdStageRelease.Init()
	addFlagProject(cmdStageRelease.Flags, cmdStageRelease.Config)
	addFlagPush(cmdStageRelease.Flags, cmdStageRelease.Config)

	return cmdStageRelease
}

func newCmdUpdateImage() *cli.Command {
	cmdUpdateImage := &cli.Command{
		Short:     "update-image",
		UsageLine: "automation update-image [flags]",
		Long:      updateImageLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			runner := newUpdateImageRunner(cmd.Config)
			return runner.run(ctx)
		},
	}

	cmdUpdateImage.Init()
	addFlagBuild(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagProject(cmdUpdateImage.Flags, cmdUpdateImage.Config)
	addFlagPush(cmdUpdateImage.Flags, cmdUpdateImage.Config)

	return cmdUpdateImage
}
