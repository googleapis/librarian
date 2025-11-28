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
	"strings"

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
	googleapisDir, err := fetchGoogleapisDir(ctx, cfg.Sources)
	if err != nil {
		return err
	}
	if err := addLibraries(cfg, googleapisDir); err != nil {
		return err
	}
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

// cleanOutput removes all files in dir except those in keep. The keep list
// should contain paths relative to dir. It returns an error if any file in
// keep does not exist.
func cleanOutput(dir string, keep []string) error {
	keepSet := make(map[string]bool)
	for _, k := range keep {
		path := filepath.Join(dir, k)
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%s: file %q in keep list does not exist", dir, k)
		}
		keepSet[k] = true
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if keepSet[rel] {
			return nil
		}
		return os.Remove(path)
	})
}

func addLibraries(cfg *config.Config, googleapisDir string) error {
	channels, err := findChannels(googleapisDir)
	if err != nil {
		return err
	}

	configured := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		for _, api := range lib.Channels {
			configured[api.Path] = true
		}
		if len(lib.Channels) == 0 {
			configured[defaultRustLibraryName(lib.Name)] = true
		}
	}

	for _, c := range channels {
		if configured[c] {
			continue
		}
		serviceConfig, err := serviceconfig.Find(googleapisDir, c)
		if err != nil {
			return err
		}
		cfg.Libraries = append(cfg.Libraries, &config.Library{
			Name: defaultRustLibraryName(c),
			Channels: []*config.Channel{{
				Path:          c,
				ServiceConfig: serviceConfig,
			}},
		})
	}
	return nil
}

// findChannels scans the googleapis directory and returns all API paths.
func findChannels(googleapisDir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(googleapisDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if !containsProtoFiles(path) {
			return nil
		}
		channel, err := filepath.Rel(googleapisDir, path)
		if err != nil {
			return err
		}
		paths = append(paths, channel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// containsProtoFiles returns true if the directory contains any .proto files.
func containsProtoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".proto") {
			return true
		}
	}
	return false
}

// defaultRustLibraryName derives an API path from a Rust library name.
// For example: google-cloud-foo-v1 -> google/cloud/foo/v1
//
// TODO(https://github.com/googleapis/librarian/issues/3058): See
// go/cloud-rust:on-crate-names for overrides and make sure they are handled
// properly.
func defaultRustLibraryName(name string) string {
	return strings.ReplaceAll(name, "-", "/")
}
