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

package conventionalcommits

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/gitrepo"
)

// ConventionalCommit represents a parsed conventional commit message.
// See https://www.conventionalcommits.org/en/v1.0.0/ for details.
type ConventionalCommit struct {
	// Type is the type of change (e.g., "feat", "fix", "docs").
	Type string `yaml:"type" json:"type"`
	// Subject is the short summary of the change.
	Subject string `yaml:"subject" json:"subject"`
	// Body is the long-form description of the change.
	Body string `yaml:"body" json:"body"`
	// LibraryID is the library ID the commit associated with.
	LibraryID string `yaml:"-" json:"-"`
	// Scope is the scope of the change.
	Scope string `yaml:"-" json:"-"`
	// Footers contain metadata (e.g,"BREAKING CHANGE", "Reviewed-by").
	Footers map[string]string `yaml:"-" json:"-"`
	// IsBreaking indicates if the commit introduces a breaking change.
	IsBreaking bool `yaml:"-" json:"-"`
	// IsNested indicates if the commit is a nested commit.
	IsNested bool `yaml:"-" json:"-"`
	// SHA is the full commit hash.
	SHA string `yaml:"-" json:"-"`
	// When is the timestamp of the commit.
	When time.Time `yaml:"-" json:"-"`
}

// MarshalJSON implements a custom JSON marshaler for ConventionalCommit.
func (c *ConventionalCommit) MarshalJSON() ([]byte, error) {
	type Alias ConventionalCommit
	return json.Marshal(&struct {
		*Alias
		PiperCLNumber    string `json:"piper_cl_number,omitempty"`
		SourceCommitHash string `json:"source_commit_hash,omitempty"`
	}{
		Alias:            (*Alias)(c),
		PiperCLNumber:    c.Footers["PiperOrigin-RevId"],
		SourceCommitHash: c.Footers["git-commit-hash"],
	})
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

const (
	beginCommitOverride = "BEGIN_COMMIT_OVERRIDE"
	endCommitOverride   = "END_COMMIT_OVERRIDE"
	beginNestedCommit   = "BEGIN_NESTED_COMMIT"
	endNestedCommit     = "END_NESTED_COMMIT"
)

func extractCommitMessageOverride(message string) string {
	beginIndex := strings.Index(message, beginCommitOverride)
	if beginIndex == -1 {
		return message
	}
	afterBegin := message[beginIndex+len(beginCommitOverride):]
	endIndex := strings.Index(afterBegin, endCommitOverride)
	if endIndex == -1 {
		return message
	}
	return strings.TrimSpace(afterBegin[:endIndex])
}

// commitPart holds the raw string of a commit message and whether it's nested.
type commitPart struct {
	message  string
	isNested bool
}

func extractCommitParts(message string) []commitPart {
	parts := strings.Split(message, beginNestedCommit)
	var commitParts []commitPart

	// The first part is the primary commit.
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		commitParts = append(commitParts, commitPart{
			message:  strings.TrimSpace(parts[0]),
			isNested: false,
		})
	}

	// The rest of the parts are nested commits.
	for i := 1; i < len(parts); i++ {
		nestedPart := parts[i]
		endIndex := strings.Index(nestedPart, endNestedCommit)
		if endIndex == -1 {
			// Malformed, ignore.
			continue
		}
		commitStr := strings.TrimSpace(nestedPart[:endIndex])
		if commitStr == "" {
			continue
		}
		commitParts = append(commitParts, commitPart{
			message:  commitStr,
			isNested: true,
		})
	}
	return commitParts
}

// ParseCommits parses a commit message into a slice of ConventionalCommit structs.
//
// It supports an override block wrapped in BEGIN_COMMIT_OVERRIDE and
// END_COMMIT_OVERRIDE. If found, this block takes precedence, and only its
// content will be parsed.
//
// The message can also contain multiple nested commits, each wrapped in
// BEGIN_NESTED_COMMIT and END_NESTED_COMMIT markers.
//
// Malformed override or nested blocks (e.g., with a missing end marker) are
// ignored. Any commit part that is found but fails to parse as a valid
// conventional commit is logged and skipped.
func ParseCommits(commit *gitrepo.Commit, libraryID string) ([]*ConventionalCommit, error) {
	message := commit.Message
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("empty commit message")
	}
	message = extractCommitMessageOverride(message)

	var commits []*ConventionalCommit

	for _, part := range extractCommitParts(message) {
		c, err := parseSimpleCommit(part, commit, libraryID)
		if err != nil {
			slog.Warn("failed to parse commit part", "commit", part.message, "error", err)
			continue
		}

		if c != nil {
			commits = append(commits, c)
		}
	}

	return commits, nil
}

// parseSimpleCommit parses a simple commit message and returns a ConventionalCommit.
// A simple commit message is commit that does not include override or nested commits.
func parseSimpleCommit(commitPart commitPart, commit *gitrepo.Commit, libraryID string) (*ConventionalCommit, error) {
	trimmedMessage := strings.TrimSpace(commitPart.message)
	if trimmedMessage == "" {
		return nil, fmt.Errorf("empty commit message")
	}
	lines := strings.Split(trimmedMessage, "\n")

	header, ok := parseHeader(lines[0])
	if !ok {
		slog.Warn("Invalid conventional commit message", "message", commitPart.message, "hash", commit.Hash.String())
		return nil, nil
	}

	bodyLines, footerLines := separateBodyAndFooters(lines[1:])

	footers, footerIsBreaking := parseFooters(footerLines)

	return &ConventionalCommit{
		Type:       header.Type,
		Scope:      header.Scope,
		Subject:    header.Description,
		Body:       strings.TrimSpace(strings.Join(bodyLines, "\n")),
		LibraryID:  libraryID,
		Footers:    footers,
		IsBreaking: header.IsBreaking || footerIsBreaking,
		IsNested:   commitPart.isNested,
		SHA:        commit.Hash.String(),
		When:       commit.When,
	}, nil
}
