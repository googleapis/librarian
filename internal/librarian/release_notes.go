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
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/conventionalcommits"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var (
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

	releaseNotesTemplate = template.Must(template.New("releaseNotes").Funcs(template.FuncMap{
		"shortSHA": func(sha string) string {
			if len(sha) < 7 {
				return sha
			}
			return sha[:7]
		},
	}).Parse(`## [{{.NewVersion}}]({{"https://github.com/"}}{{.Repo.Owner}}/{{.Repo.Name}}/compare/{{.PreviousTag}}...{{.NewTag}}) ({{.Date}})
{{- range .Sections -}}
{{- if .Commits -}}
{{- if .Heading}}

### {{.Heading}}
{{end}}

{{- range .Commits -}}
* {{.Description}} ([{{shortSHA .SHA}}]({{"https://github.com/"}}{{$.Repo.Owner}}/{{$.Repo.Name}}/commit/{{.SHA}}))
{{- end -}}
{{- end -}}
{{- end -}}`))
)

// FormatReleaseNotes generates the body for a release pull request.
func FormatReleaseNotes(repo gitrepo.Repository, state *config.LibrarianState) (string, error) {
	var body bytes.Buffer

	librarianVersion := cli.Version()
	fmt.Fprintf(&body, "Librarian Version: %s\n", librarianVersion)
	fmt.Fprintf(&body, "Language Image: %s\n\n", state.Image)

	for _, library := range state.Libraries {
		if !library.ReleaseTriggered {
			continue
		}

		notes, newVersion, err := formatLibraryReleaseNotes(repo, library)
		if err != nil {
			return "", fmt.Errorf("failed to format release notes for library %s: %w", library.ID, err)
		}
		fmt.Fprintf(&body, "<details><summary>%s: %s</summary>\n\n", library.ID, newVersion)

		body.WriteString(notes)
		body.WriteString("\n\n</details>")

		body.WriteString("\n")
	}
	return body.String(), nil
}

// formatLibraryReleaseNotes generates release notes in Markdown format for a single library.
// It returns the generated release notes and the new version string.
func formatLibraryReleaseNotes(repo gitrepo.Repository, library *config.LibraryState) (string, string, error) {
	ghRepo, err := github.FetchGitHubRepoFromRemote(repo)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch github repo from remote: %w", err)
	}
	previousTag := formatTag(library, "")
	commits, err := GetConventionalCommitsSinceLastRelease(repo, library)
	if err != nil {
		return "", "", fmt.Errorf("failed to get conventional commits for library %s: %w", library.ID, err)
	}
	newVersion, err := NextVersion(commits, library.Version, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to get next version for library %s: %w", library.ID, err)
	}
	newTag := formatTag(library, newVersion)

	commitsByType := make(map[string][]*conventionalcommits.ConventionalCommit)
	for _, commit := range commits {
		commitsByType[commit.Type] = append(commitsByType[commit.Type], commit)
	}

	type releaseNoteSection struct {
		Heading string
		Commits []*conventionalcommits.ConventionalCommit
	}
	var sections []releaseNoteSection
	// Group commits by type, according to commitTypeOrder, to be used in the release notes.
	for _, ct := range commitTypeOrder {
		displayName, headingOK := commitTypeToHeading[ct]
		typedCommits, commitsOK := commitsByType[ct]
		if headingOK && commitsOK {
			sections = append(sections, releaseNoteSection{
				Heading: displayName,
				Commits: typedCommits,
			})
		}
	}

	var out bytes.Buffer
	data := struct {
		NewVersion  string
		PreviousTag string
		NewTag      string
		Repo        *github.Repository
		Date        string
		Sections    []releaseNoteSection
	}{
		NewVersion:  newVersion,
		PreviousTag: previousTag,
		NewTag:      newTag,
		Repo:        ghRepo,
		Date:        time.Now().Format("2006-01-02"),
		Sections:    sections,
	}
	if err := releaseNotesTemplate.Execute(&out, data); err != nil {
		// This should not happen, as the template is valid and the data is structured correctly.
		return "", "", fmt.Errorf("error executing template: %v", err)
	}

	return strings.TrimSpace(out.String()), newVersion, nil
}
