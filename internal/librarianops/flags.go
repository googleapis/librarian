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

package librarianops

import (
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// parseFlags parses the command line flags for librarianops commands.
func parseFlags(cmd *cli.Command) (repoName, workDir string, verbose bool, err error) {
	workDir = cmd.String("C")
	verbose = cmd.Bool("v")
	if workDir != "" {
		// When -C is provided, infer repo name from directory basename, having
		// it to an absolute directory (to allow "-C .")
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return "", "", verbose, fmt.Errorf("cannot resolve %s: %w", workDir, err)
		}
		workDir = absWorkDir
		repoName = filepath.Base(absWorkDir)
	} else {
		// When -C is not provided, require positional repo argument.
		if cmd.Args().Len() == 0 {
			return "", "", verbose, fmt.Errorf("usage: librarianops <command> <repo> or librarianops <command> -C <dir>")
		}
		repoName = cmd.Args().Get(0)
	}
	return repoName, workDir, verbose, nil
}
