// Copyright 2025 Google LLC
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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/librarian/internal/rust"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

const googleapisRepo = "github.com/googleapis/googleapis"

var (
	errMissingLibraryOrAllFlag = errors.New("must specify library name or use --all flag")
	errBothLibraryAndAllFlag   = errors.New("cannot specify both library name and --all flag")
	errEmptySources            = errors.New("sources field is required in librarian.yaml: specify googleapis and/or discovery source commits")
)

func generateCommand() *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a client library",
		UsageText: "librarian generate [library] [--all]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "all",
				Usage: "generate all libraries",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			all := cmd.Bool("all")
			libraryName := cmd.Args().First()
			if !all && libraryName == "" {
				return errMissingLibraryOrAllFlag
			}
			if all && libraryName != "" {
				return errBothLibraryAndAllFlag
			}
			return runGenerate(ctx, all, libraryName)
		},
	}
}

func runGenerate(ctx context.Context, all bool, libraryName string) error {
	cfg, err := yaml.Read[config.Config](librarianConfigPath)
	if err != nil {
		return err
	}
	if cfg.Sources == nil {
		return errEmptySources
	}
	if all {
		return generateAll(ctx, cfg)
	}
	return generateLibrary(ctx, cfg, libraryName)
}

func generateAll(ctx context.Context, cfg *config.Config) error {
	for _, lib := range cfg.Libraries {
		if err := generateLibrary(ctx, cfg, lib.Name); err != nil {
			return err
		}
	}
	return nil
}

func generateLibrary(ctx context.Context, cfg *config.Config, libraryName string) error {
	googleapisDir, err := fetchGoogleapisDir(ctx, cfg.Sources)
	if err != nil {
		return err
	}
	for _, lib := range cfg.Libraries {
		if lib.Name == libraryName {
			for _, api := range lib.Channels {
				if api.ServiceConfig == "" {
					serviceConfig, err := serviceconfig.Find(googleapisDir, api.Path)
					if err != nil {
						return err
					}
					api.ServiceConfig = serviceConfig
				}
			}
			return generate(ctx, cfg.Language, lib, cfg.Sources)
		}
	}
	return fmt.Errorf("library %q not found", libraryName)
}

func generate(ctx context.Context, language string, library *config.Library, sources *config.Sources) error {
	var err error
	switch language {
	case "testhelper":
		err = testGenerate(library)
	case "rust":
		keep := append(library.Keep, "Cargo.toml")
		if err := cleanOutput(library.Output, keep); err != nil {
			return err
		}
		err = rust.Generate(ctx, library, sources)
	default:
		err = fmt.Errorf("generate not implemented for %q", language)
	}
	if err != nil {
		fmt.Printf("✗ Error generating %s: %v\n", library.Name, err)
		return err
	}
	fmt.Printf("✓ Successfully generated %s\n", library.Name)
	return nil
}

func fetchGoogleapisDir(ctx context.Context, sources *config.Sources) (string, error) {
	if sources == nil || sources.Googleapis == nil {
		return "", errors.New("googleapis source is required")
	}
	if sources.Googleapis.Dir != "" {
		return sources.Googleapis.Dir, nil
	}
	return fetch.RepoDir(ctx, googleapisRepo, sources.Googleapis.Commit, sources.Googleapis.SHA256)
}

// cleanOutput removes all files and directories in dir except those in keep.
// It returns an error if any file in keep does not exist.
func cleanOutput(dir string, keep []string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	existing := make(map[string]bool)
	for _, e := range entries {
		existing[e.Name()] = true
	}
	keepSet := make(map[string]bool)
	for _, k := range keep {
		if !existing[k] {
			return fmt.Errorf("%s: file %q in keep list does not exist", dir, k)
		}
		keepSet[k] = true
	}
	for _, e := range entries {
		if keepSet[e.Name()] {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
