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

package dart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	golangsemver "golang.org/x/mod/semver"
)

// PublishParams holds parameters for running the Publish function.
type PublishParams struct {
	// Config is the repository configuration.
	Config *config.Config
	// DryRun indicates whether to run publish with --dry-run.
	DryRun bool
	// DryRunKeepGoing indicates whether to run in dry-run mode without stopping on errors.
	DryRunKeepGoing bool
	// SkipSemverChecks indicates whether to skip semantic versioning checks.
	SkipSemverChecks bool
	// Verbose indicates whether to stream the output of executed commands.
	Verbose bool
	// IgnoredChanges is a list of file paths/patterns to ignore when detecting changed libraries.
	IgnoredChanges []string
}

// Publish finds all the libraries that should be published and publishes them in topological order.
func Publish(ctx context.Context, params PublishParams) error {
	if err := git.MatchesBranchPoint(ctx, command.Git, config.RemoteUpstream, config.BranchMain); err != nil {
		return err
	}

	libraryByName := make(map[string]*config.Library)
	for _, lib := range params.Config.Libraries {
		libraryByName[lib.Name] = lib
	}

	deps, repoVersions, err := getCloudDeps(ctx, params.Config.Libraries)
	if err != nil {
		return err
	}

	sorted, err := sortLibraries(libraryByName, deps)
	if err != nil {
		return err
	}

	var librariesToPublish []*config.Library
	for _, libName := range sorted {
		lib := libraryByName[libName]
		if lib.SkipRelease || lib.Version == "" {
			continue
		}
		/*
			libDir := libraryOutput(lib, params.Config.Default)

				pubspecPath := filepath.Join(libDir, "pubspec.yaml")
				if _, err := os.Stat(pubspecPath); err != nil {
					continue
				}*/

		repoVersion, ok := repoVersions[libName]
		if !ok {
			return fmt.Errorf("`dart pub get` did not return a version for %s", libName)
		}

		publishedVersion, err := getPublishedVersion(ctx, libName)
		if err != nil {
			return fmt.Errorf("failed to get published version of %s: %w", libName, err)
		}

		// The package hasn't been published yet.
		if publishedVersion == "" {
			librariesToPublish = append(librariesToPublish, lib)
			continue
		}

		comp := golangsemver.Compare("v"+publishedVersion, "v"+repoVersion)
		if comp > 0 {
			return fmt.Errorf("published version %q is greater than repo version %q for package %s", publishedVersion, repoVersion, libName)
		} else if comp < 0 {
			librariesToPublish = append(librariesToPublish, lib)
		}
	}

	if len(librariesToPublish) == 0 {
		return nil
	}

	for _, lib := range librariesToPublish {
		libDir := libraryOutput(lib, params.Config.Default)
		var args []string
		if params.DryRun || params.DryRunKeepGoing {
			args = []string{"pub", "publish", "--dry-run"}
		} else {
			args = []string{"pub", "publish", "--force"}
		}

		var runErr error
		if params.Verbose {
			runErr = command.RunStreamingInDir(ctx, libDir, command.Dart, args...)
		} else {
			runErr = command.RunInDir(ctx, libDir, command.Dart, args...)
		}

		if runErr != nil {
			if params.DryRunKeepGoing {
				fmt.Fprintf(os.Stderr, "Error publishing %s: %v\n", lib.Name, runErr)
				continue
			}
			return fmt.Errorf("failed to publish %s: %w", lib.Name, runErr)
		}
	}

	return nil
}

var pubdevAPIURL = "https://pub.dev/api/packages/"

func getPublishedVersion(ctx context.Context, libName string) (string, error) {
	apiURL := pubdevAPIURL + url.PathEscape(libName)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d from pub.dev: %s", resp.StatusCode, resp.Status)
	}

	var data struct {
		Latest struct {
			Version string `json:"version"`
		} `json:"latest"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.Latest.Version, nil
}

func getCloudDeps(ctx context.Context, libraries []*config.Library) (map[string][]string, map[string]string, error) {
	output, err := command.Output(ctx, command.Dart, "pub", "deps", "--json")
	if err != nil {
		return nil, nil, err
	}
	var data struct {
		Packages []struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			Dependencies []string `json:"dependencies"`
		} `json:"packages"`
	}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return nil, nil, fmt.Errorf("failed to parse dart pub deps output: %w", err)
	}

	libNames := make(map[string]bool)
	for _, lib := range libraries {
		libNames[lib.Name] = true
	}

	depsMap := make(map[string][]string)
	versionsMap := make(map[string]string)
	for _, pkg := range data.Packages {
		if !libNames[pkg.Name] {
			continue
		}
		versionsMap[pkg.Name] = pkg.Version
		var deps []string
		for _, dep := range pkg.Dependencies {
			if libNames[dep] {
				deps = append(deps, dep)
			}
		}
		slices.Sort(deps)
		depsMap[pkg.Name] = deps
	}

	return depsMap, versionsMap, nil
}

func sortLibraries(libraryByName map[string]*config.Library, deps map[string][]string) ([]string, error) {
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	for name := range libraryByName {
		pkgDeps := deps[name]
		inDegree[name] = len(pkgDeps)
		for _, dep := range pkgDeps {
			dependents[dep] = append(dependents[dep], name)
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	slices.Sort(queue)

	var sorted []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, curr)

		for _, dep := range dependents[curr] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
		slices.Sort(queue)
	}

	if len(sorted) < len(libraryByName) {
		return nil, fmt.Errorf("cycle detected in dependency DAG")
	}

	return sorted, nil
}

func libraryOutput(lib *config.Library, defaults *config.Default) string {
	if lib.Output != "" {
		return lib.Output
	}
	var defaultOut string
	if defaults != nil {
		defaultOut = defaults.Output
	}
	return filepath.Join(defaultOut, lib.Name)
}
