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
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	librarianDir        = ".librarian"
	librarianStateFile  = "state.yaml"
	librarianConfigFile = "config.yaml"
	defaultTagFormat    = "{name}/v{version}"
)

var (
	errRepoNotFound = errors.New("-repo flag is required")
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migrate-librarian failed", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flagSet := flag.NewFlagSet("migrate-librarian", flag.ContinueOnError)
	repoPath := flagSet.String("repo", "", "Path to the repository containing legacy .librarian configuration (required)")
	language := flagSet.String("lang", "go", "One of go and python (default: go)")
	outputPath := flagSet.String("output", "./.librarian.yaml", "Output file path (default: ./.librarian.yaml)")
	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if *repoPath == "" {
		return errRepoNotFound
	}

	librarianState, err := readState(*repoPath)
	if err != nil {
		return err
	}

	librarianConfig, err := readConfig(*repoPath)
	if err != nil {
		return err
	}

	cfg := buildConfig(librarianState, librarianConfig, *language)

	return yaml.Write(*outputPath, cfg)
}

func buildConfig(
	librarianState *legacyconfig.LibrarianState,
	librarianConfig *legacyconfig.LibrarianConfig,
	lang string) *config.Config {
	repo := "googleapis/google-cloud-go"
	if lang == "python" {
		repo = "googleapis/google-cloud-python"
	}
	cfg := &config.Config{
		Language: lang,
		Repo:     repo,
		Default: &config.Default{
			TagFormat: defaultTagFormat,
		},
	}

	cfg.Libraries = buildLibraries(librarianState, librarianConfig)

	return cfg
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
		// Go and Python monorepo only contains GAPIC libraries.
		library.SpecificationFormat = "protobuf"
		// hydrate params from library config
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
			Path:          api.Path,
			ServiceConfig: api.ServiceConfig,
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
