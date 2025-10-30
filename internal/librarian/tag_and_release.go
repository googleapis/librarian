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
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const (
	pullRequestSegments  = 7
	tagAndReleaseCmdName = "tag-and-release"
	releasePendingLabel  = "release:pending"
	releaseDoneLabel     = "release:done"
)

var (
	bulkChangeSectionRegex = regexp.MustCompile(`(feat|fix|perf|revert|docs): (.*)\nLibraries: (.*)`)
	contentRegex           = regexp.MustCompile(`### (Features|Bug Fixes|Performance Improvements|Reverts|Documentation)\n`)
	detailsRegex           = regexp.MustCompile(`(?s)<details><summary>(.*?)</summary>(.*?)</details>`)
	summaryRegex           = regexp.MustCompile(`(.*?): (v?\d+\.\d+\.\d+)`)

	libraryReleaseTemplate = template.Must(template.New("libraryRelease").Parse(`### {{.Type}}
{{ range .Messages }}
{{.}}
{{ end }}

`))
)

type tagAndReleaseRunner struct {
	ghClient    GitHubClient
	pullRequest string
}

// libraryRelease holds the parsed information from a pull request body.
type libraryRelease struct {
	// Body contains the release notes.
	Body string
	// Library is the library id of the library being released
	Library string
	// Version is the version that is being released
	Version string
}

type libraryReleaseBuilder struct {
	typeToMessages map[string][]string
	title          string
	version        string
}

func newTagAndReleaseRunner(cfg *config.Config) (*tagAndReleaseRunner, error) {
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("`%s` must be set", config.LibrarianGithubToken)
	}
	repo, err := parseRemote(cfg.Repo)
	if err != nil {
		return nil, err
	}
	ghClient := github.NewClient(cfg.GitHubToken, repo)
	// If a custom GitHub API endpoint is provided (for testing),
	// parse it and set it as the BaseURL on the GitHub client.
	if cfg.GitHubAPIEndpoint != "" {
		endpoint, err := url.Parse(cfg.GitHubAPIEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse github-api-endpoint: %w", err)
		}
		ghClient.BaseURL = endpoint
	}
	return &tagAndReleaseRunner{
		ghClient:    ghClient,
		pullRequest: cfg.PullRequest,
	}, nil
}

func parseRemote(repo string) (*github.Repository, error) {
	if isURL(repo) {
		return github.ParseRemote(repo)
	}
	// repo is a directory
	absRepoRoot, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	githubRepo, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir: absRepoRoot,
	})
	if err != nil {
		return nil, err
	}
	return GetGitHubRepositoryFromGitRepo(githubRepo)
}

func (r *tagAndReleaseRunner) run(ctx context.Context) error {
	slog.Info("running tag-and-release command")
	prs, err := r.determinePullRequestsToProcess(ctx)
	if err != nil {
		return err
	}
	if len(prs) == 0 {
		slog.Info("no pull requests to process, exiting")
		return nil
	}

	var hadErrors bool
	for _, p := range prs {
		if err := r.processPullRequest(ctx, p); err != nil {
			slog.Error("failed to process pull request", "pr", p.GetNumber(), "error", err)
			hadErrors = true
			continue
		}
		slog.Info("processed pull request", "pr", p.GetNumber())
	}
	slog.Info("finished processing all pull requests")

	if hadErrors {
		return errors.New("failed to process some pull requests")
	}
	return nil
}

func (r *tagAndReleaseRunner) determinePullRequestsToProcess(ctx context.Context) ([]*github.PullRequest, error) {
	slog.Info("determining pull requests to process")
	if r.pullRequest != "" {
		slog.Info("processing a single pull request", "pr", r.pullRequest)
		ss := strings.Split(r.pullRequest, "/")
		if len(ss) != pullRequestSegments {
			return nil, fmt.Errorf("invalid pull request format: %s", r.pullRequest)
		}
		prNum, err := strconv.Atoi(ss[pullRequestSegments-1])
		if err != nil {
			return nil, fmt.Errorf("invalid pull request number: %s", ss[pullRequestSegments-1])
		}
		pr, err := r.ghClient.GetPullRequest(ctx, prNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get pull request %d: %w", prNum, err)
		}
		return []*github.PullRequest{pr}, nil
	}

	slog.Info("searching for pull requests to tag and release")
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	query := fmt.Sprintf("label:%s merged:>=%s", releasePendingLabel, thirtyDaysAgo)
	prs, err := r.ghClient.SearchPullRequests(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search pull requests: %w", err)
	}
	return prs, nil
}

func (r *tagAndReleaseRunner) processPullRequest(ctx context.Context, p *github.PullRequest) error {
	slog.Info("processing pull request", "pr", p.GetNumber())
	releases := parsePullRequestBody(p.GetBody())
	if len(releases) == 0 {
		slog.Warn("no release details found in pull request body, skipping")
		return nil
	}

	// Load library state from remote repo
	targetBranch := *p.Base.Ref
	librarianState, err := loadRepoStateFromGitHub(ctx, r.ghClient, targetBranch)
	if err != nil {
		return err
	}

	librarianConfig, err := loadLibrarianConfigFromGitHub(ctx, r.ghClient, targetBranch)
	if err != nil {
		slog.Warn("error loading .librarian/config.yaml", slog.Any("err", err))
	}

	// Add a tag to the release commit to trigger louhi flow: "release-{pr number}".
	// See: go/sdk-librarian:louhi-trigger for details.
	commitSha := p.GetMergeCommitSHA()
	tagName := fmt.Sprintf("release-%d", p.GetNumber())
	if err := r.ghClient.CreateTag(ctx, tagName, commitSha); err != nil {
		return fmt.Errorf("failed to create tag %s: %w", tagName, err)
	}
	for _, release := range releases {
		libraryState := librarianState.LibraryByID(release.Library)
		if libraryState == nil {
			return fmt.Errorf("library %s not found", release.Library)
		}

		var libraryConfig *config.LibraryConfig
		if librarianConfig != nil {
			libraryConfig = librarianConfig.LibraryConfigFor(release.Library)
		}

		if libraryConfig != nil && libraryConfig.SkipGitHubReleaseCreation {
			slog.Info("skip creating release", "library", release.Library)
			continue
		}

		slog.Info("creating release", "library", release.Library, "version", release.Version)
		tagFormat := config.DetermineTagFormat(release.Library, libraryState, librarianConfig)
		tagName := config.FormatTag(tagFormat, release.Library, release.Version)
		releaseName := fmt.Sprintf("%s %s", release.Library, release.Version)
		if _, err := r.ghClient.CreateRelease(ctx, tagName, releaseName, release.Body, commitSha); err != nil {
			return fmt.Errorf("failed to create release: %w", err)
		}

	}
	return r.replacePendingLabel(ctx, p)
}

// parsePullRequestBody parses a string containing release notes and returns a slice of ParsedPullRequestBody.
func parsePullRequestBody(body string) []libraryRelease {
	slog.Info("parsing pull request body")
	idToBuilder := make(map[string]*libraryReleaseBuilder)
	matches := detailsRegex.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		summary := match[1]
		content := strings.TrimSpace(match[2])
		if summary == "Bulk Changes" {
			// Associated bulk changes to individual libraries.
			sections := bulkChangeSectionRegex.FindAllStringSubmatch(content, -1)
			for _, section := range sections {
				if len(section) != 4 {
					slog.Warn("bulk change does not associated with a library id", "content", section)
					continue
				}

				commitType, ok := commitTypeToHeading[strings.TrimSpace(section[1])]
				if !ok {
					slog.Warn("unrecognized commit type, skipping", "commit", section[1])
					continue
				}
				message := fmt.Sprintf("* %s", strings.TrimSpace(section[2]))
				libraries := section[3]
				for _, library := range strings.Split(libraries, ",") {
					// Bulk change doesn't have title and version, put an empty string so that
					// title and version are not overwritten, if exists.
					updateLibraryReleaseBuilder(idToBuilder, library, commitType, "", message, "")
				}
			}

			continue
		}

		summaryMatch := summaryRegex.FindStringSubmatch(summary)
		if len(summaryMatch) != 3 {
			slog.Warn("failed to parse pull request body", "match", strings.Join(match, "\n"))
			continue
		}

		slog.Info("parsed pull request body", "library", summaryMatch[1], "version", summaryMatch[2])
		library := strings.TrimSpace(summaryMatch[1])
		version := strings.TrimSpace(summaryMatch[2])
		// Split the content using commit types, e.g., Features, Bug Fixes, etc.
		// For non-bulk changes, the first match (i = 0) is the release title, the i-th match is
		// the commit messages of typeMatches[i-1].
		contentMatches := contentRegex.Split(content, -1)
		title := contentMatches[0]
		typeMatches := contentRegex.FindAllStringSubmatch(content, -1)
		if len(typeMatches) == 0 {
			// No commit message in a library.
			updateLibraryReleaseBuilder(idToBuilder, library, "", title, "", version)
		}
		for i, typeMatch := range typeMatches {
			commitType := typeMatch[1]
			contentMatch := contentMatches[i+1]
			messages := strings.Split(contentMatch, "\n\n")
			for _, message := range messages {
				message = strings.TrimSpace(message)
				if message != "" {
					updateLibraryReleaseBuilder(idToBuilder, library, commitType, title, message, version)
				}
			}
		}

	}

	var parsedBodies []libraryRelease
	for libraryID, builder := range idToBuilder {
		parsedBodies = append(parsedBodies, libraryRelease{
			Body:    buildReleaseBody(builder.typeToMessages, builder.title),
			Library: libraryID,
			Version: builder.version,
		})
	}

	sort.Slice(parsedBodies, func(i, j int) bool {
		return parsedBodies[i].Library < parsedBodies[j].Library
	})

	return parsedBodies
}

// replacePendingLabel is a helper function that replaces the `release:pending` label with `release:done`.
func (r *tagAndReleaseRunner) replacePendingLabel(ctx context.Context, p *github.PullRequest) error {
	var currentLabels []string
	for _, label := range p.Labels {
		currentLabels = append(currentLabels, label.GetName())
	}
	currentLabels = slices.DeleteFunc(currentLabels, func(s string) bool {
		return s == releasePendingLabel
	})
	currentLabels = append(currentLabels, releaseDoneLabel)
	if err := r.ghClient.ReplaceLabels(ctx, p.GetNumber(), currentLabels); err != nil {
		return fmt.Errorf("failed to replace labels: %w", err)
	}
	return nil
}

// updateLibraryReleaseBuilder finds or creates a libraryReleaseBuilder for a given library
// and updates it with new information.
func updateLibraryReleaseBuilder(idToVersionAndBody map[string]*libraryReleaseBuilder, library, commitType, title, message, version string) {
	vab, ok := idToVersionAndBody[library]
	if !ok {
		idToVersionAndBody[library] = &libraryReleaseBuilder{
			typeToMessages: map[string][]string{
				commitType: {message},
			},
			version: version,
			title:   title,
		}

		return
	}

	vab.typeToMessages[commitType] = append(vab.typeToMessages[commitType], message)
	if version == "" {
		version = vab.version
	}
	vab.version = version
	if title == "" {
		title = vab.title
	}
	vab.title = title
}

// buildReleaseBody formats the release notes for a single library.
//
// It takes a map of commit types (e.g., "Features", "Bug Fixes") to their corresponding messages and a title string.
// It returns a formatted string containing the title and all commit messages organized by type, following the order
// defined in commitTypeOrder.
func buildReleaseBody(body map[string][]string, title string) string {
	var builder strings.Builder
	builder.WriteString(title)
	for _, commitType := range commitTypeOrder {
		heading := commitTypeToHeading[commitType]
		messages, ok := body[heading]
		if !ok {
			continue
		}
		var out bytes.Buffer
		data := &struct {
			Type     string
			Messages []string
		}{
			Type:     heading,
			Messages: messages,
		}
		if err := libraryReleaseTemplate.Execute(&out, data); err != nil {
			slog.Error("error executing template", "error", err)
			continue
		}

		builder.WriteString(strings.TrimSpace(out.String()))
		builder.WriteString("\n\n")
	}

	return strings.TrimSpace(builder.String())
}
