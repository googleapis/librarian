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
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

type apiInfo struct {
	// rootDir is the directory (relative to the package root) containing
	// all the subdirectories, e.g. "google/cloud".
	rootDir string
	// commonDir is the directory (relative to rootDir) containing code which
	// is generated for all APIs in a version-neutral way, e.g. "run".
	commonDir string
	// versionDir is the directory (relative to rootDir) containing code which
	// is specific to a single version, e.g. "run_v2". This is empty if the API
	// path has no version component (e.g. for "google/shopping/type").
	versionDir string
}

var errBadAPIPath = errors.New("invalid API path")

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
			cleanGAPIC(api, lib)
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
		return fmt.Errorf("can't find files under %s: %w", dir, err)
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		name := dirEntry.Name()
		if !strings.HasSuffix(name, "_pb2.py") && !strings.HasSuffix(name, "_pb2.pyi") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("error deleting %s from %s: %w", name, dir, err)
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
	apiInfo, err := deriveAPIInfo(api, lib)
	if err != nil {
		return err
	}
	// FIXME: Maybe use commonDir instead in this case?
	// Not sure how often this happens - just google/shopping/type and
	// google/apps/script/type/{xyz}?
	if apiInfo.versionDir == "" {
		return nil
	}
	dir := filepath.Join(lib.Output, apiInfo.rootDir, apiInfo.versionDir)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	docsDir := filepath.Join(lib.Output, "docs", apiInfo.versionDir)
	return os.RemoveAll(docsDir)
}

// cleanGAPICCommon cleans the common output created for packages containing
// any GAPIC libraries.
func cleanGAPICCommon(lib *config.Library) error {
	apiInfo, err := deriveAPIInfo(lib.APIs[0], lib)
	if err != nil {
		return err
	}
	// Whole directories to delete
	if err := os.RemoveAll(filepath.Join(lib.Output, "samples", "generated")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(lib.Output, "tests", "unit", "gapic")); err != nil {
		return err
	}
	// TODO: Check that there's never anything else here.
	if err := os.RemoveAll(filepath.Join(lib.Output, "testing")); err != nil {
		return err
	}
	// Individual files to delete.
	files := []string{
		filepath.Join(apiInfo.rootDir, apiInfo.commonDir, "__init__.py"),
		filepath.Join(apiInfo.rootDir, apiInfo.commonDir, "gapic_version.py"),
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
		"setup.py",
		"docs/static/_custom.css",
		"docs/_templates/layout.html",
		"docs/conf.py",
		"docs/index.rst",
		"docs/multiprocessing.rst",
		"docs/README.rst",
		"docs/summary_overview.md",
	}
	for _, file := range files {
		if err := os.Remove(filepath.Join(lib.Output, file)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error deleting %s/%s: %w", lib.Output, file, err)
		}
	}
	return nil
}

// deriveAPIInfo derives an apiInfo for  a single API within a library, using
// the API path and the options from the configuration.
func deriveAPIInfo(api *config.API, lib *config.Library) (*apiInfo, error) {
	splitPath := strings.Split(api.Path, "/")
	if len(splitPath) < 2 {
		return nil, fmt.Errorf("not enough path segments in %s: %w", api.Path, errBadAPIPath)
	}
	// TODO: Check all of this logic with Tony.
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
	return &apiInfo{
		rootDir:    rootDir,
		commonDir:  gapicName,
		versionDir: versionDir,
	}, nil
}

func findOptArg(api *config.API, cfg *config.PythonPackage, optName string) string {
	if cfg == nil || cfg.OptArgsByAPI == nil {
		return ""
	}
	// FIXME: Remove OptArgs, or use it here.
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
