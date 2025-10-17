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
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
)

var (
	errPiperNotFound = errors.New("piper id not found")

	// maxPRBodySize is the maximum size of a pull request body. The actual max
	// is 65536 bytes, but we round down for safety.
	maxPRBodySize = 65000

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
{{- if .TrimBody }}
The release notes have been trimmed. Please refer to the individual library release notes for more details.
{{ end }}
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
	}).Parse(`BEGIN_COMMIT_OVERRIDE
{{ range .Commits }}
BEGIN_NESTED_COMMIT
{{.Type}}: {{.Subject}}
{{.Body}}

PiperOrigin-RevId: {{index .Footers "PiperOrigin-RevId"}}
Library-IDs: {{index .Footers "Library-IDs"}}
Source-link: [googleapis/googleapis@{{shortSHA .CommitHash}}](https://github.com/googleapis/googleapis/commit/{{shortSHA .CommitHash}})
END_NESTED_COMMIT
{{ end }}
END_COMMIT_OVERRIDE

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

	onboardingBodyTemplate = template.Must(template.New("onboardingBody").Parse(`feat: onboard a new library

PiperOrigin-RevId: {{.PiperID}}
Library-IDs: {{.LibraryID}}
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
	// TrimBody indicates whether the body should be trimmed to fit within
	// the maximum size.
	TrimBody bool
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
	var releaseSections []*releaseNoteSection
	// create a map to deduplicate bulk changes based on their commit hash
	// and subject
	bulkChangesMap := make(map[string]*config.Commit)
	for _, library := range state.Libraries {
		if !library.ReleaseTriggered {
			continue
		}

		for _, commit := range library.Changes {
			if commit.IsBulkCommit() {
				bulkChangesMap[commit.CommitHash+commit.Subject] = commit
			}
		}

		section := formatLibraryReleaseNotes(library)
		releaseSections = append(releaseSections, section)
	}
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

	return writeReleaseBody(data, false)
}

func writeReleaseBody(data *releasePRBody, trimmed bool) (string, error) {
	var out bytes.Buffer
	if err := releaseNotesTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}
	b := out.Bytes()
	if len(b) > maxPRBodySize {
		if trimmed {
			return "", errors.New("the pull request body is too large even when trimmed down")
		}
		slog.Warn("the pull request body is too large, trimming down")
		data.BulkChanges = nil
		for _, section := range data.NoteSections {
			section.CommitSections = nil
		}
		data.TrimBody = true
		return writeReleaseBody(data, true)
	}

	return strings.TrimSpace(out.String()), nil
}

// formatLibraryReleaseNotes generates release notes in Markdown format for a single library.
// It returns the generated release notes and the new version string.
func formatLibraryReleaseNotes(library *config.LibraryState) *releaseNoteSection {
	// The version should already be updated to the next version.
	newVersion := library.Version
	tagFormat := config.DetermineTagFormat(library.ID, library, nil)
	newTag := config.FormatTag(tagFormat, library.ID, newVersion)
	previousTag := config.FormatTag(tagFormat, library.ID, library.PreviousVersion)

	commitsByType := make(map[string][]*config.Commit)
	for _, commit := range library.Changes {
		if !commit.IsBulkCommit() {
			commitsByType[commit.Type] = append(commitsByType[commit.Type], commit)
		}
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
