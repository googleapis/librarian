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
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarianops/flagsparser"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

const (
	branchPrefix = "librarianops-generateall-"
	commitTitle  = "chore: run librarian update and generate --all"
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
  4. Run librarian tidy
  5. Run librarian update --all
  6. Run librarian generate --all
  7. Run cargo update --workspace (google-cloud-rust only)
  8. Commit changes
  9. Create a pull request`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "C",
				Usage: "work in `directory` (repo name inferred from basename)",
			},
			&cli.BoolFlag{
				Name:  "v",
				Usage: "run librarian with verbose output",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			repoName, workDir, err := flagsparser.ParseRepoFlags(cmd)
			if err != nil {
				return err
			}
			return runGenerate(ctx, repoName, workDir)
		},
	}
}

func runGenerate(ctx context.Context, repoName, repoDir string) error {
	if !supportedRepositories[repoName] {
		return fmt.Errorf("repository %q not found in supported repositories list", repoName)
	}
	return processRepo(ctx, repoName, repoDir, command.Verbose)
}

func processRepo(ctx context.Context, repoName, repoDir string, verbose bool) (err error) {
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
	version, err := getLibrarianVersionAtMain(ctx)
	if err != nil {
		return err
	}
	if err := updateLibrarianVersion(version, repoDir); err != nil {
		return err
	}
	if repoName != repoFake {
		if err := runLibrarianWithVersion(ctx, version, verbose, "tidy"); err != nil {
			return err
		}
	}
	if repoName != repoFake {
		if err := runLibrarianWithVersion(ctx, version, verbose, "update", "--all"); err != nil {
			return err
		}
	}
	if err := runLibrarianWithVersion(ctx, version, verbose, "generate", "--all"); err != nil {
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
		if err := createPR(ctx, repoName, version); err != nil {
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

func createPR(ctx context.Context, repoName, librarianVersion string) error {
	sources := "googleapis"
	if repoName == repoRust {
		sources = "googleapis and discovery-artifact-manager"
	}
	title := fmt.Sprintf("chore: update librarian, %s, and regenerate", sources)
	body := fmt.Sprintf(`Update librarian version to @main (%s).

Update %s to the latest commit and regenerate all client libraries.`, librarianVersion, sources)
	return command.Run(ctx, "gh", "pr", "create", "--title", title, "--body", body)
}

func runCargoUpdate(ctx context.Context) error {
	return command.Run(ctx, "cargo", "update", "--workspace")
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

func updateLibrarianVersion(version, repoDir string) error {
	configPath := filepath.Join(repoDir, "librarian.yaml")
	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		return err
	}
	cfg.Version = version
	return yaml.Write(configPath, cfg)
}

func runLibrarianWithVersion(ctx context.Context, version string, verbose bool, args ...string) error {
	if verbose {
		args = append([]string{"-v"}, args...)
	}
	return command.Run(ctx, "go",
		append([]string{"run", fmt.Sprintf("github.com/googleapis/librarian/cmd/librarian@%s", version)}, args...)...)
}
