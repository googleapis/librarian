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

// Command migrate is a tool for migrating .sidekick.toml or .librarian configuration to librarian.yaml.
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
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/librarian/rust"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/pelletier/go-toml/v2"
)

const (
	sidekickFile             = ".sidekick.toml"
	cargoFile                = "Cargo.toml"
	googleapisArchivePrefix  = "https://github.com/googleapis/googleapis/archive/"
	showcaseArchivePrefix    = "https://github.com/googleapis/gapic-showcase/archive/"
	protobufArchivePrefix    = "https://github.com/protocolbuffers/protobuf/archive/"
	conformanceArchivePrefix = "https://github.com/protocolbuffers/protobuf/archive/"
	tarballSuffix            = ".tar.gz"
	librarianDir             = ".librarian"
	librarianStateFile       = "state.yaml"
	librarianConfigFile      = "config.yaml"
	defaultTagFormat         = "{name}/v{version}"
	googleapisRepo           = "github.com/googleapis/googleapis"
)

var (
	errRepoNotFound                = errors.New("-repo flag is required")
	errSidekickNotFound            = errors.New(".sidekick.toml not found")
	errTidyFailed                  = errors.New("librarian tidy failed")
	errUnableToCalculateOutputPath = errors.New("unable to calculate output path")
	errFetchSource                 = errors.New("cannot fetch source")
	errLibraryNameNotFound         = errors.New("library name not found")

	fetchSource = fetchGoogleapis
)

var excludedVeneerLibraries = map[string]struct{}{
	"echo-server": {},
	"gcp-sdk":     {},
}

type PubSpec struct {
	Name            string            `yaml:"name,omitempty"`
	Description     string            `yaml:"description,omitempty"`
	Version         string            `yaml:"version,omitempty"`
	Repository      string            `yaml:"repository,omitempty"`
	IssueTracker    string            `yaml:"issue_tracker,omitempty"`
	Environment     map[string]string `yaml:"environment,omitempty"`
	Resolution      string            `yaml:"resolution,omitempty"`
	Dependencies    map[string]string `yaml:"dependencies,omitempty"`
	DevDependencies map[string]string `yaml:"dev_dependencies,omitempty"`
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("migrate", flag.ContinueOnError)
	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if flagSet.NArg() < 1 {
		return errRepoNotFound
	}

	repoPath := flagSet.Arg(0)
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	base := filepath.Base(abs)
	switch base {
	case "google-cloud-dart":
		return runSidekickMigration(ctx, abs)
	case "google-cloud-rust":
		return fmt.Errorf(".sidekick.toml files have been deleted in %q", base)
	case "google-cloud-python", "google-cloud-go":
		parts := strings.SplitN(base, "-", 3)
		return runLibrarianMigration(ctx, parts[2], abs)
	default:
		return fmt.Errorf("invalid path: %q", repoPath)
	}
}

func runSidekickMigration(ctx context.Context, repoPath string) error {
	defaults, err := readRootSidekick(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read root .sidekick.toml from %q: %w", repoPath, err)
	}

	sidekickFiles, err := findSidekickFiles(filepath.Join(repoPath, "generated"))
	if err != nil {
		return fmt.Errorf("failed to find sidekick.toml files: %w", err)
	}
	libraries, err := buildGAPIC(sidekickFiles, repoPath)
	if err != nil {
		return fmt.Errorf("failed to read sidekick.toml files: %w", err)
	}

	cfg := buildConfig(libraries, defaults)

	if err := librarian.RunTidyOnConfig(ctx, cfg); err != nil {
		return errTidyFailed
	}
	return nil
}

// readRootSidekick reads the root .sidekick.toml file and extracts defaults.
func readRootSidekick(repoPath string) (*config.Config, error) {
	rootPath := filepath.Join(repoPath, sidekickFile)
	data, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, errSidekickNotFound
	}

	// Parse as generic map to handle the dynamic package keys
	var sidekick sidekickconfig.Config
	if err := toml.Unmarshal(data, &sidekick); err != nil {
		return nil, err
	}

	version := sidekick.Codec["version"]
	apiKeys := sidekick.Codec["api-keys-environment-variables"]
	issueTrackerURL := sidekick.Codec["issue-tracker-url"]
	googleapisSHA256 := sidekick.Source["googleapis-sha256"]
	googleapisRoot := sidekick.Source["googleapis-root"]
	showcaseRoot := sidekick.Source["showcase-root"]
	showcaseSHA256 := sidekick.Source["showcase-sha256"]
	protobufRoot := sidekick.Source["protobuf-root"]
	protobufSHA256 := sidekick.Source["protobuf-sha256"]
	protobufSubDir := sidekick.Source["protobuf-subdir"]
	conformanceRoot := sidekick.Source["conformance-root"]
	conformanceSHA256 := sidekick.Source["conformance-sha256"]

	googleapisCommit := strings.TrimSuffix(strings.TrimPrefix(googleapisRoot, googleapisArchivePrefix), tarballSuffix)
	showcaseCommit := strings.TrimSuffix(strings.TrimPrefix(showcaseRoot, showcaseArchivePrefix), tarballSuffix)
	protobufCommit := strings.TrimSuffix(strings.TrimPrefix(protobufRoot, protobufArchivePrefix), tarballSuffix)
	conformanceCommit := strings.TrimSuffix(strings.TrimPrefix(conformanceRoot, conformanceArchivePrefix), tarballSuffix)

	prefix := parseKeyWithPrefix(sidekick.Codec, "prefix:")
	packages := parseKeyWithPrefix(sidekick.Codec, "package:")
	protos := parseKeyWithPrefix(sidekick.Codec, "proto:")

	cfg := &config.Config{
		Language: "dart",
		Version:  version,
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: googleapisCommit,
				SHA256: googleapisSHA256,
			},
			Showcase: &config.Source{
				Commit: showcaseCommit,
				SHA256: showcaseSHA256,
			},
			ProtobufSrc: &config.Source{
				Commit:  protobufCommit,
				SHA256:  protobufSHA256,
				Subpath: protobufSubDir,
			},
			Conformance: &config.Source{
				Commit: conformanceCommit,
				SHA256: conformanceSHA256,
			},
		},
		Default: &config.Default{
			Output: "generated/",
			Dart: &config.DartPackage{
				APIKeysEnvironmentVariables: apiKeys,
				IssueTrackerURL:             issueTrackerURL,
				Prefixes:                    prefix,
				Protos:                      protos,
				Packages:                    packages,
			},
		},
	}
	if sidekick.Release != nil {
		cfg.Release = &config.Release{
			Branch:         sidekick.Release.Branch,
			Remote:         sidekick.Release.Remote,
			IgnoredChanges: sidekick.Release.IgnoredChanges,
		}
	}
	return cfg, nil
}

// findSidekickFiles finds all .sidekick.toml files within the given path.
func findSidekickFiles(path string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == sidekickFile {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i] < files[j]
	})

	return files, nil
}

func buildGAPIC(files []string, repoPath string) (map[string]*config.Library, error) {
	libraries := make(map[string]*config.Library)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		var sidekick sidekickconfig.Config
		if err := toml.Unmarshal(data, &sidekick); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", file, err)
		}

		// Get API path
		apiPath := sidekick.General.SpecificationSource
		if apiPath == "" {
			continue
		}

		specificationFormat := sidekick.General.SpecificationFormat
		if specificationFormat == "" {
			specificationFormat = "protobuf"
		}

		dir := filepath.Dir(file)
		pubSpec, err := readPubSpec(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to read pubspec.yaml at %s: %w", dir, err)
		}

		if pubSpec.Name == "" {
			return nil, errLibraryNameNotFound
		}
		libraryName := pubSpec.Name
		lib, exists := libraries[libraryName]
		if !exists {
			lib = &config.Library{
				Name: libraryName,
			}
			libraries[libraryName] = lib
		}

		if pubSpec.Version != "" {
			lib.Version = pubSpec.Version
		}

		lib.APIs = append(lib.APIs, &config.API{
			Path: apiPath,
		})

		if copyrightYear, ok := sidekick.Codec["copyright-year"]; ok && copyrightYear != "" {
			lib.CopyrightYear = copyrightYear
		}

		if pubSpec.Description != "" {
			lib.DescriptionOverride = pubSpec.Description
		}

		relativePath, err := filepath.Rel(repoPath, dir)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate relative path: %w", errUnableToCalculateOutputPath)
		}
		lib.Output = relativePath

		if _, ok := sidekick.Codec["not-for-publication"]; ok {
			lib.SkipPublish = true
		}

		lib.SpecificationFormat = specificationFormat

		dartPackage := &config.DartPackage{}
		if apiKeys, ok := sidekick.Codec["api-keys-environment-variables"]; ok && apiKeys != "" {
			dartPackage.APIKeysEnvironmentVariables = apiKeys
		}

		if devDeps, ok := sidekick.Codec["dev-dependencies"]; ok && devDeps != "" {
			dartPackage.DevDependencies = devDeps
		}
		if extraImports, ok := sidekick.Codec["extra-imports"]; ok && extraImports != "" {
			dartPackage.ExtraImports = extraImports
		}
		if partFile, ok := sidekick.Codec["part-file"]; ok && partFile != "" {
			dartPackage.PartFile = partFile
		}
		if repoURL, ok := sidekick.Codec["repository-url"]; ok && repoURL != "" {
			dartPackage.RepositoryURL = repoURL
		}
		if afterTitle, ok := sidekick.Codec["readme-after-title-text"]; ok && afterTitle != "" {
			dartPackage.ReadmeAfterTitleText = afterTitle
		}
		if quickStart, ok := sidekick.Codec["readme-quickstart-text"]; ok && quickStart != "" {
			dartPackage.ReadmeQuickstartText = quickStart
		}

		if !isEmptyDartPackage(dartPackage) {
			lib.Dart = dartPackage
		}
	}

	return libraries, nil
}

// deriveLibraryName derives a library name from an API path.
// For Rust: see go/cloud-rust:on-crate-names.
func deriveLibraryName(apiPath string) string {
	trimmedPath := strings.TrimPrefix(apiPath, "google/")
	trimmedPath = strings.TrimPrefix(trimmedPath, "cloud/")
	trimmedPath = strings.TrimPrefix(trimmedPath, "devtools/")
	if strings.HasPrefix(trimmedPath, "api/apikeys/") {
		trimmedPath = strings.TrimPrefix(trimmedPath, "api/")
	}

	return "google-cloud-" + strings.ReplaceAll(trimmedPath, "/", "-")
}

// buildConfig builds the complete config from libraries.
func buildConfig(libraries map[string]*config.Library, defaults *config.Config) *config.Config {
	cfg := defaults
	// Convert libraries map to sorted slice, applying new schema logic
	var libList []*config.Library

	for _, lib := range libraries {
		// Get the API path for this library
		apiPath := ""
		if len(lib.APIs) > 0 {
			apiPath = lib.APIs[0].Path
		}

		// Derive expected library name from API path
		expectedName := deriveLibraryName(apiPath)
		nameMatchesConvention := lib.Name == expectedName
		// Check if library has extra configuration beyond just name/api/version
		hasExtraConfig := lib.CopyrightYear != "" ||
			(lib.Rust != nil && (lib.Rust.PerServiceFeatures || len(lib.Rust.DisabledRustdocWarnings) > 0 ||
				lib.Rust.GenerateSetterSamples != "" || lib.Rust.GenerateRpcSamples != "" ||
				len(lib.Rust.PackageDependencies) > 0 || len(lib.Rust.PaginationOverrides) > 0 ||
				lib.Rust.NameOverrides != ""))
		// Only include in libraries section if specific data needs to be retained
		if !nameMatchesConvention || hasExtraConfig || len(lib.APIs) > 1 {
			libCopy := *lib
			libList = append(libList, &libCopy)
		}
	}

	// Sort libraries by name
	sort.Slice(libList, func(i, j int) bool {
		return libList[i].Name < libList[j].Name
	})

	cfg.Libraries = libList

	return cfg
}

func parseKeyWithPrefix(codec map[string]string, prefix string) map[string]string {
	res := make(map[string]string)
	for key, value := range codec {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		res[key] = value
	}
	return res
}

func strToBool(s string) bool {
	return s == "true"
}

// strToSlice converts a comma-separated string into a slice of strings.
//
// The wantEmpty parameter controls the behavior when the input string is empty:
//   - If true: Returns an empty initialized slice (make([]string, 0)).
//   - If false: Returns nil.
func strToSlice(s string, wantEmpty bool) []string {
	if s == "" {
		if wantEmpty {
			return make([]string, 0)
		}

		return nil
	}

	return strings.Split(s, ",")
}

func isEmptyDartPackage(r *config.DartPackage) bool {
	return reflect.DeepEqual(r, &config.DartPackage{})
}

func readTOML[T any](file string) (*T, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", file, err)
	}

	var tomlData T
	if err := toml.Unmarshal(data, &tomlData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", file, err)
	}

	return &tomlData, nil
}

func readCargoConfig(dir string) (*rust.Cargo, error) {
	cargoData, err := os.ReadFile(filepath.Join(dir, cargoFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read cargo: %w", err)
	}
	cargo := rust.Cargo{
		Package: &rust.CrateInfo{
			Publish: true,
		},
	}
	if err := toml.Unmarshal(cargoData, &cargo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cargo: %w", err)
	}
	return &cargo, nil
}

func readPubSpec(dir string) (*PubSpec, error) {
	return yaml.Read[PubSpec](filepath.Join(dir, "pubspec.yaml"))
}
