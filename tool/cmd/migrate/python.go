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
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

// pythonGapicInfo contains information about the py_gapic_library target
// from BUILD.bazel.
type pythonGapicInfo struct {
	// transport is the transport specified in the BUILD.bazel file as a
	// top-level attribute. The transport may instead be specified in optArgs;
	// this is handled in the generator.
	transport string

	// optArgs is the value of the opt_args attribute in the BUILD.bazel file,
	// if any. If a rest_numeric_enums attribute is specified as False, this is
	// included in optArgs as rest-numeric-enums.
	optArgs []string
}

// buildPythonLibraries builds a set of librarian libraries from legacylibrarian
// libraries and the googleapis directory used to find settings in service
// config files, BUILD.bazel files etc.
func buildPythonLibraries(input *MigrationInput, googleapisDir string) ([]*config.Library, error) {
	var libraries []*config.Library
	// No need to use legacyconfig.LibraryConfig - the only thing in
	// the python config is a single global file entry.

	for _, libState := range input.librarianState.Libraries {
		library := &config.Library{
			Name:    libState.ID,
			Version: libState.Version,
		}
		if libState.APIs != nil {
			library.APIs = toAPIs(libState.APIs)
		}
		// Convert "preserve" regexes into "keep" paths, sorted for ease
		// of testing.
		keep, err := transformPreserveToKeep(input.repoPath, libState)
		if err != nil {
			return nil, err
		}
		slices.Sort(keep)
		library.Keep = keep

		// Apply any information from the .repo-metadata JSON file, e.g.
		// overriding the description or library stability.
		repoMetadataPath := filepath.Join(input.repoPath, "packages", libState.ID, ".repo-metadata.json")
		library, err = applyRepoMetadata(repoMetadataPath, googleapisDir, library)
		if err != nil {
			return nil, err
		}

		// Apply any information from the BUILD.bazel files in the various API
		// directories. This also detects if there are any non-GAPIC APIs (e.g
		// pure proto "type" packages) as they cannot currently be migrated.
		library, err = applyBuildBazelConfig(library, googleapisDir)
		if err != nil {
			return nil, err
		}
		// Skip anything that can't be migrated yet.
		if library == nil {
			continue
		}

		libraries = append(libraries, library)
	}
	return libraries, nil
}

// applyBuildBazelConfig applies the information from BUILD.bazel files
// associated with the APIs in the library, adding GAPIC generator arguments,
// discovering non-default transports etc. If any APIs within the library are
// not GAPIC APIs (e.g. they're just protos), applyBuildBazelConfig returns nil
// instead of returning the a pointer to the library config; the caller should
// then skip this library as it cannot yet be migrated.
func applyBuildBazelConfig(library *config.Library, googleapisDir string) (*config.Library, error) {
	pythonConfig := &config.PythonPackage{
		OptArgsByAPI: make(map[string][]string),
	}
	allTransports := make(map[string]bool)
	transportsByApi := make(map[string]string)

	for _, api := range library.APIs {
		bazelGapicInfo, err := parseBazelPythonInfo(googleapisDir, api.Path)
		if err != nil {
			return nil, err
		}
		if bazelGapicInfo == nil {
			pythonConfig.ProtoOnlyAPIs = append(pythonConfig.ProtoOnlyAPIs, api.Path)
			continue
		}
		transportsByApi[api.Path] = bazelGapicInfo.transport
		allTransports[bazelGapicInfo.transport] = true
		if len(bazelGapicInfo.optArgs) != 0 {
			pythonConfig.OptArgsByAPI[api.Path] = bazelGapicInfo.optArgs
		}
	}
	if len(allTransports) == 1 {
		// One consistent transport; set it library-wide if it's not the default.
		// This assumes that where there's a mixture of GAPIC and non-GAPIC, the
		// first path is a GAPIC API, but that happens to be true for now (and
		// we don't care what happens post-migration).
		transport := transportsByApi[library.APIs[0].Path]
		if transport != "grpc+rest" {
			library.Transport = transport
		}
	} else {
		// Transport differs by API version. Add it into OptArgsByAPI, but only
		// for non-proto-only APIs. (Proto-only APIs don't have a transport
		// anyway.)
		for _, api := range library.APIs {
			if slices.Contains(pythonConfig.ProtoOnlyAPIs, api.Path) {
				continue
			}
			optArgs := pythonConfig.OptArgsByAPI[api.Path]
			optArgs = append(optArgs, fmt.Sprintf("transport=%s", transportsByApi[api.Path]))
			pythonConfig.OptArgsByAPI[api.Path] = optArgs
		}
	}

	if len(pythonConfig.OptArgsByAPI) > 0 || len(pythonConfig.ProtoOnlyAPIs) > 0 {
		if len(pythonConfig.OptArgsByAPI) == 0 {
			pythonConfig.OptArgsByAPI = nil
		}
		library.Python = pythonConfig
	}
	return library, nil
}

// transformPreserveToKeep converts the "preserve" entries in a legacylibrarian
// state file into "keep" entries in a new library config. Differences:
//   - Preserve entries are repo-root-relative; keep entries are
//     library-output-relative.
//   - Preserve entries are regular expressions; keep entries are just filenames.
//   - Preserve entries don't have to match anything; keep entries must exist.
//
// transformPreserveToKeep finds all files which would *currently* be kept, and
// creates a keep entry for each of them.
func transformPreserveToKeep(rootDir string, libState *legacyconfig.LibraryState) ([]string, error) {
	if len(libState.PreserveRegex) == 0 {
		return nil, nil
	}
	if len(libState.SourceRoots) != 1 {
		return nil, fmt.Errorf("cannot migrate %s with %d source roots and %d preserve regexes", libState.ID, len(libState.SourceRoots), len(libState.PreserveRegex))
	}
	sourceRoot := libState.SourceRoots[0]

	// relPaths contains a list of files in the source root, but relative to
	// rootDir.
	relPaths, err := findSubDirRelPaths(rootDir, filepath.Join(rootDir, sourceRoot))
	if err != nil {
		return nil, err
	}

	// Apply all the preserve regular expressions to all the files we've found,
	// adding each match to the keep list.
	preserveRegexps, err := compileRegexps(libState.PreserveRegex)
	if err != nil {
		return nil, err
	}
	var keepPaths []string
	for _, path := range filterPathsByRegex(relPaths, preserveRegexps) {
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			// This should not happen given the logic in findSubDirRelPaths
			return nil, fmt.Errorf("could not make path %q relative to %q: %w", path, sourceRoot, err)
		}
		keepPaths = append(keepPaths, relative)
	}
	return keepPaths, nil
}

// compileRegexps takes a slice of string patterns and compiles each one into a
// regular expression. It returns a slice of compiled regexps or an error if any
// pattern is invalid.
// This function is copied from legacylibrarian. Neither this code nor
// legacylibrarian is expected to be long-lived, so it's not worth refactoring
// for reuse without duplication.
func compileRegexps(patterns []string) ([]*regexp.Regexp, error) {
	var regexps []*regexp.Regexp
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		regexps = append(regexps, re)
	}
	return regexps, nil
}

// findSubDirRelPaths walks the subDir tree returns a slice of all file and
// directory paths relative to the dir. This is repeated for all nested
// directories. subDir must be under or the same as dir.
// This function is copied from legacylibrarian. Neither this code nor
// legacylibrarian is expected to be long-lived, so it's not worth refactoring
// for reuse without duplication.
func findSubDirRelPaths(dir, subDir string) ([]string, error) {
	dirRelPath, err := filepath.Rel(dir, subDir)
	if err != nil {
		return nil, fmt.Errorf("cannot establish the relationship between %s and %s: %w", dir, subDir, err)
	}
	// '..' signifies that the subDir exists outside of dir
	if strings.HasPrefix(dirRelPath, "..") {
		return nil, fmt.Errorf("subDir is not nested within the dir: %s, %s", subDir, dir)
	}

	var paths []string
	err = filepath.WalkDir(subDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// error is ignored as we have confirmed that subDir is child or equal to rootDir
		relPath, _ := filepath.Rel(dir, path)
		// Special case when subDir is equal to dir. Drop the "." as it references itself
		if relPath != "." {
			paths = append(paths, relPath)
		}
		return nil
	})
	return paths, err
}

// filterPathsByRegex returns a new slice containing only the paths from the
// input slice that match at least one of the provided regular expressions.
// This function is copied from legacylibrarian. Neither this code nor
// legacylibrarian is expected to be long-lived, so it's not worth refactoring
// for reuse without duplication.
func filterPathsByRegex(paths []string, regexps []*regexp.Regexp) []string {
	var filtered []string
	for _, path := range paths {
		for _, re := range regexps {
			if re.MatchString(path) {
				filtered = append(filtered, path)
				break
			}
		}
	}
	return filtered
}

// applyRepoMetadata loads the existing .repo-metadata.json file for a library
// and applies the information within it to the specified library.
func applyRepoMetadata(metadataPath, googleapisDir string, library *config.Library) (*config.Library, error) {
	defaultTitle := ""
	// Load the service config file for the first API if there is one, and
	// use that
	if len(library.APIs) > 0 {
		apiInfo, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, serviceconfig.LangPython)
		if err != nil {
			return nil, err
		}
		defaultTitle = apiInfo.Title
	}

	// Load the current repo metadata and apply overrides for anything that
	// isn't going to get the right value by default.
	generatorInputRepoMetadata, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}
	repoMetadata := &repometadata.RepoMetadata{}
	if err := json.Unmarshal(generatorInputRepoMetadata, repoMetadata); err != nil {
		return nil, err
	}
	if repoMetadata.ReleaseLevel != "stable" {
		library.ReleaseLevel = repoMetadata.ReleaseLevel
	}
	if repoMetadata.APIDescription != defaultTitle {
		library.DescriptionOverride = repoMetadata.APIDescription
	}
	return library, nil
}

// parseBazelPythonInfo reads a BUILD.bazel file from the specified API
// directory (relative to googleapisDir) and populates a pythonGapicInfo with
// information based on the attributes. This function fails with an error if
// there's no such BUILD.bazel file, it's invalid, or it includes multiple
// py_gapic_library rules. If the file exists with no py_gapic_library rules,
// the function succeeds but returns a nil pointer (to indicate there's no
// GAPIC library).
func parseBazelPythonInfo(googleapisDir, apiDir string) (*pythonGapicInfo, error) {
	path := filepath.Join(googleapisDir, apiDir, "BUILD.bazel")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, err := build.ParseBuild(path, data)
	if err != nil {
		return nil, err
	}
	rules := file.Rules("py_gapic_library")
	if len(rules) == 0 {
		return nil, nil
	}
	if len(rules) > 1 {
		return nil, fmt.Errorf("file %s contains multiple py_gapic_library rules", path)
	}
	rule := rules[0]
	optArgs := rule.AttrStrings("opt_args")
	if rule.AttrLiteral("rest_numeric_enums") == "False" {
		optArgs = append(optArgs, "rest-numeric-enums=False")
	}
	transport := rule.AttrString("transport")
	if transport == "" {
		transport = "grpc+rest"
	}
	return &pythonGapicInfo{
		transport: transport,
		optArgs:   optArgs,
	}, nil
}
