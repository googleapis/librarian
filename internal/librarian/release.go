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

import (
	"github.com/googleapis/librarian/internal/cli"
)

// cmdRelease is the command for the `release` subcommand.
var cmdRelease = &cli.Command{
	Short:     "release manages releases of libraries.",
	UsageLine: "librarian release <command> [flags]",
	Long: `The release command orchestrate the creation of a release pull request.
The subcommands organize the tasks involed in creating a release: parsing conventional commits, 
determining the correct semantic version, and generating a changelog.`,
}

func init() {
	cmdRelease.Init()
	cmdRelease.Commands = append(cmdRelease.Commands,
		cmdInit,
		cmdTagAndRelease,
	)
}
