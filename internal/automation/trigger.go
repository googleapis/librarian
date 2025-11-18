// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package automation

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"strings"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
)

var triggerNameByCommandName = map[string]string{
	"generate":        "generate",
	"stage-release":   "stage-release",
	"publish-release": "publish-release",
	"update-image":    "update-image",
}

var defaultAPISource = &APISourceRepo{
	Name:   "googleapis",
	Branch: "master",
}

const region = "global"

// GitHubClient handles communication with the GitHub API.
type GitHubClient interface {
	FindMergedPullRequestsWithPendingReleaseLabel(ctx context.Context, owner, repo string) ([]*github.PullRequest, error)
}

type wrappedCloudBuildClient struct {
	client *cloudbuild.Client
}

// RunBuildTrigger executes the RPC to trigger a Cloud Build trigger.
func (c *wrappedCloudBuildClient) RunBuildTrigger(ctx context.Context, req *cloudbuildpb.RunBuildTriggerRequest, opts ...gax.CallOption) error {
	resp, err := c.client.RunBuildTrigger(ctx, req, opts...)
	if err != nil {
		return err
	}

	slog.Debug("triggered", slog.String("LRO Name", resp.Name()))
	return err
}

// ListBuildTriggers executes the RPC to list Cloud Build triggers.
func (c *wrappedCloudBuildClient) ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) iter.Seq2[*cloudbuildpb.BuildTrigger, error] {
	return c.client.ListBuildTriggers(ctx, req, opts...).All()
}

// RunCommand triggers a command for each registered repository that supports it.
func RunCommand(ctx context.Context, command string, projectId string, push bool, build bool) error {
	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating cloudbuild client: %w", err)
	}
	defer c.Close()
	wrappedClient := &wrappedCloudBuildClient{
		client: c,
	}
	ghClient := github.NewClient(os.Getenv(config.LibrarianGithubToken), nil)
	return runCommandWithClient(ctx, wrappedClient, ghClient, command, projectId, push, build)
}

func runCommandWithClient(ctx context.Context, client CloudBuildClient, ghClient GitHubClient, command string, projectId string, push bool, build bool) error {
	config, err := loadRepositoriesConfig()
	if err != nil {
		slog.Error("error loading repositories config", slog.Any("err", err))
		return err
	}
	return runCommandWithConfig(ctx, client, ghClient, command, projectId, push, build, config)
}

func runCommandWithConfig(ctx context.Context, client CloudBuildClient, ghClient GitHubClient, command string, projectId string, push bool, build bool, config *RepositoriesConfig) error {
	// validate command is allowed
	triggerName := triggerNameByCommandName[command]
	if triggerName == "" {
		return fmt.Errorf("unsupported command: %s", command)
	}

	errs := make([]error, 0)

	repositories := config.RepositoriesForCommand(command)
	for _, repository := range repositories {
		slog.Debug("running command", "command", command, "repository", repository.Name)

		gitUrl, err := repository.GitURL()
		if err != nil {
			slog.Error("repository has no configured git url", slog.Any("repository", repository))
			return err
		}

		apiSource := repository.APISourceRepo
		if apiSource == nil {
			apiSource = defaultAPISource
		}

		apiSourceGitUrl, err := apiSource.GitURL()
		if err != nil {
			slog.Error("repository API source has no configured git url", slog.Any("repository", repository))
			return err
		}

		apiSourceBranch := apiSource.Branch
		if apiSourceBranch == "" {
			apiSourceBranch = defaultAPISource.Branch
		}

		substitutions := map[string]string{
			"_REPOSITORY":               repository.Name,
			"_FULL_REPOSITORY":          gitUrl,
			"_GITHUB_TOKEN_SECRET_NAME": repository.SecretName,
			"_PUSH":                     fmt.Sprintf("%v", push),
			"_API_SOURCE_REPOSITORY":    apiSourceGitUrl,
			"_API_SOURCE_BRANCH":        apiSourceBranch,
		}
		if repository.Branch != "" {
			substitutions["_BRANCH"] = repository.Branch
		}

		if command == "publish-release" {
			parts := strings.Split(gitUrl, "/")
			repositoryOwner := parts[len(parts)-2]
			prs, err := ghClient.FindMergedPullRequestsWithPendingReleaseLabel(ctx, repositoryOwner, repository.Name)
			if err != nil {
				slog.Error("error finding merged pull requests for publish-release", slog.Any("err", err), slog.String("repository", repository.Name))
				errs = append(errs, err)
				continue
			}
			if len(prs) == 0 {
				slog.Info("no pull requests with label 'release:pending' found. Skipping 'publish-release' trigger.", slog.String("repository", repository.Name))
				continue
			} else {
				substitutions["_PR"] = fmt.Sprintf("%v", prs[0].GetHTMLURL())
			}
		} else if command == "generate" || command == "update-image" {
			substitutions["_BUILD"] = fmt.Sprintf("%v", build)
		}
		err = runCloudBuildTriggerByName(ctx, client, projectId, region, triggerName, substitutions)
		if err != nil {
			slog.Error("error triggering cloudbuild", slog.Any("err", err))
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
