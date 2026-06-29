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

package librarian

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/python"
)

const (
	bulkManifestFile            = ".release-please-bulk-manifest.json"
	bulkConfigFile              = "release-please-bulk-config.json"
	defaultManifestFile         = ".release-please-manifest.json"
	defaultConfigFile           = "release-please-config.json"
	defaultReleasePleaseVersion = "0.0.0"
)

func hasBulkReleasePleaseConfigs(dir string, cfg *config.Config) bool {
	manifestFile, configFile := releasePleaseFiles(cfg)
	_, errM := os.Stat(filepath.Join(dir, manifestFile))
	_, errC := os.Stat(filepath.Join(dir, configFile))
	return !errors.Is(errM, fs.ErrNotExist) && !errors.Is(errC, fs.ErrNotExist)
}

// releasePleaseFiles returns the file names for the Release Please manifest file
// and config file in this order, depending on the SDK language.
func releasePleaseFiles(cfg *config.Config) (string, string) {
	// google-cloud-node uses the default Release Please files to add a new library.
	// google-cloud-python and google-cloud-go use the "-bulk-" files.
	manifestFile := bulkManifestFile
	configFile := bulkConfigFile
	if cfg.Language == config.LanguageNodejs {
		// google-cloud-node uses the default files
		manifestFile = defaultManifestFile
		configFile = defaultConfigFile
	}
	return manifestFile, configFile
}

// syncToReleasePlease updates the release-please configuration files with the
// onboarded library's package name, initial version, and language-specific
// extra files to track for release version bumps.
func syncToReleasePlease(dir string, cfg *config.Config, name string) error {
	lib, err := FindLibrary(cfg, name)
	if err != nil {
		return err
	}

	manifestFile, configFile := releasePleaseFiles(cfg)
	manifestPath := filepath.Join(dir, manifestFile)
	manifest, err := readJSONFile[map[string]string](manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}
	if manifest == nil {
		manifest = make(map[string]string)
	}

	configPath := filepath.Join(dir, configFile)
	bulkConfig, err := readJSONFile[map[string]any](configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	if bulkConfig == nil {
		bulkConfig = make(map[string]any)
	}
	packagesRaw, pkgsExist := bulkConfig["packages"]
	packages, isMap := packagesRaw.(map[string]any)
	if pkgsExist && !isMap {
		return fmt.Errorf("'packages' in %s is not an object: %v",
			configPath, packagesRaw)
	}
	if !isMap || packages == nil {
		packages = make(map[string]any)
		bulkConfig["packages"] = packages
	}

	var extraFiles []any
	pkgPath := lib.Name
	switch cfg.Language {
	case config.LanguagePython:
		pkgPath = python.ReleasePleasePkgPrefix + lib.Name
		extraFiles = python.ReleasePleaseExtraFiles(lib)
	case config.LanguageNodejs:
		pkgPath = "packages/" + lib.Name
	}

	component := lib.Name
	if cfg.Language == config.LanguageNodejs {
		// google-cloud-node does not need to override
		// component value in package.
		component = ""
	}

	if err := syncPackageToReleasePlease(manifest, packages, pkgPath, lib.Version, component, extraFiles); err != nil {
		return err
	}

	manifestOut, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, manifestOut, 0644); err != nil {
		return err
	}

	configOut, err := json.MarshalIndent(bulkConfig, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, configOut, 0644); err != nil {
		return err
	}

	return nil
}

// readJSONFile reads a file and parses its JSON content into a new instance of T.
func readJSONFile[T any](path string) (T, error) {
	var val T
	b, err := os.ReadFile(path)
	if err != nil {
		return val, err
	}
	if err := json.Unmarshal(b, &val); err != nil {
		return val, err
	}
	return val, nil
}

// syncPackageToReleasePlease registers a package's version in the manifest and
// merges its configuration/extra-files into the package configuration map.
func syncPackageToReleasePlease(manifest map[string]string, packages map[string]any, pkgPath, version, component string, extraFiles []any) error {
	v := defaultReleasePleaseVersion
	if version != "" {
		v = version
	}
	if _, ok := manifest[pkgPath]; !ok {
		manifest[pkgPath] = v
	}

	pkgRaw, ok := packages[pkgPath]
	pkgCfg, isMap := pkgRaw.(map[string]any)
	if ok && !isMap {
		return fmt.Errorf("package configuration for %q is not an object: %v", pkgPath, pkgRaw)
	}
	if !ok || !isMap || pkgCfg == nil {
		pkgCfg = make(map[string]any)
		packages[pkgPath] = pkgCfg
	}

	if component != "" {
		// Python and Go set component names for packages in the config file.
		// NodeJS does not do this and passes an empty string in the argument.
		pkgCfg["component"] = component
	}

	if len(extraFiles) > 0 {
		var existing []any
		if e, ok := pkgCfg["extra-files"].([]any); ok {
			existing = e
		}
		merged, err := mergeExtraFiles(existing, extraFiles)
		if err != nil {
			return err
		}
		pkgCfg["extra-files"] = merged
	}
	return nil
}

type extraFile struct {
	path   string
	isMap  bool
	rawMap map[string]any
}

// mergeExtraFiles merges existing and derived extra-files list while removing duplicates.
//
// Items can be string paths or maps with a "path" key. Deduplication and sorting (strings
// first, maps second) are done using the path string.
func mergeExtraFiles(existing []any, derived []any) ([]any, error) {
	seen := make(map[string]*extraFile)
	if err := addUniqueExtraFiles(seen, existing); err != nil {
		return nil, err
	}
	if err := addUniqueExtraFiles(seen, derived); err != nil {
		return nil, err
	}

	files := make([]*extraFile, 0, len(seen))
	for _, ef := range seen {
		files = append(files, ef)
	}

	sort.Slice(files, func(i, j int) bool {
		iF := files[i]
		jF := files[j]

		if iF.isMap != jF.isMap {
			return !iF.isMap
		}
		return iF.path < jF.path
	})

	result := make([]any, 0, len(files))
	for _, ef := range files {
		if ef.isMap {
			result = append(result, ef.rawMap)
		} else {
			result = append(result, ef.path)
		}
	}
	return result, nil
}

// addUniqueExtraFiles normalizes and adds items to the seen map, prioritizing
// map entries over strings and returning an error on conflicts.
func addUniqueExtraFiles(seen map[string]*extraFile, items []any) error {
	for _, item := range items {
		var ef *extraFile
		switch val := item.(type) {
		case string:
			ef = &extraFile{path: val, isMap: false}
		case map[string]any:
			p, ok := val["path"].(string)
			if !ok || p == "" {
				return fmt.Errorf("extra-files item map missing 'path' key or not a string: %v", val)
			}
			ef = &extraFile{path: p, isMap: true, rawMap: val}
		default:
			return fmt.Errorf("invalid extra-files item type %T: %v", item, item)
		}

		existingEF, exists := seen[ef.path]
		if !exists {
			seen[ef.path] = ef
			continue
		}
		if ef.isMap && !existingEF.isMap {
			seen[ef.path] = ef
			continue
		}
		if ef.isMap && existingEF.isMap && !reflect.DeepEqual(ef.rawMap, existingEF.rawMap) {
			return fmt.Errorf("conflicting configurations for extra-file %q:\nexisting: %v\nnew: %v", ef.path, existingEF.rawMap, ef.rawMap)
		}
	}
	return nil
}
