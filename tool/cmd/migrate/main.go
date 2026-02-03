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
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/pelletier/go-toml/v2"
)

const (
	sidekickFile             = ".sidekick.toml"
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

	fetchSource = fetchGoogleapis

	pathToName = map[string]string{
		"google_cloud_protojson_conformance": "google_cloud_protobuf_test_messages_proto3",
		"google_cloud_showcase_v1beta1":      "google_cloud_showcase_v1beta1",
	}

	libraryToKeep = map[string][]string{
		"google_cloud_showcase_v1beta1": {"dart_test.yaml"},
		"google_cloud_rpc": {
			"lib/src/exceptions.dart",
			"lib/src/versions.dart",
			"lib/src/vm.dart",
			"lib/src/web.dart",
			"lib/exceptions.dart",
			"lib/service_client.dart",
		},
	}
)

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
		Version:  librarian.Version(),
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
				Version:                     version,
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

func buildGAPIC(files []string, repoPath string) ([]*config.Library, error) {
	var libraries []*config.Library

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}

		var sidekick sidekickconfig.Config
		if err := toml.Unmarshal(data, &sidekick); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", file, err)
		}

		apiPath := sidekick.General.SpecificationSource
		if apiPath == "" {
			continue
		}

		specificationFormat := sidekick.General.SpecificationFormat
		if specificationFormat == "" {
			specificationFormat = "protobuf"
		}

		var libraryName string
		if name, ok := pathToName[filepath.Base(filepath.Dir(file))]; ok {
			libraryName = name
		} else {
			// Library name or package name is derived from api path by packageName function in dart package.
			// However, each library in the librarian configuration should have a name.
			libraryName = genLibraryName(apiPath)
		}
		lib := &config.Library{
			Name: libraryName,
			APIs: []*config.API{
				{
					Path: apiPath,
				},
			},
		}

		dir := filepath.Dir(file)
		keep, err := parseKeep(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}
		if additionalKeeps, ok := libraryToKeep[libraryName]; ok {
			keep = append(keep, additionalKeeps...)
		}
		slices.Sort(keep)
		lib.Keep = keep
		if copyrightYear, ok := sidekick.Codec["copyright-year"]; ok && copyrightYear != "" {
			lib.CopyrightYear = copyrightYear
		}
		if descriptionOverride, ok := sidekick.Source["description-override"]; ok && descriptionOverride != "" {
			lib.DescriptionOverride = descriptionOverride
		}
		relativePath, err := filepath.Rel(repoPath, dir)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate relative path: %w", errUnableToCalculateOutputPath)
		}
		lib.Output = relativePath
		if _, ok := sidekick.Codec["not-for-publication"]; ok {
			lib.SkipPublish = true
		}
		// in Dart .sidekick.toml, the protobuf source root is protobuf, while in Rust is protobuf-src.
		// Convert protobuf to protobuf-src to reuse parsing logic in sidekick tool.
		if roots, ok := sidekick.Source["roots"]; ok && roots != "" {
			lib.Roots = strings.Split(roots, ",")
			for i, root := range lib.Roots {
				if root == "protobuf" {
					lib.Roots[i] = "protobuf-src"
				}
			}
		}
		lib.SpecificationFormat = specificationFormat

		dartPackage := &config.DartPackage{}
		if titleOverride, ok := sidekick.Source["title-override"]; ok && titleOverride != "" {
			dartPackage.TitleOverride = titleOverride
		}
		if nameOverride, ok := sidekick.Source["name-override"]; ok && nameOverride != "" {
			dartPackage.NameOverride = nameOverride
		}

		if apiKeys, ok := sidekick.Codec["api-keys-environment-variables"]; ok && apiKeys != "" {
			dartPackage.APIKeysEnvironmentVariables = apiKeys
		}
		if deps, ok := sidekick.Codec["dependencies"]; ok && deps != "" {
			dartPackage.Dependencies = deps
		}
		if devDeps, ok := sidekick.Codec["dev-dependencies"]; ok && devDeps != "" {
			dartPackage.DevDependencies = devDeps
		}
		if extraImports, ok := sidekick.Codec["extra-imports"]; ok && extraImports != "" {
			dartPackage.ExtraImports = extraImports
		}
		if partFile, ok := sidekick.Codec["part-file"]; ok && partFile != "" {
			// part-file in .sidekick.toml starts with src/, however, the file path is
			// actually {output}/lib/src/*.
			path := filepath.Join(dir, "lib", partFile)
			if _, err := os.Stat(path); err != nil {
				return nil, fmt.Errorf("failed to stat %s: %w", path, err)
			}
			partFileRelPath, err := filepath.Rel(dir, path)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate relative path: %w", errUnableToCalculateOutputPath)
			}
			lib.Keep = append(lib.Keep, partFileRelPath)
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
		prefix := parseKeyWithPrefix(sidekick.Codec, "prefix:")
		if len(prefix) != 0 {
			dartPackage.Prefixes = prefix
		}

		if !isEmptyDartPackage(dartPackage) {
			lib.Dart = dartPackage
		}

		libraries = append(libraries, lib)
	}

	sort.Slice(libraries, func(i, j int) bool {
		return libraries[i].Name < libraries[j].Name
	})

	return libraries, nil
}

// buildConfig builds the complete config from libraries.
func buildConfig(libraries []*config.Library, defaults *config.Config) *config.Config {
	defaults.Libraries = libraries
	return defaults
}

func genLibraryName(path string) string {
	path = strings.TrimPrefix(path, "google/cloud/")
	path = strings.TrimPrefix(path, "google/")
	return "google_cloud_" + strings.ReplaceAll(path, "/", "_")
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

func isEmptyDartPackage(r *config.DartPackage) bool {
	return reflect.DeepEqual(r, &config.DartPackage{})
}

// parseKeep generate files that should keep before library generation.
// Only files inside example/ and test/ directories are considered, if existed.
func parseKeep(base string) ([]string, error) {
	var res []string
	for _, sub := range []string{"example", "test"} {
		subDir := filepath.Join(base, sub)
		err := filepath.WalkDir(subDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}

			if !d.IsDir() {
				relativePath, err := filepath.Rel(base, path)
				if err != nil {
					return err
				}
				res = append(res, relativePath)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
