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

// Publish all the libraries that should be published.
//
// The algorithm is:
//  1. Find the dependencies of all packages in the repository.
//  2. Order the packages so that no package appears before any of its dependencies.
//  3. For each package in order, check if it should be published:
//     a. If it has not been published before, it should be published.
//     b. If it has been published before, check if the published version is less than the
//     repository version. If it is, it should be published.
//  4. If the package is being published, check for semver violations using dart-apitool
//     (https://github.com/bmw-tech/dart_apitool).
//  5. For each package in order that should be published, run `dart pub publish`.
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

	deps, repoVersions, err := getDeps(ctx, params.Config.Libraries)
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
				if err := runSemverCheck(ctx, lib, params.Config.Default, params.Verbose); err != nil {
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

func runSemverCheck(ctx context.Context, lib *config.Library, defaultCfg *config.Default, verbose bool) error {
	libDir := libraryOutput(lib, defaultCfg)
	args := []string{"diff", "--old", "pub://" + lib.Name, "--new", "."}

	var runErr error
	if verbose {
		runErr = command.RunStreamingInDir(ctx, libDir, command.DartAPITool, args...)
	} else {
		runErr = command.RunInDir(ctx, libDir, command.DartAPITool, args...)

	}
	if runErr != nil {
		return fmt.Errorf("dart-apitool failed for %s: %w", lib.Name, runErr)
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
