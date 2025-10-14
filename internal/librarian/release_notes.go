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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
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

type generationPRRequest struct {
	sourceRepo      gitrepo.Repository
	languageRepo    gitrepo.Repository
	state           *config.LibrarianState
	idToCommits     map[string]string
	failedLibraries []string
}

type onboardPRRequest struct {
	sourceRepo gitrepo.Repository
	state      *config.LibrarianState
	api        string
	library    string
}

type generationPRBody struct {
	StartSHA         string
	EndSHA           string
	LibrarianVersion string
	ImageVersion     string
	Commits          []*gitrepo.ConventionalCommit
	FailedLibraries  []string
}

type onboardingPRBody struct {
	ImageVersion     string
	LibrarianVersion string
	LibraryID        string
	PiperID          string
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
	Commits []*gitrepo.ConventionalCommit
}

// formatOnboardPRBody creates the body of an onboarding pull request.
func formatOnboardPRBody(opt *onboardPRRequest) (string, error) {
	piperID, err := getPiperID(opt.state, opt.sourceRepo, opt.api, opt.library)
	if err != nil {
		return "", err
	}

	data := &onboardingPRBody{
		LibrarianVersion: cli.Version(),
		ImageVersion:     opt.state.Image,
		LibraryID:        opt.library,
		PiperID:          piperID,
	}

	var out bytes.Buffer
	if err := onboardingBodyTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// formatGenerationPRBody creates the body of a generation pull request.
// Only consider libraries whose ID appears in idToCommits.
func formatGenerationPRBody(request *generationPRRequest) (string, error) {
	var allCommits []*gitrepo.ConventionalCommit
	languageRepoChanges, err := languageRepoChangedFiles(request.languageRepo)
	if err != nil {
		return "", fmt.Errorf("failed to fetch changes in language repo: %w", err)
	}
	for _, library := range request.state.Libraries {
		lastGenCommit, ok := request.idToCommits[library.ID]
		if !ok {
			continue
		}
		// If nothing has changed that would be significant in a release for this library,
		// we don't look at the API changes either.
		if !shouldIncludeForRelease(languageRepoChanges, library.SourceRoots, library.ReleaseExcludePaths) {
			continue
		}

		commits, err := getConventionalCommitsSinceLastGeneration(request.sourceRepo, library, lastGenCommit)
		if err != nil {
			return "", fmt.Errorf("failed to fetch conventional commits for library, %s: %w", library.ID, err)
		}
		allCommits = append(allCommits, commits...)
	}

	if len(allCommits) == 0 {
		return "No commit is found since last generation", nil
	}

	startCommit, err := findLatestGenerationCommit(request.sourceRepo, request.state, request.idToCommits)
	if err != nil {
		return "", fmt.Errorf("failed to find the start commit: %w", err)
	}
	// Even though startCommit might be nil, it shouldn't happen in production
	// because this function will return early if no conventional commit is found
	// since last generation.
	startSHA := startCommit.Hash.String()
	groupedCommits := groupByIDAndSubject(allCommits)
	// Sort the slice by commit time in reverse order,
	// so that the latest commit appears first.
	sort.Slice(groupedCommits, func(i, j int) bool {
		return groupedCommits[i].When.After(groupedCommits[j].When)
	})
	endSHA := groupedCommits[0].CommitHash
	librarianVersion := cli.Version()
	data := &generationPRBody{
		StartSHA:         startSHA,
		EndSHA:           endSHA,
		LibrarianVersion: librarianVersion,
		ImageVersion:     request.state.Image,
		Commits:          groupedCommits,
		FailedLibraries:  request.failedLibraries,
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

// groupByIDAndSubject aggregates conventional commits for ones have the same Piper ID and subject in the footer.
func groupByIDAndSubject(commits []*gitrepo.ConventionalCommit) []*gitrepo.ConventionalCommit {
	var res []*gitrepo.ConventionalCommit
	idToCommits := make(map[string][]*gitrepo.ConventionalCommit)
	for _, commit := range commits {
		// a commit is not considering for grouping if it doesn't have a footer or
		// the footer doesn't have a Piper ID.
		if commit.Footers == nil {
			commit.Footers = make(map[string]string)
			commit.Footers["Library-IDs"] = commit.LibraryID
			res = append(res, commit)
			continue
		}

		id, ok := commit.Footers["PiperOrigin-RevId"]
		if !ok {
			commit.Footers["Library-IDs"] = commit.LibraryID
			res = append(res, commit)
			continue
		}

		key := fmt.Sprintf("%s-%s", id, commit.Subject)
		idToCommits[key] = append(idToCommits[key], commit)
	}

	for _, groupCommits := range idToCommits {
		var ids []string
		for _, commit := range groupCommits {
			ids = append(ids, commit.LibraryID)
		}
		firstCommit := groupCommits[0]
		firstCommit.Footers["Library-IDs"] = strings.Join(ids, ",")
		res = append(res, firstCommit)
	}

	return res
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

	commitsByType := make(map[string][]*gitrepo.ConventionalCommit)
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

// getPiperID extracts the Piper ID from the commit message that onboarded the API.
func getPiperID(state *config.LibrarianState, sourceRepo gitrepo.Repository, apiPath, library string) (string, error) {
	libraryState := findLibraryByID(state, library)
	serviceYaml := ""
	for _, api := range libraryState.APIs {
		if api.Path == apiPath {
			serviceYaml = api.ServiceConfig
			break
		}
	}

	initialCommit, err := sourceRepo.GetLatestCommit(filepath.Join(apiPath, serviceYaml))
	if err != nil {
		return "", err
	}

	id, err := findPiperIDFrom(initialCommit, library)
	if err != nil {
		return "", err
	}

	slog.Info("found piper id in the commit message", "piperID", id)
	return id, nil
}

func findPiperIDFrom(commit *gitrepo.Commit, libraryID string) (string, error) {
	commits, err := gitrepo.ParseCommits(commit, libraryID)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 || commits[0].Footers == nil {
		return "", errPiperNotFound
	}

	id, ok := commits[0].Footers["PiperOrigin-RevId"]
	if !ok {
		return "", errPiperNotFound
	}

	return id, nil
}

// languageRepoChangedFiles returns the paths of files changed in the repo as part
// of the current librarian run - either in the head commit if the repo is clean,
// or the outstanding changes otherwise.
func languageRepoChangedFiles(languageRepo gitrepo.Repository) ([]string, error) {
	clean, err := languageRepo.IsClean()
	if err != nil {
		return nil, err
	}
	if clean {
		headHash, err := languageRepo.HeadHash()
		if err != nil {
			return nil, err
		}
		return languageRepo.ChangedFilesInCommit(headHash)
	}
	// The commit or push flag is not set, get all locally changed files.
	return languageRepo.ChangedFiles()
}
