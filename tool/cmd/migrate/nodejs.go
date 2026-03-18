// Copyright 2026 Google LLC
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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

// nodejsPackageJSON represents the fields we need from a Node.js package.json.
type nodejsPackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

// owlBotYAML represents the fields we need from an .OwlBot.yaml.
type owlBotYAML struct {
	DeepCopyRegex []owlBotCopyRule `yaml:"deep-copy-regex"`
}

// owlBotCopyRule represents a copy rule in .OwlBot.yaml.
type owlBotCopyRule struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
}

// nodejsGapicInfo contains information from the nodejs_gapic_library rule.
type nodejsGapicInfo struct {
	packageName           string
	bundleConfig          string
	extraProtocParameters []string
	handwrittenLayer      bool
	mainService           string
	mixins                string
}

// owlBotSourceRegex extracts the base API path from an .OwlBot.yaml
// deep-copy-regex source pattern. The pattern is usually of the form:
// /some/path/(version-regex)/.*-nodejs, or /some/path/[^/]+-nodejs,
// or /some_path-nodejs.
var owlBotSourceRegex = regexp.MustCompile(`^/(?:(.+?)/(?:\(|v\d|[^/]+-nodejs)|([^/]+)-nodejs)`)

func runNodejsMigration(ctx context.Context, repoPath string) error {
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}

	libraries, err := buildNodejsLibraries(repoPath, src.Dir)
	if err != nil {
		return err
	}

	sort.Slice(libraries, func(i, j int) bool {
		return libraries[i].Name < libraries[j].Name
	})

	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
		Sources: &config.Sources{
			Googleapis: src,
		},
		Default: &config.Default{
			Output:       "packages",
			Keep:         []string{"CHANGELOG.md"},
			ReleaseLevel: "stable",
		},
		Libraries: libraries,
	}
	cfg.Sources.Googleapis.Dir = ""

	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return fmt.Errorf("librarian tidy failed: %w", err)
	}
	return nil
}

func buildNodejsLibraries(repoPath, googleapisDir string) ([]*config.Library, error) {
	packagesDir := filepath.Join(repoPath, "packages")
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, err
	}

	var libraries []*config.Library
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		libraryName := entry.Name()
		pkgDir := filepath.Join(packagesDir, libraryName)

		// Read package.json.
		pkgJSON, err := readNodejsPackageJSON(filepath.Join(pkgDir, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("reading package.json for %s: %w", libraryName, err)
		}

		library := &config.Library{
			Name:    libraryName,
			Version: pkgJSON.Version,
		}

		// Read .OwlBot.yaml to get API paths.
		owlBotPath := filepath.Join(pkgDir, ".OwlBot.yaml")
		if _, statErr := os.Stat(owlBotPath); statErr == nil {
			owlBot, err := yaml.Read[owlBotYAML](owlBotPath)
			if err != nil {
				return nil, fmt.Errorf("reading .OwlBot.yaml for %s: %w", libraryName, err)
			}
			apis, err := parseOwlBotAPIPaths(owlBot, googleapisDir, pkgJSON.Name)
			if err != nil {
				return nil, fmt.Errorf("parsing API paths for %s: %w", libraryName, err)
			}
			library.APIs = apis
		}

		// Extract copyright year from existing generated source files.
		if year := extractCopyrightYear(pkgDir); year != "" {
			library.CopyrightYear = year
		}

		// Check if the npm package name needs to be set explicitly.
		derivedName := deriveNpmPackageName(libraryName)
		if pkgJSON.Name != derivedName {
			ensureNodejsPackage(library).PackageName = pkgJSON.Name
		}

		// Extract extra dependencies (beyond google-gax).
		extraDeps := make(map[string]string)
		for dep, version := range pkgJSON.Dependencies {
			if dep == "google-gax" {
				continue
			}
			extraDeps[dep] = version
		}
		if len(extraDeps) > 0 {
			ensureNodejsPackage(library).Dependencies = extraDeps
		}

		// Apply BUILD.bazel fields to the library config.
		if len(library.APIs) > 0 {
			info, err := parseBazelNodejsInfo(googleapisDir, library.APIs[0].Path)
			if err == nil && info != nil {
				if info.bundleConfig != "" || len(info.extraProtocParameters) > 0 ||
					info.handwrittenLayer || info.mainService != "" || info.mixins != "" {
					pkg := ensureNodejsPackage(library)
					if info.bundleConfig != "" {
						pkg.BundleConfig = info.bundleConfig
					}
					if len(info.extraProtocParameters) > 0 {
						pkg.ExtraProtocParameters = info.extraProtocParameters
					}
					if info.handwrittenLayer {
						pkg.HandwrittenLayer = true
					}
					if info.mainService != "" {
						pkg.MainService = info.mainService
					}
					if info.mixins != "" {
						pkg.Mixins = info.mixins
					}
				}
			}
		}

		libraries = append(libraries, library)
	}
	return libraries, nil
}

func readNodejsPackageJSON(path string) (*nodejsPackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkg := &nodejsPackageJSON{}
	if err := json.Unmarshal(data, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

// parseOwlBotAPIPaths extracts API paths from .OwlBot.yaml deep-copy-regex
// source patterns by finding the base path and then discovering version
// directories in googleapis that contain a nodejs_gapic_library rule matching
// the provided npm package name.
func parseOwlBotAPIPaths(owlBot *owlBotYAML, googleapisDir, pkgName string) ([]*config.API, error) {
	if len(owlBot.DeepCopyRegex) == 0 {
		return nil, nil
	}
	// Use the first copy rule to find the base API path.
	source := owlBot.DeepCopyRegex[0].Source
	matches := owlBotSourceRegex.FindStringSubmatch(source)
	if len(matches) < 2 {
		return nil, fmt.Errorf("cannot parse API path from .OwlBot.yaml source: %q", source)
	}
	basePath := matches[1]
	if basePath == "" {
		basePath = matches[2]
	}

	// Find version directories in googleapis by walking the base path.
	dir := filepath.Join(googleapisDir, basePath)
	var apis []*config.API
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // ignore inaccessible directories
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "BUILD.bazel" {
			return nil
		}
		apiDir := filepath.Dir(path)
		apiPath, err := filepath.Rel(googleapisDir, apiDir)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", apiDir, err)
		}
		info, err := parseBazelNodejsInfo(googleapisDir, apiPath)
		if err != nil {
			return fmt.Errorf("parsing bazel info for %s: %w", apiPath, err)
		}
		if info == nil {
			return nil
		}
		// Match the npm package name.
		if info.packageName == pkgName {
			apis = append(apis, &config.API{Path: apiPath})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking googleapis directory %s: %w", dir, err)
	}
	sort.Slice(apis, func(i, j int) bool {
		return apis[i].Path < apis[j].Path
	})
	return apis, nil
}

// parseBazelNodejsInfo reads a BUILD.bazel file from the specified API
// directory (relative to googleapisDir) and extracts information from the
// nodejs_gapic_library rule. Returns nil if no such rule exists.
func parseBazelNodejsInfo(googleapisDir, apiDir string) (*nodejsGapicInfo, error) {
	file, err := parseBazel(googleapisDir, apiDir)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	rules := file.Rules("nodejs_gapic_library")
	if len(rules) == 0 {
		return nil, nil
	}
	if len(rules) > 1 {
		return nil, fmt.Errorf("file %s/BUILD.bazel contains multiple nodejs_gapic_library rules", apiDir)
	}
	rule := rules[0]
	info := &nodejsGapicInfo{
		packageName:           rule.AttrString("package_name"),
		bundleConfig:          rule.AttrString("bundle_config"),
		extraProtocParameters: rule.AttrStrings("extra_protoc_parameters"),
		mainService:           rule.AttrString("main_service"),
		mixins:                rule.AttrString("mixins"),
	}
	if rule.AttrLiteral("handwritten_layer") == "True" {
		info.handwrittenLayer = true
	}
	return info, nil
}

// ensureNodejsPackage returns the Nodejs field of the library, initializing
// it if nil.
func ensureNodejsPackage(l *config.Library) *config.NodejsPackage {
	if l.Nodejs == nil {
		l.Nodejs = &config.NodejsPackage{}
	}
	return l.Nodejs
}

// copyrightYearRegex matches "Copyright YYYY Google" in a file header.
var copyrightYearRegex = regexp.MustCompile(`Copyright (\d{4}) Google`)

// extractCopyrightYear reads the copyright year from src/index.ts.
func extractCopyrightYear(pkgDir string) string {
	data, err := os.ReadFile(filepath.Join(pkgDir, "src", "index.ts"))
	if err != nil {
		return ""
	}
	if m := copyrightYearRegex.FindSubmatch(data); len(m) > 1 {
		return string(m[1])
	}
	return ""
}

// deriveNpmPackageName derives the expected npm package name from a library
// directory name. For example, "google-cloud-batch" becomes
// "@google-cloud/batch".
func deriveNpmPackageName(libraryName string) string {
	idx := strings.Index(libraryName, "-")
	if idx == -1 {
		return libraryName
	}
	idx2 := strings.Index(libraryName[idx+1:], "-")
	if idx2 == -1 {
		return libraryName
	}
	idx2 += idx + 1
	scope := libraryName[:idx2]
	name := libraryName[idx2+1:]
	return fmt.Sprintf("@%s/%s", scope, name)
}
