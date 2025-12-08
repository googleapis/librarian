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

// Command migrate-librarian is a tool for migrating .librarian/state.yaml and .librarian/config.yaml to librarian
// configuration.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	librarianDir        = ".librarian"
	librarianStateFile  = "state.yaml"
	librarianConfigFile = "config.yaml"
	defaultTagFormat    = "{name}/v{version}"
	googleapisRepo      = "github.com/googleapis/googleapis"
)

var (
	errFetchSource      = errors.New("cannot fetch source")
	errLangNotSupported = errors.New("only go and python are supported")
	errRepoNotFound     = errors.New("exactly one repo path argument is required")
	errTidyFailed       = errors.New("librarian tidy failed")

	fetchSource = fetchGoogleapis
)

// RepoConfig represents the .librarian/generator-input/repo-config.yaml file in google-cloud-go repository.
type RepoConfig struct {
	Modules []*RepoConfigModule `yaml:"modules"`
}

// RepoConfigModule represents a module in repo-config.yaml.
type RepoConfigModule struct {
	Name                        string           `yaml:"name"`
	ModulePathVersion           string           `yaml:"module_path_version,omitempty"`
	DeleteGenerationOutputPaths []string         `yaml:"delete_generation_output_paths,omitempty"`
	APIs                        []*RepoConfigAPI `yaml:"apis,omitempty"`
}

// RepoConfigAPI represents an API in repo-config.yaml.
type RepoConfigAPI struct {
	Path            string   `yaml:"path"`
	ClientDirectory string   `yaml:"client_directory,omitempty"`
	DisableGAPIC    bool     `yaml:"disable_gapic,omitempty"`
	NestedProtos    []string `yaml:"nested_protos,omitempty"`
	ProtoPackage    string   `yaml:"proto_package,omitempty"`
}

type Config struct {
	librarianState  *legacyconfig.LibrarianState
	librarianConfig *legacyconfig.LibrarianConfig
	repoConfig      *RepoConfig
	lang            string
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatalf("migrate-librarian failed: %q", err)
	}
}

func run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("migrate-librarian", flag.ContinueOnError)
	outputPath := flagSet.String("output", "./librarian.yaml", "Output file path (default: ./librarian.yaml)")
	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if flagSet.NArg() != 1 {
		return errRepoNotFound
	}
	repoPath := flagSet.Arg(0)

	language, err := deriveLanguage(repoPath)
	if err != nil {
		return err
	}

	librarianState, err := readState(repoPath)
	if err != nil {
		return err
	}

	librarianConfig, err := readConfig(repoPath)
	if err != nil {
		return err
	}

	repoConfig, err := readRepoConfig(repoPath)
	if err != nil {
		return err
	}

	cfg, err := buildConfig(ctx, &Config{
		librarianState:  librarianState,
		librarianConfig: librarianConfig,
		repoConfig:      repoConfig,
		lang:            language,
	})

	if err != nil {
		return err
	}

	if err := yaml.Write(*outputPath, cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := librarian.RunTidy(); err != nil {
		return errTidyFailed
	}

	return nil
}

func deriveLanguage(repoPath string) (string, error) {
	base := filepath.Base(repoPath)
	switch {
	case strings.HasSuffix(base, "go"):
		return "go", nil
	case strings.HasSuffix(base, "python"):
		return "python", nil
	default:
		return "", errLangNotSupported
	}
}

func buildConfig(
	ctx context.Context,
	languageConfig *Config) (*config.Config, error) {
	repo := "googleapis/google-cloud-go"
	if languageConfig.lang == "python" {
		repo = "googleapis/google-cloud-python"
	}

	src, err := fetchSource(ctx)
	if err != nil {
		return nil, errFetchSource
	}

	cfg := &config.Config{
		Language: languageConfig.lang,
		Repo:     repo,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Default: &config.Default{
			TagFormat: defaultTagFormat,
		},
	}

	cfg.Libraries = buildLibraries(languageConfig)

	return cfg, nil
}

func fetchGoogleapis(ctx context.Context) (*config.Source, error) {
	endpoint := &fetch.Endpoints{
		API:      "https://api.github.com",
		Download: "https://github.com",
	}
	repo := &fetch.Repo{
		Org:  "googleapis",
		Repo: "googleapis",
	}
	latestCommit, sha256, err := fetch.LatestCommitAndChecksum(endpoint, repo)
	if err != nil {
		return nil, err
	}

	dir, err := fetch.RepoDir(ctx, googleapisRepo, latestCommit, sha256)
	if err != nil {
		return nil, err
	}

	return &config.Source{
		Commit: latestCommit,
		SHA256: sha256,
		Dir:    dir,
	}, nil
}

func buildLibraries(cfg *Config) []*config.Library {
	var libraries []*config.Library
	idToLibraryState := sliceToMap[legacyconfig.LibraryState](
		cfg.librarianState.Libraries,
		func(lib *legacyconfig.LibraryState) string {
			return lib.ID
		})

	idToLibraryConfig := sliceToMap[legacyconfig.LibraryConfig](
		cfg.librarianConfig.Libraries,
		func(lib *legacyconfig.LibraryConfig) string {
			return lib.LibraryID
		})

	idToGoModule := make(map[string]*RepoConfigModule)
	if cfg.lang == "go" {
		idToGoModule = sliceToMap[RepoConfigModule](
			cfg.repoConfig.Modules,
			func(mod *RepoConfigModule) string {
				return mod.Name
			})
	}

	// Iterate libraries from idToLibraryState because librarianConfig.Libraries is a
	// subset of librarianState.Libraries.
	for id, libState := range idToLibraryState {
		library := &config.Library{}
		library.Name = id
		library.Version = libState.Version
		if libState.APIs != nil {
			library.Channels = toChannels(libState.APIs)
		}
		library.Keep = libState.PreserveRegex

		libCfg, ok := idToLibraryConfig[id]
		if ok {
			library.SkipGenerate = libCfg.GenerateBlocked
			library.SkipRelease = libCfg.ReleaseBlocked
		}

		libGoModule, ok := idToGoModule[id]
		if ok {
			var goAPIs []*config.GoAPI
			for _, oneApi := range libGoModule.APIs {
				goAPIs = append(goAPIs, &config.GoAPI{
					Path:            oneApi.Path,
					ClientDirectory: oneApi.ClientDirectory,
					DisableGAPIC:    oneApi.DisableGAPIC,
					NestedProtos:    oneApi.NestedProtos,
					ProtoPackage:    oneApi.ProtoPackage,
				})
			}

			goModule := &config.GoModule{
				DeleteGenerationOutputPaths: libGoModule.DeleteGenerationOutputPaths,
				GoAPIs:                      goAPIs,
				ModulePathVersion:           libGoModule.ModulePathVersion,
			}

			if !isEmptyGoModule(goModule) {
				library.Go = goModule
			}
		}

		libraries = append(libraries, library)
	}

	sort.Slice(libraries, func(i, j int) bool {
		return libraries[i].Name < libraries[j].Name
	})

	return libraries
}

func sliceToMap[T any](slice []*T, keyFunc func(t *T) string) map[string]*T {
	res := make(map[string]*T, len(slice))
	for _, t := range slice {
		key := keyFunc(t)
		res[key] = t
	}

	return res
}

func toChannels(apis []*legacyconfig.API) []*config.Channel {
	channels := make([]*config.Channel, 0, len(apis))
	for _, api := range apis {
		channels = append(channels, &config.Channel{
			Path: api.Path,
		})
	}

	return channels
}

func isEmptyGoModule(mod *config.GoModule) bool {
	return reflect.DeepEqual(mod, &config.GoModule{})
}

func readState(path string) (*legacyconfig.LibrarianState, error) {
	stateFile := filepath.Join(path, librarianDir, librarianStateFile)
	return yaml.Read[legacyconfig.LibrarianState](stateFile)
}

func readConfig(path string) (*legacyconfig.LibrarianConfig, error) {
	configFile := filepath.Join(path, librarianDir, librarianConfigFile)
	return yaml.Read[legacyconfig.LibrarianConfig](configFile)
}

func readRepoConfig(path string) (*RepoConfig, error) {
	configFile := filepath.Join(path, librarianDir, "generator-input/repo-config.yaml")
	if _, err := os.Stat(configFile); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	return yaml.Read[RepoConfig](configFile)
}
