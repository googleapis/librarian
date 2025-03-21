// Copyright 2024 Google LLC
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

package command

import (
	"slices"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// CommitMessage represents the result of parsing a git commit message,
// interpreting conventional commits for "feat", "docs" and "fix" (while
// ignoring other types such as "refactor" and "chore" which are not expected
// to be interesting to consumers).
type CommitMessage struct {
	Features     []string
	Docs         []string
	Fixes        []string
	PiperOrigins []string
	SourceLinks  []string
	// TODO: Keep a list of the breaking change lines?
	Breaking   bool
	CommitHash string
	// Libraries which should be released by this commit, even if they would not be normally.
	TriggerLibraries []string
	// Libraries which not should be released by this commit, even if they would be normally.
	NoTriggerLibraries []string
}

// Parses a message from a commit, remembering the hash of the commit it
// came from and extracting conventional commit information
// (and PiperOrigin-RevId/SourceLink details) from it.
// Currently ignores any line of the message which does not contain a conventional commit.
func ParseCommit(commit object.Commit) *CommitMessage {
	const PiperPrefix = "PiperOrigin-RevId: "
	const SourceLinkPrefix = "SourceLink: "
	const TriggerReleasePrefix = "TriggerRelease: "
	const NoTriggerReleasePrefix = "NoTriggerRelease: "

	features := []string{}
	docs := []string{}
	fixes := []string{}
	piperOrigins := []string{}
	sourceLinks := []string{}
	triggerLibraries := []string{}
	noTriggerLibraries := []string{}
	breaking := false

	messageLines := strings.Split(commit.Message, "\n")
	for _, line := range messageLines {
		// Handle any known prefixes that we just want to keep in a simple way.
		if maybeAppendString(line, PiperPrefix, &piperOrigins) ||
			maybeAppendString(line, SourceLinkPrefix, &sourceLinks) ||
			maybeAppendString(line, TriggerReleasePrefix, &triggerLibraries) ||
			maybeAppendString(line, NoTriggerReleasePrefix, &noTriggerLibraries) {
			continue
		}

		// Now see if the line represents a conventional commit.
		colon := strings.Index(line, ":")
		if colon == -1 {
			continue
		}
		prefix := line[:colon]
		// Remember whether this line represents a breaking change. (We don't want to just
		// change "breaking" yet, as it may not be a conventional commit at all.)
		lineBreaking := strings.Contains(prefix, "!")
		// Remove any ! now that we've seen whether the prefix contains it.
		prefix = strings.ReplaceAll(prefix, "!", "")
		// Remove anything after a bracket, e.g. feat(spanner) just becomes feat
		prefix = strings.Split(prefix, "(")[0]
		var slice *[]string
		switch prefix {
		case "feat":
			slice = &features
		case "doc":
		case "docs":
			slice = &docs
		case "fixes":
			slice = &fixes
		case "refactor":
		case "tools":
		case "chore":
		case "test":
		case "tests":
		case "deps":
			// Conventional commit type we know about, but don't keep.
			// TODO: Maybe we should keep deps?
			slice = nil
		default:
			// Not a conventional commit line (that we recognise, anyway) - ignore it
			continue
		}
		if slice != nil {
			*slice = append(*slice, strings.TrimSpace(line[colon+1:]))
		}
		breaking = breaking || lineBreaking
	}

	return &CommitMessage{
		Features:           features,
		Docs:               docs,
		Fixes:              fixes,
		PiperOrigins:       piperOrigins,
		SourceLinks:        sourceLinks,
		TriggerLibraries:   triggerLibraries,
		NoTriggerLibraries: noTriggerLibraries,
		Breaking:           breaking,
		CommitHash:         commit.Hash.String(),
	}
}

// Returns whether this commit should trigger a release for the given library.
func IsReleaseWorthy(message *CommitMessage, libraryId string) bool {
	if slices.Contains(message.NoTriggerLibraries, libraryId) {
		return false
	}
	if slices.Contains(message.TriggerLibraries, libraryId) {
		return true
	}
	return len(message.Features) > 0 || len(message.Fixes) > 0 || message.Breaking
}

func maybeAppendString(line, prefix string, slice *[]string) bool {
	if !strings.HasPrefix(line, prefix) {
		return false
	}
	*slice = append(*slice, strings.TrimSpace(line[len(prefix):]))
	return true
}
