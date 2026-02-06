// Copyright 2026 Google LLC
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

package flags_parser

import (
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/urfave/cli/v3"
)

// Parses the command line flags for librarianops commands.
func ParseRepoFlags(cmd *cli.Command) (repoName, workDir string, err error) {
	workDir = cmd.String("C")
	command.Verbose = cmd.Bool("v")

	if workDir != "" {
		// When -C is provided, infer repo name from directory basename.
		repoName = filepath.Base(workDir)
	} else {
		// When -C is not provided, require positional repo argument.
		if cmd.Args().Len() == 0 {
			return "", "", fmt.Errorf("usage: librarianops <command> <repo> or librarianops <command> -C <dir>")
		}
		repoName = cmd.Args().Get(0)
	}
	return repoName, workDir, nil
}
