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
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	// regexAPIPath matches OwlBot deep-copy-regex source paths for Ruby libraries.
	// Capturing groups:
	//   1: Base API path (e.g. "google/cloud/automl")
	//   2: API version (e.g. "v1") or empty string for unversioned wrapper libraries
	//   3: Gem directory name token (e.g. "[^/]+" or "google-cloud-automl-v1")
	//   4: Trailing path contents after "-ruby/"
	regexAPIPath = regexp.MustCompile(`^/(.+?)(?:/(v\d+\w*))?/(\[\^/\]\+|[^/]+)-ruby/(.*)$`)
	// Skip these directories when searching for libraries.
	skippedDirs = []string{".github"}
)

type owlbotYaml struct {
	DeepCopyRegex []owlbotSrc `yaml:"deep-copy-regex"`
}

type owlbotSrc struct {
	Source string `yaml:"source"`
}

// WrapperBuild represents build configuration parsed from BUILD.bazel for an unversioned Ruby wrapper library.
type WrapperBuild struct {
	Path   string
	Params *ExtraProtoParams
}

// ExtraProtoParams represents extra protoc parameters parsed from BUILD.bazel for a Ruby API version.
type ExtraProtoParams struct {
	EnvPrefix          string
	ExtraDeps          string
	GemNamespace       string
	NamespaceOverride  string
	PathOverride       string
	ServiceOverride    string
	WrapperGemOverride string
	YardStrict         string
}

func runRubyMigration(ctx context.Context, repoPath string) error {
	src, err := fetchSource(ctx)
	if err != nil {
		return errFetchSource
	}
	cfg := &config.Config{
		Language: config.LanguageRuby,
		Sources: &config.Sources{
			Googleapis: src,
		},
		Tools: &config.Tools{
			Gem: []*config.GemTool{
				{
					Name:    "gapic-generator-cloud",
					Version: "0.49.0",
				},
				{
					Name:    "grpc",
					Version: "1.78.1",
				},
			},
			Protoc: &config.Protoc{
				Version: "33.2",
				SHA256:  "b24b53f87c151bfd48b112fe4c3a6e6574e5198874f38036aff41df3456b8caf",
			},
		},
	}
	if err := mergeConfig(cfg, repoPath); err != nil {
		return err
	}
	existingLibs, err := parseExistingLibraries(repoPath)
	if err != nil {
		return err
	}
	libs, err := findRubyLibraries(src.Dir, repoPath)
	if err != nil {
		return err
	}
	cfg.Libraries = mergeLibs(existingLibs, libs)
	// The directory name in Googleapis is present for migration code to look
	// up API details. It shouldn't be persisted.
	cfg.Sources.Googleapis.Dir = ""
	if err := librarian.RunTidyOnConfig(ctx, repoPath, cfg); err != nil {
		return fmt.Errorf("%w: %w", errTidyFailed, err)
	}
	log.Printf("Successfully migrated Ruby libraries configuration")
	return nil
}

func parseExistingLibraries(repoPath string) ([]*config.Library, error) {
	cfg, err := readExistingConfig(repoPath)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}
	return cfg.Libraries, nil
}

func findRubyLibraries(googleapisPath, repoPath string) ([]*config.Library, error) {
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, fmt.Errorf("reading repository directory: %w", err)
	}
	var libraries []*config.Library
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if slices.Contains(skippedDirs, name) {
			continue
		}
		owlBotPath := filepath.Join(repoPath, name, ".OwlBot.yaml")
		if _, err := os.Stat(owlBotPath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("checking OwlBot config: %w", err)
		}
		lib := &config.Library{
			Name: name,
		}
		api, isWrapper, err := parseAPIFromOwlBot(owlBotPath)
		if err != nil {
			return nil, err
		}
		if api != "" {
			if !isWrapper {
				lib.APIs = []*config.API{
					{
						Path: api,
					},
				}
				vb, err := parseVersionedBuild(googleapisPath, api)
				if err != nil {
					return nil, err
				}
				if vb != nil {
					lib.APIs[0].Ruby = &config.RubyAPI{
						RubyCloudOpts: &config.RubyCloudOpts{
							EnvPrefix:          vb.EnvPrefix,
							ExtraDependencies:  vb.ExtraDeps,
							GemNamespace:       vb.GemNamespace,
							NamespaceOverride:  vb.NamespaceOverride,
							PathOverride:       vb.PathOverride,
							ServiceOverride:    vb.ServiceOverride,
							WrapperGemOverride: vb.WrapperGemOverride,
							YardStrict:         vb.YardStrict,
						},
					}
				}
			} else {
				wb, err := parseUnversionedBuild(googleapisPath, api)
				if err != nil {
					return nil, err
				}
				if wb != nil {
					rubyAPI := &config.API{
						Path: wb.Path,
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{
								EnvPrefix:          wb.Params.EnvPrefix,
								ExtraDependencies:  wb.Params.ExtraDeps,
								GemNamespace:       wb.Params.GemNamespace,
								NamespaceOverride:  wb.Params.NamespaceOverride,
								PathOverride:       wb.Params.PathOverride,
								ServiceOverride:    wb.Params.ServiceOverride,
								WrapperGemOverride: wb.Params.WrapperGemOverride,
								YardStrict:         wb.Params.YardStrict,
							},
						},
					}
					lib.APIs = append(lib.APIs, rubyAPI)
				}
			}
			libraries = append(libraries, lib)
		}
	}
	parseWrapperOf(libraries)
	return libraries, nil
}

// parseAPIFromOwlBot parses API details from OwlBot config and determines if the library is a wrapper.
// It returns the API path, whether the library is a wrapper, and an error if parsing fails.
func parseAPIFromOwlBot(owlBotPath string) (string, bool, error) {
	data, err := os.ReadFile(owlBotPath)
	if err != nil {
		return "", false, fmt.Errorf("reading OwlBot config %s: %w", owlBotPath, err)
	}
	owlbot, err := yaml.Unmarshal[owlbotYaml](data)
	if err != nil {
		return "", false, fmt.Errorf("parsing OwlBot config %s: %w", owlBotPath, err)
	}
	// Skip .github/.Owlbot.yaml.
	if len(owlbot.DeepCopyRegex) == 0 {
		return "", false, nil
	}
	src := owlbot.DeepCopyRegex[0].Source
	matches := regexAPIPath.FindStringSubmatch(src)
	if len(matches) != 5 {
		return "", false, nil
	}
	basePath := matches[1]
	version := matches[2]
	if version == "" {
		// Unversioned wrapper library
		return basePath, true, nil
	}
	// Versioned library
	return basePath + "/" + version, false, nil
}

// parseWrapperOf sets the WrapperOf field for wrapper libraries.
func parseWrapperOf(libraries []*config.Library) {
	slices.SortFunc(libraries, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})
	for i, lib := range libraries {
		var wrapperOf []string
		prefix := lib.Name + "-"
		// Since libraries are sorted by name, the wrapped libraries
		// are guaranteed to appear after the wrapper library.
		for j := i + 1; j < len(libraries); j++ {
			other := libraries[j]
			if !strings.HasPrefix(other.Name, prefix) {
				// Since libraries are sorted by name, the wrapped libraries
				// must be consecutive.
				break
			}
			suffix := strings.TrimPrefix(other.Name, prefix)
			// Verify that the suffix after the prefix represents a valid version,
			// e.g., starting with v followed by a digit.
			// We use simple, string comparison because the migration tool will
			// be removed after language onboarding.
			if len(suffix) > 1 && suffix[0] == 'v' && suffix[1] >= '0' && suffix[1] <= '9' {
				wrapperOf = append(wrapperOf, other.Name)
			}
		}
		if len(wrapperOf) > 0 {
			lib.Ruby = &config.RubyPackage{
				WrapperOf: wrapperOf,
			}
		}
	}
}

func parseVersionedBuild(googleapisDir, apiPath string) (*ExtraProtoParams, error) {
	file, err := parseBazel(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	return parseExtraProtoParams(file)
}

func parseExtraProtoParams(file *build.File) (*ExtraProtoParams, error) {
	vb := &ExtraProtoParams{}
	rules := file.Rules("ruby_cloud_gapic_library")
	if len(rules) == 0 || rules[0].Attr("extra_protoc_parameters") == nil {
		return vb, nil
	}
	for _, dep := range extractStrings(rules[0].Attr("extra_protoc_parameters")) {
		switch {
		case strings.HasPrefix(dep, "ruby-cloud-env-prefix="):
			vb.EnvPrefix, _ = strings.CutPrefix(dep, "ruby-cloud-env-prefix=")
		case strings.HasPrefix(dep, "ruby-cloud-extra-dependencies="):
			vb.ExtraDeps, _ = strings.CutPrefix(dep, "ruby-cloud-extra-dependencies=")
		case strings.HasPrefix(dep, "ruby-cloud-gem-namespace="):
			vb.GemNamespace, _ = strings.CutPrefix(dep, "ruby-cloud-gem-namespace=")
		case strings.HasPrefix(dep, "ruby-cloud-namespace-override="):
			vb.NamespaceOverride, _ = strings.CutPrefix(dep, "ruby-cloud-namespace-override=")
		case strings.HasPrefix(dep, "ruby-cloud-path-override="):
			vb.PathOverride, _ = strings.CutPrefix(dep, "ruby-cloud-path-override=")
		case strings.HasPrefix(dep, "ruby-cloud-service-override="):
			vb.ServiceOverride, _ = strings.CutPrefix(dep, "ruby-cloud-service-override=")
		case strings.HasPrefix(dep, "ruby-cloud-wrapper-gem-override="):
			vb.WrapperGemOverride, _ = strings.CutPrefix(dep, "ruby-cloud-wrapper-gem-override=")
		case strings.HasPrefix(dep, "ruby-cloud-yard-strict="):
			vb.YardStrict, _ = strings.CutPrefix(dep, "ruby-cloud-yard-strict=")

		}
	}
	return vb, nil
}

// mergeLibs merges existing libraries with discovered libraries. Existing libraries' configurations
// are preserved and discovered libraries are appended to the result.
func mergeLibs(existingLibs []*config.Library, libs []*config.Library) []*config.Library {
	existingMap := toMap(existingLibs)
	res := make([]*config.Library, len(existingLibs))
	copy(res, existingLibs)
	for _, lib := range libs {
		if _, ok := existingMap[lib.Name]; !ok {
			res = append(res, lib)
		}
	}
	return res
}

func toMap(libs []*config.Library) map[string]*config.Library {
	libraryMap := make(map[string]*config.Library)
	for _, lib := range libs {
		libraryMap[lib.Name] = lib
	}
	return libraryMap
}

func mergeConfig(cfg *config.Config, repoPath string) error {
	existingConfig, err := readExistingConfig(repoPath)
	if err != nil {
		return err
	}
	if existingConfig != nil {
		if existingConfig.Sources != nil {
			cfg.Sources = existingConfig.Sources
		}
		if existingConfig.Tools != nil {
			cfg.Tools = existingConfig.Tools
		}
	}
	return nil
}

func readExistingConfig(repoPath string) (*config.Config, error) {
	absConfigPath, err := filepath.Abs(filepath.Join(repoPath, config.LibrarianYAML))
	if err != nil {
		return nil, fmt.Errorf("getting absolute path to %s: %w", config.LibrarianYAML, err)
	}
	cfg, err := yaml.Read[config.Config](absConfigPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseUnversionedBuild(googleapisDir, apiPath string) (*WrapperBuild, error) {
	file, err := parseBazel(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	api := parseAPIFromWrapperBuild(file)
	if api == "" {
		return nil, nil
	}
	params, err := parseExtraProtoParams(file)
	if err != nil {
		return nil, err
	}
	return &WrapperBuild{
		Path:   api,
		Params: params,
	}, nil
}

func parseAPIFromWrapperBuild(file *build.File) string {
	rules := file.Rules("ruby_cloud_gapic_library")
	if len(rules) == 0 || rules[0].Attr("srcs") == nil {
		return ""
	}
	srcs := extractStrings(rules[0].Attr("srcs"))
	if len(srcs) == 0 {
		return ""
	}
	res := srcs[0]
	parts := strings.SplitN(res, ":", 2)
	if len(parts) > 0 {
		res = parts[0]
	}
	res, _ = strings.CutPrefix(res, "//")
	return res
}
