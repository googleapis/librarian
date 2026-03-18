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

// Command build-docker-images builds and tag multiple language-specific
// Docker images based on a librarian version.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"

	"github.com/googleapis/librarian/internal/command"
	"github.com/urfave/cli/v3"
)

var supportedLanguages = []string{
	"go",
	"python",
	"rust",
}

func main() {
	ctx := context.Background()
	cmd := &cli.Command{
		Name:  "build-docker-images",
		Usage: "builds docker images for the specified Librarian version",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "version",
				Usage:    "librarian version, as specified in librarian.yaml config",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "revision",
				Usage:    "revision of librarian to check out; defaults to version flag value",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:  "language",
				Usage: "language to build image for; may be repeated; defaults to all supported languages",
			},
		}, Action: func(ctx context.Context, cmd *cli.Command) error {
			return buildDockerImages(ctx, cmd.String("version"), cmd.String("revision"), cmd.StringSlice("language"))
		},
	}
	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Fatalf("build-docker-images: %v", err)
	}
}

func buildDockerImages(ctx context.Context, version, revision string, languages []string) error {
	repoDir, err := os.MkdirTemp("", "librarian-docker-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	if revision == "" {
		revision = version
	}
	if err := cloneRepo(ctx, repoDir, revision); err != nil {
		return fmt.Errorf("failed to clone librarian repo: %w", err)
	}
	if err := os.Chdir(repoDir); err != nil {
		return fmt.Errorf("failed to change to temp directory: %w", err)
	}
	if len(languages) == 0 {
		languages = supportedLanguages
	}
	for _, language := range languages {
		if err := buildDockerImage(ctx, version, language); err != nil {
			return err
		}
	}
	return nil
}

func buildDockerImage(ctx context.Context, version, language string) error {
	slog.Info("building image", "language", language)
	args := []string{
		"build",
		"-t",
		fmt.Sprintf("librarian-%s:%s", language, version),
		"--target",
		language,
		"-f",
		"cmd/librarian/Dockerfile",
		".",
	}
	// Execute docker build with streaming output
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image for language %s: %w", language, err)
	}
	return nil
}

func cloneRepo(ctx context.Context, repoDir, revision string) error {
	args := []string{
		"clone",
		"https://github.com/googleapis/librarian",
		"--depth=1",
		"--revision=" + revision,
		repoDir,
	}
	return command.Run(ctx, "git", args...)
}
