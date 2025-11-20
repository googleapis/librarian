// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/sideflip/config"
	"github.com/pelletier/go-toml/v2"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var repoPath string

	flag.StringVar(&repoPath, "repo", "", "Path to the google-cloud-rust repository (optional)")
	flag.Parse()

	if flag.NArg() != 1 {
		return fmt.Errorf("usage: sidekick-migrate [flags] <output-file>")
	}
	outputPath := flag.Arg(0)

	var err error
	if repoPath == "" {
		repoPath, err = downloadAndExtractTarball("googleapis/google-cloud-rust", "main")
		if err != nil {
			return fmt.Errorf("failed to download google-cloud-rust: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "Reading sidekick.toml files from %s...\n", repoPath)

	rootDefaults, err := readRootSidekick(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read root .sidekick.toml: %w", err)
	}

	if rootDefaults.GoogleapisCommit != "" {
		_, err = downloadAndExtractTarball("googleapis/googleapis", rootDefaults.GoogleapisCommit)
		if err != nil {
			return fmt.Errorf("failed to download googleapis: %w", err)
		}
	}

	sidekickFiles, err := findSidekickFiles(repoPath)
	if err != nil {
		return fmt.Errorf("failed to find sidekick.toml files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d sidekick.toml files\n", len(sidekickFiles))

	libraries, err := readSidekickFiles(sidekickFiles)
	if err != nil {
		return fmt.Errorf("failed to read sidekick files: %w", err)
	}

	cfg := buildConfig(libraries, rootDefaults)

	if err := cfg.Write(outputPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Wrote config to %s\n", outputPath)

	return nil
}

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
	GoogleapisCommit        string
	DiscoveryCommit         string
}

// readRootSidekick reads the root .sidekick.toml file and extracts defaults.
func readRootSidekick(repoPath string) (*RootDefaults, error) {
	rootPath := filepath.Join(repoPath, ".sidekick.toml")
	data, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, err
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

	// Extract source section
	if source, ok := raw["source"].(map[string]interface{}); ok {
		if commit, ok := source["googleapis"].(string); ok {
			defaults.GoogleapisCommit = commit
		}
		if commit, ok := source["discovery"].(string); ok {
			defaults.DiscoveryCommit = commit
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
			return err
		}
		if !info.IsDir() && info.Name() == ".sidekick.toml" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// CargoConfig represents relevant fields from Cargo.toml.
type CargoConfig struct {
	Package struct {
		Name    string      `toml:"name"`
		Version string      `toml:"version"`
		Publish interface{} `toml:"publish"` // Can be bool or array of strings
	} `toml:"package"`
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

		// Add API path
		if lib.Channel == "" && len(lib.Channels) == 0 {
			lib.Channel = apiPath
		} else if lib.Channel != "" && lib.Channel != apiPath {
			// Convert to multi-API library
			lib.Channels = []string{lib.Channel, apiPath}
			lib.Channel = ""
		} else if len(lib.Channels) > 0 && !contains(lib.Channels, apiPath) {
			lib.Channels = append(lib.Channels, apiPath)
		}

		// Set version from Cargo.toml (more authoritative than sidekick)
		if cargo.Package.Version != "" {
			lib.Version = cargo.Package.Version
		} else if version, ok := sidekick.Codec["version"].(string); ok && lib.Version == "" {
			lib.Version = version
		}

		// Set publish disabled from Cargo.toml
		if publishValue, ok := cargo.Package.Publish.(bool); ok && !publishValue {
			if lib.Publish == nil {
				lib.Publish = &config.LibraryPublish{}
			}
			lib.Publish.Disabled = true
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
		var packageDeps []config.RustPackageDependency
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
				packageDeps = append(packageDeps, *dep)
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
// For Rust: google/cloud/secretmanager/v1 -> google-cloud-secretmanager-v1.
func deriveLibraryName(apiPath string) string {
	return strings.ReplaceAll(apiPath, "/", "-")
}

// buildConfig builds the complete config from libraries.
func buildConfig(libraries map[string]*config.Library, rootDefaults *RootDefaults) *config.Config {
	cfg := &config.Config{
		Version:  "v1",
		Language: "rust",
		Default: &config.Default{
			Output: "src/generated/",
			Generate: &config.DefaultGenerate{
				Auto:          true,
				OneLibraryPer: "channel",
				ReleaseLevel:  "stable",
			},
			Release: &config.DefaultRelease{
				TagFormat: "{name}/v{version}",
				Remote:    rootDefaults.Remote,
				Branch:    rootDefaults.Branch,
			},
			Rust: &config.RustDefault{
				DisabledRustdocWarnings: rootDefaults.DisabledRustdocWarnings,
				PackageDependencies:     rootDefaults.PackageDependencies,
			},
		},
	}

	// Add sources from root .sidekick.toml
	if rootDefaults.GoogleapisCommit != "" {
		cfg.Sources = &config.Sources{
			Googleapis: &config.Source{
				Commit: rootDefaults.GoogleapisCommit,
			},
		}
	}
	if rootDefaults.DiscoveryCommit != "" {
		if cfg.Sources == nil {
			cfg.Sources = &config.Sources{}
		}
		cfg.Sources.Discovery = &config.Source{
			Commit: rootDefaults.DiscoveryCommit,
		}
	}

	// Convert libraries map to sorted slice, applying new schema logic
	var libList []*config.Library
	versions := make(map[string]string)

	for name, lib := range libraries {
		// Track versions for ALL libraries
		if lib.Version != "" {
			versions[name] = lib.Version
		}

		// Get the API path for this library
		apiPath := lib.Channel
		if len(lib.Channels) > 0 {
			apiPath = lib.Channels[0]
		}

		// Derive expected library name from API path
		// For Rust with one_library_per: channel, expected name is the API path with / replaced by -
		expectedName := deriveLibraryName(apiPath)

		// Check if library has extra configuration beyond just name/api/version
		hasExtraConfig := lib.Generate != nil || lib.Publish != nil || lib.CopyrightYear != "" ||
			(lib.Rust != nil && (lib.Rust.PerServiceFeatures || len(lib.Rust.DisabledRustdocWarnings) > 0 ||
				len(lib.Rust.PackageDependencies) > 0 || lib.Rust.GenerateSetterSamples ||
				len(lib.Rust.PaginationOverrides) > 0 || lib.Rust.NameOverrides != ""))

		// Only include in libraries section if:
		// 1. Name doesn't match expected naming convention (name override)
		// 2. Library has extra configuration
		// 3. Library spans multiple APIs
		nameMatchesConvention := lib.Name == expectedName

		if !nameMatchesConvention || hasExtraConfig || len(lib.Channels) > 1 {
			// Clear version from library (it goes in versions section)
			libCopy := *lib
			libCopy.Version = ""
			libList = append(libList, &libCopy)
		}
	}

	// Sort libraries by name
	sort.Slice(libList, func(i, j int) bool {
		return libList[i].Name < libList[j].Name
	})

	cfg.Libraries = libList
	cfg.Versions = versions

	return cfg
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// downloadAndExtractTarball downloads a GitHub tarball and extracts it to the cache.
func downloadAndExtractTarball(repo, commit string) (string, error) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".librarian", "cache", repo+"@"+commit)

	if _, err := os.Stat(cacheDir); err == nil {
		return cacheDir, nil
	}

	url := fmt.Sprintf("https://github.com/%s/archive/%s.tar.gz", repo, commit)
	fmt.Fprintf(os.Stderr, "Downloading %s...\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download tarball: HTTP %d", resp.StatusCode)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	tmpDir := cacheDir + ".tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	var rootDir string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to read tar: %w", err)
		}

		if rootDir == "" && header.Typeflag == tar.TypeDir {
			rootDir = header.Name
		}

		target := filepath.Join(tmpDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				os.RemoveAll(tmpDir)
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				os.RemoveAll(tmpDir)
				return "", fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				os.RemoveAll(tmpDir)
				return "", fmt.Errorf("failed to write file: %w", err)
			}
			f.Close()
		}
	}

	extractedRoot := filepath.Join(tmpDir, rootDir)
	if err := os.Rename(extractedRoot, cacheDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to move extracted directory: %w", err)
	}

	os.RemoveAll(tmpDir)
	fmt.Fprintf(os.Stderr, "Extracted to %s\n", cacheDir)
	return cacheDir, nil
}
