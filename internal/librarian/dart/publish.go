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
	"log/slog"
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
}

// Publish finds all the libraries that should be published and publishes them in topological order.
func Publish(ctx context.Context, params PublishParams) error {
	if err := git.MatchesBranchPoint(ctx, command.Git, config.RemoteUpstream, config.BranchMain); err != nil {
		if params.DryRunKeepGoing {
			slog.Warn("Branch point check failed, but continuing due to --keep-going", "error", err)
		} else {
			return err
		}
	}

	libraryByName := make(map[string]*config.Library)
	for _, lib := range params.Config.Libraries {
		libraryByName[lib.Name] = lib
	}

	deps, repoVersions, err := getCloudDeps(ctx, params.Config.Libraries)
	if err != nil {
		return err
	}

	sortedLibraryNames, err := sortByDeps(libraryByName, deps)
	if err != nil {
		return err
	}

	publishedVersions := make(map[string]string, len(libraryByName))
	var librariesToPublish []*config.Library
	for _, libName := range sortedLibraryNames {
		lib := libraryByName[libName]
		if lib.SkipRelease || lib.Version == "" {
			continue
		}

		repoVersion, ok := repoVersions[libName]
		if !ok {
			return fmt.Errorf("no local version for %s", libName)
		}

		publishedVersion, err := getPublishedVersion(ctx, libName)
		if err != nil {
			return fmt.Errorf("failed to get published version of %s: %w", libName, err)
		}

		if publishedVersion == "" {
			librariesToPublish = append(librariesToPublish, lib)
			continue
		}

		comp := golangsemver.Compare("v"+publishedVersion, "v"+repoVersion)
		if comp > 0 {
			if params.DryRunKeepGoing {
				slog.Warn("published version greater than repo version, but continuing due to --keep-going",
					"package", libName, "published", publishedVersion, "repo", repoVersion)
			} else {
				return fmt.Errorf("published version %q is greater than repo version %q for package %s",
					publishedVersion, repoVersion, libName)
			}
		} else if comp < 0 {
			librariesToPublish = append(librariesToPublish, lib)
			publishedVersions[libName] = repoVersion
		}
	}

	if len(librariesToPublish) == 0 {
		return nil
	}

	if !params.SkipSemverChecks {
		for _, lib := range librariesToPublish {
			if _, ok := publishedVersions[lib.Name]; ok {
				if err := runSemverCheck(ctx, lib, params.Config.Default); err != nil {
					if params.DryRunKeepGoing {
						slog.Warn("semver check failed but continuing due to --keep-going",
							"package", lib.Name, "error", err)
					} else {
						return err
					}
				}
			}
		}
	}

	for _, lib := range librariesToPublish {
		libDir := libraryOutput(lib, params.Config.Default)
		var args []string
		if params.DryRun || params.DryRunKeepGoing {
			args = []string{"pub", "publish", "--skip-validation", "--dry-run"}
		} else {
			args = []string{"pub", "publish", "--skip-validation", "--force"}
		}

		var runErr error
		if params.Verbose {
			runErr = command.RunStreamingInDir(ctx, libDir, command.Dart, args...)
		} else {
			runErr = command.RunInDir(ctx, libDir, command.Dart, args...)
		}

		if runErr != nil {
			if params.DryRunKeepGoing {
				slog.Warn("publishing failed but continuing due to --keep-going",
					"package", lib.Name, "error", runErr)
				continue
			}
			return fmt.Errorf("failed to publish %s: %w", lib.Name, runErr)
		}
	}

	return nil
}

func runSemverCheck(ctx context.Context, lib *config.Library, defaultCfg *config.Default) error {
	libDir := libraryOutput(lib, defaultCfg)
	reportPath := filepath.Join(os.TempDir(), fmt.Sprintf("report-%s-publish.json", lib.Name))
	output, err := command.Output(ctx, "dart-apitool", "diff", "--old", "pub://"+lib.Name, "--new", libDir,
		"--report-format", "json", "--report-file-path", reportPath)
	if err != nil {
		return fmt.Errorf("dart-apitool failed for %s: %w (output: %s, see %s)", lib.Name, err, output, reportPath)
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

func sortByDeps(libraryByName map[string]*config.Library, deps map[string][]string) ([]string, error) {
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
