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
	"slices"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/container"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"github.com/googleapis/librarian/internal/utils"
	"google.golang.org/protobuf/encoding/protojson"
)

const releaseIDEnvVarName = "_RELEASE_ID"

type Command struct {
	Name  string
	Short string
	// Obtains a language repo where appropriate, cloning or using
	// flags where necessary. May return a nil pointer if the command
	// does not use a language repo.
	maybeGetLanguageRepo func(workRoot string) (*gitrepo.Repo, error)
	// Executes the command with the given pre-populated context.
	execute func(*CommandContext) error
	// Functions to execute when initializing the flag set for the command.
	flagFunctions []func(fs *flag.FlagSet)

	// The flag set used to parse the flags and construct usage messages.
	// This is populated for each command in init().
	flags *flag.FlagSet
}

// Information used when executing a command. This is set up by RunCommand,
// then passed into Command.execute.
type CommandContext struct {
	// Context for operations requiring cancellation etc
	ctx context.Context
	// The time at which the command started executing, to be used as a consistent
	// timestamp for anything which needs one.
	startTime time.Time
	// Temporary directory created under /tmp by default (but can be specified via a flag)
	// All files created by Librarian live under this directory unless otherwise a location
	// (e.g. a repo root) is specified via a flag.
	workRoot string
	// The language repo for the command, where appropriate.
	languageRepo *gitrepo.Repo
	// The pipeline configuration, loaded from the language repo if there is one.
	// (This is nil if languageRepo is nil.)
	pipelineConfig *statepb.PipelineConfig
	// The pipeline state, loaded from the language repo if there is one.
	// (This is nil if languageRepo is nil.)
	pipelineState *statepb.PipelineState
	// Configuration for running container commands.
	containerConfig *container.ContainerConfig
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

func cloneOrOpenLanguageRepo(workRoot string) (*gitrepo.Repo, error) {
	var languageRepo *gitrepo.Repo
	if flagRepoRoot != "" && flagRepoUrl != "" {
		return nil, errors.New("do not specify both repo-root and repo-url")
	}
	if flagRepoUrl != "" {
		// Take the last part of the URL as the directory name. It feels very
		// unlikely that will clash with anything else (e.g. "output")
		bits := strings.Split(flagRepoUrl, "/")
		repoName := bits[len(bits)-1]
		repoPath := filepath.Join(workRoot, repoName)
		return gitrepo.CloneOrOpen(repoPath, flagRepoUrl)
	}
	if flagRepoRoot == "" {
		languageRepoURL := fmt.Sprintf("https://github.com/googleapis/google-cloud-%s", flagLanguage)
		repoPath := filepath.Join(workRoot, fmt.Sprintf("google-cloud-%s", flagLanguage))
		return gitrepo.CloneOrOpen(repoPath, languageRepoURL)
	}
	repoRoot, err := filepath.Abs(flagRepoRoot)
	if err != nil {
		return nil, err
	}
	languageRepo, err = gitrepo.Open(repoRoot)
	if err != nil {
		return nil, err
	}
	clean, err := gitrepo.IsClean(languageRepo)
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, errors.New("language repo must be clean")
	}
	return languageRepo, nil
}

func RunCommand(c *Command, ctx context.Context) error {
	if c.flags.Lookup("language") != nil {
		if err := validateLanguage(); err != nil {
			return err
		}
	}
	startTime := time.Now()
	workRoot, err := createWorkRoot(startTime)
	if err != nil {
		return err
	}
	languageRepo, err := c.maybeGetLanguageRepo(workRoot)
	if err != nil {
		return err
	}
	var state *statepb.PipelineState = nil
	var config *statepb.PipelineConfig = nil
	if languageRepo != nil {
		state, err = loadPipelineState(languageRepo)
		if err != nil {
			return err
		}
		config, err = loadPipelineConfig(languageRepo)
		if err != nil {
			return err
		}
	}

	image := deriveImage(state)
	containerConfig, err := container.NewContainerConfig(ctx, workRoot, image, flagSecretsProject, config)
	if err != nil {
		return err
	}

	cmdContext := &CommandContext{
		ctx:             ctx,
		startTime:       startTime,
		workRoot:        workRoot,
		languageRepo:    languageRepo,
		pipelineConfig:  config,
		pipelineState:   state,
		containerConfig: containerConfig,
	}
	return c.execute(cmdContext)
}

func appendResultEnvironmentVariable(ctx *CommandContext, name, value string) error {
	envFile := flagEnvFile
	if envFile == "" {
		envFile = filepath.Join(ctx.workRoot, "env-vars.txt")
	}

	return utils.AppendToFile(envFile, fmt.Sprintf("%s=%s\n", name, value))
}

func deriveImage(state *statepb.PipelineState) string {
	if flagImage != "" {
		return flagImage
	}

	defaultRepository := os.Getenv(defaultRepositoryEnvironmentVariable)
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

// Finds a library which includes code generated from the given API path.
// If there are no such libraries, an empty string is returned.
// If there are multiple such libraries, the first match is returned.
func findLibraryIDByApiPath(state *statepb.PipelineState, apiPath string) string {
	for _, library := range state.Libraries {
		if slices.Contains(library.ApiPaths, apiPath) {
			return library.Id
		}
	}
	return ""
}

func findLibraryByID(state *statepb.PipelineState, libraryID string) *statepb.LibraryState {
	for _, library := range state.Libraries {
		if library.Id == libraryID {
			return library
		}
	}
	return nil
}

func loadPipelineState(languageRepo *gitrepo.Repo) (*statepb.PipelineState, error) {
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

func loadPipelineConfig(languageRepo *gitrepo.Repo) (*statepb.PipelineConfig, error) {
	path := filepath.Join(languageRepo.Dir, "generator-input", "pipeline-config.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &statepb.PipelineConfig{}
	err = protojson.Unmarshal(bytes, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func savePipelineState(ctx *CommandContext) error {
	path := filepath.Join(ctx.languageRepo.Dir, "generator-input", "pipeline-state.json")
	// Marshal the protobuf message as JSON...
	unformatted, err := protojson.Marshal(ctx.pipelineState)
	if err != nil {
		return err
	}
	// ... then reformat it
	var formatted bytes.Buffer
	err = json.Indent(&formatted, unformatted, "", "    ")
	if err != nil {
		return err
	}
	// The file mode is likely to be irrelevant, given that the permissions aren't changed
	// if the file exists, which we expect it to anyway.
	err = os.WriteFile(path, formatted.Bytes(), os.FileMode(0644))
	return err
}

func formatTimestamp(t time.Time) string {
	const yyyyMMddHHmmss = "20060102T150405" // Expected format by time library
	return t.Format(yyyyMMddHHmmss)
}

func createWorkRoot(t time.Time) (string, error) {
	if flagWorkRoot != "" {
		slog.Info(fmt.Sprintf("Using specified working directory: %s", flagWorkRoot))
		return flagWorkRoot, nil
	}

	path := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-%s", formatTimestamp(t)))

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
func commitAll(repo *gitrepo.Repo, msg string) error {
	status, err := gitrepo.AddAll(repo)
	if err != nil {
		return err
	}
	if status.IsClean() {
		slog.Info("No modifications to commit.")
		return nil
	}

	return gitrepo.Commit(repo, msg, flagGitUserName, flagGitUserEmail)
}

// Push the contents of the language repo and create a new pull request.
func pushAndCreatePullRequest(ctx *CommandContext, title string, description string) (*githubrepo.PullRequestMetadata, error) {
	if !flagPush {
		return nil, nil
	}

	gitHubRepo, err := gitrepo.GetGitHubRepoFromRemote(ctx.languageRepo)
	if err != nil {
		return nil, err
	}

	branch := fmt.Sprintf("librarian-%s", formatTimestamp(ctx.startTime))
	err = gitrepo.PushBranch(ctx.languageRepo, branch, githubrepo.GetAccessToken())
	if err != nil {
		slog.Info(fmt.Sprintf("Received error pushing branch: '%s'", err))
		return nil, err
	}
	pr, err := githubrepo.CreatePullRequest(ctx.ctx, gitHubRepo, branch, title, description)
	if pr != nil {
		return pr, err
	}
	return nil, err
}

// Log details of an error which prevents a single API or library from being configured/released, but without
// halting the overall process. Return a brief description to the errors to include in the PR.
// We don't include detailed errors in the PR, as this could reveal sensitive information.
// The action should describe what failed, e.g. "configuring", "building", "generating".
func logPartialError(id string, err error, action string) string {
	slog.Warn(fmt.Sprintf("Error while %s %s: %s", action, id, err))
	return fmt.Sprintf("Error while %s %s", action, id)
}

var Commands = []*Command{
	CmdConfigure,
	CmdGenerate,
	CmdUpdateApis,
	CmdCreateReleasePR,
	CmdUpdateImageTag,
	CmdRelease,
	CmdMergeReleasePR,
}

func init() {
	for _, c := range Commands {
		c.flags = flag.NewFlagSet(c.Name, flag.ContinueOnError)
		c.flags.Usage = constructUsage(c.flags, c.Name)
		for _, fn := range c.flagFunctions {
			fn(c.flags)
		}
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

func formatReleaseTag(libraryID, version string) string {
	return libraryID + "-" + version
}
