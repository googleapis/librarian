// Copyright 2024 Google LLC
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

// Package migrate_sidekick provides tools to create a librarian configuration from .sidekick.toml
// in a repository.
package migrate_sidekick

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/pelletier/go-toml/v2"
	yamlv3 "gopkg.in/yaml.v3"
)

const (
	sidekickFile = ".sidekick.toml"
)

var (
	// gitDir is the directory stores git configuration.
	// Define it as a variable, rather than a const because we don't want to
	// commit a .git directory. The directory is overwritten in the tests.
	gitDir              = ".git"
	errBranchNotFound   = errors.New("branch not found")
	errHeadNotFound     = errors.New("HEAD not found")
	errSidekickNotFound = errors.New(".sidekick.toml not found")
	errSrcNotFound      = errors.New("src/generated directory not found")
)

// RootSidekickConfig represents the structure of the root .sidekick.toml file.
type RootSidekickConfig struct {
	Codec struct {
		DisabledRustdocWarnings string            `toml:"disabled-rustdoc-warnings"`
		Packages                map[string]string `toml:",remain"`
	} `toml:"codec"`
	Release struct {
		Remote string `toml:"remote"`
		Branch string `toml:"branch"`
	} `toml:"release"`
}

// SidekickConfig represents the structure of a .sidekick.toml file.
type SidekickConfig struct {
	General struct {
		SpecificationSource string `toml:"specification-source"`
		ServiceConfig       string `toml:"service-config"`
	} `toml:"general"`
	Codec               map[string]interface{} `toml:"codec"` // Use map to capture all fields including package:*
	PaginationOverrides []struct {
		ID        string `toml:"id"`
		ItemField string `toml:"item-field"`
	} `toml:"pagination-overrides"`
}

// RootDefaults contains defaults extracted from root .sidekick.toml.
type RootDefaults struct {
	DisabledRustdocWarnings []string
	PackageDependencies     []*config.RustPackageDependency
	Remote                  string
	Branch                  string
}

// CargoConfig represents relevant fields from Cargo.toml.
type CargoConfig struct {
	Package struct {
		Name    string      `toml:"name"`
		Version string      `toml:"version"`
		Publish interface{} `toml:"publish"` // Can be bool or array of strings
	} `toml:"package"`
}

func main() {
	if err := run(); err != nil {
		slog.Error("migrate-sidekick command failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		repoPath       string
		outputPath     string
		googleapisPath string
	)

	flag.StringVar(&repoPath, "repo", "", "Path to the google-cloud-rust repository (required)")
	flag.StringVar(&outputPath, "output", "", "Output file path (default: stdout)")
	flag.StringVar(&googleapisPath, "googleapis", "", "Path to googleapis repository")
	flag.Parse()

	if repoPath == "" {
		return fmt.Errorf("-repo flag is required")
	}

	slog.Info("Reading sidekick.toml...", "path", repoPath)

	// Read root .sidekick.toml for defaults
	rootDefaults, err := readRootSidekick(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read root .sidekick.toml: %w", err)
	}

	// Find all .sidekick.toml files
	sidekickFiles, err := findSidekickFiles(repoPath)
	if err != nil {
		return fmt.Errorf("failed to find sidekick.toml files: %w", err)
	}

	// Read all sidekick.toml files
	libraries, err := readSidekickFiles(sidekickFiles)
	if err != nil {
		return fmt.Errorf("failed to read sidekick.toml files: %w", err)
	}

	// Build config
	cfg := buildConfig(libraries, googleapisPath, rootDefaults)

	// Write output
	if outputPath == "" {
		// Write to stdout
		enc := yamlv3.NewEncoder(os.Stdout)
		enc.SetIndent(2)
		defer enc.Close()

		if err := enc.Encode(cfg); err != nil {
			return fmt.Errorf("failed to encode config: %w", err)
		}
	} else {
		if err := yaml.Write(outputPath, cfg); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
		slog.Info("Wrote config to output file", "path", outputPath)
	}

	return nil
}

// readRootSidekick reads the root .sidekick.toml file and extracts defaults.
func readRootSidekick(repoPath string) (*RootDefaults, error) {
	rootPath := filepath.Join(repoPath, sidekickFile)
	data, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, errSidekickNotFound
	}

	// Parse as generic map to handle the dynamic package keys
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	defaults := &RootDefaults{}

	// Extract codec section
	if codec, ok := raw["codec"].(map[string]interface{}); ok {
		// Parse disabled warnings
		if warnings, ok := codec["disabled-rustdoc-warnings"].(string); ok {
			defaults.DisabledRustdocWarnings = strings.Split(warnings, ",")
		}

		// Parse package dependencies
		for key, value := range codec {
			if !strings.HasPrefix(key, "package:") {
				continue
			}
			pkgName := strings.TrimPrefix(key, "package:")
			pkgSpec := value.(string)

			dep := parsePackageDependency(pkgName, pkgSpec)
			if dep != nil {
				defaults.PackageDependencies = append(defaults.PackageDependencies, dep)
			}
		}
	}

	// Extract release section
	if release, ok := raw["release"].(map[string]interface{}); ok {
		if remote, ok := release["remote"].(string); ok {
			defaults.Remote = remote
		}
		if branch, ok := release["branch"].(string); ok {
			defaults.Branch = branch
		}
	}

	// Sort package dependencies by name
	sort.Slice(defaults.PackageDependencies, func(i, j int) bool {
		return defaults.PackageDependencies[i].Name < defaults.PackageDependencies[j].Name
	})

	return defaults, nil
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
		}
	}

	return dep
}

// findSidekickFiles finds all .sidekick.toml files in the repository.
func findSidekickFiles(repoPath string) ([]string, error) {
	var files []string

	generatedPath := filepath.Join(repoPath, "src", "generated")
	err := filepath.Walk(generatedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errSrcNotFound
		}
		if !info.IsDir() && info.Name() == sidekickFile {
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

// readSidekickFiles reads all sidekick.toml files and extracts library information.
func readSidekickFiles(files []string) (map[string]*config.Library, error) {
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
		// Get Service config
		serviceConfig := sidekick.General.ServiceConfig

		// Read Cargo.toml in the same directory to get the actual library name
		dir := filepath.Dir(file)
		cargoPath := filepath.Join(dir, "Cargo.toml")
		cargoData, err := os.ReadFile(cargoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", cargoPath, err)
		}

		var cargo CargoConfig
		if err := toml.Unmarshal(cargoData, &cargo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", cargoPath, err)
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

		// Parse Rust-specific configuration from sidekick.toml codec section
		perServiceFeatures, _ := sidekick.Codec["per-service-features"].(string)
		disabledRustdocWarnings, _ := sidekick.Codec["disabled-rustdoc-warnings"].(string)
		generateSetterSamples, _ := sidekick.Codec["generate-setter-samples"].(string)
		nameOverrides, _ := sidekick.Codec["name-overrides"].(string)

		// Parse package dependencies
		var packageDeps []*config.RustPackageDependency
		for key, value := range sidekick.Codec {
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

		// Sort package dependencies by name for consistent output
		sort.Slice(packageDeps, func(i, j int) bool {
			return packageDeps[i].Name < packageDeps[j].Name
		})

		// Parse pagination overrides
		var paginationOverrides []config.RustPaginationOverride
		for _, po := range sidekick.PaginationOverrides {
			paginationOverrides = append(paginationOverrides, config.RustPaginationOverride{
				ID:        po.ID,
				ItemField: po.ItemField,
			})
		}

		// Set Rust-specific configuration
		if perServiceFeatures != "" || disabledRustdocWarnings != "" || len(packageDeps) > 0 ||
			generateSetterSamples != "" || len(paginationOverrides) > 0 || nameOverrides != "" {
			if lib.Rust == nil {
				lib.Rust = &config.RustCrate{}
			}

			// Per-service features
			if perServiceFeatures == "true" {
				lib.Rust.PerServiceFeatures = true
			}

			// Disabled rustdoc warnings (comma-separated string)
			if disabledRustdocWarnings != "" {
				lib.Rust.DisabledRustdocWarnings = strings.Split(disabledRustdocWarnings, ",")
			}

			// Package dependencies
			if len(packageDeps) > 0 {
				lib.Rust.PackageDependencies = packageDeps
			}

			// Generate setter samples
			if generateSetterSamples == "true" {
				lib.Rust.GenerateSetterSamples = true
			}

			// Pagination overrides
			if len(paginationOverrides) > 0 {
				lib.Rust.PaginationOverrides = paginationOverrides
			}

			// Name overrides (codec-level, for renaming types/services)
			if nameOverrides != "" {
				lib.Rust.NameOverrides = nameOverrides
			}
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
func buildConfig(libraries map[string]*config.Library, googleapisPath string, rootDefaults *RootDefaults) *config.Config {
	cfg := &config.Config{
		Language: "rust",
		Default: &config.Default{
			Output: "src/generated/",
			Rust: &config.RustDefault{
				DisabledRustdocWarnings: rootDefaults.DisabledRustdocWarnings,
				PackageDependencies:     rootDefaults.PackageDependencies,
			},
		},
	}

	// Add googleapis source if provided
	if googleapisPath != "" {
		// Try to get the current commit
		commit, err := getGitCommit(googleapisPath)
		if err == nil {
			cfg.Sources = &config.Sources{
				Googleapis: &config.Source{
					Commit: commit,
				},
			}
		}
	}

	// Convert libraries map to sorted slice, applying new schema logic
	var libList []*config.Library

	for _, lib := range libraries {
		// Get the API path for this library
		apiPath := ""
		if len(lib.Channels) > 0 {
			apiPath = lib.Channels[0].Path
		}
		
		// Derive expected library name from API path
		expectedName := deriveLibraryName(apiPath)
		nameMatchesConvention := lib.Name == expectedName
		// Check if library has extra configuration beyond just name/api/version
		hasExtraConfig := lib.CopyrightYear != "" ||
			(lib.Rust != nil && (lib.Rust.PerServiceFeatures || len(lib.Rust.DisabledRustdocWarnings) > 0 ||
				len(lib.Rust.PackageDependencies) > 0 || lib.Rust.GenerateSetterSamples ||
				len(lib.Rust.PaginationOverrides) > 0 || lib.Rust.NameOverrides != ""))
		// Only include in libraries section if:
		// 1. Name doesn't match expected naming convention (name override)
		// 2. Library has extra configuration
		// 3. Library spans multiple APIs
		if !nameMatchesConvention || hasExtraConfig || len(lib.Channels) > 1 {
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

// getGitCommit gets the current git commit hash from a repository.
func getGitCommit(repoPath string) (string, error) {
	// Read .git/HEAD to get current commit
	headPath := filepath.Join(repoPath, gitDir, "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", errHeadNotFound
	}

	head := strings.TrimSpace(string(data))
	if strings.HasPrefix(head, "ref: ") {
		// HEAD points to a branch
		refPath := strings.TrimPrefix(head, "ref: ")
		refFullPath := filepath.Join(repoPath, gitDir, refPath)
		commitData, err := os.ReadFile(refFullPath)
		if err != nil {
			return "", errBranchNotFound
		}
		return strings.TrimSpace(string(commitData)), nil
	}

	// HEAD is a direct commit hash
	return head, nil
}
