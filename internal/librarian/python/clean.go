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

package python

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// gapicGenerationInfo contains useful information about the expected directory
// structure of files created by the GAPIC generator. The fields are exported
// for ease of testing, but the type itself is not exported - it is not expected
// to be useful outside the python package.
type gapicGenerationInfo struct {
	// RootDir is the directory (relative to the package root) containing
	// all the subdirectories, e.g. "google/cloud".
	RootDir string
	// NeutralDir is the directory (relative to RootDir) containing code which
	// is generated for all APIs in a version-neutral way, e.g. "run".
	NeutralDir string
	// VersionDir is the directory (relative to RootDir) containing code which
	// is specific to a single version, e.g. "run_v2". This is empty if the API
	// path has no version component (e.g. for "google/shopping/type").
	VersionDir string
}

const neutralSourcePlaceholder = "{neutral-source}"

var (
	errBadAPIPath               = errors.New("invalid API path")
	errNoCommonGAPICFilesConfig = errors.New("when cleaning a GAPIC package, a config with common GAPIC paths must be provided")
)

// CleanLibrary removes all generated code from beneath the given library's
// output directory. If the output directory does not currently exist, this
// function is a no-op.
func CleanLibrary(lib *config.Library) error {
	_, err := os.Stat(lib.Output)
	if os.IsNotExist(err) {
		return nil
	}

	if len(lib.APIs) == 0 {
		return nil
	}

	anyGAPIC := false
	for _, api := range lib.APIs {
		if isProtoOnly(api, lib) {
			if err := cleanProtoOnly(api, lib); err != nil {
				return err
			}
		} else {
			if err := cleanGAPIC(api, lib); err != nil {
				return err
			}
			anyGAPIC = true
		}
	}
	if anyGAPIC {
		if err := cleanGAPICCommon(lib); err != nil {
			return err
		}
	}
	return nil
}

// cleanProtoOnly cleans the output of a proto-only API. This is expected to
// be the directory corresponding to the API path, under the output directory.
// All .proto files and files ending with "_pb2.py" and "_pb2.pyi" are deleted;
// any subdirectories are ignored.
func cleanProtoOnly(api *config.API, lib *config.Library) error {
	dir := filepath.Join(lib.Output, api.Path)
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("can't find files under %s: %w", dir, err)
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		name := dirEntry.Name()
		if !strings.HasSuffix(name, "_pb2.py") && !strings.HasSuffix(name, "_pb2.pyi") && !strings.HasSuffix(name, ".proto") {
			continue
		}
		if err := deleteUnlessKept(lib, filepath.Join(api.Path, name)); err != nil {
			return fmt.Errorf("error deleting %s/%s: %w", api.Path, name, err)
		}
	}
	return nil
}

// cleanGAPIC cleans the generated output from a single GAPIC API, but only
// files that will be uniquely generated for that API. (See cleanGAPICCommon for
// cleaning files that will be generated for multiple APIs and overlaid.) The
// directory corresponding to the API version (e.g. "google/cloud/run_v2") will
// be completely deleted. Likewise the documentation directory (e.g.
// "docs/run_v2") will be completely deleted.
func cleanGAPIC(api *config.API, lib *config.Library) error {
	generationInfo, err := deriveGAPICGenerationInfo(api, lib)
	if err != nil {
		return err
	}
	// Unusual, but it does happen, e.g. google/shopping/type and
	// google/apps/script/type/{xyz}. We'll delete files as "common" GAPIC
	// files instead.
	if generationInfo.VersionDir == "" {
		return nil
	}
	srcDir := filepath.Join(generationInfo.RootDir, generationInfo.VersionDir)
	if err := deleteUnlessKept(lib, srcDir); err != nil {
		return err
	}
	docsDir := filepath.Join("docs", generationInfo.VersionDir)
	return deleteUnlessKept(lib, docsDir)
}

// cleanGAPICCommon cleans the common output created for packages containing
// any GAPIC libraries.
func cleanGAPICCommon(lib *config.Library) error {
	apiInfo, err := deriveGAPICGenerationInfo(lib.APIs[0], lib)
	if err != nil {
		return err
	}
	if lib.Python == nil {
		return errNoCommonGAPICFilesConfig
	}
	if len(lib.Python.CommonGAPICPaths) == 0 {
		return errNoCommonGAPICFilesConfig
	}
	neutralDir := filepath.Join(apiInfo.RootDir, apiInfo.NeutralDir)
	for _, path := range lib.Python.CommonGAPICPaths {
		replacedPath := strings.ReplaceAll(path, neutralSourcePlaceholder, neutralDir)
		if err := deleteUnlessKept(lib, replacedPath); err != nil {
			return fmt.Errorf("error deleting %s: %w", replacedPath, err)
		}
	}
	return nil
}

// deleteUnlessKept deletes the specified path unless it's preserved by the
// Keep configuration of the specifified library. If the path is a directory,
// the function recurses, deleting all files below the directory (including
// files in child directories). Directories themselves are never deleted. If
// a directory appears in a Keep list, no child files are deleted.
// The path is expected to be relative to the library's output directory.
// No error is reported if the given path is not found.
func deleteUnlessKept(lib *config.Library, path string) error {
	if slices.Contains(lib.Keep, path) {
		return nil
	}
	fullPath := filepath.Join(lib.Output, path)
	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !stat.IsDir() {
		return os.Remove(fullPath)
	}
	children, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := deleteUnlessKept(lib, filepath.Join(path, child.Name())); err != nil {
			return err
		}
	}
	return nil
}

// deriveGAPICGenerationInfo derives an apiInfo for  a single API within a library, using
// the API path and the options from the configuration.
func deriveGAPICGenerationInfo(api *config.API, lib *config.Library) (*gapicGenerationInfo, error) {
	splitPath := strings.Split(api.Path, "/")
	if len(splitPath) < 2 {
		return nil, fmt.Errorf("not enough path segments in %s: %w", api.Path, errBadAPIPath)
	}
	namespace := findOptArg(api, lib.Python, "python-gapic-namespace")
	gapicName := findOptArg(api, lib.Python, "python-gapic-name")

	lastElement := splitPath[len(splitPath)-1]
	version := ""
	if strings.HasPrefix(lastElement, "v") {
		version = lastElement
		splitPath = splitPath[:len(splitPath)-1]
	}
	var rootDir string
	if namespace == "" {
		rootDir = strings.Join(splitPath[:len(splitPath)-1], "/")
	} else {
		rootDir = strings.ReplaceAll(namespace, ".", "/")
	}
	if gapicName == "" {
		gapicName = splitPath[len(splitPath)-1]
	}
	versionDir := ""
	if version != "" {
		versionDir = fmt.Sprintf("%s_%s", gapicName, version)
	}
	return &gapicGenerationInfo{
		RootDir:    rootDir,
		NeutralDir: gapicName,
		VersionDir: versionDir,
	}, nil
}

// findOptArg finds the value for the named option within the configuration for
// a Python package, with respect to a specific API path.
// Note: this does not use PythonPackage.OptArgs, only OptArgsByAPI.
// TODO(https://github.com/googleapis/librarian/issues/4107): remove the above
// comment when OptArgs doesn't exist, or change this function to use OptArgs
// if we decide to keep it.
func findOptArg(api *config.API, cfg *config.PythonPackage, optName string) string {
	if cfg == nil || cfg.OptArgsByAPI == nil {
		return ""
	}
	args, ok := cfg.OptArgsByAPI[api.Path]
	if !ok {
		return ""
	}
	prefix := optName + "="
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return arg[len(prefix):]
		}
	}
	return ""
}
