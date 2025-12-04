// Copyright 2025 Google LLC
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

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/googleapis/librarian/internal/fetch"
)

var latestCommitAndChecksum = fetch.LatestCommitAndChecksum

const (
	defaultGitHubAPI = "https://api.github.com"
	defaultGitHubDn  = "https://github.com"
	defaultRoot      = "googleapis"
)

// UpdateRootConfig updates the root configuration file with the latest SHA from GitHub.
func UpdateRootConfig(rootConfig *Config, rootName string) error {
	if rootName == "" {
		rootName = defaultRoot
	}
	endpoints := githubConfig(rootConfig)
	repo, err := githubRepoFromTarballLink(rootConfig, rootName)
	if err != nil {
		return err
	}

	latestSha, newSha256, err := latestCommitAndChecksum(endpoints, repo)
	if err != nil {
		return err
	}
	fmt.Printf("updating %s\n", configName)

	contents, err := os.ReadFile(configName)
	if err != nil {
		return err
	}
	newContents, err := updateRootConfigContents(rootName, contents, endpoints, repo, latestSha, newSha256)
	if err != nil {
		return err
	}
	return os.WriteFile(configName, newContents, 0644)
}

// githubConfig returns the GitHub API and download endpoints.
// In tests, these are replaced with a fake.
func githubConfig(rootConfig *Config) *fetch.Endpoints {
	api, ok := rootConfig.Source["github-api"]
	if !ok {
		api = defaultGitHubAPI
	}
	download, ok := rootConfig.Source["github"]
	if !ok {
		download = defaultGitHubDn
	}
	return &fetch.Endpoints{
		API:      api,
		Download: download,
	}
}

// githubRepoFromRoot extracts the gitHub account and repository (such as
// `googleapis/googleapis`, or `googleapis/google-cloud-rust`) from the tarball
// link.
func githubRepoFromTarballLink(rootConfig *Config, rootName string) (*fetch.Repo, error) {
	config := githubConfig(rootConfig)
	root, ok := rootConfig.Source[fmt.Sprintf("%s-root", rootName)]
	if !ok {
		return nil, fmt.Errorf("missing %s root configuration", rootName)
	}
	return fetch.RepoFromTarballLink(config.Download, root)
}

func updateRootConfigContents(rootName string, contents []byte, endpoints *fetch.Endpoints, repo *fetch.Repo, latestSha, newSha256 string) ([]byte, error) {
	newLink := fetch.TarballLink(endpoints.Download, repo, latestSha)

	var output strings.Builder
	updatedRoot := 0
	updatedSha256 := 0
	updatedExtractedName := 0
	lines := strings.Split(string(contents), "\n")
	for idx, line := range lines {
		switch {
		case strings.HasPrefix(line, fmt.Sprintf("%s-root ", rootName)):
			s := strings.SplitN(line, "=", 2)
			if len(s) != 2 {
				return nil, fmt.Errorf("invalid %s-root line, expected = separator, got=%q", rootName, line)
			}
			fmt.Fprintf(&output, "%s= '%s'\n", s[0], newLink)
			updatedRoot += 1
		case strings.HasPrefix(line, fmt.Sprintf("%s-sha256 ", rootName)):
			s := strings.SplitN(line, "=", 2)
			if len(s) != 2 {
				return nil, fmt.Errorf("invalid %s-sha256 line, expected = separator, got=%q", rootName, line)
			}
			fmt.Fprintf(&output, "%s= '%s'\n", s[0], newSha256)
			updatedSha256 += 1
		case strings.HasPrefix(line, fmt.Sprintf("%s-extracted-name ", rootName)):
			s := strings.SplitN(line, "=", 2)
			if len(s) != 2 {
				return nil, fmt.Errorf("invalid %s-extracted-name line, expected = separator, got=%q", rootName, line)
			}
			fmt.Fprintf(&output, "%s= '%s-%s'\n", s[0], repo.Repo, latestSha)
			updatedExtractedName += 1
		default:
			if idx != len(lines)-1 {
				fmt.Fprintf(&output, "%s\n", line)
			} else {
				fmt.Fprintf(&output, "%s", line)
			}
		}
	}
	newContents := output.String()
	if updatedRoot == 0 && updatedSha256 == 0 {
		return []byte(newContents), nil
	}
	if updatedRoot != 1 || updatedSha256 != 1 || updatedExtractedName > 1 {
		return nil, fmt.Errorf("too many changes to Root or Sha256 for %s", rootName)
	}
	return []byte(newContents), nil
}
