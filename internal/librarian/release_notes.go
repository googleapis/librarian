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

package librarian

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/conventionalcommits"
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

	// The order in which commit types should appear in release notes.
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
	}).Parse(`## [{{.NewVersion}}]({{"https://github.com/"}}{{.RepoOwner}}/{{.RepoName}}/compare/{{.PreviousTag}}...{{.NewTag}}) ({{.Date}})
{{- range .CommitTypes -}}
{{- if .Commits -}}
{{- if .Heading}}

### {{.Heading}}
{{end}}

{{- range .Commits -}}
* {{.Description}} ([{{shortSHA .SHA}}]({{"https://github.com/"}}{{$.RepoOwner}}/{{$.RepoName}}/commit/{{.SHA}}))
{{- end -}}
{{- end -}}
{{- end -}}`))
)

// LibraryRelease holds information for a single library's release.
type LibraryRelease struct {
	PreviousTag string
	NewTag      string
	NewVersion  string
	Commits     []*conventionalcommits.ConventionalCommit
}

// FormatReleaseNotes generates the body for a release pull request.
func FormatReleaseNotes(releases map[string]*LibraryRelease, repoOwner, repoName, librarianVersion, languageImage string) string {
	var body bytes.Buffer
	if librarianVersion != "" {
		fmt.Fprintf(&body, "Librarian Version: %s\n", librarianVersion)
	}
	if languageImage != "" {
		fmt.Fprintf(&body, "Language Image: %s\n\n", languageImage)
	}

	// Sort library names for consistent output
	libraryNames := make([]string, 0, len(releases))
	for name := range releases {
		libraryNames = append(libraryNames, name)
	}
	sort.Strings(libraryNames)

	for i, name := range libraryNames {
		release := releases[name]
		fmt.Fprintf(&body, "<details><summary>%s: %s</summary>\n\n", name, release.NewVersion)
		notes := formatLibraryReleaseNotes(release.Commits, release.PreviousTag, release.NewTag, release.NewVersion, repoOwner, repoName)
		body.WriteString(notes)
		body.WriteString("\n\n</details>")
		if i < len(libraryNames)-1 {
			body.WriteString("\n")
		}
	}

	return body.String()
}

// formatLibraryReleaseNotes generates release notes in Markdown format for a single library.
func formatLibraryReleaseNotes(commits []*conventionalcommits.ConventionalCommit, previousTag, newTag, newVersion, repoOwner, repoName string) string {
	commitsByType := make(map[string][]*conventionalcommits.ConventionalCommit)
	for _, commit := range commits {
		commitsByType[commit.Type] = append(commitsByType[commit.Type], commit)
	}

	type commitType struct {
		Heading string
		Commits []*conventionalcommits.ConventionalCommit
	}
	var commitTypes []commitType
	for _, ct := range commitTypeOrder {
		if displayName, ok := commitTypeToHeading[ct]; ok {
			if typedCommits, ok := commitsByType[ct]; ok {
				commitTypes = append(commitTypes, commitType{
					Heading: displayName,
					Commits: typedCommits,
				})
			}
		}
	}

	var out bytes.Buffer
	data := struct {
		NewVersion  string
		PreviousTag string
		NewTag      string
		RepoOwner   string
		RepoName    string
		Date        string
		CommitTypes []commitType
	}{
		NewVersion:  newVersion,
		PreviousTag: previousTag,
		NewTag:      newTag,
		RepoOwner:   repoOwner,
		RepoName:    repoName,
		Date:        time.Now().Format("2006-01-02"),
		CommitTypes: commitTypes,
	}
	if err := releaseNotesTemplate.Execute(&out, data); err != nil {
		// This should not happen, as the template is valid and the data is structured correctly.
		return fmt.Sprintf("Error executing template: %v", err)
	}

	return strings.TrimSpace(out.String())
}
