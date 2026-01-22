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

// Package librarianops provides orchestration for running librarian across
// multiple repositories.
package librarianops

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/urfave/cli/v3"
)

const (
	repoRust = "google-cloud-rust"
	repoFake = "fake-repo" // used for testing

	branchPrefix = "librarianops-generateall-"
	commitTitle  = "chore: run librarian update and generate --all"
)

var supportedRepositories = map[string]bool{
	repoRust: true,
	repoFake: true, // used for testing
}

// Run executes the librarianops command with the given arguments.
func Run(ctx context.Context, args ...string) error {
	cmd := &cli.Command{
		Name:      "librarianops",
		Usage:     "orchestrate librarian operations across multiple repositories",
		UsageText: "librarianops [command]",
		Commands: []*cli.Command{
			generateCommand(),
		},
	}
	return cmd.Run(ctx, args)
}

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate libraries across repositories",
		UsageText: "librarianops generate [<repo> | --all]",
		Description: `Examples:
  librarianops generate google-cloud-rust
  librarianops generate --all
  librarianops generate -C ~/workspace/google-cloud-rust google-cloud-rust

Specify a repository name (e.g., google-cloud-rust) to process a single repository,
or use --all to process all repositories.

Use -C to work in a specific directory (assumes repository already exists there).

For each repository, librarianops will:
  1. Clone the repository to a temporary directory
  2. Create a branch: librarianops-generateall-YYYY-MM-DD
  3. Run librarian update discovery (google-cloud-rust only)
  4. Run librarian update googleapis
  5. Run librarian generate --all
  6. Run cargo update --workspace (google-cloud-rust only)
  7. Commit changes
  8. Create a pull request (pushes branch automatically)`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "process all repositories",
			},
			&cli.StringFlag{
				Name:  "C",
				Usage: "work in `directory` (assumes repo exists)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")
			workDir := cmd.String("C")
			repoName := ""
			if cmd.Args().Len() > 0 {
				repoName = cmd.Args().Get(0)
			}
			if all && repoName != "" {
				return fmt.Errorf("cannot specify both <repo> and --all")
			}
			if !all && repoName == "" {
				return fmt.Errorf("usage: librarianops generate [<repo> | --all]")
			}
			if all && workDir != "" {
				return fmt.Errorf("cannot use -C with --all")
			}
			return runGenerate(ctx, all, repoName, workDir)
		},
	}
}

func runGenerate(ctx context.Context, all bool, repoName, repoDir string) error {
	if all {
		for name := range supportedRepositories {
			if err := processRepo(ctx, name, ""); err != nil {
				return err
			}
		}
		return nil
	}

	if !supportedRepositories[repoName] {
		return fmt.Errorf("repository %q not found in supported repositories list", repoName)
	}
	return processRepo(ctx, repoName, repoDir)
}

func processRepo(ctx context.Context, repoName, repoDir string) (err error) {
	if repoDir == "" {
		repoDir, err = os.MkdirTemp("", "librarianops-"+repoName+"-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer func() {
			cerr := os.RemoveAll(repoDir)
			if err == nil {
				err = cerr
			}
		}()
		if err := cloneRepo(ctx, repoDir, repoName); err != nil {
			return err
		}
	}
	originalWD, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := os.Chdir(repoDir); err != nil {
		return fmt.Errorf("failed to change directory to %q: %w", repoDir, err)
	}
	defer os.Chdir(originalWD)

	if err := createBranch(ctx, time.Now()); err != nil {
		return err
	}
	if repoName == repoRust {
		if err := librarian.Run(ctx, "librarian", "update", "discovery"); err != nil {
			return err
		}
	}
	if err := librarian.Run(ctx, "librarian", "update", "googleapis"); err != nil {
		return err
	}
	if err := librarian.Run(ctx, "librarian", "generate", "--all"); err != nil {
		return err
	}
	if repoName == repoRust {
		if err := runCargoUpdate(ctx); err != nil {
			return err
		}
	}
	if err := commitChanges(ctx); err != nil {
		return err
	}
	if repoName != repoFake {
		if err := pushBranch(ctx); err != nil {
			return err
		}
		if err := createPR(ctx, repoName); err != nil {
			return err
		}
	}
	return nil
}

func cloneRepo(ctx context.Context, repoDir, repoName string) error {
	return command.Run(ctx, "gh", "repo", "clone", fmt.Sprintf("googleapis/%s", repoName), repoDir)
}

func createBranch(ctx context.Context, now time.Time) error {
	branchName := fmt.Sprintf("%s%s", branchPrefix, now.Format("2006-01-02"))
	return command.Run(ctx, "git", "checkout", "-b", branchName)
}

func commitChanges(ctx context.Context) error {
	if err := command.Run(ctx, "git", "add", "."); err != nil {
		return err
	}
	return command.Run(ctx, "git", "commit", "-m", commitTitle)
}

func pushBranch(ctx context.Context) error {
	return command.Run(ctx, "git", "push", "-u", "origin", "HEAD")
}

func createPR(ctx context.Context, repoName string) error {
	var body string
	if repoName == repoRust {
		body = `Update googleapis/googleapis and googleapis/discovery-artifact-manager
to the latest commit and regenerate all client libraries.`
	} else {
		body = `Update googleapis/googleapis to the latest commit and regenerate all client libraries.`
	}
	return command.Run(ctx, "gh", "pr", "create", "--title", commitTitle, "--body", body)
}

func runCargoUpdate(ctx context.Context) error {
	return command.Run(ctx, "cargo", "update", "--workspace")
}
