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
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

var cmdUpdateApis = &cli.Command{
	Short:     "update-apis regenerates APIs in a language repo with new specifications",
	UsageLine: "librarian update-apis -language=<language> [flags]",
	Long: `Specify the language, and optional flags to use non-default repositories, e.g. for testing.
A pull request will only be created if -push is specified, in which case the LIBRARIAN_GITHUB_TOKEN
environment variable must be populated with an access token which has write access to the
language repo in which the pull request will be created.

After acquiring the API and language repositories, each configured library is potentially regenerated.

The state for each library is used to determine:
- The generation mode for the library (blocked, manual, automatic)
- The last API specification commit that was generated (if any)
- The paths within the API repository which contribute to the library

The command immediately skips any library which:
- Does not specify any paths within the API repository
- Has a generation mode of "blocked"
- Is not the one specified by the -library-id flag, when that has been specified
- Has not changed in terms of API specifications since the API commit at
  which it was last generated

For any other library, the command determines the last API commit containing
a change to any of the paths within the API repository, along with a message describing
all the changes to those paths since the last API specification that was generated
(or a "initial generation" if this is the first time the library has been generated).

The command runs the following language container commands for each library:
- "generate-library" to generate the source code for the library into an empty directory
- "clean" to clean any previously-generated source code from the language repository
- "build-library" (after copying the newly-generated code into place in the repository)

If all of these steps succeed, a commit is created (still on a per library basis)
with the results of the regeneration, and updated state to indicate the new
"last generated API commit" for that library.

If any container command fails, the error is reported, and the repository is reset as
if generation hadn't occurred for that library.

After iterating across all libraries, if the -push flag has been specified and any
libraries were successfully regenerated, a pull request is created in the
language repository, containing the generated commits. The pull request description
includes an overview list of what's in each commit, along with any failures in other
libraries. (The details of the failures are not included; consult the logs for
the command to see exactly what happened.)

If the -push flag has not been specified but a pull request would have been created,
the description of the pull request that would have been created is included in the
output of the command. Even if a pull request isn't created, any successful regeneration
commits will still be present in the language repo.
`,
	Run: runUpdateAPIs,
}

func init() {
	cmdUpdateApis.InitFlags()
	addFlagImage(cmdUpdateApis.Flags)
	addFlagWorkRoot(cmdUpdateApis.Flags)
	addFlagAPIRoot(cmdUpdateApis.Flags)
	addFlagBranch(cmdUpdateApis.Flags)
	addFlagGitUserEmail(cmdUpdateApis.Flags)
	addFlagGitUserName(cmdUpdateApis.Flags)
	addFlagLanguage(cmdUpdateApis.Flags)
	addFlagLibraryID(cmdUpdateApis.Flags)
	addFlagPush(cmdUpdateApis.Flags)
	addFlagRepoRoot(cmdUpdateApis.Flags)
	addFlagRepoUrl(cmdUpdateApis.Flags)
	addFlagSecretsProject(cmdUpdateApis.Flags)
	addFlagCi(cmdUpdateApis.Flags)
}

func runUpdateAPIs(ctx context.Context, cfg *config.Config) error {
	state, err := createCommandStateForLanguage(ctx, cfg.WorkRoot, cfg.RepoRoot, cfg.RepoURL, cfg.Language, cfg.Image,
		os.Getenv(defaultRepositoryEnvironmentVariable), cfg.SecretsProject, cfg.CI)
	if err != nil {
		return err
	}
	return updateAPIs(ctx, state, cfg)
}

func updateAPIs(ctx context.Context, state *commandState, cfg *config.Config) error {
	var apiRepo *gitrepo.Repository
	cleanWorkingTreePostGeneration := true
	if cfg.APIRoot == "" {
		var err error
		apiRepo, err = cloneGoogleapis(state.workRoot, cfg.CI)
		if err != nil {
			return err
		}
	} else {
		apiRoot, err := filepath.Abs(cfg.APIRoot)
		slog.Info(fmt.Sprintf("Using apiRoot: %s", apiRoot))
		if err != nil {
			slog.Info(fmt.Sprintf("Error retrieving apiRoot: %s", err))
			return err
		}
		apiRepo, err = gitrepo.NewRepository(&gitrepo.RepositoryOptions{
			Dir: apiRoot,
			CI:  cfg.CI,
		})
		if err != nil {
			return err
		}
		clean, err := apiRepo.IsClean()
		if err != nil {
			return err
		}
		if !clean {
			cleanWorkingTreePostGeneration = false
			slog.Warn("API repo has modifications, so will not be cleaned after generation")
		}
	}

	outputDir := filepath.Join(state.workRoot, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Code will be generated in %s", outputDir))

	// Root for generator-input defensive copies
	if err := os.Mkdir(filepath.Join(state.workRoot, config.GeneratorInputDir), 0755); err != nil {
		return err
	}

	prContent := new(PullRequestContent)
	// Perform "generate, clean, commit, build" on each library.
	for _, library := range state.pipelineState.Libraries {
		err := updateLibrary(state, apiRepo, outputDir, library, prContent, cfg.LibraryID, cfg.GitUserName, cfg.GitUserEmail)
		if err != nil {
			return err
		}
	}

	// Clean  the API repo in case it was changed, but not if it was already dirty before the command.
	if cleanWorkingTreePostGeneration {
		apiRepo.CleanWorkingTree()
	}
	_, err := createPullRequest(ctx, state, prContent, "feat: API regeneration", "", "regen", cfg.GitHubToken, cfg.Push)
	return err
}

func updateLibrary(state *commandState, apiRepo *gitrepo.Repository, outputRoot string, library *statepb.LibraryState,
	prContent *PullRequestContent, libraryID, gitUserName, gitUserEmail string) error {
	cc := state.containerConfig
	languageRepo := state.languageRepo

	if libraryID != "" && libraryID != library.Id {
		// If LibraryID has been passed in, we only act on that library.
		return nil
	}

	if len(library.ApiPaths) == 0 {
		slog.Info(fmt.Sprintf("Skipping non-generated library: '%s'", library.Id))
		return nil
	}

	if library.GenerationAutomationLevel == statepb.AutomationLevel_AUTOMATION_LEVEL_BLOCKED {
		slog.Info(fmt.Sprintf("Skipping generation-blocked library: '%s'", library.Id))
		return nil
	}

	initialGeneration := library.LastGeneratedCommit == ""
	commits, err := apiRepo.GetCommitsForPathsSinceCommit(library.ApiPaths, library.LastGeneratedCommit)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		slog.Info(fmt.Sprintf("Library '%s' has no changes.", library.Id))
		return nil
	}
	slog.Info(fmt.Sprintf("Generating '%s' with %d new commit(s)", library.Id, len(commits)))

	// Now that we know the API has at least one new API commit, regenerate it, update the state, commit the change and build the output.

	// We create an output directory separately for each API.
	outputDir := filepath.Join(outputRoot, library.Id)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Take a defensive copy of the generator input directory from the language repo.
	// This needs to be done per library, as the previous iteration may have updated generator-input in a meaningful way.
	// We could potentially just keep a single copy and update it, but it's clearer diagnostically if we can tell
	// what state we passed into the container.
	generatorInput := filepath.Join(state.workRoot, config.GeneratorInputDir, library.Id)
	if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(state.languageRepo.Dir, config.GeneratorInputDir))); err != nil {
		return err
	}

	if err := cc.GenerateLibrary(apiRepo.Dir, outputDir, generatorInput, library.Id); err != nil {
		addErrorToPullRequest(prContent, library.Id, err, "generating")
		return nil
	}
	if err := cc.Clean(languageRepo.Dir, library.Id); err != nil {
		addErrorToPullRequest(prContent, library.Id, err, "cleaning")
		// Clean up any changes before starting the next iteration.
		if err := languageRepo.CleanWorkingTree(); err != nil {
			return err
		}
		return nil
	}
	if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
		return err
	}

	library.LastGeneratedCommit = commits[0].Hash.String()
	if err := savePipelineState(state); err != nil {
		return err
	}

	// Note that as we've updated the state, we'll definitely have something to commit, even if no
	// generated code changed. This avoids us regenerating no-op changes again and again, and reflects
	// that we really are at the latest state. We could skip the build step here if there are no changes
	// prior to updating the state, but it's probably not worth the additional complexity (and it does
	// no harm to check the code is still "healthy").
	var msg string
	if initialGeneration {
		// If this is the first time we've generated this library, it's not worth listing all the previous
		// changes separately.
		msg = fmt.Sprintf("feat: Initial generation for %s", library.Id)
	} else {
		msg = createCommitMessage(library.Id, commits)
	}
	if err := commitAll(languageRepo, msg,
		gitUserName, gitUserEmail); err != nil {
		return err
	}

	// Once we've committed, we can build - but then check that nothing has changed afterwards.
	// We consider a "something changed" error as fatal, whereas a build error just needs to
	// undo the commit, report the failure and continue
	buildErr := cc.BuildLibrary(languageRepo.Dir, library.Id)
	clean, err := languageRepo.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("building '%s' created changes in the repo", library.Id)
	}

	if buildErr != nil {
		addErrorToPullRequest(prContent, library.Id, err, "building")
		if err = languageRepo.CleanAndRevertHeadCommit(); err != nil {
			return err
		}
		return nil
	}

	addSuccessToPullRequest(prContent, fmt.Sprintf("Generated %s", library.Id))
	return nil
}

func createCommitMessage(libraryID string, commits []*gitrepo.Commit) string {
	const PiperPrefix = "PiperOrigin-RevId: "
	var builder strings.Builder

	// Start the commit with a line on its own saying what's being regenerated.
	builder.WriteString(fmt.Sprintf("regen: Regenerate %s at API commit %s", libraryID, commits[0].Hash.String()[0:7]))
	builder.WriteString("\n")
	builder.WriteString("\n")

	piperRevIdLines := []string{}
	sourceLinkLines := []string{}
	// Consume the commits in reverse order, so that they're in normal chronological order,
	// accumulating PiperOrigin-RevId and Source-Link lines.
	for i := len(commits) - 1; i >= 0; i-- {
		commit := commits[i]
		messageLines := strings.Split(commit.Message, "\n")
		sourceLinkLines = append(sourceLinkLines, fmt.Sprintf("Source-Link: https://github.com/googleapis/googleapis/commit/%s", commit.Hash.String()))
		for _, line := range messageLines {
			if strings.HasPrefix(line, PiperPrefix) {
				piperRevIdLines = append(piperRevIdLines, line)
			} else {
				builder.WriteString(line)
				builder.WriteString("\n")
			}

		}
	}
	for _, revIdLine := range piperRevIdLines {
		builder.WriteString(revIdLine)
		builder.WriteString("\n")
	}
	for _, sourceLinkLine := range sourceLinkLines {
		builder.WriteString(sourceLinkLine)
		builder.WriteString("\n")
	}
	return builder.String()
}
