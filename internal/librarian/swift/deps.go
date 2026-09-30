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

package swift

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"golang.org/x/sync/errgroup"
)

var localOrRemotePathRegex = regexp.MustCompile(`(?s)localOrRemotePackage\s*\([^\)]*?path:\s*"([^"]+)"`)

// extractInternalPackageDependencies parses Package.swift content and returns
// the paths declared in localOrRemotePackage(..., path: "...", ...) calls.
func extractInternalPackageDependencies(manifestContent string) []string {
	matches := localOrRemotePathRegex.FindAllStringSubmatch(manifestContent, -1)
	var paths []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		p := filepath.ToSlash(filepath.Clean(strings.TrimSpace(m[1])))
		if p != "" && !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	return paths
}

func dumpPackageDependencies(ctx context.Context, swiftExe, pkgDir string) ([]string, error) {
	out, err := command.OutputWithEnv(ctx, map[string]string{"GOOGLE_CLOUD_SWIFT_LOCAL_DEPS": "1"}, swiftExe, "package", "--package-path", pkgDir, "dump-package")
	if err != nil {
		return nil, err
	}
	var dump struct {
		Dependencies []struct {
			FileSystem []struct {
				Identity string `json:"identity"`
				Path     string `json:"path"`
			} `json:"fileSystem"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(out), &dump); err != nil {
		return nil, err
	}
	var paths []string
	for _, dep := range dump.Dependencies {
		for _, fs := range dep.FileSystem {
			if fs.Path != "" {
				paths = append(paths, fs.Path)
			} else if fs.Identity != "" {
				paths = append(paths, fs.Identity)
			}
		}
	}
	return paths, nil
}

func extractPackageDependencies(ctx context.Context, swiftExe, pkgDir string) ([]string, error) {
	if swiftExe != "" {
		if _, err := exec.LookPath(swiftExe); err == nil {
			paths, err := dumpPackageDependencies(ctx, swiftExe, pkgDir)
			if err == nil {
				return paths, nil
			}
		}
	}
	manifestPath := filepath.Join(pkgDir, "Package.swift")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", manifestPath, err)
	}
	return extractInternalPackageDependencies(string(content)), nil
}

// buildDependencyGraph reads or dumps package manifests for each library and returns a map of
// library name to the names of its internal monorepo dependencies.
func buildDependencyGraph(ctx context.Context, cfg *config.Config, libraries []*config.Library, swiftExe string) (map[string][]string, error) {
	pathToLib := make(map[string]*config.Library)
	for _, lib := range cfg.Libraries {
		pkgDir := libraryPackageDirectory(lib, cfg.Default)
		if pkgDir == "" {
			continue
		}
		cleanDir := filepath.ToSlash(filepath.Clean(pkgDir))
		pathToLib[cleanDir] = lib
		pathToLib[filepath.Base(cleanDir)] = lib
		if base := SplitRepoName(cleanDir); base != "" {
			pathToLib[base] = lib
		}
		if lib.Output != "" {
			pathToLib[filepath.ToSlash(filepath.Clean(lib.Output))] = lib
		}
	}

	var mu sync.Mutex
	deps := make(map[string][]string)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for _, lib := range libraries {
		pkgDir := libraryPackageDirectory(lib, cfg.Default)
		if pkgDir == "" {
			mu.Lock()
			deps[lib.Name] = nil
			mu.Unlock()
			continue
		}

		g.Go(func() error {
			rawPaths, err := extractPackageDependencies(gctx, swiftExe, pkgDir)
			if err != nil {
				return err
			}
			var libDeps []string
			for _, rawPath := range rawPaths {
				depLib := resolveDependencyLibrary(rawPath, pathToLib)
				if depLib != nil && depLib.Name != lib.Name && !slices.Contains(libDeps, depLib.Name) {
					libDeps = append(libDeps, depLib.Name)
				}
			}
			slices.Sort(libDeps)

			mu.Lock()
			deps[lib.Name] = libDeps
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return deps, nil
}

func resolveDependencyLibrary(rawPath string, pathToLib map[string]*config.Library) *config.Library {
	cleanPath := filepath.ToSlash(filepath.Clean(rawPath))
	if lib, ok := pathToLib[cleanPath]; ok {
		return lib
	}
	base := filepath.Base(cleanPath)
	if lib, ok := pathToLib[base]; ok {
		return lib
	}
	if repo := SplitRepoName(cleanPath); repo != "" {
		if lib, ok := pathToLib[repo]; ok {
			return lib
		}
	}
	for dir, lib := range pathToLib {
		if strings.HasSuffix(cleanPath, "/"+dir) {
			return lib
		}
	}
	return nil
}

// topologicalLevels groups libraries into discrete dependency levels L0..Lk using Kahn's algorithm.
// Level 0 has no internal dependencies; level i+1 depends only on libraries in levels 0..i.
func topologicalLevels(libraries []*config.Library, deps map[string][]string) ([][]*config.Library, error) {
	libMap := make(map[string]*config.Library, len(libraries))
	for _, lib := range libraries {
		libMap[lib.Name] = lib
	}

	inDegree := make(map[string]int, len(libraries))
	dependents := make(map[string][]string, len(libraries))

	for _, lib := range libraries {
		var validDeps []string
		for _, depName := range deps[lib.Name] {
			if _, ok := libMap[depName]; ok {
				validDeps = append(validDeps, depName)
				dependents[depName] = append(dependents[depName], lib.Name)
			}
		}
		inDegree[lib.Name] = len(validDeps)
	}

	var currentLevel []*config.Library
	for _, lib := range libraries {
		if inDegree[lib.Name] == 0 {
			currentLevel = append(currentLevel, lib)
		}
	}
	slices.SortFunc(currentLevel, func(a, b *config.Library) int {
		return strings.Compare(a.Name, b.Name)
	})

	var levels [][]*config.Library
	processedCount := 0

	for len(currentLevel) > 0 {
		levels = append(levels, currentLevel)
		processedCount += len(currentLevel)

		var nextLevel []*config.Library
		for _, lib := range currentLevel {
			for _, dep := range dependents[lib.Name] {
				inDegree[dep]--
				if inDegree[dep] == 0 {
					nextLevel = append(nextLevel, libMap[dep])
				}
			}
		}
		slices.SortFunc(nextLevel, func(a, b *config.Library) int {
			return strings.Compare(a.Name, b.Name)
		})
		currentLevel = nextLevel
	}

	if processedCount < len(libraries) {
		var cycleLibs []string
		for _, lib := range libraries {
			if inDegree[lib.Name] > 0 {
				cycleLibs = append(cycleLibs, lib.Name)
			}
		}
		slices.Sort(cycleLibs)
		return nil, fmt.Errorf("cycle detected in Swift dependency graph among libraries: %s", strings.Join(cycleLibs, ", "))
	}

	return levels, nil
}
