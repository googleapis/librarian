// Copyright 2025 Google LLC
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

package legacyautomation

import (
	"context"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacycli"
)

func newAutomationCommand() *legacycli.Command {
	commands := []*legacycli.Command{
		newCmdGenerate(),
		newCmdPublishRelease(),
		newCmdStageRelease(),
	}

	return legacycli.NewCommandSet(
		commands,
		"automation manages Cloud Build resources to run Librarian CLI.",
		"automation <command> [arguments]",
		automationLongHelp)
}

func newCmdGenerate() *legacycli.Command {
	cmdGenerate := &legacycli.Command{
		Short:     "generate",
		UsageLine: "automation generate [flags]",
		Long:      generateLongHelp,
		Action: func(ctx context.Context, cmd *legacycli.Command) error {
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

func newCmdPublishRelease() *legacycli.Command {
	cmdPublishRelease := &legacycli.Command{
		Short:     "publish-release",
		UsageLine: "automation publish-release [flags]",
		Long:      publishLongHelp,
		Action: func(ctx context.Context, cmd *legacycli.Command) error {
			runner := newPublishRunner(cmd.Config)
			return runner.run(ctx)
		},
	}

	cmdPublishRelease.Init()
	addFlagProject(cmdPublishRelease.Flags, cmdPublishRelease.Config)

	return cmdPublishRelease
}

func newCmdStageRelease() *legacycli.Command {
	cmdStageRelease := &legacycli.Command{
		Short:     "stage-release",
		UsageLine: "automation stage-release [flags]",
		Long:      stageLongHelp,
		Action: func(ctx context.Context, cmd *legacycli.Command) error {
			runner := newStageRunner(cmd.Config)
			return runner.run(ctx)
		},
	}

	cmdStageRelease.Init()
	addFlagProject(cmdStageRelease.Flags, cmdStageRelease.Config)
	addFlagPush(cmdStageRelease.Flags, cmdStageRelease.Config)

	return cmdStageRelease
}
