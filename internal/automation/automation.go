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

package automation

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/cli"
)

func newAutomationCommand() *cli.Command {
	cmdVersion := &cli.Command{
		Short:     "version prints the version information",
		UsageLine: "automation version",
		Long:      versionLongHelp,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(cli.Version())
			return nil
		},
	}
	cmdVersion.Init()

	cmd := &cli.Command{
		Short:     "automation manages Cloud Build resources to run Librarian CLI.",
		UsageLine: "automation <command> [arguments]",
		Long:      automationLongHelp,
		Commands: []*cli.Command{
			newCmdGenerate(),
			newCmdPublishRelease(),
			cmdVersion,
		},
	}

	cmd.Init()
	return cmd
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
