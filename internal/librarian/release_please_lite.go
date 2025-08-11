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
	"fmt"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// ConventionalCommit represents a parsed conventional commit message.
// See https://www.conventionalcommits.org/en/v1.0.0/ for details.
type ConventionalCommit struct {
	Type        string
	Scope       string
	Description string
	Body        string
	Footers     map[string]string
	IsBreaking  bool
	SHA         string
}

var commitRegex = regexp.MustCompile(`^(?P<type>\w+)(?:\((?P<scope>.*)\))?(?P<breaking>!)?:\s(?P<description>.*)`)
var footerRegex = regexp.MustCompile(`^([A-Za-z-]+|BREAKING CHANGE):\s(.*)`)

// ParseCommit parses a single commit message and returns a ConventionalCommit.
// If the commit message does not follow the conventional commit format,
// nil is returned.
func ParseCommit(message, hashString string) (*ConventionalCommit, error) {
	lines := strings.Split(strings.TrimSpace(message), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty commit message")
	}
	match := commitRegex.FindStringSubmatch(lines[0])
	if len(match) == 0 {
		return nil, fmt.Errorf("invalid commit message: %s", message)
	}

	result := make(map[string]string)
	for i, name := range commitRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	cc := &ConventionalCommit{
		Type:        result["type"],
		Scope:       result["scope"],
		Description: result["description"],
		IsBreaking:  result["breaking"] == "!",
		Footers:     make(map[string]string),
		SHA:         hashString,
	}

	var bodyLines []string
	var footerLines []string
	inFooterSection := false

	for i := 1; i < len(lines); i++ {
		line := lines[i]

		if !inFooterSection {
			if strings.TrimSpace(line) == "" {
				// Potential separator. Check if subsequent lines are footers.
				// If they are, this blank line is the separator.
				// If not, this blank line is part of the body.
				isSeparator := false
				for j := i + 1; j < len(lines); j++ {
					if strings.TrimSpace(lines[j]) != "" {
						if footerRegex.MatchString(lines[j]) {
							isSeparator = true
						}
						break
					}
				}
				if isSeparator {
					inFooterSection = true
					continue // Skip the blank separator line
				}
			}
			bodyLines = append(bodyLines, line)
		} else {
			footerLines = append(footerLines, line)
		}
	}

	// Process footers with multi-line support.
	var lastKey string
	for _, line := range footerLines {
		footerMatches := footerRegex.FindStringSubmatch(line)
		if len(footerMatches) > 0 {
			key := strings.TrimSpace(footerMatches[1])
			value := strings.TrimSpace(footerMatches[2])
			cc.Footers[key] = value
			lastKey = key
			if key == "BREAKING CHANGE" {
				cc.IsBreaking = true
			}
		} else if lastKey != "" && strings.TrimSpace(line) != "" {
			// This is a continuation of the previous footer.
			cc.Footers[lastKey] += "\n" + line
		}
	}

	cc.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	return cc, nil
}

// GetCommits returns all conventional commits for the given library since the
// version specified in the state file.
func GetCommits(repo *gitrepo.LocalRepository, state *config.LibrarianState, library *config.LibraryState) ([]*ConventionalCommit, error) {
	var paths []string
	paths = append(paths, library.SourceRoots...)

	commits, err := repo.GetCommitsForPathsSinceTag(paths, library.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %s: %w", library.ID, err)
	}
	conventionalCommits := []*ConventionalCommit{}
	for _, commit := range commits {
		files, err := repo.ChangedFilesInCommit(commit.Hash.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get changed files for commit %s: %w", commit.Hash.String(), err)
		}
		if shouldExclude(files, library.ReleaseExcludePaths) {
			continue
		}
		conventionalCommit, err := ParseCommit(commit.Message, commit.Hash.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse commit %s: %w", commit.Hash.String(), err)
		}
		conventionalCommits = append(conventionalCommits, conventionalCommit)
	}
	return conventionalCommits, nil
}

func shouldExclude(files, excludePaths []string) bool {
	for _, file := range files {
		excluded := false
		for _, excludePath := range excludePaths {
			if strings.HasPrefix(file, excludePath) {
				excluded = true
				break
			}
		}
		if !excluded {
			return false
		}
	}
	return true
}
