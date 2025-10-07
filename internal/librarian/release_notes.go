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
	}).Parse(`Librarian Version: {{.LibrarianVersion}}
Language Image: {{.ImageVersion}}

{{- range .NoteSections -}}
{{ $noteSection := . }}
<details><summary>{{.LibraryID}}: {{.NewVersion}}</summary>

## [{{.NewVersion}}]({{"https://github.com/"}}{{.RepoOwner}}/{{.RepoName}}/compare/{{.PreviousTag}}...{{.NewTag}}) ({{.Date}})
{{ range .CommitSections }}
### {{.Heading}}
{{ range .Commits }}
{{ if index .Footers "PiperOrigin-RevId" -}}
* {{.Subject}} (PiperOrigin-RevId: {{index .Footers "PiperOrigin-RevId"}}) ([{{shortSHA .CommitHash}}]({{"https://github.com/"}}{{$noteSection.RepoOwner}}/{{$noteSection.RepoName}}/commit/{{shortSHA .CommitHash}}))
{{- else -}}
* {{.Subject}} ([{{shortSHA .CommitHash}}]({{"https://github.com/"}}{{$noteSection.RepoOwner}}/{{$noteSection.RepoName}}/commit/{{shortSHA .CommitHash}}))
{{- end }}
{{ end }}

{{- end }}
</details>

{{ end }}
`))

	genBodyTemplate = template.Must(template.New("genBody").Funcs(template.FuncMap{
		"shortSHA": shortSHA,
	}).Parse(`BEGIN_COMMIT_OVERRIDE
{{ range .Commits }}
BEGIN_NESTED_COMMIT
{{.Type}}: [{{.LibraryID}}] {{.Subject}}
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
)

type generationPRBody struct {
	StartSHA         string
	EndSHA           string
	LibrarianVersion string
	ImageVersion     string
	Commits          []*conventionalcommits.ConventionalCommit
	FailedLibraries  []string
}

type releaseNote struct {
	LibrarianVersion string
	ImageVersion     string
	NoteSections     []*releaseNoteSection
}

type releaseNoteSection struct {
	RepoOwner      string
	RepoName       string
	LibraryID      string
	PreviousTag    string
	NewTag         string
	NewVersion     string
	Date           string
	CommitSections []*commitSection
}

type commitSection struct {
	Heading string
	Commits []*conventionalcommits.ConventionalCommit
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
	allCommits = groupByPiperID(allCommits)
	// Sort the slice by commit time in reverse order,
	// so that the latest commit appears first.
	sort.Slice(allCommits, func(i, j int) bool {
		return allCommits[i].When.After(allCommits[j].When)
	})
	endSHA := allCommits[0].CommitHash
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
// A library is skipped if the last generated commit is empty.
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

// groupByPiperID aggregates conventional commits for ones have the same Piper ID in the footer.
func groupByPiperID(commits []*conventionalcommits.ConventionalCommit) []*conventionalcommits.ConventionalCommit {
	idToCommits := make(map[string][]*conventionalcommits.ConventionalCommit)
	var singletons []*conventionalcommits.ConventionalCommit
	for i := range len(commits) {
		commit := commits[i]
		// a commit is not considering for grouping if it doesn't have a footer or
		// the footer doesn't have a Piper ID.
		if commit.Footers == nil {
			commit.Footers["Library-IDs"] = commit.LibraryID
			singletons = append(singletons, commit)
			continue
		}

		id, ok := commit.Footers["PiperOrigin-RevId"]
		if !ok {
			commit.Footers["Library-IDs"] = commit.LibraryID
			singletons = append(singletons, commit)
			continue
		}

		idToCommits[id] = append(idToCommits[id], commit)
	}

	var res []*conventionalcommits.ConventionalCommit
	for _, groupCommits := range idToCommits {
		var ids []string
		for _, commit := range groupCommits {
			ids = append(ids, commit.LibraryID)
		}
		firstCommit := groupCommits[0]
		firstCommit.Footers["Library-IDs"] = strings.Join(ids, ",")
		res = append(res, firstCommit)
	}

	return append(res, singletons...)
}

// formatReleaseNotes generates the body for a release pull request.
func formatReleaseNotes(state *config.LibrarianState, ghRepo *github.Repository) (string, error) {
	librarianVersion := cli.Version()
	var releaseSections []*releaseNoteSection
	for _, library := range state.Libraries {
		if !library.ReleaseTriggered {
			continue
		}

		section := formatLibraryReleaseNotes(library, ghRepo)
		releaseSections = append(releaseSections, section)
	}

	data := &releaseNote{
		LibrarianVersion: librarianVersion,
		ImageVersion:     state.Image,
		NoteSections:     releaseSections,
	}

	var out bytes.Buffer
	if err := releaseNotesTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// formatLibraryReleaseNotes generates release notes in Markdown format for a single library.
// It returns the generated release notes and the new version string.
func formatLibraryReleaseNotes(library *config.LibraryState, ghRepo *github.Repository) *releaseNoteSection {
	// The version should already be updated to the next version.
	newVersion := library.Version
	tagFormat := config.DetermineTagFormat(library.ID, library, nil)
	newTag := config.FormatTag(tagFormat, library.ID, newVersion)
	previousTag := config.FormatTag(tagFormat, library.ID, library.PreviousVersion)

	commitsByType := make(map[string][]*conventionalcommits.ConventionalCommit)
	for _, commit := range library.Changes {
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
		RepoOwner:      ghRepo.Owner,
		RepoName:       ghRepo.Name,
		LibraryID:      library.ID,
		NewVersion:     newVersion,
		PreviousTag:    previousTag,
		NewTag:         newTag,
		Date:           time.Now().Format("2006-01-02"),
		CommitSections: sections,
	}

	return section
}
