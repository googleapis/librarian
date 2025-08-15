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

package gitrepo

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// ConventionalCommit represents a parsed conventional commit message.
// See https://www.conventionalcommits.org/en/v1.0.0/ for details.
type ConventionalCommit struct {
	// Type is the type of change (e.g., "feat", "fix", "docs").
	Type string
	// Scope is the scope of the change.
	Scope string
	// Description is the short summary of the change.
	Description string
	// Body is the long-form description of the change.
	Body string
	// Footers contain metadata (e.g.,"BREAKING CHANGE", "Reviewed-by").
	Footers map[string]string
	// IsBreaking indicates if the commit introduces a breaking change.
	IsBreaking bool
	// SHA is the full commit hash.
	SHA string
}

const breakingChangeKey = "BREAKING CHANGE"

var commitRegex = regexp.MustCompile(`^(?P<type>\w+)(?:\((?P<scope>.*)\))?(?P<breaking>!)?:\s(?P<description>.*)`)
var footerRegex = regexp.MustCompile(`^([A-Za-z-]+|` + breakingChangeKey + `):\s(.*)`)

// ParseCommit parses a single commit message and returns a ConventionalCommit.
// If the commit message does not follow the conventional commit format, it
// logs a warning and returns a nil commit and no error.
func ParseCommit(message, hashString string) (*ConventionalCommit, error) {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil, fmt.Errorf("empty commit message")
	}
	lines := strings.Split(trimmedMessage, "\n")
	match := commitRegex.FindStringSubmatch(lines[0])
	if len(match) == 0 {
		slog.Warn("Invalid conventional commit message", "message", message, "hash", hashString)
		return nil, nil
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
			if key == breakingChangeKey {
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
