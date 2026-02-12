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
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
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
	ImportPath      string   `yaml:"import_path,omitempty"`
	NestedProtos    []string `yaml:"nested_protos,omitempty"`
	ProtoPackage    string   `yaml:"proto_package,omitempty"`
}

// MigrationInput holds all intermediate configuration and state necessary for migration from legacy files.
type MigrationInput struct {
	librarianState  *legacyconfig.LibrarianState
	librarianConfig *legacyconfig.LibrarianConfig
	repoConfig      *RepoConfig
	lang            string
	repoPath        string
}

var (
	addGoModules = map[string]*RepoConfigModule{
		"ai": {
			APIs: []*RepoConfigAPI{
				{
					Path:            "google/ai/generativelanguage/v1",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
				{
					Path:            "google/ai/generativelanguage/v1alpha",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
				{
					Path:            "google/ai/generativelanguage/v1beta",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
				{
					Path:            "google/ai/generativelanguage/v1beta2",
					ClientDirectory: "generativelanguage",
					ImportPath:      "ai/generativelanguage",
				},
			},
		},
	}

	libraryOverrides = map[string]*config.Library{
		"ai": {
			ReleaseLevel: "beta",
		},
	}
)

func runLibrarianMigration(ctx context.Context, language, repoPath string) error {
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

	cfg, err := buildConfigFromLibrarian(ctx, &MigrationInput{
		librarianState:  librarianState,
		librarianConfig: librarianConfig,
		repoConfig:      repoConfig,
		lang:            language,
		repoPath:        repoPath,
	})
	if err != nil {
		return err
	}
	if err := librarian.RunTidyOnConfig(ctx, cfg); err != nil {
		return errTidyFailed
	}
	return nil
}

func buildConfigFromLibrarian(ctx context.Context, input *MigrationInput) (*config.Config, error) {
	repo := "googleapis/google-cloud-go"
	if input.lang == "python" {
		repo = "googleapis/google-cloud-python"
	}

	src, err := fetchSource(ctx)
	if err != nil {
		return nil, errFetchSource
	}

	cfg := &config.Config{
		Language: input.lang,
		Repo:     repo,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Default: &config.Default{
			TagFormat: defaultTagFormat,
		},
	}

	if input.lang == "python" {
		cfg.Libraries, err = buildPythonLibraries(input, src.Dir)
		if err != nil {
			return nil, err
		}
		cfg.Default.Output = "packages"
		cfg.Default.ReleaseLevel = "stable"
		cfg.Default.Transport = "grpc+rest"
	} else {
		cfg.Default.Output = "."
		cfg.Default.ReleaseLevel = "ga"
		cfg.Libraries, err = buildGoLibraries(input)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(cfg.Libraries, func(i, j int) bool {
		return cfg.Libraries[i].Name < cfg.Libraries[j].Name
	})

	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""

	return cfg, nil
}

func fetchGoogleapis(ctx context.Context) (*config.Source, error) {
	endpoint := &fetch.Endpoints{
		API:      "https://api.github.com",
		Download: "https://github.com",
	}
	repo := &fetch.Repo{
		Org:    "googleapis",
		Repo:   "googleapis",
		Branch: fetch.DefaultBranchMaster,
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

func buildGoLibraries(input *MigrationInput) ([]*config.Library, error) {
	var libraries []*config.Library
	idToLibraryState := sliceToMap(
		input.librarianState.Libraries,
		func(lib *legacyconfig.LibraryState) string {
			return lib.ID
		})

	idToLibraryConfig := sliceToMap(
		input.librarianConfig.Libraries,
		func(lib *legacyconfig.LibraryConfig) string {
			return lib.LibraryID
		})

	idToGoModule := make(map[string]*RepoConfigModule)
	if input.repoConfig != nil {
		idToGoModule = sliceToMap(
			input.repoConfig.Modules,
			func(mod *RepoConfigModule) string {
				return mod.Name
			})
	}
	maps.Copy(idToGoModule, addGoModules)
	libraryNames, err := libraryWithAliasshim(input.repoPath)
	if err != nil {
		return nil, err
	}
	// Iterate libraries from idToLibraryState because librarianConfig.Libraries is a
	// subset of librarianState.Libraries.
	for id, libState := range idToLibraryState {
		library := &config.Library{}
		library.Name = id
		library.Version = libState.Version
		if libState.APIs != nil {
			library.APIs = toAPIs(libState.APIs)
		}
		library.Keep = append(library.Keep, libState.PreserveRegex...)
		library.Keep = append(library.Keep, filepath.Join(id, "CHANGES.md"))
		if libraryNames[id] {
			library.Keep = append(library.Keep, filepath.Join(id, "aliasshim", "aliasshim.go"))
		}
		slices.Sort(library.Keep)

		libCfg, ok := idToLibraryConfig[id]
		if ok {
			library.SkipGenerate = libCfg.GenerateBlocked
			library.SkipRelease = libCfg.ReleaseBlocked
		}
		// The source of truth of release level is BUILD.bazel, use a map to store the special value.
		if override, ok := libraryOverrides[id]; ok {
			library.ReleaseLevel = override.ReleaseLevel
		}

		libGoModule, ok := idToGoModule[id]
		if ok {
			var goAPIs []*config.GoAPI
			for _, api := range libGoModule.APIs {
				goAPIs = append(goAPIs, &config.GoAPI{
					Path:            api.Path,
					ClientDirectory: api.ClientDirectory,
					DisableGAPIC:    api.DisableGAPIC,
					ImportPath:      api.ImportPath,
					NestedProtos:    api.NestedProtos,
					ProtoPackage:    api.ProtoPackage,
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

	return libraries, nil
}

func sliceToMap[T any](slice []*T, keyFunc func(t *T) string) map[string]*T {
	res := make(map[string]*T, len(slice))
	for _, t := range slice {
		key := keyFunc(t)
		res[key] = t
	}

	return res
}

func toAPIs(legacyapis []*legacyconfig.API) []*config.API {
	apis := make([]*config.API, 0, len(legacyapis))
	for _, api := range legacyapis {
		apis = append(apis, &config.API{
			Path: api.Path,
		})
	}
	return apis
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

// libraryWithAliasshim traverses the repoPath to find repoPath/{libraryName}/aliasshim/aliasshim.go.
// Returns all name of the library in a map for faster look up.
func libraryWithAliasshim(repoPath string) (map[string]bool, error) {
	files, err := aliasshim(repoPath)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool)
	for _, file := range files {
		parentDir := filepath.Dir(filepath.Dir(file))
		names[filepath.Base(parentDir)] = true
	}
	return names, nil
}

func aliasshim(repoPath string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "aliasshim.go" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
