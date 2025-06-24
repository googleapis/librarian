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
	"github.com/googleapis/librarian/internal/config"
)

var CmdLibrarian = &cli.Command{
	Short: "librarian manages client libraries for Google APIs",
	Usage: "librarian <command> [arguments]",
	Long:  "Librarian manages client libraries for Google APIs.",
}

func init() {
	CmdLibrarian.Commands = append(CmdLibrarian.Commands,
		CmdConfigure,
		CmdGenerate,
		CmdUpdateApis,
		CmdCreateReleasePR,
		CmdUpdateImageTag,
		CmdMergeReleasePR,
		CmdCreateReleaseArtifacts,
		CmdPublishReleaseArtifacts,
		CmdVersion,
	)
}

func Run(ctx context.Context, arg ...string) error {
	CmdLibrarian.InitFlags()
	if err := CmdLibrarian.Parse(arg); err != nil {
		return err
	}
	if len(arg) == 0 {
		CmdLibrarian.Help()
		return fmt.Errorf("command not specified")
	}
	cmd, err := CmdLibrarian.Lookup(arg[0])
	if err != nil {
		CmdLibrarian.Help()
		return err
	}
	if err := cmd.Parse(arg[1:]); err != nil {
		return err
	}
	slog.Info("librarian", "arguments", arg)

	cfg := config.New()
	applyFlags(cfg)
	return cmd.Run(ctx, cfg)
}
