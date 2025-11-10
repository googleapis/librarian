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
	cmd := &cli.Command{
		Short:     "",
		UsageLine: "",
		Long:      "",
		Commands: []*cli.Command{
			newCmdPublishRelease(),
		},
	}

	cmd.Init()
	return cmd
}

func newCmdPublishRelease() *cli.Command {
	cmdPublishRelease := &cli.Command{
		Short:     "",
		UsageLine: "",
		Long:      "",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if _, err := cmd.Config.IsValid(); err != nil {
				return fmt.Errorf("failed to validate config: %s", err)
			}
			runner := newPublishRunner(cmd.Config)
			return runner.run(ctx)
		},
	}

	cmdPublishRelease.Init()
	addFlagForceRun(cmdPublishRelease.Flags, cmdPublishRelease.Config)
	addFlagProject(cmdPublishRelease.Flags, cmdPublishRelease.Config)
	addFlagPush(cmdPublishRelease.Flags, cmdPublishRelease.Config)

	return cmdPublishRelease
}
