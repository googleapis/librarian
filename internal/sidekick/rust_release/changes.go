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

package rustrelease

import (
	"fmt"
	"os/exec"

	"github.com/googleapis/librarian/internal/sidekick/config"
)

func matchesBranchPoint(config *config.Release) error {
	branch := fmt.Sprintf("%s/%s", config.Remote, config.Branch)
	delta := fmt.Sprintf("%s...HEAD", branch)
	cmd := exec.Command(gitExe(config), "diff", "--name-only", delta)
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	if len(output) != 0 {
		return fmt.Errorf("the local repository does not match is branch point from %s, change files:\n%s", branch, string(output))
	}
	return nil
}
