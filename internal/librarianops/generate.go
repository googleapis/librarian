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
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

const (
	branchPrefix = "librarianops-generateall-"
	commitTitle  = "chore: run librarian update and generate --all"
	// librarianImageTemplate is a template string to format a language and
	// version into the name of a Docker image to run when the --docker flag
	// has been specified.
	// TODO(https://github.com/googleapis/librarian/issues/4464): change this
	// to an Artifact Registry image when we publish automatically.
	librarianImageTemplate = "docker.io/library/librarian-{language}:{version}"
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
  3. Run librarian tidy
  4. Run librarian update for configured sources (discovery, googleapis)
  5. Run librarian generate --all
  6. Run cargo update --workspace (google-cloud-rust only)
  7. Commit changes
  8. Create a pull request`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "C",
				Usage: "work in `directory` (repo name inferred from basename)",
			},
			&cli.BoolFlag{
				Name:  "v",
				Usage: "run librarian with verbose output",
			},
			&cli.BoolFlag{
				Name:  "docker",
				Usage: "run librarian in Docker",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			repoName, workDir, verbose, err := parseFlags(cmd)
			if err != nil {
				return err
			}
			command.Verbose = verbose
			return runGenerate(ctx, repoName, workDir, cmd.Bool("docker"))
		},
	}
}

func runGenerate(ctx context.Context, repoName, repoDir string, runInDocker bool) error {
	if !supportedRepositories[repoName] {
		return fmt.Errorf("repository %q not found in supported repositories list", repoName)
	}
	return processRepo(ctx, repoName, repoDir, "", command.Verbose, runInDocker)
}

func processRepo(ctx context.Context, repoName, repoDir, librarianBin string, verbose, runInDocker bool) (err error) {
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
	cfg, err := yaml.Read[config.Config]("librarian.yaml")
	if err != nil {
		return err
	}
	if librarianBin == "" && cfg.Version == "" {
		return errors.New("librarian.yaml must specify the librarian version")
	}
	run := func(args ...string) error {
		if librarianBin != "" {
			return runLibrarianBin(ctx, librarianBin, verbose, args...)
		}
		if runInDocker {
			return runLibrarianInDocker(ctx, cfg.Language, cfg.Version, verbose, args...)
		}
		return runLibrarianWithVersion(ctx, cfg.Version, verbose, args...)
	}
	if repoName != repoFake {
		if err := run("tidy"); err != nil {
			return err
		}
		sources := sourcesToUpdate(cfg)
		if len(sources) > 0 {
			args := append([]string{"update"}, sources...)
			if err := run(args...); err != nil {
				return err
			}
		}
	}
	if err := run("generate", "--all"); err != nil {
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
	branchName := fmt.Sprintf("%s%s", branchPrefix, now.UTC().Format("20060102T150405Z"))
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
	sources := "googleapis"
	if repoName == repoRust {
		sources = "googleapis and discovery-artifact-manager"
	}
	title := fmt.Sprintf("chore: update %s and regenerate", sources)
	body := fmt.Sprintf("Update %s to the latest commit and regenerate all client libraries.", sources)
	return command.Run(ctx, "gh", "pr", "create", "--title", title, "--body", body)
}

func runCargoUpdate(ctx context.Context) error {
	return command.Run(ctx, "cargo", "update", "--workspace")
}

func runLibrarianWithVersion(ctx context.Context, version string, verbose bool, args ...string) error {
	if verbose {
		args = append([]string{"-v"}, args...)
	}
	return command.Run(ctx, "go",
		append([]string{"run", fmt.Sprintf("github.com/googleapis/librarian/cmd/librarian@%s", version)}, args...)...)
}

func runLibrarianInDocker(ctx context.Context, language, version string, verbose bool, args ...string) error {
	if verbose {
		args = append([]string{"-v"}, args...)
	}
	dockerImage := strings.NewReplacer("{language}", language, "{version}", version).Replace(librarianImageTemplate)
	currentUser, err := user.Current()
	if err != nil {
		return err
	}
	homeCache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dockerArgs := []string{
		"run",
		// Clean up the container afterwards.
		"--rm",
		// Run as the current user in the container, so that files are still
		// owned appropriately.
		"-u",
		fmt.Sprintf("%s:%s", currentUser.Uid, currentUser.Gid),
		// Map the current working directory to /repo.
		"-v",
		".:/repo",
		// Map the cache directory (avoids fetching sources multiple times).
		"-v",
		homeCache + ":/.cache",
		// Use /repo as the working directory.
		"-w",
		"/repo",
		dockerImage,
	}
	return command.Run(ctx, "docker", append(dockerArgs, args...)...)
}

// runLibrarianBin runs a pre-built librarian binary with the given arguments.
func runLibrarianBin(ctx context.Context, bin string, verbose bool, args ...string) error {
	if verbose {
		args = append([]string{"-v"}, args...)
	}
	return command.Run(ctx, bin, args...)
}

func sourcesToUpdate(cfg *config.Config) []string {
	if cfg.Sources == nil {
		return nil
	}
	var sources []string
	if cfg.Sources.Discovery != nil {
		sources = append(sources, "discovery")
	}
	if cfg.Sources.Googleapis != nil {
		sources = append(sources, "googleapis")
	}
	return sources
}
