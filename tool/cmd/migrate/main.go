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
	"github.com/pelletier/go-toml/v2"
)

const (
	sidekickFile             = ".sidekick.toml"
	cargoFile                = "Cargo.toml"
	discoveryArchivePrefix   = "https://github.com/googleapis/discovery-artifact-manager/archive/"
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
)

var excludedVeneerLibraries = map[string]struct{}{
	"echo-server": {},
	"gcp-sdk":     {},
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

	sidekickFiles, err := findSidekickFiles(filepath.Join(repoPath, "src", "generated"))
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

// parsePackageDependency parses a package dependency spec.
// Format: "package=name,source=path,force-used=true,used-if=condition".
func parsePackageDependency(name, spec string) *config.RustPackageDependency {
	dep := &config.RustPackageDependency{
		Name: name,
	}

	parts := strings.Split(spec, ",")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}
		key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])

		switch key {
		case "package":
			dep.Package = value
		case "source":
			dep.Source = value
		case "force-used":
			dep.ForceUsed = value == "true"
		case "used-if":
			dep.UsedIf = value
		case "feature":
			dep.Feature = value
		case "ignore":
			dep.Ignore = value == "true"
		}
	}

	return dep
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
		if specificationFormat == "disco" {
			specificationFormat = "discovery"
		}

		// Read Cargo.toml in the same directory to get the actual library name
		dir := filepath.Dir(file)
		cargo, err := readCargoConfig(dir)
		if err != nil {
			return nil, err
		}

		libraryName := cargo.Package.Name
		if libraryName == "" {
			continue
		}

		// Create or update library
		lib, exists := libraries[libraryName]
		if !exists {
			lib = &config.Library{
				Name: libraryName,
			}
			libraries[libraryName] = lib
		}
		lib.SpecificationFormat = specificationFormat
		relativePath, err := filepath.Rel(repoPath, dir)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate relative path: %w", errUnableToCalculateOutputPath)
		}
		lib.Output = relativePath
		lib.APIs = append(lib.APIs, &config.API{
			Path: apiPath,
		})

		// Set version from Cargo.toml (more authoritative than sidekick)
		if cargo.Package.Version != "" && cargo.Package.Version != "0.0.0" {
			lib.Version = cargo.Package.Version
		} else if version, ok := sidekick.Codec["version"]; ok && lib.Version == "" && version != "0.0.0" {
			lib.Version = version
		}

		// Set publish disabled from Cargo.toml
		if !cargo.Package.Publish {
			lib.SkipPublish = true
		}

		// Parse library-level configuration
		if copyrightYear, ok := sidekick.Codec["copyright-year"]; ok && copyrightYear != "" {
			lib.CopyrightYear = copyrightYear
		}

		if extraModules, ok := sidekick.Codec["extra-modules"]; ok {
			for _, module := range strToSlice(extraModules, false) {
				if module == "" {
					continue
				}
				lib.Keep = append(lib.Keep, fmt.Sprintf("src/%s.rs", module))
			}
		}

		// Parse Rust-specific configuration from .sidekick.toml source section
		if descriptionOverride, ok := sidekick.Source["description-override"]; ok {
			lib.DescriptionOverride = descriptionOverride
		}

		includeList := sidekick.Source["include-list"]
		includeIds := sidekick.Source["include-ids"]
		skippedIds := sidekick.Source["skipped-ids"]
		roots := sidekick.Source["roots"]

		// Parse Rust-specific configuration from sidekick.toml codec section
		disabledRustdocWarnings := sidekick.Codec["disabled-rustdoc-warnings"]
		perServiceFeatures := sidekick.Codec["per-service-features"]
		modulePath := sidekick.Codec["module-path"]
		templateOverride := sidekick.Codec["template-override"]
		packageNameOverride := sidekick.Codec["package-name-override"]
		rootName := sidekick.Codec["root-name"]
		defaultFeatures := sidekick.Codec["default-features"]
		disabledClippyWarnings := sidekick.Codec["disabled-clippy-warnings"]
		hasVeneer := sidekick.Codec["has-veneer"]
		routingRequired := sidekick.Codec["routing-required"]
		includeGrpcOnlyMethods := sidekick.Codec["include-grpc-only-methods"]
		generateSetterSamples := sidekick.Codec["generate-setter-samples"]
		generateRpcSamples := sidekick.Codec["generate-rpc-samples"]
		postProcessProtos := sidekick.Codec["post-process-protos"]
		detailedTracingAttributes := sidekick.Codec["detailed-tracing-attributes"]
		nameOverrides := sidekick.Codec["name-overrides"]

		// Parse package dependencies
		packageDeps := parsePackageDependencies(sidekick.Codec)

		// Parse documentation overrides
		var documentationOverrides []config.RustDocumentationOverride
		for _, do := range sidekick.CommentOverrides {
			if strings.HasPrefix(do.Replace, "\n") {
				// this ensures that newline is preserved in yaml format
				do.Replace = " " + do.Replace
			}
			documentationOverrides = append(documentationOverrides, config.RustDocumentationOverride{
				ID:      do.ID,
				Match:   do.Match,
				Replace: do.Replace,
			})
		}

		// Parse pagination overrides
		var paginationOverrides []config.RustPaginationOverride
		for _, po := range sidekick.PaginationOverrides {
			paginationOverrides = append(paginationOverrides, config.RustPaginationOverride{
				ID:        po.ID,
				ItemField: po.ItemField,
			})
		}

		// Set Rust-specific configuration only if there's actual config
		rustCrate := &config.RustCrate{
			RustDefault: config.RustDefault{
				PackageDependencies:     packageDeps,
				DisabledRustdocWarnings: strToSlice(disabledRustdocWarnings, false),
				GenerateSetterSamples:   generateSetterSamples,
				GenerateRpcSamples:      generateRpcSamples,
			},
			PerServiceFeatures:        strToBool(perServiceFeatures),
			ModulePath:                modulePath,
			TemplateOverride:          templateOverride,
			PackageNameOverride:       packageNameOverride,
			RootName:                  rootName,
			Roots:                     strToSlice(roots, false),
			DefaultFeatures:           strToSlice(defaultFeatures, false),
			IncludeList:               strToSlice(includeList, false),
			IncludedIds:               strToSlice(includeIds, false),
			SkippedIds:                strToSlice(skippedIds, false),
			DisabledClippyWarnings:    strToSlice(disabledClippyWarnings, false),
			HasVeneer:                 strToBool(hasVeneer),
			RoutingRequired:           strToBool(routingRequired),
			IncludeGrpcOnlyMethods:    strToBool(includeGrpcOnlyMethods),
			PostProcessProtos:         postProcessProtos,
			DetailedTracingAttributes: strToBool(detailedTracingAttributes),
			DocumentationOverrides:    documentationOverrides,
			PaginationOverrides:       paginationOverrides,
			NameOverrides:             nameOverrides,
		}

		if sidekick.Discovery != nil {
			if lib.Rust == nil {
				lib.Rust = &config.RustCrate{}
			}
			pollers := make([]config.RustPoller, len(sidekick.Discovery.Pollers))
			for i, p := range sidekick.Discovery.Pollers {
				pollers[i] = config.RustPoller{
					Prefix:   p.Prefix,
					MethodID: p.MethodID,
				}
			}
			rustCrate.Discovery = &config.RustDiscovery{
				OperationID: sidekick.Discovery.OperationID,
				Pollers:     pollers,
			}
		}

		if !isEmptyRustCrate(rustCrate) {
			lib.Rust = rustCrate
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

// findCargos returns all Cargo.toml files within the given path.
//
// A file is filtered if the file lives in a path that contains src/generated.
func findCargos(path string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && strings.Contains(path, "src/generated") {
			return filepath.SkipDir
		}

		if d.IsDir() || d.Name() != cargoFile {
			return nil
		}

		files = append(files, path)

		return nil
	})
	return files, err
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

func parsePackageDependencies(codec map[string]string) []*config.RustPackageDependency {
	var packageDeps []*config.RustPackageDependency
	for key, value := range codec {
		if !strings.HasPrefix(key, "package:") {
			continue
		}
		pkgName := strings.TrimPrefix(key, "package:")

		dep := parsePackageDependency(pkgName, value)
		if dep != nil {
			packageDeps = append(packageDeps, dep)
		}
	}

	// Sort package dependencies by name
	sort.Slice(packageDeps, func(i, j int) bool {
		return packageDeps[i].Name < packageDeps[j].Name
	})

	return packageDeps
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

func isEmptyRustCrate(r *config.RustCrate) bool {
	return reflect.DeepEqual(r, &config.RustCrate{})
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
