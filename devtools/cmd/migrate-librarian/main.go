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
	errRepoNotFound     = errors.New("repo path argument is required")
	errLangNotSupported = errors.New("only go and python are supported")
	errFetchSource      = errors.New("cannot fetch source")
	errTidyFailed       = errors.New("librarian tidy failed")

	fetchSource = fetchGoogleapis
)

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
	if flagSet.NArg() < 1 {
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

	cfg, err := buildConfig(ctx, librarianState, librarianConfig, language)
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
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(absPath, "go") {
		return "go", nil
	}

	if strings.HasSuffix(absPath, "python") {
		return "python", nil
	}

	return "", errLangNotSupported
}

func buildConfig(
	ctx context.Context,
	librarianState *legacyconfig.LibrarianState,
	librarianConfig *legacyconfig.LibrarianConfig,
	lang string) (*config.Config, error) {
	repo := "googleapis/google-cloud-go"
	if lang == "python" {
		repo = "googleapis/google-cloud-python"
	}

	src, err := fetchSource(ctx)
	if err != nil {
		return nil, errFetchSource
	}

	cfg := &config.Config{
		Language: lang,
		Repo:     repo,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Default: &config.Default{
			TagFormat: defaultTagFormat,
		},
	}

	cfg.Libraries = buildLibraries(librarianState, librarianConfig)

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

func buildLibraries(
	librarianState *legacyconfig.LibrarianState,
	librarianConfig *legacyconfig.LibrarianConfig) []*config.Library {
	var libraries []*config.Library
	idToLibraryState := sliceToMap[legacyconfig.LibraryState](
		librarianState.Libraries,
		func(lib *legacyconfig.LibraryState) string {
			return lib.ID
		})
	idToLibraryConfig := sliceToMap[legacyconfig.LibraryConfig](
		librarianConfig.Libraries,
		func(lib *legacyconfig.LibraryConfig) string {
			return lib.LibraryID
		})

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

func readState(path string) (*legacyconfig.LibrarianState, error) {
	stateFile := filepath.Join(path, librarianDir, librarianStateFile)
	return yaml.Read[legacyconfig.LibrarianState](stateFile)
}

func readConfig(path string) (*legacyconfig.LibrarianConfig, error) {
	configFile := filepath.Join(path, librarianDir, librarianConfigFile)
	return yaml.Read[legacyconfig.LibrarianConfig](configFile)
}
