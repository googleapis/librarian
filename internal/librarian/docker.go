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

package librarian

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

var (
	errNoImage = errors.New("image not specified, and no version to derive image from")
)

const (
	versionedLibrarianImagePrefix = "docker.io/library/librarian-"
)

func dockerCommand() *cli.Command {
	return &cli.Command{
		Name:      "docker",
		Usage:     "runs librarian in a docker image",
		UsageText: "librarian docker [--image=...] --args=\"[normal librarian command line arguments]\"",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "image",
				Usage: "the image to use; when not set, defaults to the version specified in librarian.yaml",
			},
			// TODO: Improve this. (We don't really want to use a StringSliceFlag... it
			// would be nice to be able to run "librarian docker -- generate --all") or
			// similar... basically, stop parsing flags after "--".
			&cli.StringFlag{
				Name:     "args",
				Usage:    "the arguments to pass to the nested librarian invocation",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := yaml.Read[config.Config](librarianConfigPath)
			if err != nil {
				return err
			}
			args := strings.Split(cmd.String("args"), " ")
			return runInDocker(ctx, cfg, cmd.String("image"), args)
		},
	}
}

func runInDocker(ctx context.Context, cfg *config.Config, image string, args []string) error {
	dockerExe := "docker"
	// TODO: allow this to be overridden?
	// TODO: File bug to create a top-level "tools" section, rather than it
	// just being in release.

	if image == "" {
		if cfg.Version == "" {
			return errNoImage
		}
		// TODO: Use an Artifact Registry location...
		image = versionedLibrarianImagePrefix + cfg.Version
	}
	// TODO: Figure out what to do if $LIBRARIAN_CACHE is set. (Other tools
	// need ~/.cache as well...)

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
		"-u",
		fmt.Sprintf("%s:%s", currentUser.Uid, currentUser.Gid),
		"-v",
		".:/repo",
		"-v",
		homeCache + ":/.cache",
		"-w",
		"/repo",
		image,
	}
	dockerArgs = append(dockerArgs, args...)
	// TODO: Add this functionality to our command package?
	cmd := exec.CommandContext(ctx, dockerExe, dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
