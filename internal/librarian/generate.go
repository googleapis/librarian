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
	"github.com/googleapis/librarian/internal/librarian/internal/python"
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
	lib, err := generateLibrary(ctx, cfg, libraryName)
	if err != nil {
		return err
	}
	return formatLibrary(cfg.Language, lib)
}

func generateAll(ctx context.Context, cfg *config.Config) error {
	googleapisDir, err := fetchGoogleapisDir(ctx, cfg.Sources)
	if err != nil {
		return err
	}

	libraries, err := deriveDefaultLibraries(cfg, googleapisDir)
	if err != nil {
		return err
	}
	cfg.Libraries = append(cfg.Libraries, libraries...)
	for _, lib := range cfg.Libraries {
		lib, err := generateLibrary(ctx, cfg, lib.Name)
		if err != nil {
			return err
		}
		if err := formatLibrary(cfg.Language, lib); err != nil {
			return err
		}
	}
	return nil
}

// deriveDefaultLibraries finds libraries for allowed channels that are not
// explicitly configured in librarian.yaml.
//
// For each allowed channel without configuration, it derives default values
// for the library name and output path. If the output directory exists, the
// library is added for generation. Channels whose output directories do not
// exist in the librarian.yaml but should be generated are returned.
func deriveDefaultLibraries(cfg *config.Config, googleapisDir string) ([]*config.Library, error) {
	if cfg.Default == nil {
		return nil, nil
	}

	configured := make(map[string]bool)
	for _, lib := range cfg.Libraries {
		for _, ch := range lib.Channels {
			configured[ch.Path] = true
		}
	}

	var derived []*config.Library
	for channel := range serviceconfig.Allowlist {
		if configured[channel] {
			continue
		}
		name := defaultLibraryName(cfg.Language, channel)
		output := defaultOutput(cfg.Language, channel, cfg.Default.Output)
		if !dirExists(output) {
			continue
		}
		sc, err := serviceconfig.Find(googleapisDir, channel)
		if err != nil {
			return nil, err
		}
		derived = append(derived, &config.Library{
			Name:   name,
			Output: output,
			Channels: []*config.Channel{{
				Path:          channel,
				ServiceConfig: sc,
			}},
		})
	}
	return derived, nil
}

func defaultLibraryName(language, channel string) string {
	switch language {
	case "rust":
		return rust.DefaultLibraryName(channel)
	default:
		return channel
	}
}

func defaultOutput(language, channel, defaultOut string) string {
	switch language {
	case "rust":
		return rust.DefaultOutput(channel, defaultOut)
	default:
		return defaultOut
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func generateLibrary(ctx context.Context, cfg *config.Config, libraryName string) (*config.Library, error) {
	googleapisDir, err := fetchGoogleapisDir(ctx, cfg.Sources)
	if err != nil {
		return nil, err
	}
	for _, lib := range cfg.Libraries {
		if lib.Name == libraryName {
			if lib.SkipGenerate {
				fmt.Printf("⊘ Skipping %s (skip_generate is set)\n", lib.Name)
				return nil, nil
			}
			lib, err := prepareLibrary(cfg.Language, lib, cfg.Default)
			if err != nil {
				return nil, err
			}
			for _, api := range lib.Channels {
				if api.ServiceConfig == "" {
					serviceConfig, err := serviceconfig.Find(googleapisDir, api.Path)
					if err != nil {
						return nil, err
					}
					api.ServiceConfig = serviceConfig
				}
			}
			return generate(ctx, cfg.Language, lib, cfg.Sources)
		}
	}
	return nil, fmt.Errorf("library %q not found", libraryName)
}

// prepareLibrary applies language-specific derivations and fills defaults.
// For Rust libraries without an explicit output path, it derives the output
// from the first channel path.
func prepareLibrary(language string, lib *config.Library, defaults *config.Default) (*config.Library, error) {
	if lib.Output == "" {
		if lib.Veneer {
			return nil, fmt.Errorf("veneer %q requires an explicit output path", lib.Name)
		}
		if len(lib.Channels) == 0 {
			return nil, fmt.Errorf("library %q has no channels, cannot determine default output", lib.Name)
		}
		lib.Output = defaultOutput(language, lib.Channels[0].Path, defaults.Output)
	}
	return fillDefaults(lib, defaults), nil
}

func generate(ctx context.Context, language string, library *config.Library, sources *config.Sources) (*config.Library, error) {
	switch language {
	case "testhelper":
		if err := testGenerate(library); err != nil {
			return nil, err
		}
	case "rust":
		keep, err := rust.Keep(library)
		if err != nil {
			return nil, fmt.Errorf("library %s: %w", library.Name, err)
		}
		if err := cleanOutput(library.Output, keep); err != nil {
			return nil, fmt.Errorf("library %s: %w", library.Name, err)
		}
		if err := rust.Generate(ctx, library, sources); err != nil {
			return nil, err
		}
	case "python":
		if err := cleanOutput(library.Output, library.Keep); err != nil {
			return nil, err
		}
		if err := python.Generate(ctx, library, sources); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("generate not implemented for %q", language)
	}
	fmt.Printf("✓ Successfully generated %s\n", library.Name)
	return library, nil
}

func formatLibrary(language string, library *config.Library) error {
	switch language {
	case "testhelper":
		return nil
	case "rust":
		return rust.Format(library)
	}
	return fmt.Errorf("format not implemented for %q", language)
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
// should contain paths relative to dir. It returns an error if the directory
// does not exist or any file in keep does not exist.
func cleanOutput(dir string, keep []string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("output directory %q does not exist; check that the output field in librarian.yaml is correct", dir)
		}
		return fmt.Errorf("failed to stat output directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output path %q is not a directory", dir)
	}

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
