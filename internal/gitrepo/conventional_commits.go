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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

const (
	// beginCommit marks the start of the commit message block. It is used in conjunction with endCommit.
	beginCommit = "BEGIN_COMMIT"
	// beginCommitOverride marks the start of the commit message block. It is used in conjunction with endCommitOverride.
	// Deprecated: use beginCommit instead.
	beginCommitOverride = "BEGIN_COMMIT_OVERRIDE"
	// endCommit marks the end of the commit message block.
	endCommit = "END_COMMIT"
	// endCommitOverride marks the end of the commit message block.
	// Deprecated: use endCommit instead.
	endCommitOverride = "END_COMMIT_OVERRIDE"
	beginNestedCommit = "BEGIN_NESTED_COMMIT"
	endNestedCommit   = "END_NESTED_COMMIT"
	breakingChangeKey = "BREAKING CHANGE"
	sourceLinkKey     = "Source-Link"
)

var (
	commitRegex = regexp.MustCompile(`^(?P<type>\w+)(?:\((?P<scope>.*)\))?(?P<breaking>!)?:\s(?P<description>.*)`)
	// footerRegex defines the format for a conventional commit footer.
	// A footer key consists of letters and hyphens, or is the "BREAKING CHANGE"
	// literal. The key is followed by ":" and then the value.
	// e.g., "Reviewed-by: G. Gemini" or "BREAKING CHANGE: an API was changed".
	footerRegex     = regexp.MustCompile(`^([A-Za-z-]+|` + breakingChangeKey + `):(.*)`)
	sourceLinkRegex = regexp.MustCompile(`^\[googleapis/googleapis@(?P<shortSHA>.*)]\(https://github\.com/googleapis/googleapis/commit/(?P<sha>.*)\)$`)
	// libraryIDRegex extracts the libraryID from the commit message in a generation PR.
	// For a generation PR, each commit is expected to have the libraryID in brackets
	// ('[]').
	libraryIDRegex = regexp.MustCompile(`\[([^]]+)]`)
)

// ErrEmptyCommitMessage returns when the commit message is empty.
var ErrEmptyCommitMessage = errors.New("empty commit message")

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
	// Footers contain metadata (e.g,"BREAKING CHANGE", "Reviewed-by").
	// Repeated footer keys not supported, only first value is kept
	Footers map[string]string `yaml:"-" json:"-"`
	// IsBreaking indicates if the commit introduces a breaking change.
	IsBreaking bool `yaml:"-" json:"-"`
	// IsNested indicates if the commit is a nested commit.
	IsNested bool `yaml:"-" json:"-"`
	// CommitHash is the full commit hash.
	CommitHash string `yaml:"-" json:"commit_hash,omitempty"`
	// When is the timestamp of the commit.
	When time.Time `yaml:"-" json:"-"`
}

// parsedHeader holds the result of parsing the header line.
type parsedHeader struct {
	Type        string
	Scope       string
	Description string
	IsBreaking  bool
}

// extractLibraryID pulls the text between '[]' as the commit's libraryID
// A non generation PR may not have the associated libraryID between [] and
// will return an empty libraryID.
func (header *parsedHeader) extractLibraryID() string {
	matches := libraryIDRegex.FindStringSubmatch(header.Description)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

// commitPart holds the raw string of a commit message and whether it's nested.
type commitPart struct {
	message  string
	isNested bool
}

// MarshalJSON implements a custom JSON marshaler for ConventionalCommit.
func (c *ConventionalCommit) MarshalJSON() ([]byte, error) {
	type Alias ConventionalCommit
	return json.Marshal(&struct {
		*Alias
		PiperCLNumber string `json:"piper_cl_number,omitempty"`
	}{
		Alias:         (*Alias)(c),
		PiperCLNumber: c.Footers["PiperOrigin-RevId"],
	})
}

// ParseCommits parses a commit message into a slice of ConventionalCommit structs.
//
// It supports a top-level commit wrapped in BEGIN_COMMIT and END_COMMIT.
// If found, this block takes precedence, and only its content will be parsed.
//
// The message can also contain multiple nested commits, each wrapped in
// BEGIN_NESTED_COMMIT and END_NESTED_COMMIT markers.
//
// Malformed override or nested blocks (e.g., with a missing end marker) are
// ignored. Any commit part that is found but fails to parse as a valid
// conventional commit is logged and skipped.
func ParseCommits(commit *Commit, libraryID string) ([]*ConventionalCommit, error) {
	message := commit.Message
	if strings.TrimSpace(message) == "" {
		return nil, ErrEmptyCommitMessage
	}
	message = extractBeginCommitMessage(message)

	var commits []*ConventionalCommit

	for _, part := range extractCommitParts(message) {
		c, err := parseSimpleCommit(part, commit, libraryID)
		if err != nil {
			slog.Warn("failed to parse commit part", "commit", part.message, "error", err)
			continue
		}

		if c != nil {
			commits = append(commits, c...)
		}
	}

	return commits, nil
}

func extractBeginCommitMessage(message string) string {
	// Search the deprecated marker first because beginCommit is the prefix
	// of beginCommitOverride.
	beginMarker := beginCommitOverride
	endMarker := endCommitOverride
	beginIndex := strings.Index(message, beginMarker)
	if beginIndex == -1 {
		beginMarker = beginCommit
		endMarker = endCommit
		beginIndex = strings.Index(message, beginMarker)
	}
	if beginIndex == -1 {
		return message
	}

	afterBegin := message[beginIndex+len(beginMarker):]
	endIndex := strings.Index(afterBegin, endMarker)
	if endIndex == -1 {
		return message
	}

	return strings.TrimSpace(afterBegin[:endIndex])
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

// parseSimpleCommit parses a simple commit message and returns a slice of ConventionalCommit.
// A simple commit message is commit that does not include override or nested commits.
func parseSimpleCommit(commitPart commitPart, commit *Commit, libraryID string) ([]*ConventionalCommit, error) {
	trimmedMessage := strings.TrimSpace(commitPart.message)
	if trimmedMessage == "" {
		return nil, fmt.Errorf("empty commit message")
	}

	lines := strings.Split(trimmedMessage, "\n")
	bodyLines, footerLines := separateBodyAndFooters(lines)
	footers, footerIsBreaking := parseFooters(footerLines)
	processFooters(footers)

	var commits []*ConventionalCommit
	// Hold the subjects of each commit.
	var subjects [][]string
	// Hold the body of each commit.
	var body [][]string
	// Whether it has seen an empty line.
	var foundSeparator bool
	// If the body lines have multiple headers, separate them into different conventional commit, all associated with
	// the same commit sha.
	for _, bodyLine := range bodyLines {
		header, ok := parseHeader(bodyLine)
		if !ok {
			if len(commits) == 0 {
				// This should not happen as we expect a conventional commit message inside a nested commit.
				slog.Warn("bodyLine is not a header, not in a commit", "bodyLine", bodyLine, "hash", commit.Hash.String())
				continue
			}

			bodyLine = strings.TrimSpace(bodyLine)
			if bodyLine == "" {
				foundSeparator = true
				continue
			}

			if foundSeparator {
				// Since we have seen a separator, the rest of the lines are body lines of the commit.
				body[len(body)-1] = append(body[len(body)-1], bodyLine)
			} else {
				// We haven't seen a separator, this line is the continuation of the title.
				subjects[len(subjects)-1] = append(subjects[len(subjects)-1], bodyLine)
			}

			continue
		}

		subjects = append(subjects, []string{})
		body = append(body, []string{})
		foundSeparator = false
		// If there is an association for the commit (i.e. the commit has '[LIBRARY_ID]' in the
		// description), then use that libraryID. Otherwise, use the libraryID passed as the default.
		headerLibraryID := header.extractLibraryID()
		if headerLibraryID != "" {
			libraryID = headerLibraryID
		}

		commits = append(commits, &ConventionalCommit{
			Type:       header.Type,
			Subject:    header.Description,
			LibraryID:  libraryID,
			Footers:    footers,
			IsBreaking: header.IsBreaking || footerIsBreaking,
			IsNested:   commitPart.isNested,
			CommitHash: commit.Hash.String(),
			When:       commit.When,
		})
	}

	for i, commit := range commits {
		sub := fmt.Sprintf("%s %s", commit.Subject, strings.Join(subjects[i], " "))
		commit.Subject = strings.TrimSpace(sub)
		commit.Body = strings.Join(body[i], "\n")
	}

	return commits, nil
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
// It identifies the footer section by looking for a blank line followed by a
// line that matches the conventional commit footer format.
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
		if _, ok := footers[key]; ok {
			// Key already exists. Invalidate lastKey to prevent any subsequent
			// continuation lines from being appended to the wrong footer.
			lastKey = ""
			continue
		}
		value := strings.TrimSpace(footerMatches[2])
		footers[key] = value
		lastKey = key
		if key == breakingChangeKey {
			isBreaking = true
		}
	}
	for key, value := range footers {
		footers[key] = strings.TrimSpace(value)
	}
	return footers, isBreaking
}

// processFooters format value of certain keys.
func processFooters(footers map[string]string) {
	for key, value := range footers {
		if key == sourceLinkKey {
			// Exact commit sha from the googleapis commit.
			matches := sourceLinkRegex.FindStringSubmatch(value)
			if len(matches) == 0 {
				continue
			}
			footers[key] = matches[2]
		}
	}
}
