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
// WITHOUT WARRANTIES, OR CONDITIONS OF ANY KIND, either express or implied.
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

// footerRegex defines the format for a conventional commit footer.
// A footer key consists of letters and hyphens, or is the "BREAKING CHANGE"
// literal. The key is followed by ": " and then the value.
// e.g., "Reviewed-by: G. Gemini" or "BREAKING CHANGE: an API was changed".
var footerRegex = regexp.MustCompile(`^([A-Za-z-]+|` + breakingChangeKey + `):\s(.*)`)

// parsedHeader holds the result of parsing the header line.
type parsedHeader struct {
	Type        string
	Scope       string
	Description string
	IsBreaking  bool
}

// parseHeader parses the header line of a commit message.
func parseHeader(headerLine string) (*parsedHeader, bool) {
	match := commitRegex.FindStringSubmatch(headerLine)
	if len(match) == 0 {
		return nil, false
	}

	capturesMap := make(map[string]string)
	for i, name := range commitRegex.SubexpNames()[1:] {
		if name != "" {
			capturesMap[name] = match[i+1]
		}
	}

	return &parsedHeader{
		Type:        capturesMap["type"],
		Scope:       capturesMap["scope"],
		Description: capturesMap["description"],
		IsBreaking:  capturesMap["breaking"] == "!",
	}, true
}

// separateBodyAndFooters splits the lines after the header into body and footer sections.
func separateBodyAndFooters(lines []string) (bodyLines, footerLines []string) {
	inFooterSection := false
	for i, line := range lines {
		if inFooterSection {
			footerLines = append(footerLines, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			isSeparator := false
			// Look ahead at the next non-blank line.
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
				continue // Skip the blank separator line.
			}
		}
		bodyLines = append(bodyLines, line)
	}
	return bodyLines, footerLines
}

// parseFooters parses footer lines from a conventional commit message into a map
// of key-value pairs. It supports multi-line footers and also returns a
// boolean indicating if a breaking change was detected.
func parseFooters(footerLines []string) (footers map[string]string, isBreaking bool) {
	footers = make(map[string]string)
	var lastKey string
	for _, line := range footerLines {
		footerMatches := footerRegex.FindStringSubmatch(line)
		if len(footerMatches) == 0 {
			// Not a new footer. If we have a previous key and the line is not
			// empty, append it to the last value.
			if lastKey != "" && strings.TrimSpace(line) != "" {
				footers[lastKey] += "\n" + line
			}
			continue
		}
		// This is a new footer.
		key := strings.TrimSpace(footerMatches[1])
		value := strings.TrimSpace(footerMatches[2])
		footers[key] = value
		lastKey = key
		if key == breakingChangeKey {
			isBreaking = true
		}
	}
	return footers, isBreaking
}

// ParseCommit parses a single commit message and returns a ConventionalCommit.
// If the commit message does not follow the conventional commit format, it
// logs a warning and returns a nil commit and no error.
func ParseCommit(message, hashString string) (*ConventionalCommit, error) {
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil, fmt.Errorf("empty commit message")
	}
	lines := strings.Split(trimmedMessage, "\n")

	header, ok := parseHeader(lines[0])
	if !ok {
		slog.Warn("Invalid conventional commit message", "message", message, "hash", hashString)
		return nil, nil
	}

	bodyLines, footerLines := separateBodyAndFooters(lines[1:])

	footers, footerIsBreaking := parseFooters(footerLines)

	return &ConventionalCommit{
		Type:        header.Type,
		Scope:       header.Scope,
		Description: header.Description,
		Body:        strings.TrimSpace(strings.Join(bodyLines, "\n")),
		Footers:     footers,
		IsBreaking:  header.IsBreaking || footerIsBreaking,
		SHA:         hashString,
	}, nil
}
