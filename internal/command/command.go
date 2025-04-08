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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/protobuf/encoding/protojson"
)

const releaseIDEnvVarName = "RELEASE_ID"

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
func findLibrary(state *statepb.PipelineState, apiPath string) string {

	for _, library := range state.Libraries {
		if slices.Contains(library.ApiPaths, apiPath) {
			return library.Id
		}
	}
	return ""
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

func loadConfig(languageRepo *gitrepo.Repo) (*statepb.PipelineConfig, error) {
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

func saveState(languageRepo *gitrepo.Repo, state *statepb.PipelineState) error {
	path := filepath.Join(languageRepo.Dir, "generator-input", "pipeline-state.json")
	// Marshal the protobuf message as JSON...
	unformatted, err := protojson.Marshal(state)
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

func createTmpWorkingRoot(t time.Time) (string, error) {
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

func pushAndCreatePullRequest(ctx context.Context, repo *gitrepo.Repo, startOfRun time.Time, title string, description string) (*gitrepo.PullRequestMetadata, error) {
	if !flagPush {
		return nil, nil
	}

	// This should already have been validated to be non-empty by validatePush
	gitHubAccessToken := os.Getenv(gitHubTokenEnvironmentVariable)
	branch := fmt.Sprintf("librarian-%s", formatTimestamp(startOfRun))
	err := gitrepo.PushBranch(repo, branch, gitHubAccessToken)
	if err != nil {
		slog.Info(fmt.Sprintf("Received error pushing branch: '%s'", err))
		return nil, err
	}
	pr, err := gitrepo.CreatePullRequest(ctx, repo, branch, gitHubAccessToken, title, description)
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
		addFlagGitUserEmail,
		addFlagGitUserName,
		addFlagLanguage,
		addFlagPush,
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
		addFlagBuild,
	} {
		fn(fs)
	}

	fs = CmdUpdateApis.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagAPIRoot,
		addFlagBranch,
		addFlagGitUserEmail,
		addFlagGitUserName,
		addFlagLanguage,
		addFlagLibraryID,
		addFlagPush,
		addFlagRepoRoot,
	} {
		fn(fs)
	}

	fs = CmdCreateReleasePR.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagSecretsProject,
		addFlagWorkRoot,
		addFlagLanguage,
		addFlagPush,
		addFlagGitUserEmail,
		addFlagGitUserName,
		addFlagRepoRoot,
		addFlagSkipBuild,
		addFlagEnvFile,
	} {
		fn(fs)
	}

	fs = CmdUpdateImageTag.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagWorkRoot,
		addFlagAPIRoot,
		addFlagBranch,
		addFlagGitUserEmail,
		addFlagGitUserName,
		addFlagLanguage,
		addFlagPush,
		addFlagRepoRoot,
		addFlagTag,
	} {
		fn(fs)
	}

	fs = CmdRelease.flags
	for _, fn := range []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagWorkRoot,
		addFlagLanguage,
		addFlagRepoRoot,
		addFlagReleaseID,
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
