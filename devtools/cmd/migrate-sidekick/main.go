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

// Command migrate-sidekick is a tool for migrating .sidekick.toml to librarian configuration.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
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
)

var (
	errRepoNotFound                = errors.New("-repo flag is required")
	errSidekickNotFound            = errors.New(".sidekick.toml not found")
	errTidyFailed                  = errors.New("librarian tidy failed")
	errUnableToCalculateOutputPath = errors.New("unable to calculate output path")
)

var excludedVeneerLibraries = map[string]struct{}{
	"echo-server": {},
	"gcp-sdk":     {},
}

// SidekickConfig represents the structure of a .sidekick.toml file.
type SidekickConfig struct {
	General struct {
		SpecificationSource string `toml:"specification-source"`
		ServiceConfig       string `toml:"service-config"`
		SpecificationFormat string `toml:"specification-format"`
	} `toml:"general"`
	Source                 map[string]interface{} `toml:"source"`
	Codec                  map[string]interface{} `toml:"codec"`
	DocumentationOverrides []struct {
		ID      string `toml:"id"`
		Match   string `toml:"match"`
		Replace string `toml:"replace"`
	} `toml:"documentation-overrides"`
	PaginationOverrides []struct {
		ID        string `toml:"id"`
		ItemField string `toml:"item-field"`
	} `toml:"pagination-overrides"`
}

// CargoConfig represents relevant fields from Cargo.toml.
type CargoConfig struct {
	Package struct {
		Name    string      `toml:"name"`
		Version string      `toml:"version"`
		Publish interface{} `toml:"publish"` // Can be bool or array of strings
	} `toml:"package"`
}

// RustOverrideData is used to store RustDocumentationOverride data to fix newlines.
type RustOverrideData struct {
	Match   string
	Replace string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migrate-sidekick failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flagSet := flag.NewFlagSet("migrate-sidekick", flag.ContinueOnError)
	outputPath := flagSet.String("output", "./librarian.yaml", "Output file path (default: ./librarian.yaml)")
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if flagSet.NArg() < 1 {
		return errRepoNotFound
	}
	repoPath := flagSet.Arg(0)

	// Read root .sidekick.toml for defaults
	defaults, err := readRootSidekick(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read root .sidekick.toml: %w", err)
	}

	// Find all .sidekick.toml files for GAPIC libraries.
	sidekickFiles, err := findSidekickFiles(filepath.Join(repoPath, "src", "generated"))
	if err != nil {
		return fmt.Errorf("failed to find sidekick.toml files: %w", err)
	}

	libraries, err := buildGAPIC(sidekickFiles, repoPath)
	if err != nil {
		return fmt.Errorf("failed to read sidekick.toml files: %w", err)
	}

	cargoFiles, err := findCargos(filepath.Join(repoPath, "src"))
	if err != nil {
		return fmt.Errorf("failed to find Cargo.toml files: %w", err)
	}

	veneers, err := buildVeneer(cargoFiles)
	if err != nil {
		return fmt.Errorf("failed to build veneers: %w", err)
	}

	allLibraries := make(map[string]*config.Library, len(libraries)+len(veneers))
	maps.Copy(allLibraries, libraries)
	maps.Copy(allLibraries, veneers)

	cfg := buildConfig(allLibraries, defaults)

	if err := yaml.Write(*outputPath, cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	slog.Info("Wrote config to output file", "path", *outputPath)

	if err := librarian.RunTidy(); err != nil {
		slog.Error(errTidyFailed.Error(), "error", err)
		return errTidyFailed
	}

	return fixDocumentOverrideNewLines(*outputPath, cfg)
}

// readRootSidekick reads the root .sidekick.toml file and extracts defaults.
func readRootSidekick(repoPath string) (*config.Config, error) {
	rootPath := filepath.Join(repoPath, sidekickFile)
	data, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, errSidekickNotFound
	}

	// Parse as generic map to handle the dynamic package keys
	var sidekick SidekickConfig
	if err := toml.Unmarshal(data, &sidekick); err != nil {
		return nil, err
	}

	releaseLevel, _ := sidekick.Codec["release-level"].(string)
	warnings, _ := sidekick.Codec["disabled-rustdoc-warnings"].(string)
	discoverySHA256, _ := sidekick.Source["discovery-sha256"].(string)
	discoveryRoot, _ := sidekick.Source["discovery-root"].(string)
	googleapisSHA256, _ := sidekick.Source["googleapis-sha256"].(string)
	googleapisRoot, _ := sidekick.Source["googleapis-root"].(string)
	showcaseRoot, _ := sidekick.Source["showcase-root"].(string)
	showcaseSHA256, _ := sidekick.Source["showcase-sha256"].(string)
	protobufRoot, _ := sidekick.Source["protobuf-src-root"].(string)
	protobufSHA256, _ := sidekick.Source["protobuf-src-sha256"].(string)
	protobufSubDir, _ := sidekick.Source["protobuf-src-subdir"].(string)
	conformanceRoot, _ := sidekick.Source["conformance-root"].(string)
	conformanceSHA256, _ := sidekick.Source["conformance-sha256"].(string)
	generateSetterSamples, _ := sidekick.Codec["generate-setter-samples"].(string)

	discoveryCommit := strings.TrimSuffix(strings.TrimPrefix(discoveryRoot, discoveryArchivePrefix), tarballSuffix)
	googleapisCommit := strings.TrimSuffix(strings.TrimPrefix(googleapisRoot, googleapisArchivePrefix), tarballSuffix)
	showcaseCommit := strings.TrimSuffix(strings.TrimPrefix(showcaseRoot, showcaseArchivePrefix), tarballSuffix)
	protobufCommit := strings.TrimSuffix(strings.TrimPrefix(protobufRoot, protobufArchivePrefix), tarballSuffix)
	conformanceCommit := strings.TrimSuffix(strings.TrimPrefix(conformanceRoot, conformanceArchivePrefix), tarballSuffix)

	// Parse package dependencies
	packageDependencies := parsePackageDependencies(sidekick.Codec)

	cfg := &config.Config{
		Language: "rust",
		Sources: &config.Sources{
			Discovery: &config.Source{
				Commit: discoveryCommit,
				SHA256: discoverySHA256,
			},
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
			Output:       "src/generated/",
			ReleaseLevel: releaseLevel,
			Rust: &config.RustDefault{
				PackageDependencies:     packageDependencies,
				DisabledRustdocWarnings: strToSlice(warnings, false),
				GenerateSetterSamples:   generateSetterSamples,
			},
		},
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

		var sidekick SidekickConfig
		if err := toml.Unmarshal(data, &sidekick); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", file, err)
		}

		// Get API path
		apiPath := sidekick.General.SpecificationSource
		if apiPath == "" {
			continue
		}

		serviceConfig := sidekick.General.ServiceConfig

		specificationFormat := sidekick.General.SpecificationFormat
		if specificationFormat == "disco" {
			specificationFormat = "discovery"
		}

		// Read Cargo.toml in the same directory to get the actual library name
		dir := filepath.Dir(file)
		cargo, err := readTOML[CargoConfig](filepath.Join(dir, cargoFile))
		if err != nil {
			return nil, fmt.Errorf("failed to read cargo: %w", err)
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

		// Add channels
		lib.Channels = append(lib.Channels, &config.Channel{
			Path:          apiPath,
			ServiceConfig: serviceConfig,
		})

		// Set version from Cargo.toml (more authoritative than sidekick)
		if cargo.Package.Version != "" {
			lib.Version = cargo.Package.Version
		} else if version, ok := sidekick.Codec["version"].(string); ok && lib.Version == "" {
			lib.Version = version
		}

		// Set publish disabled from Cargo.toml
		if publishValue, ok := cargo.Package.Publish.(bool); ok && !publishValue {
			lib.SkipPublish = true
		}

		// Parse library-level configuration
		if copyrightYear, ok := sidekick.Codec["copyright-year"].(string); ok && copyrightYear != "" {
			lib.CopyrightYear = copyrightYear
		}

		if extraModules, ok := sidekick.Codec["extra-modules"].(string); ok {
			for _, module := range strToSlice(extraModules, false) {
				if module == "" {
					continue
				}
				lib.Keep = append(lib.Keep, fmt.Sprintf("src/%s.rs", module))
			}
		}

		// Parse Rust-specific configuration from .sidekick.toml source section
		if descriptionOverride, ok := sidekick.Source["description-override"].(string); ok {
			lib.DescriptionOverride = descriptionOverride
		}

		titleOverride, _ := sidekick.Source["title-override"].(string)
		roots, _ := sidekick.Source["roots"].(string)
		lib.Roots = strToSlice(roots, false)

		includeList, _ := sidekick.Source["include-list"].(string)
		includeIds, _ := sidekick.Source["include-ids"].(string)
		skippedIds, _ := sidekick.Source["skipped-ids"].(string)

		// Parse Rust-specific configuration from sidekick.toml codec section
		disabledRustdocWarnings, _ := sidekick.Codec["disabled-rustdoc-warnings"].(string)
		perServiceFeatures, _ := sidekick.Codec["per-service-features"].(string)
		modulePath, _ := sidekick.Codec["module-path"].(string)
		templateOverride, _ := sidekick.Codec["template-override"].(string)
		packageNameOverride, _ := sidekick.Codec["package-name-override"].(string)
		rootName, _ := sidekick.Codec["root-name"].(string)
		defaultFeatures, _ := sidekick.Codec["default-features"].(string)
		disabledClippyWarnings, _ := sidekick.Codec["disabled-clippy-warnings"].(string)
		hasVeneer, _ := sidekick.Codec["has-veneer"].(string)
		routingRequired, _ := sidekick.Codec["routing-required"].(string)
		includeGrpcOnlyMethods, _ := sidekick.Codec["include-grpc-only-methods"].(string)
		generateSetterSamples, _ := sidekick.Codec["generate-setter-samples"].(string)
		generateRpcSamples, _ := sidekick.Codec["generate-rpc-samples"].(string)
		postProcessProtos, _ := sidekick.Codec["post-process-protos"].(string)
		detailedTracingAttributes, _ := sidekick.Codec["detailed-tracing-attributes"].(string)
		nameOverrides, _ := sidekick.Codec["name-overrides"].(string)

		// Parse package dependencies
		packageDeps := parsePackageDependencies(sidekick.Codec)

		// Parse documentation overrides
		var documentationOverrides []config.RustDocumentationOverride
		for _, do := range sidekick.DocumentationOverrides {
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
			},
			PerServiceFeatures:        strToBool(perServiceFeatures),
			ModulePath:                modulePath,
			TemplateOverride:          templateOverride,
			TitleOverride:             titleOverride,
			PackageNameOverride:       packageNameOverride,
			RootName:                  rootName,
			DefaultFeatures:           strToSlice(defaultFeatures, false),
			IncludeList:               strToSlice(includeList, false),
			IncludedIds:               strToSlice(includeIds, false),
			SkippedIds:                strToSlice(skippedIds, false),
			DisabledClippyWarnings:    strToSlice(disabledClippyWarnings, false),
			HasVeneer:                 strToBool(hasVeneer),
			RoutingRequired:           strToBool(routingRequired),
			IncludeGrpcOnlyMethods:    strToBool(includeGrpcOnlyMethods),
			GenerateRpcSamples:        strToBool(generateRpcSamples),
			PostProcessProtos:         postProcessProtos,
			DetailedTracingAttributes: strToBool(detailedTracingAttributes),
			DocumentationOverrides:    documentationOverrides,
			PaginationOverrides:       paginationOverrides,
			NameOverrides:             nameOverrides,
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

func buildVeneer(files []string) (map[string]*config.Library, error) {
	veneers := make(map[string]*config.Library)
	for _, file := range files {
		cargo, err := readTOML[CargoConfig](file)
		if err != nil {
			return nil, err
		}

		if _, ok := excludedVeneerLibraries[cargo.Package.Name]; ok {
			slog.Info("Excluding hardcoded veneer library", "name", cargo.Package.Name)
			continue
		}

		dir := filepath.Dir(file)
		rustModules, err := buildModules(dir)
		if err != nil {
			return nil, err
		}
		name := cargo.Package.Name
		veneers[name] = &config.Library{
			Name:          name,
			Veneer:        true,
			Output:        dir,
			Version:       cargo.Package.Version,
			CopyrightYear: "2025",
		}
		if len(rustModules) > 0 {
			veneers[name].Rust = &config.RustCrate{
				Modules: rustModules,
			}
		} else {
			veneers[name].SkipGenerate = true
		}
	}

	return veneers, nil
}

func buildModules(rootDir string) ([]*config.RustModule, error) {
	var modules []*config.RustModule
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || d.Name() != sidekickFile {
			return nil
		}

		if strings.Contains(path, "tests") {
			// Only use a .sidekick.toml in tests directory to represent a rust module if the following directory
			// exists:
			//
			// |-src
			// |  |-generated
			// |     |-.sidekick.toml
			// |-Cargo.toml
			// Only one pair of Cargo.toml and .sidekick.toml (within tests directory) is not comply this structure in
			// google-cloud-rust, which is wkt/Cargo.toml and src/wkt/tests/common/src/generated/.sidekick.toml.
			srcDir := strings.TrimSuffix(path, fmt.Sprintf("/src/generated/%s", sidekickFile))
			if rootDir != srcDir {
				return nil
			}
		}

		sidekick, err := readTOML[SidekickConfig](path)
		if err != nil {
			return err
		}

		includedIds, _ := sidekick.Source["included-ids"].(string)
		includeList, _ := sidekick.Source["include-list"].(string)
		skippedIds, _ := sidekick.Source["skipped-ids"].(string)
		titleOverride, _ := sidekick.Source["title-override"].(string)
		moduleRoots := make(map[string]string)
		roots, ok := sidekick.Source["roots"].(string)
		if ok {
			for _, root := range strings.Split(roots, ",") {
				root = fmt.Sprintf("%s-root", root)
				modPath, ok := sidekick.Source[root].(string)
				if ok {
					moduleRoots[root] = modPath
				}
			}
		}

		hasVeneer, _ := sidekick.Codec["has-veneer"].(string)
		includeGrpcOnlyMethods, _ := sidekick.Codec["include-grpc-only-methods"].(string)
		routingRequired, _ := sidekick.Codec["routing-required"].(string)
		extendGrpcTransport, _ := sidekick.Codec["extend-grpc-transport"].(string)
		modulePath, _ := sidekick.Codec["module-path"].(string)
		nameOverrides, _ := sidekick.Codec["name-overrides"].(string)
		postProcessProtos, _ := sidekick.Codec["post-process-protos"].(string)
		templateOverride, _ := sidekick.Codec["template-override"].(string)

		generateSetterSamples, ok := sidekick.Codec["generate-setter-samples"].(string)
		if !ok {
			generateSetterSamples = "true"
		}

		module := &config.RustModule{
			GenerateSetterSamples:  strToBool(generateSetterSamples),
			HasVeneer:              strToBool(hasVeneer),
			IncludedIds:            strToSlice(includedIds, false),
			IncludeGrpcOnlyMethods: strToBool(includeGrpcOnlyMethods),
			IncludeList:            includeList,
			ModulePath:             modulePath,
			NameOverrides:          nameOverrides,
			Output:                 filepath.Dir(path),
			PostProcessProtos:      postProcessProtos,
			RoutingRequired:        strToBool(routingRequired),
			ExtendGrpcTransport:    strToBool(extendGrpcTransport),
			ServiceConfig:          sidekick.General.ServiceConfig,
			SkippedIds:             strToSlice(skippedIds, false),
			Source:                 sidekick.General.SpecificationSource,
			Template:               strings.TrimPrefix(templateOverride, "templates/"),
			TitleOverride:          titleOverride,
		}

		if len(moduleRoots) > 0 {
			module.ModuleRoots = moduleRoots
		}

		disabledRustdocWarnings, ok := sidekick.Codec["disabled-rustdoc-warnings"].(string)
		if ok {
			module.DisabledRustdocWarnings = strToSlice(disabledRustdocWarnings, true)
		}

		modules = append(modules, module)

		return nil
	})

	return modules, err
}

// buildConfig builds the complete config from libraries.
func buildConfig(libraries map[string]*config.Library, defaults *config.Config) *config.Config {
	cfg := defaults
	// Convert libraries map to sorted slice, applying new schema logic
	var libList []*config.Library

	for _, lib := range libraries {
		// Get the API path for this library
		apiPath := ""
		serviceConfigDoesNotExist := false
		if len(lib.Channels) > 0 {
			apiPath = lib.Channels[0].Path
			for _, ch := range lib.Channels {
				if ch.ServiceConfig == "" {
					ch.ServiceConfigDoesNotExist = true
					serviceConfigDoesNotExist = true
				}
			}
		}

		// Derive expected library name from API path
		expectedName := deriveLibraryName(apiPath)
		nameMatchesConvention := lib.Name == expectedName
		// Check if library has extra configuration beyond just name/api/version
		hasExtraConfig := lib.CopyrightYear != "" ||
			(lib.Rust != nil && (lib.Rust.PerServiceFeatures || len(lib.Rust.DisabledRustdocWarnings) > 0 ||
				lib.Rust.GenerateSetterSamples != "" || lib.Rust.GenerateRpcSamples ||
				len(lib.Rust.PackageDependencies) > 0 || len(lib.Rust.PaginationOverrides) > 0 ||
				lib.Rust.NameOverrides != ""))
		// Only include in libraries section if specific data needs to be retained
		if !nameMatchesConvention || hasExtraConfig || len(lib.Channels) > 1 || serviceConfigDoesNotExist {
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

func parsePackageDependencies(codec map[string]interface{}) []*config.RustPackageDependency {
	var packageDeps []*config.RustPackageDependency
	for key, value := range codec {
		if !strings.HasPrefix(key, "package:") {
			continue
		}
		pkgName := strings.TrimPrefix(key, "package:")
		pkgSpec, ok := value.(string)
		if !ok {
			continue
		}

		dep := parsePackageDependency(pkgName, pkgSpec)
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

// fixDocumentOverrideNewLines takes any toml content in rust.DocumentationOverrides that contains
// newlines and appropriately adjusts them in the yaml file.
func fixDocumentOverrideNewLines(yamlFile string, config *config.Config) error {
	input, err := os.ReadFile(yamlFile)
	if err != nil {
		return err
	}

	content := string(input)
	lookup := make(map[string]RustOverrideData)
	for _, lib := range config.Libraries {
		if lib.Rust == nil {
			continue
		}
		for _, o := range lib.Rust.DocumentationOverrides {
			key := o.ID + "|" + strings.Trim(o.Match, " \n\r")
			lookup[key] = RustOverrideData{
				Match:   o.Match,
				Replace: o.Replace,
			}
		}
	}
	lines := strings.Split(content, "\n")
	var newLines []string
	var currentID string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- id: ") {
			currentID = strings.TrimPrefix(trimmed, "- id: ")
			newLines = append(newLines, line)
			continue
		}
		if strings.Contains(line, "match: ") {
			parts := strings.SplitN(line, "match: ", 2)
			indent := parts[0]
			currentVal := strings.Trim(parts[1], " '\"")

			lookupKey := currentID + "|" + currentVal

			if data, ok := lookup[lookupKey]; ok {
				if strings.Contains(data.Match, "\n") {
					newLines = append(newLines, fmt.Sprintf("%smatch: |", indent))
					newLines = append(newLines, "") // Leading newline
					newLines = append(newLines, indent+"  "+strings.TrimSpace(data.Match))
				} else {
					newLines = append(newLines, line) // Keep original if no newline
				}

				// --- HANDLE REPLACE FIELD ---
				// We assume the next line in the file is the 'replace' line
				// We skip the original replace line by incrementing the loop counter 'i'
				if i+1 < len(lines) && strings.Contains(lines[i+1], "replace: ") {
					i++ // Skip the original line
					if strings.Contains(data.Replace, "\n") {
						newLines = append(newLines, fmt.Sprintf("%sreplace: |", indent))
						newLines = append(newLines, "") // Leading newline
						newLines = append(newLines, indent+"  "+strings.TrimSpace(data.Replace))
					} else {
						newLines = append(newLines, lines[i])
					}
				}
				continue
			}
		}

		newLines = append(newLines, line)
	}

	content = strings.Join(newLines, "\n")
	err = os.WriteFile(yamlFile, []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}
