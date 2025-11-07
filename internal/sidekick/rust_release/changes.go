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
	"bytes"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
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

func isNewFile(config *config.Release, ref, name string) bool {
	delta := fmt.Sprintf("%s..HEAD", ref)
	cmd := exec.Command(gitExe(config), "diff", "--summary", delta, "--", name)
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	expected := fmt.Sprintf(" create mode 100644 %s", name)
	return bytes.HasPrefix(output, []byte(expected))
}

func filesChangedSince(config *config.Release, ref string) ([]string, error) {
	cmd := exec.Command(gitExe(config), "diff", "--name-only", ref)
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return filesFilter(config, strings.Split(string(output), "\n")), nil
}

func filesFilter(config *config.Release, files []string) []string {
	var patterns []gitignore.Pattern
	for _, p := range config.IgnoredChanges {
		patterns = append(patterns, gitignore.ParsePattern(p, nil))
	}
	matcher := gitignore.NewMatcher(patterns)

	files = slices.DeleteFunc(files, func(a string) bool {
		if a == "" {
			return true
		}
		return matcher.Match(strings.Split(a, "/"), false)
	})
	return files
}
