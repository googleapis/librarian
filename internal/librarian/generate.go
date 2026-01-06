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
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/urfave/cli/v3"
)

const (
	discoveryRepo  = "github.com/googleapis/discovery-artifact-manager"
	googleapisRepo = "github.com/googleapis/googleapis"
	protobufRepo   = "github.com/protocolbuffers/protobuf"
	showcaseRepo   = "github.com/googleapis/gapic-showcase"
)

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
	googleapisDir, err := fetchSource(ctx, cfg.Sources.Googleapis, googleapisRepo)
	if err != nil {
		return err
	}
	return routeGenerate(ctx, all, cfg, googleapisDir, libraryName)
}

func routeGenerate(ctx context.Context, all bool, cfg *config.Config, googleapisDir, libraryName string) error {
	if all {
		return generateAll(ctx, cfg, googleapisDir)
	}
	lib, err := generateLibrary(ctx, cfg, googleapisDir, libraryName)
	if err != nil {
		return err
	}
	if lib == nil {
		// Skip formatting if generation skipped.
		return nil
	}
	return formatLibrary(ctx, cfg.Language, lib)
}

func generateAll(ctx context.Context, cfg *config.Config, googleapisDir string) error {
	for _, lib := range cfg.Libraries {
		lib, err := generateLibrary(ctx, cfg, googleapisDir, lib.Name)
		if err != nil {
			return err
		}
		if lib == nil {
			// Skip formatting if generation skipped.
			continue
		}
		if err := formatLibrary(ctx, cfg.Language, lib); err != nil {
			return err
		}
	}
	return nil
}

func defaultOutput(language, channel, defaultOut string) string {
	switch language {
	case languageRust:
		return rust.DefaultOutput(channel, defaultOut)
	default:
		return defaultOut
	}
}

func deriveChannelPath(language, name string) string {
	switch language {
	case languageRust:
		return rust.DeriveChannelPath(name)
	default:
		return strings.ReplaceAll(name, "-", "/")
	}
}

func generateLibrary(ctx context.Context, cfg *config.Config, googleapisDir, libraryName string) (*config.Library, error) {
	for _, lib := range cfg.Libraries {
		if lib.Name == libraryName {
			if lib.SkipGenerate {
				return nil, nil
			}
			lib, err := prepareLibrary(cfg.Language, lib, cfg.Default, googleapisDir)
			if err != nil {
				return nil, err
			}
			return generate(ctx, cfg.Language, lib, cfg.Sources)
		}
	}
	return nil, fmt.Errorf("library %q not found", libraryName)
}

// prepareLibrary applies language-specific derivations and fills defaults.
// For Rust libraries without an explicit output path, it derives the output
// from the first channel path.
func prepareLibrary(language string, lib *config.Library, defaults *config.Default, googleapisDir string) (*config.Library, error) {
	if len(lib.Channels) == 0 {
		// If no channels are specified, create an empty channel first
		lib.Channels = append(lib.Channels, &config.Channel{})
	}

	// The googleapis path of a veneer library lives in language-specific configurations,
	// so we only need to derive the path and service config for non-veneer libraries.
	if !lib.Veneer {
		for _, ch := range lib.Channels {
			if ch.Path == "" {
				ch.Path = deriveChannelPath(language, lib.Name)
			}
			if ch.ServiceConfig == "" {
				sc, err := serviceconfig.Find(googleapisDir, ch.Path)
				if err != nil {
					return nil, err
				}
				ch.ServiceConfig = sc
			}
		}
	}

	if lib.Output == "" {
		if lib.Veneer {
			return nil, fmt.Errorf("veneer %q requires an explicit output path", lib.Name)
		}
		lib.Output = defaultOutput(language, lib.Channels[0].Path, defaults.Output)
	}
	return fillDefaults(lib, defaults), nil
}

func generate(ctx context.Context, language string, library *config.Library, cfgSources *config.Sources) (_ *config.Library, err error) {
	googleapisDir, err := fetchSource(ctx, cfgSources.Googleapis, googleapisRepo)
	if err != nil {
		return nil, err
	}

	switch language {
	case languageFake:
		if err := fakeGenerate(library); err != nil {
			return nil, err
		}
	case languagePython:
		if err := cleanOutput(library.Output, library.Keep); err != nil {
			return nil, err
		}
		if err := python.Generate(ctx, library, googleapisDir); err != nil {
			return nil, err
		}
	case languageRust:
		sources := &rust.Sources{
			Googleapis: googleapisDir,
		}
		sources.Discovery, err = fetchSource(ctx, cfgSources.Discovery, discoveryRepo)
		if err != nil {
			return nil, err
		}
		sources.Conformance, err = fetchSource(ctx, cfgSources.Conformance, protobufRepo)
		if err != nil {
			return nil, err
		}
		sources.Showcase, err = fetchSource(ctx, cfgSources.Showcase, showcaseRepo)
		if err != nil {
			return nil, err
		}
		if cfgSources.ProtobufSrc != nil {
			dir, err := fetchSource(ctx, cfgSources.ProtobufSrc, protobufRepo)
			if err != nil {
				return nil, err
			}
			sources.ProtobufSrc = filepath.Join(dir, cfgSources.ProtobufSrc.Subpath)
		}
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
	default:
		return nil, fmt.Errorf("generate not implemented for %q", language)
	}
	fmt.Printf("✓ Successfully generated %s\n", library.Name)
	return library, nil
}

func formatLibrary(ctx context.Context, language string, library *config.Library) error {
	switch language {
	case languageFake:
		return fakeFormat(library)
	case languageRust:
		return rust.Format(ctx, library)
	}
	return fmt.Errorf("format not implemented for %q", language)
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
		// Effectively get a canonical relative path. While in most cases
		// this will be equal to k, it might not be - in particular,
		// on Windows the directory separator in paths returned by Rel
		// will be a backslash.
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		keepSet[rel] = true
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
