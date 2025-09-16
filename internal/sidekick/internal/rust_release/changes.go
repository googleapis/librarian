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
	"os/exec"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
)

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
