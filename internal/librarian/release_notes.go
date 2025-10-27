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
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
)

var (
	errPiperNotFound = errors.New("piper id not found")

	commitTypeToHeading = map[string]string{
		"feat":     "Features",
		"fix":      "Bug Fixes",
		"perf":     "Performance Improvements",
		"revert":   "Reverts",
		"docs":     "Documentation",
		"style":    "Styles",
		"chore":    "Miscellaneous Chores",
		"refactor": "Code Refactoring",
		"test":     "Tests",
		"build":    "Build System",
		"ci":       "Continuous Integration",
	}

	// commitTypeOrder is the order in which commit types should appear in release notes.
	// Only these listed are included in release notes.
	commitTypeOrder = []string{
		"feat",
		"fix",
		"perf",
		"revert",
		"docs",
	}

	shortSHA = func(sha string) string {
		if len(sha) < 8 {
			return sha
		}
		return sha[:8]
	}

	releaseNotesTemplate = template.Must(template.New("releaseNotes").Funcs(template.FuncMap{
		"shortSHA": shortSHA,
	}).Parse(`Librarian Version: {{.LibrarianVersion}}
Language Image: {{.ImageVersion}}
{{ $prInfo := . }}
{{- range .NoteSections -}}
<details><summary>{{.LibraryID}}: {{.NewVersion}}</summary>

## [{{.NewVersion}}]({{"https://github.com/"}}{{$prInfo.RepoOwner}}/{{$prInfo.RepoName}}/compare/{{.PreviousTag}}...{{.NewTag}}) ({{$prInfo.Date}})
{{ range .CommitSections }}
### {{.Heading}}
{{ range .Commits }}
{{ if not .IsBulkCommit -}}
{{ if .PiperCLNumber -}}
* {{.Subject}} (PiperOrigin-RevId: {{.PiperCLNumber}}) ([{{shortSHA .CommitHash}}]({{"https://github.com/"}}{{$prInfo.RepoOwner}}/{{$prInfo.RepoName}}/commit/{{shortSHA .CommitHash}}))
{{- else -}}
* {{.Subject}} ([{{shortSHA .CommitHash}}]({{"https://github.com/"}}{{$prInfo.RepoOwner}}/{{$prInfo.RepoName}}/commit/{{shortSHA .CommitHash}}))
{{- end }}
{{- end }}
{{ end }}

{{- end }}
</details>


{{ end }}
{{- if .BulkChanges -}}
<details><summary>Bulk Changes</summary>
{{ range .BulkChanges }}
{{ if .PiperCLNumber -}}
* {{.Type}}: {{.Subject}} (PiperOrigin-RevId: {{.PiperCLNumber}}) ([{{shortSHA .CommitHash}}]({{"https://github.com/"}}{{$prInfo.RepoOwner}}/{{$prInfo.RepoName}}/commit/{{shortSHA .CommitHash}}))
  Libraries: {{.LibraryIDs}}
{{- else -}}
* {{.Type}}: {{.Subject}} ([{{shortSHA .CommitHash}}]({{"https://github.com/"}}{{$prInfo.RepoOwner}}/{{$prInfo.RepoName}}/commit/{{shortSHA .CommitHash}}))
  Libraries: {{.LibraryIDs}}
{{- end }}
{{- end }}
</details>
{{ end }}
`))

	genBodyTemplate = template.Must(template.New("genBody").Funcs(template.FuncMap{
		"shortSHA": shortSHA,
	}).Parse(`BEGIN_COMMIT
{{ range .Commits }}
BEGIN_NESTED_COMMIT
{{.Type}}: {{.Subject}}
{{.Body}}

PiperOrigin-RevId: {{index .Footers "PiperOrigin-RevId"}}
Library-IDs: {{index .Footers "Library-IDs"}}
Source-link: [googleapis/googleapis@{{shortSHA .CommitHash}}](https://github.com/googleapis/googleapis/commit/{{shortSHA .CommitHash}})
END_NESTED_COMMIT
{{ end }}
END_COMMIT

This pull request is generated with proto changes between
[googleapis/googleapis@{{shortSHA .StartSHA}}](https://github.com/googleapis/googleapis/commit/{{.StartSHA}})
(exclusive) and
[googleapis/googleapis@{{shortSHA .EndSHA}}](https://github.com/googleapis/googleapis/commit/{{.EndSHA}})
(inclusive).

Librarian Version: {{.LibrarianVersion}}
Language Image: {{.ImageVersion}}

{{- if .FailedLibraries }}

## Generation failed for
{{- range .FailedLibraries }}
- {{ . }}
{{- end -}}
{{- end }}
`))

	onboardingBodyTemplate = template.Must(template.New("onboardingBody").Parse(`BEGIN_COMMIT

feat: onboard a new library

PiperOrigin-RevId: {{.PiperID}}
Library-IDs: {{.LibraryID}}

END_COMMIT

Librarian Version: {{.LibrarianVersion}}
Language Image: {{.ImageVersion}}
`))
)

type releasePRBody struct {
	LibrarianVersion string
	ImageVersion     string
	RepoOwner        string
	RepoName         string
	Date             string
	NoteSections     []*releaseNoteSection
	BulkChanges      []*config.Commit
}

type releaseNoteSection struct {
	LibraryID      string
	PreviousTag    string
	NewTag         string
	NewVersion     string
	CommitSections []*commitSection
}

type commitSection struct {
	Heading string
	Commits []*config.Commit
}

// formatReleaseNotes generates the body for a release pull request.
func formatReleaseNotes(state *config.LibrarianState, ghRepo *github.Repository) (string, error) {
	librarianVersion := cli.Version()
	// Separate commits to bulk changes (affects multiple libraries) or library-specific changes because they
	// appear in different section in the release notes.
	bulkChangesMap, libraryChanges := separateCommits(state)
	// Process library specific changes.
	var releaseSections []*releaseNoteSection
	for _, library := range state.Libraries {
		if !library.ReleaseTriggered {
			continue
		}
		// No need to check the existence of the key, library.ID, because a library without library-specific changes
		// may appear in the release notes, i.e., in the bulk changes section.
		commits := libraryChanges[library.ID]
		section := formatLibraryReleaseNotes(library, commits)
		releaseSections = append(releaseSections, section)
	}
	// Process bulk changes
	var bulkChanges []*config.Commit
	for _, commit := range bulkChangesMap {
		bulkChanges = append(bulkChanges, commit)
	}
	sort.Slice(bulkChanges, func(i, j int) bool {
		return bulkChanges[i].CommitHash < bulkChanges[j].CommitHash
	})

	data := &releasePRBody{
		LibrarianVersion: librarianVersion,
		Date:             time.Now().Format("2006-01-02"),
		RepoOwner:        ghRepo.Owner,
		RepoName:         ghRepo.Name,
		ImageVersion:     state.Image,
		NoteSections:     releaseSections,
		BulkChanges:      bulkChanges,
	}

	var out bytes.Buffer
	if err := releaseNotesTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// formatLibraryReleaseNotes generates release notes in Markdown format for a single library.
// It returns the generated release notes and the new version string.
func formatLibraryReleaseNotes(library *config.LibraryState, commits []*config.Commit) *releaseNoteSection {
	// The version should already be updated to the next version.
	newVersion := library.Version
	tagFormat := config.DetermineTagFormat(library.ID, library, nil)
	newTag := config.FormatTag(tagFormat, library.ID, newVersion)
	previousTag := config.FormatTag(tagFormat, library.ID, library.PreviousVersion)

	sort.Slice(commits, func(i, j int) bool {
		return commits[i].CommitHash < commits[j].CommitHash
	})
	commitsByType := make(map[string][]*config.Commit)
	for _, commit := range commits {
		commitsByType[commit.Type] = append(commitsByType[commit.Type], commit)
	}

	var sections []*commitSection
	// Group commits by type, according to commitTypeOrder, to be used in the release notes.
	for _, ct := range commitTypeOrder {
		displayName, headingOK := commitTypeToHeading[ct]
		typedCommits, commitsOK := commitsByType[ct]
		if headingOK && commitsOK {
			sections = append(sections, &commitSection{
				Heading: displayName,
				Commits: typedCommits,
			})
		}
	}

	section := &releaseNoteSection{
		LibraryID:      library.ID,
		NewVersion:     newVersion,
		PreviousTag:    previousTag,
		NewTag:         newTag,
		CommitSections: sections,
	}

	return section
}

// separateCommits analyzes all commits associated with triggered releases in the
// given state and categorizes them into two groups:
//
// 1. Bulk Changes: Commits that affect multiple libraries. This includes:
//   - Commits identified by IsBulkCommit() (e.g., librarian generation PRs).
//   - Commits that appear in multiple libraries' change sets but are not
//     marked as bulk commits (e.g., dependency updates, README changes).
//     The Library-IDs for these are concatenated.
//
// 2. Library Changes: Commits that are unique to a single library.
//
// It returns two maps:
//   - The first map contains bulk changes, keyed by a composite of commit hash and subject.
//   - The second map contains library-specific changes, keyed by LibraryID.
func separateCommits(state *config.LibrarianState) (map[string]*config.Commit, map[string][]*config.Commit) {
	maybeBulkChanges := make(map[string][]*config.Commit)
	for _, library := range state.Libraries {
		if !library.ReleaseTriggered {
			continue
		}

		for _, commit := range library.Changes {
			key := commit.CommitHash + commit.Subject
			maybeBulkChanges[key] = append(maybeBulkChanges[key], commit)
		}
	}

	bulkChanges := make(map[string]*config.Commit)
	libraryChanges := make(map[string][]*config.Commit)
	for key, commits := range maybeBulkChanges {
		// A commit has multiple library IDs in the footer, this should come from librarian generation PR.
		// All commits should be identical.
		if commits[0].IsBulkCommit() {
			bulkChanges[key] = commits[0]
			continue
		}
		// More than ten commits have the same commit subject and sha, this should come from other sources,
		// e.g., dependency updates, README updates, etc.
		// All commits should be identical except for the library id.
		// We assume this type of commits has only one library id in Footers and each id is unique among all
		// commits.
		if len(commits) >= config.BulkChangeThreshold {
			bulkChanges[key] = concatenateLibraryIDs(commits)
			continue
		}
		// We assume the rest of commits are library-specific.
		for _, commit := range commits {
			// Non-bulk commits may have 1 - 9 library IDs.
			libraryIDs := strings.Split(commit.LibraryIDs, ",")
			for _, libraryID := range libraryIDs {
				if libraryID == "" {
					continue
				}
				libraryChanges[libraryID] = append(libraryChanges[libraryID], commit)
			}
		}
	}

	return bulkChanges, libraryChanges
}

// concatenateLibraryIDs merges the LibraryIDs from a slice of commits into the first commit.
func concatenateLibraryIDs(commits []*config.Commit) *config.Commit {
	var libraryIDs []string
	for _, commit := range commits {
		libraryIDs = append(libraryIDs, commit.LibraryIDs)
	}

	sort.Slice(libraryIDs, func(i, j int) bool {
		return libraryIDs[i] < libraryIDs[j]
	})
	commits[0].LibraryIDs = strings.Join(libraryIDs, ",")
	return commits[0]
}
