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
	"log/slog"
	"sort"
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

	shortSHA = func(sha string) string {
		if len(sha) < 7 {
			return sha
		}
		return sha[:7]
	}

	releaseNotesTemplate = template.Must(template.New("releaseNotes").Funcs(template.FuncMap{
		"shortSHA": shortSHA,
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

	genBodyTemplate = template.Must(template.New("genBody").Funcs(template.FuncMap{
		"shortSHA": shortSHA,
	}).Parse(`This pull request is generated with proto changes between
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

BEGIN_COMMIT_OVERRIDE
{{ range .Commits }}
BEGIN_NESTED_COMMIT
{{.Type}}: [{{.LibraryID}}] {{.Description}}
{{.Body}}

PiperOrigin-RevId: {{index .Footers "PiperOrigin-RevId"}}

Source-link: [googleapis/googleapis@{{shortSHA .SHA}}](https://github.com/googleapis/googleapis/commit/{{.SHA}})
END_NESTED_COMMIT
{{ end }}
END_COMMIT_OVERRIDE
`))
)

type generationPRBody struct {
	StartSHA         string
	EndSHA           string
	LibrarianVersion string
	ImageVersion     string
	Commits          []*conventionalcommits.ConventionalCommit
	FailedLibraries  []string
}

// formatGenerationPRBody creates the body of a generation pull request.
// Only consider libraries whose ID appears in idToCommits.
func formatGenerationPRBody(repo gitrepo.Repository, state *config.LibrarianState, idToCommits map[string]string, failedLibraries []string) (string, error) {
	var allCommits []*conventionalcommits.ConventionalCommit
	for _, library := range state.Libraries {
		lastGenCommit, ok := idToCommits[library.ID]
		if !ok {
			continue
		}

		commits, err := getConventionalCommitsSinceLastGeneration(repo, library, lastGenCommit)
		if err != nil {
			return "", fmt.Errorf("failed to fetch conventional commits for library, %s: %w", library.ID, err)
		}
		allCommits = append(allCommits, commits...)
	}

	if len(allCommits) == 0 {
		return "No commit is found since last generation", nil
	}

	startCommit, err := findLatestGenerationCommit(repo, state, idToCommits)
	if err != nil {
		return "", fmt.Errorf("failed to find the start commit: %w", err)
	}
	// Even though startCommit might be nil, it shouldn't happen in production
	// because this function will return early if no conventional commit is found
	// since last generation.
	startSHA := startCommit.Hash.String()

	// Sort the slice by commit time in reverse order,
	// so that the latest commit appears first.
	sort.Slice(allCommits, func(i, j int) bool {
		return allCommits[i].When.After(allCommits[j].When)
	})
	endSHA := allCommits[0].SHA
	librarianVersion := cli.Version()
	data := &generationPRBody{
		StartSHA:         startSHA,
		EndSHA:           endSHA,
		LibrarianVersion: librarianVersion,
		ImageVersion:     state.Image,
		Commits:          allCommits,
		FailedLibraries:  failedLibraries,
	}
	var out bytes.Buffer
	if err := genBodyTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// findLatestGenerationCommit returns the latest commit among the last generated
// commit of all the libraries.
// A libray is skipped if the last generated commit is empty.
//
// Note that it is possible that the returned commit is nil.
func findLatestGenerationCommit(repo gitrepo.Repository, state *config.LibrarianState, idToCommits map[string]string) (*gitrepo.Commit, error) {
	latest := time.UnixMilli(0) // the earliest timestamp.
	var res *gitrepo.Commit
	for _, library := range state.Libraries {
		commitHash, ok := idToCommits[library.ID]
		if !ok || commitHash == "" {
			slog.Info("skip getting last generated commit", "library", library.ID)
			continue
		}
		commit, err := repo.GetCommit(commitHash)
		if err != nil {
			return nil, fmt.Errorf("can't find last generated commit for %s: %w", library.ID, err)
		}
		if latest.Before(commit.When) {
			latest = commit.When
			res = commit
		}
	}

	if res == nil {
		slog.Warn("no library has non-empty last generated commit")
	}

	return res, nil
}

// formatReleaseNotes generates the body for a release pull request.
func formatReleaseNotes(repo gitrepo.Repository, state *config.LibrarianState) (string, error) {
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
