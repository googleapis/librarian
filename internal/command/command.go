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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/protobuf/encoding/protojson"
)

type Command struct {
	Name  string
	Short string
	Run   func(ctx context.Context) error

	flags *flag.FlagSet
}

func (c *Command) Parse(args []string) error {
	return c.flags.Parse(args)
}

func Lookup(name string) (*Command, error) {
	var cmd *Command
	for _, sub := range Commands {
		if sub.Name == name {
			cmd = sub
		}
	}
	if cmd == nil {
		return nil, fmt.Errorf("invalid command: %q", name)
	}
	return cmd, nil
}

var CmdConfigure = &Command{
	Name:  "configure",
	Short: "Configure a new API in a given language",
	Run: func(ctx context.Context) error {
		if flagAPIPath == "" {
			return fmt.Errorf("-api-path is not provided")
		}
		if !supportedLanguages[flagLanguage] {
			return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
		}
		if flagPush && flagGitHubToken == "" {
			return fmt.Errorf("-github-token must be provided if -push is set to true")
		}

		startOfRun := time.Now()
		// tmpRoot is a newly-created working directory under /tmp
		// We do any cloning or copying under there. Currently this is only
		// actually needed in generate if the user hasn't specified an output directory
		// - we could potentially only create it in that case, but always creating it
		// is a more general case.
		tmpRoot, err := createTmpWorkingRoot(startOfRun)
		if err != nil {
			return err
		}

		var apiRoot string
		if flagAPIRoot == "" {
			repo, err := cloneGoogleapis(ctx, tmpRoot)
			if err != nil {
				return err
			}
			apiRoot = repo.Dir
		} else {
			// We assume it's okay not to take a defensive copy of apiRoot in the configure command,
			// as "vanilla" configuration/generation shouldn't need to edit any protos. (That's just an escape hatch.)
			apiRoot, err = filepath.Abs(flagAPIRoot)
			if err != nil {
				return err
			}
		}

		var languageRepo *gitrepo.Repo
		if flagRepoRoot == "" {
			languageRepo, err = cloneLanguageRepo(ctx, flagLanguage, tmpRoot)
			if err != nil {
				return err
			}
		} else {
			repoRoot, err := filepath.Abs(flagRepoRoot)
			if err != nil {
				return err
			}
			languageRepo, err = gitrepo.Open(ctx, repoRoot)
			if err != nil {
				return err
			}
		}

		state, err := loadState(languageRepo)
		if err != nil {
			return err
		}

		image := deriveImage(state)

		generatorInput := filepath.Join(languageRepo.Dir, "generator-input")
		if err := container.Configure(ctx, image, apiRoot, flagAPIPath, generatorInput); err != nil {
			return err
		}

		// After configuring, we run quite a lot of the same code as in CmdUpdateApis.Run.
		outputDir := filepath.Join(tmpRoot, "output")
		if err := os.Mkdir(outputDir, 0755); err != nil {
			return err
		}

		// Take a defensive copy of the generator input directory from the language repo.
		// Note that we didn't do this earlier, as the container.Configure step is *intended* to modify
		// generator input in the repo. Any changes during generation aren't intended to be persisted though.
		generatorInput = filepath.Join(tmpRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		if err := container.Generate(ctx, image, apiRoot, outputDir, generatorInput, flagAPIPath); err != nil {
			return err
		}
		// We don't need to clean the newly-configured API, but we *do* need to clean any non-API-specific files.
		if err := container.Clean(ctx, image, languageRepo.Dir, "none"); err != nil {
			return err
		}
		if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
			return err
		}
		msg := fmt.Sprintf("Configured API %s", flagAPIPath) // TODO: Improve info using googleapis commits and version info
		if err := commitAll(ctx, languageRepo, msg); err != nil {
			return err
		}
		if err := container.Build(ctx, image, "repo-root", languageRepo.Dir, flagAPIPath); err != nil {
			return err
		}

		return push(ctx, languageRepo, startOfRun)
	},
}

var CmdGenerate = &Command{
	Name:  "generate",
	Short: "Generate client library code for an API",
	Run: func(ctx context.Context) error {
		if flagAPIPath == "" {
			return fmt.Errorf("-api-path is not provided")
		}
		if !supportedLanguages[flagLanguage] {
			return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
		}
		if flagAPIRoot == "" {
			return fmt.Errorf("-api-root is not provided")
		}

		apiRoot, err := filepath.Abs(flagAPIRoot)
		if err != nil {
			return err
		}

		// tmpRoot is a newly-created working directory under /tmp
		// We do any cloning or copying under there. Currently this is only
		// actually needed in generate if the user hasn't specified an output directory
		// - we could potentially only create it in that case, but always creating it
		// is a more general case.
		tmpRoot, err := createTmpWorkingRoot(time.Now())
		if err != nil {
			return err
		}

		var outputDir string
		if flagOutput == "" {
			outputDir = filepath.Join(tmpRoot, "output")
			if err := os.Mkdir(outputDir, 0755); err != nil {
				return err
			}
			slog.Info(fmt.Sprintf("No output directory specified. Defaulting to %s", outputDir))
		} else {
			outputDir, err = filepath.Abs(flagOutput)
			if err != nil {
				return err
			}
		}

		image := deriveImage(nil)
		// The final empty string argument is for generator input - we don't have any
		if err := container.Generate(ctx, image, apiRoot, outputDir, "", flagAPIPath); err != nil {
			return err
		}

		if flagBuild {
			if err := container.Build(ctx, image, "generator-output", outputDir, flagAPIPath); err != nil {
				return err
			}
		}
		return nil
	},
}

var CmdUpdateApis = &Command{
	Name:  "update-apis",
	Short: "Update a language repo by regenerating configured APIs",
	Run: func(ctx context.Context) error {

		if !supportedLanguages[flagLanguage] {
			return fmt.Errorf("invalid -language flag specified: %q", flagLanguage)
		}
		if flagPush && flagGitHubToken == "" {
			return fmt.Errorf("-github-token must be provided if -push is set to true")
		}

		startOfRun := time.Now()

		// tmpRoot is a newly-created working directory under /tmp
		// We do any cloning or copying under there.
		tmpRoot, err := createTmpWorkingRoot(startOfRun)
		if err != nil {
			return err
		}

		var apiRepo *gitrepo.Repo
		hardResetApiRepo := true
		if flagAPIRoot == "" {
			apiRepo, err = cloneGoogleapis(ctx, tmpRoot)
			if err != nil {
				return err
			}
		} else {
			apiRoot, err := filepath.Abs(flagAPIRoot)
			slog.Info(fmt.Sprintf("Using apiRoot: %s", apiRoot))
			if err != nil {
				slog.Info(fmt.Sprintf("Error retrieving apiRoot: %s", err))
				return err
			}
			apiRepo, err = gitrepo.Open(ctx, apiRoot)
			if err != nil {
				return err
			}
			clean, err := gitrepo.IsClean(ctx, apiRepo)
			if err != nil {
				return err
			}
			if !clean {
				hardResetApiRepo = false
				slog.Warn("API repo has modifications, so will not be reset after generation")
			}
		}

		var outputDir string
		if flagOutput == "" {
			outputDir = filepath.Join(tmpRoot, "output")
			if err := os.Mkdir(outputDir, 0755); err != nil {
				return err
			}
			slog.Info(fmt.Sprintf("No output directory specified. Defaulting to %s", outputDir))
		} else {
			outputDir, err = filepath.Abs(flagOutput)
			if err != nil {
				return err
			}
		}

		var languageRepo *gitrepo.Repo
		if flagRepoRoot == "" {
			languageRepo, err = cloneLanguageRepo(ctx, flagLanguage, tmpRoot)
			if err != nil {
				return err
			}
		} else {
			repoRoot, err := filepath.Abs(flagRepoRoot)
			if err != nil {
				return err
			}
			languageRepo, err = gitrepo.Open(ctx, repoRoot)
			if err != nil {
				return err
			}
			clean, err := gitrepo.IsClean(ctx, apiRepo)
			if err != nil {
				return err
			}
			if !clean {
				return errors.New("language repo must be clean before update")
			}
		}

		state, err := loadState(languageRepo)
		if err != nil {
			return err
		}

		image := deriveImage(state)

		// Take a defensive copy of the generator input directory from the language repo.
		generatorInput := filepath.Join(tmpRoot, "generator-input")
		if err := os.CopyFS(generatorInput, os.DirFS(filepath.Join(languageRepo.Dir, "generator-input"))); err != nil {
			return err
		}

		hashBefore, err := gitrepo.HeadHash(ctx, languageRepo)
		if err != nil {
			return err
		}

		// Perform "generate, clean, commit, build" on each element in ApiGenerationStates.
		for _, apiState := range state.ApiGenerationStates {
			err = updateApi(ctx, apiRepo, languageRepo, generatorInput, image, outputDir, state, apiState)
			if err != nil {
				return err
			}
		}

		// Reset the API repo in case it was changed, but not if it was already dirty before the command.
		if hardResetApiRepo {
			gitrepo.ResetHard(ctx, apiRepo)
		}

		if !flagPush {
			slog.Info("Pushing not specified; update complete.")
			return nil
		}

		hashAfter, err := gitrepo.HeadHash(ctx, languageRepo)
		if err != nil {
			return err
		}
		if hashBefore == hashAfter {
			slog.Info("No changes generated; nothing to push.")
			return nil
		}

		return push(ctx, languageRepo, startOfRun)
	},
}

var CmdCreateReleasePR = &Command{
	Name:  "create-release-pr",
	Short: "Generate a PR for release",
	Run: func(ctx context.Context) error {

		image := os.Getenv("LIBRARIAN_REPOSITORY")

		if err := container.CreateReleasePR(image); err != nil {
			return err
		}
		return nil
	},
}

func updateApi(ctx context.Context, apiRepo *gitrepo.Repo, languageRepo *gitrepo.Repo, generatorInput string, image string, outputRoot string, repoState *statepb.PipelineState, apiState *statepb.ApiGenerationState) error {
	if flagAPIPath != "" && flagAPIPath != apiState.Id {
		// If flagAPIPath has been passed in, we only act on that API.
		return nil
	}

	if apiState.AutomationLevel == statepb.AutomationLevel_AUTOMATION_LEVEL_BLOCKED {
		slog.Info(fmt.Sprintf("Ignoring blocked API: '%s'", apiState.Id))
		return nil
	}
	commits, err := gitrepo.GetApiCommits(ctx, apiRepo, apiState.Id, apiState.LastGeneratedCommit)
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		slog.Info(fmt.Sprintf("API '%s' has no changes.", apiState.Id))
		return nil
	}
	slog.Info(fmt.Sprintf("Generating '%s' with %d new commit(s)", apiState.Id, len(commits)))

	// Now that we know the API has at least one new API commit, regenerate it, update the state, commit the change and build the output.

	// We create an output directory separately for each API.
	outputDir := filepath.Join(outputRoot, apiState.Id)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	if err := container.Generate(ctx, image, apiRepo.Dir, outputDir, generatorInput, apiState.Id); err != nil {
		return err
	}
	if err := container.Clean(ctx, image, languageRepo.Dir, apiState.Id); err != nil {
		return err
	}
	if err := os.CopyFS(languageRepo.Dir, os.DirFS(outputDir)); err != nil {
		return err
	}

	apiState.LastGeneratedCommit = commits[0].Hash.String()
	if err := saveState(languageRepo, repoState); err != nil {
		return err
	}

	// Note that as we've updated the state, we'll definitely have something to commit, even if no
	// generated code changed. This avoids us regenerating no-op changes again and again, and reflects
	// that we really are at the latest state. We could skip the build step here if there are no changes
	// prior to updating the state, but it's probably not worth the additional complexity (and it does
	// no harm to check the code is still "healthy").
	var msg = createCommitMessage(commits)
	if err := commitAll(ctx, languageRepo, msg); err != nil {
		return err
	}

	// Once we've committed, we can build - but then check that nothing has changed afterwards.
	if err := container.Build(ctx, image, "repo-root", languageRepo.Dir, apiState.Id); err != nil {
		return err
	}
	clean, err := gitrepo.IsClean(ctx, languageRepo)
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("building '%s' created changes in the repo", apiState.Id)
	}
	return nil
}

func createCommitMessage(commits []object.Commit) string {
	const PiperPrefix = "PiperOrigin-RevId: "
	var builder strings.Builder
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

func deriveImage(state *statepb.PipelineState) string {
	if flagImage != "" {
		return flagImage
	}

	defaultRepository := os.Getenv("LIBRARIAN_REPOSITORY")
	relativeImage := fmt.Sprintf("google-cloud-%s-generator", flagLanguage)

	var tag string
	if state == nil {
		tag = "latest"
	} else {
		tag = state.ImageTag
	}
	if defaultRepository == "" {
		return fmt.Sprintf("%s:%s", relativeImage, tag)
	} else {
		return fmt.Sprintf("%s/%s:%s", defaultRepository, relativeImage, tag)
	}
}

func loadState(languageRepo *gitrepo.Repo) (*statepb.PipelineState, error) {
	path := filepath.Join(languageRepo.Dir, "generator-input", "pipeline-state.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	state := &statepb.PipelineState{}
	err = protojson.Unmarshal(bytes, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func saveState(languageRepo *gitrepo.Repo, state *statepb.PipelineState) error {
	path := filepath.Join(languageRepo.Dir, "generator-input", "pipeline-state.json")
	// Marshal the protobuf message as JSON...
	unformatted, err := protojson.Marshal(state)
	if err != nil {
		return err
	}
	// ... then reformat it
	var formatted bytes.Buffer
	err = json.Indent(&formatted, unformatted, "", "  ")
	if err != nil {
		return err
	}
	// The file mode is likely to be irrelevant, given that the permissions aren't changed
	// if the file exists, which we expect it to anyway.
	err = os.WriteFile(path, formatted.Bytes(), os.FileMode(0644))
	return err
}

func createTmpWorkingRoot(t time.Time) (string, error) {
	if flagWorkRoot != "" {
		slog.Info(fmt.Sprintf("Using specified working directory: %s", flagWorkRoot))
		return flagWorkRoot, nil
	}

	const yyyyMMddHHmmss = "20060102T150405" // Expected format by time library

	path := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-%s", t.Format(yyyyMMddHHmmss)))

	_, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		if err := os.Mkdir(path, 0755); err != nil {
			return "", fmt.Errorf("unable to create temporary working directory '%s': %w", path, err)
		}
	case err == nil:
		return "", fmt.Errorf("temporary working directory already exists: %s", path)
	default:
		return "", fmt.Errorf("unable to check directory '%s': %w", path, err)
	}

	slog.Info(fmt.Sprintf("Temporary working directory: %s", path))
	return path, nil
}

// No commit is made if there are no file modifications.
func commitAll(ctx context.Context, repo *gitrepo.Repo, msg string) error {
	status, err := gitrepo.AddAll(ctx, repo)
	if err != nil {
		return err
	}
	if status.IsClean() {
		slog.Info("No modifications to commit.")
		return nil
	}

	gitrepo.PrintStatus(ctx, repo)
	return gitrepo.Commit(ctx, repo, msg)
}

func push(ctx context.Context, repo *gitrepo.Repo, startOfRun time.Time) error {
	if !flagPush {
		return nil
	}
	if flagGitHubToken == "" {
		return fmt.Errorf("no GitHub token supplied for push")
	}
	const yyyyMMddHHmmss = "20060102T150405" // Expected format by time library
	timestamp := startOfRun.Format(yyyyMMddHHmmss)
	branch := fmt.Sprintf("librarian-%s", timestamp)
	err := gitrepo.PushBranch(ctx, repo, branch, flagGitHubToken)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("feat: API regeneration: %s", timestamp)
	return gitrepo.CreatePullRequest(ctx, repo, branch, flagGitHubToken, title)
}

var Commands = []*Command{
	CmdConfigure,
	CmdGenerate,
	CmdUpdateApis,
	CmdCreateReleasePR,
}

func init() {
	for _, c := range Commands {
		c.flags = flag.NewFlagSet(c.Name, flag.ContinueOnError)
		c.flags.Usage = constructUsage(c.flags, c.Name)
	}

	fs := CmdConfigure.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagPush,
		addFlagGitHubToken,
		addFlagRepoRoot,
	} {
		fn(fs)
	}

	fs = CmdGenerate.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagLanguage,
		addFlagOutput,
		addFlagBuild,
	} {
		fn(fs)
	}

	fs = CmdUpdateApis.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIPath,
		addFlagAPIRoot,
		addFlagBranch,
		addFlagGitHubToken,
		addFlagLanguage,
		addFlagOutput,
		addFlagPush,
		addFlagRepoRoot,
	} {
		fn(fs)
	}
}

func constructUsage(fs *flag.FlagSet, name string) func() {
	output := fmt.Sprintf("Usage:\n\n  librarian %s [arguments]\n", name)
	output += "\nFlags:\n\n"
	return func() {
		fmt.Fprint(fs.Output(), output)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\n\n")
	}
}
