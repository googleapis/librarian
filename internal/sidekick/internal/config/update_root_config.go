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
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	defaultGitHubApi = "https://api.github.com"
	defaultGitHubDn  = "https://github.com"
	branch           = "master"
	defaultRoot      = "googleapis"
)

// githubEndpoints defines the endpoints used to access GitHub.
type githubEndpoints struct {
	// Api defines the endpoint used to make API calls.
	Api string
	// Download defines the endpoint to download tarballs.
	Download string
}

// githubRepo represents a GitHub repository name.
type githubRepo struct {
	// Org defines the GitHub organization (or user), that owns the repository.
	Org string
	// Repo is the name of the repository, such as `googleapis` or `google-cloud-rust`.
	Repo string
}

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

	query := fmt.Sprintf("%s/repos/%s/%s/commits/%s", endpoints.Api, repo.Org, repo.Repo, branch)
	fmt.Printf("getting latest SHA from %q\n", query)
	latestSha, err := getLatestSha(query)
	if err != nil {
		return err
	}

	newLink := newTarballLink(endpoints, repo, latestSha)
	fmt.Printf("computing SHA256 for %q\n", newLink)
	newSha256, err := getSha256(newLink)
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

// githubConfig returns the API endpoint the browser endpoint for GitHub.
// In tests, these are replaced with a fake.
func githubConfig(rootConfig *Config) *githubEndpoints {
	api, ok := rootConfig.Source["github-api"]
	if !ok {
		api = defaultGitHubApi
	}
	download, ok := rootConfig.Source["github"]
	if !ok {
		download = defaultGitHubDn
	}
	return &githubEndpoints{
		Api:      api,
		Download: download,
	}
}

// githubRepoFromRoot extracts the gitHub account and repository (such as
// `googleapis/googleapis`, or `googleapis/google-cloud-rust`) from the tarball
// link.
func githubRepoFromTarballLink(rootConfig *Config, rootName string) (*githubRepo, error) {
	config := githubConfig(rootConfig)
	root, ok := rootConfig.Source[fmt.Sprintf("%s-root", rootName)]
	if !ok {
		return nil, fmt.Errorf("missing %s root configuration", rootName)
	}
	urlPath := strings.TrimPrefix(root, config.Download)
	urlPath = strings.TrimPrefix(urlPath, "/")
	components := strings.Split(urlPath, "/")
	if len(components) < 2 {
		return nil, fmt.Errorf("url path for %s root configuration is missing components", rootName)
	}
	repo := &githubRepo{
		Org:  components[0],
		Repo: components[1],
	}
	return repo, nil
}

func newTarballLink(endpoints *githubEndpoints, repo *githubRepo, latestSha string) string {
	return fmt.Sprintf("%s/%s/%s/archive/%s.tar.gz", endpoints.Download, repo.Org, repo.Repo, latestSha)
}

func updateRootConfigContents(rootName string, contents []byte, endpoints *githubEndpoints, repo *githubRepo, latestSha, newSha256 string) ([]byte, error) {
	newLink := newTarballLink(endpoints, repo, latestSha)

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

func getSha256(query string) (string, error) {
	response, err := http.Get(query)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, response.Body); err != nil {
		return "", err
	}
	got := fmt.Sprintf("%x", hasher.Sum(nil))
	return got, nil
}

func getLatestSha(query string) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, query, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github.VERSION.sha")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}
