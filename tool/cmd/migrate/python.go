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
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	pythonIgnoredChanges = []string{
		".repo-metadata.json",
		"docs/README.rst",
	}

	pythonDefaultCommonGAPICPaths = []string{
		"samples/generated_samples",
		"tests/unit/gapic",
		"testing",
		"{neutral-source}/__init__.py",
		"{neutral-source}/gapic_version.py",
		"{neutral-source}/py.typed",
		"tests/unit/__init__.py",
		"tests/__init__.py",
		"setup.py",
		"noxfile.py",
		".coveragerc",
		".flake8",
		".repo-metadata.json",
		"mypy.ini",
		"README.rst",
		"LICENSE",
		"MANIFEST.in",
		"docs/_static/custom.css",
		"docs/_templates/layout.html",
		"docs/conf.py",
		"docs/index.rst",
		"docs/multiprocessing.rst",
		"docs/README.rst",
		"docs/summary_overview.md",
	}

	pythonExtraKeepLists = map[string][]string{
		"google-cloud-firestore": {
			"docs/firestore_admin_v1/admin_client.rst",
			"docs/firestore_v1/aggregation.rst",
			"docs/firestore_v1/batch.rst",
			"docs/firestore_v1/bulk_writer.rst",
			"docs/firestore_v1/client.rst",
			"docs/firestore_v1/collection.rst",
			"docs/firestore_v1/document.rst",
			"docs/firestore_v1/field_path.rst",
			"docs/firestore_v1/query.rst",
			"docs/firestore_v1/transaction.rst",
			"docs/firestore_v1/transforms.rst",
			"docs/firestore_v1/types.rst",
		},
		"google-cloud-spanner": {
			"docs/spanner_v1/batch.rst",
			"docs/spanner_v1/client.rst",
			"docs/spanner_v1/database.rst",
			"docs/spanner_v1/instance.rst",
			"docs/spanner_v1/keyset.rst",
			"docs/spanner_v1/session.rst",
			"docs/spanner_v1/snapshot.rst",
			"docs/spanner_v1/streamed.rst",
			"docs/spanner_v1/table.rst",
			"docs/spanner_v1/transaction.rst",
			"tests/unit/gapic/conftest.py",
		},
	}
)

const (
	pythonDefaultLibraryType = repometadata.GAPICAutoLibraryType
	pythonTagFormat          = "{name}: v{version}"
)

// pythonGapicInfo contains information about the py_gapic_library target
// from BUILD.bazel.
type pythonGapicInfo struct {
	// transport is the transport specified in the BUILD.bazel file as a
	// top-level attribute. The transport may instead be specified in optArgs;
	// this is handled in the generator.
	transport string

	// optArgs is the value of the opt_args attribute in the BUILD.bazel file,
	// if any.
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
			Python:  &config.PythonPackage{},
		}
		if len(libState.APIs) > 0 {
			library.APIs = toAPIs(libState.APIs)
		}
		// Convert "preserve" regexes into "keep" paths, sorted for ease
		// of testing.
		keep, err := transformPreserveToKeep(input.repoPath, libState)
		if err != nil {
			return nil, err
		}

		if extraKeepList, ok := pythonExtraKeepLists[library.Name]; ok {
			keep = append(keep, extraKeepList...)
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
		// directories.
		library, err = applyBuildBazelConfig(library, googleapisDir)
		if err != nil {
			return nil, err
		}
		// Skip copying the readme file if it doesn't already exist.
		_, err = os.Stat(filepath.Join(input.repoPath, "packages", library.Name, "docs", "README.rst"))
		if err != nil {
			if os.IsNotExist(err) {
				library.Python.SkipReadmeCopy = true
			} else {
				return nil, err
			}
		}

		// Canonicalize to avoid odd empty collections etc.
		library, err = canonicalizePythonLibrary(library)
		if err != nil {
			return nil, err
		}

		libraries = append(libraries, library)
	}
	return libraries, nil
}

// applyBuildBazelConfig applies the information from BUILD.bazel files
// associated with the APIs in the library, adding GAPIC generator arguments,
// discovering non-default transports etc.
func applyBuildBazelConfig(library *config.Library, googleapisDir string) (*config.Library, error) {
	pythonConfig := library.Python
	pythonConfig.OptArgsByAPI = make(map[string][]string)

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
	if len(allTransports) != 1 {
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
	defaultDocumentationURI := ""
	defaultIssueTracker := ""
	defaultAPIShortname := ""
	defaultAPIID := ""
	// Load the service config file for the first API if there is one, and
	// use that to work out what will be generated in .repo-metadata.json by
	// default.
	if len(library.APIs) > 0 {
		apiInfo, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, config.LanguagePython)
		if err != nil {
			return nil, err
		}
		defaultTitle = strings.TrimSuffix(strings.TrimSpace(apiInfo.Title), " API")
		defaultDocumentationURI = apiInfo.DocumentationURI
		defaultIssueTracker = apiInfo.NewIssueURI
		defaultAPIShortname = apiInfo.ShortName
		defaultAPIID = apiInfo.ServiceName
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
	if repoMetadata.APIDescription != defaultTitle {
		library.DescriptionOverride = repoMetadata.APIDescription
	}
	if repoMetadata.NamePretty != defaultTitle {
		library.Python.NamePrettyOverride = repoMetadata.NamePretty
	}
	if base, _, found := strings.Cut(defaultDocumentationURI, "/docs/"); found {
		defaultDocumentationURI = base + "/"
	}
	if repoMetadata.ProductDocumentation != defaultDocumentationURI {
		library.Python.ProductDocumentationOverride = repoMetadata.ProductDocumentation
	}
	if repoMetadata.Name != library.Name {
		library.Python.MetadataNameOverride = repoMetadata.Name
	}
	if repoMetadata.LibraryType != pythonDefaultLibraryType {
		library.Python.LibraryType = repoMetadata.LibraryType
	}
	if repoMetadata.ClientDocumentation != python.BuildClientDocumentationURI(library.Name, repoMetadata.Name) {
		library.Python.ClientDocumentationOverride = repoMetadata.ClientDocumentation
	}
	if repoMetadata.IssueTracker != defaultIssueTracker {
		library.Python.IssueTrackerOverride = repoMetadata.IssueTracker
	}
	if repoMetadata.APIShortname != defaultAPIShortname {
		library.Python.APIShortnameOverride = repoMetadata.APIShortname
	}
	if repoMetadata.APIID != defaultAPIID {
		library.Python.APIIDOverride = repoMetadata.APIID
	}
	// Always populate the DefaultVersion field, even if we could have inferred
	// it. The default version affects the final code, and changes to it should
	// be explicit - if adding a new version of an API changes the inferred
	// default version, that would cause compatibility issues. This in itself is
	// far from ideal; keeping the default version is "safe" but toilsome
	// operationally.
	// TODO(https://github.com/googleapis/librarian/issues/4772): design away
	// from default versions.
	library.Python.DefaultVersion = repoMetadata.DefaultVersion

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
	transport := rule.AttrString("transport")
	if transport == "" {
		transport = "grpc+rest"
	}
	return &pythonGapicInfo{
		transport: transport,
		optArgs:   optArgs,
	}, nil
}

// canonicalizePythonLibrary sets empty collections in PythonPackage to nil,
// and removes the PythonPackage entirely if there are no values.
func canonicalizePythonLibrary(library *config.Library) (*config.Library, error) {
	// Convert empty collections to nil
	if len(library.Python.CommonGAPICPaths) == 0 {
		library.Python.CommonGAPICPaths = nil
	}
	if len(library.Python.OptArgsByAPI) == 0 {
		library.Python.OptArgsByAPI = nil
	}
	if len(library.Python.ProtoOnlyAPIs) == 0 {
		library.Python.ProtoOnlyAPIs = nil
	}
	// If there are no overrides, remove the Python-specific config.
	// Detecting this by serializing the configuration is more robust than
	// checking each field.
	pythonConfigBytes, err := yaml.Marshal(library.Python)
	if err != nil {
		return nil, err
	}
	if string(pythonConfigBytes) == "{}\n" {
		library.Python = nil
	}
	return library, nil
}
