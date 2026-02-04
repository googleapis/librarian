// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarianops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

const (
	generateBranchPrefix = "librarianops-generateall-"
	commitTitle          = "chore: run librarian update and generate --all"
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate libraries across repositories",
		UsageText: "librarianops generate [<repo> | -C <dir>]",
		Description: `Examples:
  librarianops generate google-cloud-rust
  librarianops generate -C ~/workspace/google-cloud-rust

Specify a repository name to clone and process, or use -C to work in a specific
directory (repo name is inferred from the directory basename).

For each repository, librarianops will:
  1. Clone the repository to a temporary directory (or use existing directory with -C)
  2. Create a branch: librarianops-generateall-YYYY-MM-DD
  3. Resolve librarian version from @main and update version field in librarian.yaml
  4. Run librarian update --all
  5. Run librarian tidy
  6. Run librarian generate --all
  7. Run cargo update --workspace (google-cloud-rust only)
  8. Commit changes
  9. Create a pull request`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "C",
				Usage: "work in existing `directory` (repo name inferred from basename)",
			},
			&cli.BoolFlag{
				Name:  "v",
				Usage: "run librarian with verbose output",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			repoName, workDir, err := parseRepoFlags(cmd)
			if err != nil {
				return err
			}
			return runGenerate(ctx, repoName, workDir)
		},
	}
}

func parseRepoFlags(cmd *cli.Command) (repoName, workDir string, err error) {
	workDir = cmd.String("C")
	command.Verbose = cmd.Bool("v")

	if workDir != "" {
		// When -C is provided, infer repo name from directory basename.
		// Convert to absolute path first to handle relative paths like ".".
		absPath, err := filepath.Abs(workDir)
		if err != nil {
			return "", "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		repoName = filepath.Base(absPath)
		workDir = absPath
	} else {
		// When -C is not provided, require positional repo argument.
		if cmd.Args().Len() == 0 {
			return "", "", fmt.Errorf("usage: librarianops <command> <repo> or librarianops <command> -C <dir>")
		}
		repoName = cmd.Args().Get(0)
	}
	if !supportedRepositories[repoName] {
		return "", "", fmt.Errorf("repository %q not found in supported repositories list", repoName)
	}
	return repoName, workDir, nil
}

func runGenerate(ctx context.Context, repoName, repoDir string) error {
	repoDir, cfg, cleanup, err := setupRepo(ctx, repoName, repoDir, generateBranchPrefix)
	if err != nil {
		return err
	}
	defer cleanup()

	// Get and update librarian version to @main.
	version, err := getLibrarianVersionAtMain(ctx)
	if err != nil {
		return err
	}
	configPath := filepath.Join(repoDir, "librarian.yaml")
	if err := updateLibrarianVersion(cfg, configPath, version); err != nil {
		return err
	}

	// Run update --all before tidy/generate.
	if repoName != repoFake {
		if err := runLibrarianWithVersion(ctx, repoDir, version, command.Verbose, "update", "--all"); err != nil {
			return err
		}
	}

	// Add all APIs that are not already configured.
	addedAPIs, err := addAllAPIs(ctx, cfg, repoDir, version, command.Verbose)
	if err != nil {
		return err
	}

	return processRepo(ctx, repoName, repoDir, version, commitTitle, func(repoName, version string) (string, string) {
		sources := "googleapis"
		if repoName == repoRust {
			sources = "googleapis and discovery-artifact-manager"
		}
		var added stringc
		if len(addedAPIs) > 0 {
			added = "add new APIs, "
		}
		title := fmt.Sprintf("chore: %supdate librarian, %s, and regenerate", added, sources)
		body := fmt.Sprintf(`Update librarian version to @main (%s).

Update %s to the latest commit and regenerate all client libraries.`, version, sources)

		if len(addedAPIs) > 0 {
			body += "\n\nNew APIs added:\n"
			for _, api := range addedAPIs {
				body += fmt.Sprintf("- %s\n", api)
			}
		}
		return title, body
	})
}

// setupRepo prepares a repository for processing by cloning (if needed),
// creating a branch, and reading the config.
// It returns the repo directory, config, and a cleanup function.
func setupRepo(ctx context.Context, repoName, repoDir, branchPrefix string) (dir string, cfg *config.Config, cleanup func(), err error) {
	cleanup = func() {} // no-op by default

	if repoDir == "" {
		repoDir, err = os.MkdirTemp("", "librarianops-"+repoName+"-*")
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		cleanup = func() {
			os.RemoveAll(repoDir)
		}
		if err := cloneRepo(ctx, repoDir, repoName); err != nil {
			cleanup()
			return "", nil, nil, err
		}
	}

	if err := createBranch(ctx, repoDir, branchPrefix, time.Now()); err != nil {
		cleanup()
		return "", nil, nil, err
	}
	configPath := filepath.Join(repoDir, "librarian.yaml")
	cfg, err = yaml.Read[config.Config](configPath)
	if err != nil {
		cleanup()
		return "", nil, nil, err
	}

	return repoDir, cfg, cleanup, nil
}

// prInfoFunc returns the PR title and body given the repo name and version.
type prInfoFunc func(repoName, version string) (title, body string)

// processRepo runs the common processing steps: tidy, generate, cargo update
// (for rust), commit, push, and create PR.
func processRepo(ctx context.Context, repoName, repoDir, version, commitMsg string, prInfo prInfoFunc) error {
	if repoName != repoFake {
		if err := runLibrarianWithVersion(ctx, repoDir, version, command.Verbose, "tidy"); err != nil {
			return err
		}
	}
	if err := runLibrarianWithVersion(ctx, repoDir, version, command.Verbose, "generate", "--all"); err != nil {
		return err
	}
	if repoName == repoRust {
		if err := runCargoUpdate(ctx, repoDir); err != nil {
			return err
		}
	}
	if err := commitChanges(ctx, repoDir, commitMsg); err != nil {
		return err
	}
	if repoName != repoFake {
		if err := pushBranch(ctx, repoDir); err != nil {
			return err
		}
		title, body := prInfo(repoName, version)
		if err := createPR(ctx, repoDir, title, body); err != nil {
			return err
		}
	}
	return nil
}

func cloneRepo(ctx context.Context, repoDir, repoName string) error {
	return command.Run(ctx, "gh", "repo", "clone", fmt.Sprintf("googleapis/%s", repoName), repoDir)
}

func createBranch(ctx context.Context, repoDir, branchPrefix string, now time.Time) error {
	branchName := fmt.Sprintf("%s%s", branchPrefix, now.Format("2006-01-02"))
	return command.Run(ctx, "git", "-C", repoDir, "checkout", "-b", branchName)
}

func commitChanges(ctx context.Context, repoDir, msg string) error {
	if err := command.Run(ctx, "git", "-C", repoDir, "add", "."); err != nil {
		return err
	}
	return command.Run(ctx, "git", "-C", repoDir, "commit", "-m", msg)
}

func pushBranch(ctx context.Context, repoDir string) error {
	return command.Run(ctx, "git", "-C", repoDir, "push", "-u", "origin", "HEAD")
}

func createPR(ctx context.Context, repoDir, title, body string) error {
	return command.Run(ctx, "sh", "-c", fmt.Sprintf("cd %s && gh pr create --title %q --body %q", repoDir, title, body))
}

func runCargoUpdate(ctx context.Context, repoDir string) error {
	return command.Run(ctx, "sh", "-c", fmt.Sprintf("cd %s && cargo update --workspace", repoDir))
}

func getLibrarianVersionAtMain(ctx context.Context) (string, error) {
	output, err := command.Output(ctx, "go", "list", "-m", "-json", "github.com/googleapis/librarian@main")
	if err != nil {
		return "", fmt.Errorf("go list: %w", err)
	}
	var mod struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal([]byte(output), &mod); err != nil {
		return "", fmt.Errorf("parsing go list output: %w", err)
	}
	if mod.Version == "" {
		return "", fmt.Errorf("no version in go list output: %s", output)
	}
	return mod.Version, nil
}

func updateLibrarianVersion(cfg *config.Config, configPath, version string) error {
	cfg.Version = version
	return yaml.Write(configPath, cfg)
}

func runLibrarianWithVersion(ctx context.Context, repoDir, version string, verbose bool, args ...string) error {
	if verbose {
		args = append([]string{"-v"}, args...)
	}
	cmdArgs := append([]string{"run", fmt.Sprintf("github.com/googleapis/librarian/cmd/librarian@%s", version)}, args...)
	shellCmd := fmt.Sprintf("cd %s && go %s", repoDir, shellJoin(cmdArgs))
	return command.Run(ctx, "sh", "-c", shellCmd)
}

// shellJoin joins arguments into a shell-safe command string.
func shellJoin(args []string) string {
	var result string
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += shellQuote(arg)
	}
	return result
}

// shellQuote quotes a string for safe use in a shell command.
func shellQuote(s string) string {
	// If the string contains no special characters, return it as-is.
	needsQuoting := false
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '\n' || c == '"' || c == '\'' || c == '\\' || c == '$' || c == '`' || c == '!' || c == '*' || c == '?' || c == '[' || c == ']' || c == '(' || c == ')' || c == '{' || c == '}' || c == '<' || c == '>' || c == '|' || c == '&' || c == ';' || c == '#' || c == '~' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	// Use single quotes, escaping any single quotes in the string.
	var result string
	result = "'"
	for _, c := range s {
		if c == '\'' {
			result += "'\\''"
		} else {
			result += string(c)
		}
	}
	result += "'"
	return result
}

// addAllAPIs adds all APIs from serviceconfig.APIs that are not already in the
// librarian.yaml config.
func addAllAPIs(ctx context.Context, cfg *config.Config, repoDir, version string, verbose bool) ([]string, error) {
	// Build a set of existing API paths from all libraries.
	// Check:
	// 1. Explicit API paths in lib.APIs
	// 2. Derived API paths from library names
	// 3. For Rust: rust.modules[].source paths
	existingAPIs := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		// Add explicit API paths.
		for _, api := range lib.APIs {
			existingAPIs[api.Path] = true
		}
		// Derive API path from library name (e.g., google-cloud-foo-v1 -> google/cloud/foo/v1).
		derivedPath := librarian.DeriveAPIPath(cfg.Language, lib.Name)
		existingAPIs[derivedPath] = true
		// For Rust, also check rust.modules[].source.
		if lib.Rust != nil {
			for _, mod := range lib.Rust.Modules {
				if mod.Source != "" {
					existingAPIs[mod.Source] = true
				}
			}
		}
	}

	var addedAPIs []string
	for _, api := range serviceconfig.APIs {
		// Skip APIs that already exist in the config.
		if existingAPIs[api.Path] {
			continue
		}
		// Skip APIs that are restricted to other languages.
		if len(api.Languages) > 0 && !slices.Contains(api.Languages, cfg.Language) {
			continue
		}
		if err := runLibrarianWithVersion(ctx, repoDir, version, verbose, "add", api.Path); err != nil {
			return nil, err
		}
		addedAPIs = append(addedAPIs, api.Path)
	}
	return addedAPIs, nil
}
