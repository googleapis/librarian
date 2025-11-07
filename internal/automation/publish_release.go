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
	"log/slog"
	"os"
	"strings"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"
	"google.golang.org/api/option"
)

const (
	publishCmdName = "publish-release"
)

type publishRunner struct {
	cloudBuildClient CloudBuildClient
	ghClient         GitHubClient
	repoConfig       *RepositoriesConfig
	forceRun         bool
	projectID        string
	push             bool
}

var clientFactory = func(ctx context.Context, opts ...option.ClientOption) (*cloudbuild.Client, error) {
	return cloudbuild.NewClient(ctx, opts...)
}

func newPublishRunner(ctx context.Context, cfg *config.Config) (*publishRunner, error) {
	client, err := clientFactory(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating cloudbuild client: %w", err)
	}
	wrappedClient := &wrappedCloudBuildClient{
		client: client,
	}
	ghClient := github.NewClient(os.Getenv(config.LibrarianGithubToken), nil)
	repoConfig, err := loadRepositoriesConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading repositories config: %w", err)
	}
	return &publishRunner{
		cloudBuildClient: wrappedClient,
		ghClient:         ghClient,
		repoConfig:       repoConfig,
		forceRun:         cfg.ForceRun,
		projectID:        cfg.Project,
		push:             cfg.Push,
	}, nil
}

func (r *publishRunner) run(ctx context.Context) error {
	errs := make([]error, 0)
	repositories := r.repoConfig.RepositoriesForCommand(publishCmdName)
	for _, repository := range repositories {
		slog.Debug("running command", "command", publishCmdName, "repository", repository.Name)
		gitUrl, err := repository.GitURL()
		if err != nil {
			slog.Error("repository has no configured git url", slog.Any("repository", repository))
			return err
		}

		substitutions := map[string]string{
			"_REPOSITORY":               repository.Name,
			"_FULL_REPOSITORY":          gitUrl,
			"_GITHUB_TOKEN_SECRET_NAME": repository.SecretName,
			"_PUSH":                     fmt.Sprintf("%v", r.push),
		}
		parts := strings.Split(gitUrl, "/")
		repositoryOwner := parts[len(parts)-2]
		prs, err := r.ghClient.FindMergedPullRequestsWithPendingReleaseLabel(ctx, repositoryOwner, repository.Name)
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
		err = runCloudBuildTriggerByName(ctx, r.cloudBuildClient, r.projectID, region, publishCmdName, substitutions)
		if err != nil {
			slog.Error("error triggering cloudbuild", slog.Any("err", err))
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
