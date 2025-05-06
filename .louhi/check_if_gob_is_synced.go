package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// GerritConfig holds the Gerrit API configuration
type GerritConfig struct {
	BaseURL string
	token   string
}

// GitConfig holds the local Git repository path
type GitConfig struct {
	RepoPath string
}

// CommitInfo represents the structure of a Gerrit commit object (simplified)
type CommitInfo struct {
	Commit string `json:"commit"`
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: go run check_if_gob_is_synced.go <gerrit_repo_url> <token> <git_repo_path>")
		os.Exit(1)
	}

	gerritRepoURL := os.Args[1]
	gerritAuthToken := os.Args[2]
	gitRepoPath := os.Args[3]

	// Configure Gerrit access (you might want to get these from environment variables or a config file)
	gerritConfig := GerritConfig{
		BaseURL: strings.TrimSuffix(gerritRepoURL, "/"), // Ensure no trailing slash
		token:   gerritAuthToken,
	}

	// Configure Git repository path
	gitConfig := GitConfig{
		RepoPath: gitRepoPath,
	}

	// Fetch the latest commit hash from the Git repository
	latestCommitHash, err := getLatestGitCommit(gitConfig)
	if err != nil {
		fmt.Printf("Error getting latest Git commit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Latest Git commit hash: %s\n", latestCommitHash)

	// Check if the commit exists in the Gerrit repository
	exists, err := checkCommitExistsInGerrit(gerritConfig, latestCommitHash)
	if err != nil {
		fmt.Printf("Error checking commit in Gerrit: %v\n", err)
		os.Exit(1)
	}

	if exists {
		fmt.Printf("Commit '%s' exists in the Gerrit repository.\n", latestCommitHash)
	} else {
		fmt.Printf("Commit '%s' does NOT exist in the Gerrit repository.\n", latestCommitHash)
	}
}

// getLatestGitCommit executes a Git command to get the latest commit hash.
func getLatestGitCommit(config GitConfig) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = config.RepoPath
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running git rev-parse: %v", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// checkCommitExistsInGerrit uses the Gerrit API to check if a commit exists.
func checkCommitExistsInGerrit(config GerritConfig, commitHash string) (bool, error) {
	url := fmt.Sprintf("%s/changes/%s", config.BaseURL, commitHash)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating HTTP request: %v", err)
	}

	if config.Username != "" && config.Password != "" {
		req.Header.Add("Authorization", "Bearer "+config.token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making HTTP request to Gerrit: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// The commit likely exists. We can try to decode a minimal response to confirm.
		var commitInfo CommitInfo
		err = json.NewDecoder(resp.Body).Decode(&commitInfo)
		if err == nil && commitInfo.Commit == commitHash {
			return true, nil
		}
		// Even if decoding fails, a 200 OK suggests the commit is there in some form.
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("Gerrit API returned unexpected status: %d - %s", resp.StatusCode, string(bodyBytes))
	}
}
