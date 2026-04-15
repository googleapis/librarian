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

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	githubEndpoints = &fetch.Endpoints{
		API:      "https://api.github.com",
		Download: "https://github.com",
	}
)

// MigrationInput holds all intermediate configuration and state necessary for migration from legacy files.
type MigrationInput struct {
	librarianState  *legacyconfig.LibrarianState
	librarianConfig *legacyconfig.LibrarianConfig
	lang            string
	repoPath        string
}

func runLibrarianMigration(ctx context.Context, language string, repoPath string, librariesToMigrate []string) error {
	cfg, err := runCompleteCleanLibrarianMigration(ctx, language, repoPath)
	if err != nil {
		return err
	}

	if len(librariesToMigrate) > 0 {
		cfg, err = filterLibraries(cfg, librariesToMigrate)
		if err != nil {
			return err
		}
	}

	// If we already have a config, we just replace the libraries which are new.
	// (Everything else about the existing configuration is maintained.)
	existingConfig, err := yaml.Read[config.Config](filepath.Join(repoPath, "librarian.yaml"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to load existing librarian.yaml: %w", err)
	}
	if existingConfig != nil {
		existingConfig.Libraries = mergeLibraries(existingConfig, cfg)
		cfg = existingConfig
	}

	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return errTidyFailed
	}
	if err := blockLegacyGeneration(repoPath, cfg); err != nil {
		return err
	}
	if err := augmentLegacyReleaseExcludePaths(repoPath, cfg); err != nil {
		return err
	}
	return nil
}

// runCompleteCleanLibrarianMigration runs migration procedures assuming there's
// no existing librarian.yaml file, and that all libraries should be migrated.
func runCompleteCleanLibrarianMigration(ctx context.Context, language string, repoPath string) (*config.Config, error) {
	librarianState, err := readState(repoPath)
	if err != nil {
		return nil, err
	}

	librarianConfig, err := readLegacyConfig(repoPath)
	if err != nil {
		return nil, err
	}
	cfg, err := buildConfigFromLibrarian(ctx, &MigrationInput{
		librarianState:  librarianState,
		librarianConfig: librarianConfig,
		lang:            language,
		repoPath:        repoPath,
	})
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func buildConfigFromLibrarian(ctx context.Context, input *MigrationInput) (*config.Config, error) {
	repo := "googleapis/google-cloud-python"

	src, err := fetchSource(ctx)
	if err != nil {
		return nil, errFetchSource
	}

	cfg := &config.Config{
		Language: input.lang,
		Repo:     repo,
		Version:  librarian.Version(),
		Sources: &config.Sources{
			Googleapis: src,
		},
		Default: &config.Default{
			TagFormat: defaultTagFormat,
		},
		Release: &config.Release{},
	}

	cfg.Default.Python = &config.PythonDefault{
		// Declared in python.go.
		CommonGAPICPaths: pythonDefaultCommonGAPICPaths,
		LibraryType:      pythonDefaultLibraryType,
	}
	cfg.Libraries, err = buildPythonLibraries(input, src.Dir)
	if err != nil {
		return nil, err
	}
	cfg.Default.Output = "packages"
	cfg.Default.TagFormat = pythonTagFormat
	sort.Slice(cfg.Libraries, func(i, j int) bool {
		return cfg.Libraries[i].Name < cfg.Libraries[j].Name
	})
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""

	return cfg, nil
}

// blockLegacyGeneration ensures that all libraries in the librarian config have generation blocked in the legacy
// config, by rewriting .librarian/config.yaml.
// This was previously a file maintained by hand, so a comment line is added at the start. This function assumes that
// the current directory is the repository root.
func blockLegacyGeneration(repoPath string, cfg *config.Config) error {
	legacyConfig, err := readLegacyConfig(repoPath)
	if err != nil {
		return err
	}
	for _, lib := range cfg.Libraries {
		legacyLib := legacyConfig.LibraryConfigFor(lib.Name)
		if legacyLib == nil {
			legacyLib = &legacyconfig.LibraryConfig{
				LibraryID: lib.Name,
			}
			legacyConfig.Libraries = append(legacyConfig.Libraries, legacyLib)
		}
		legacyLib.GenerateBlocked = true
	}
	configYaml, err := yaml.Marshal(legacyConfig)
	if err != nil {
		return err
	}
	comment := "# This file is being migrated to librarian@latest, and is no longer maintained by hand.\n\n"
	configYaml = append([]byte(comment), configYaml...)
	configFile := filepath.Join(repoPath, librarianDir, librarianConfigFile)
	if err := os.WriteFile(configFile, configYaml, 0644); err != nil {
		return err
	}
	return nil
}

// augmentLegacyReleaseExcludePaths ensures that every library in the
// legacylibrarian state file that is also specified in the librarian config
// has release-exclude paths matching the ones in the release "ignored changes"
// config section. This will reduce the number of unnecessary releases
// immediately after migration.
func augmentLegacyReleaseExcludePaths(repoPath string, cfg *config.Config) error {
	state, err := readState(repoPath)
	if err != nil {
		return err
	}
	for _, lib := range cfg.Libraries {
		libraryState := state.LibraryByID(lib.Name)
		if libraryState == nil {
			continue
		}
		for _, path := range cfg.Release.IgnoredChanges {
			repoRelativePath := filepath.Join(cfg.Default.Output, lib.Name, path)
			if !slices.Contains(libraryState.ReleaseExcludePaths, repoRelativePath) {
				libraryState.ReleaseExcludePaths = append(libraryState.ReleaseExcludePaths, repoRelativePath)
			}
		}
	}
	stateFile := filepath.Join(repoPath, librarianDir, librarianStateFile)
	return yaml.Write(stateFile, state)
}

func fetchGoogleapis(ctx context.Context) (*config.Source, error) {
	return fetchGoogleapisWithCommit(ctx, githubEndpoints, fetch.DefaultBranchMaster)
}

func fetchGoogleapisWithCommit(ctx context.Context, endpoints *fetch.Endpoints, commitish string) (*config.Source, error) {
	repo := &fetch.RepoRef{
		Org:    "googleapis",
		Name:   "googleapis",
		Branch: commitish,
	}
	commit, sha256, err := fetch.LatestCommitAndChecksum(endpoints, repo)
	if err != nil {
		return nil, err
	}

	dir, err := fetch.Repo(ctx, googleapisRepo, commit, sha256)
	if err != nil {
		return nil, err
	}

	return &config.Source{
		Commit: commit,
		SHA256: sha256,
		Dir:    dir,
	}, nil
}

// filterLibraries reduces the list of libraries in config to those specified in
// librariesToMigrate, returning an error if any libraries which were specified
// to be migrated are not present. The configuration is modified in-place.
func filterLibraries(cfg *config.Config, librariesToMigrate []string) (*config.Config, error) {
	var result []*config.Library
	for _, name := range librariesToMigrate {
		library, err := librarian.FindLibrary(cfg, name)
		if err != nil {
			return nil, err
		}
		result = append(result, library)
	}
	cfg.Libraries = result
	return cfg, nil
}

// mergeLibraries returns a merged slice of libraries, containing all libraries from existingConfig,
// and libraries in newConfig which don't already exist in existingConfig. The order of the libraries
// in the slice is the libraries in existingConfig, followed by newly-merged libraries from newConfig,
// in the order in which they appear in the two configurations.
func mergeLibraries(existingConfig *config.Config, newConfig *config.Config) []*config.Library {
	existingSet := make(map[string]bool)
	for _, lib := range existingConfig.Libraries {
		existingSet[lib.Name] = true
	}
	merged := existingConfig.Libraries
	for _, lib := range newConfig.Libraries {
		if !existingSet[lib.Name] {
			merged = append(merged, lib)
		}
	}
	return merged
}

func toAPIs(legacyapis []*legacyconfig.API) []*config.API {
	apis := make([]*config.API, 0, len(legacyapis))
	for _, api := range legacyapis {
		apis = append(apis, &config.API{
			Path: api.Path,
		})
	}
	// Formatting the library will sort the APIs by path later anyway, so let's
	// do that now. That way the migration code will observe the list of APIs
	// in the same order that it will eventually be saved.
	serviceconfig.SortAPIs(apis)
	return apis
}

// readLegacyState reads the legacylibrarian state file for the given
// repository root directory.
func readState(path string) (*legacyconfig.LibrarianState, error) {
	stateFile := filepath.Join(path, librarianDir, librarianStateFile)
	return yaml.Read[legacyconfig.LibrarianState](stateFile)
}

// readLegacyConfig reads the legacylibrarian configuration file for the given
// repository root directory.
func readLegacyConfig(repoPath string) (*legacyconfig.LibrarianConfig, error) {
	configFile := filepath.Join(repoPath, librarianDir, librarianConfigFile)
	return yaml.Read[legacyconfig.LibrarianConfig](configFile)
}
