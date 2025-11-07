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
	"strings"
	"testing"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v69/github"
	"github.com/googleapis/librarian/internal/config"
	"google.golang.org/api/option"
)

func TestNewPublishRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		cfg           *config.Config
		clientFactory func(ctx context.Context, opts ...option.ClientOption) (*cloudbuild.Client, error)
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name: "create_a_runner",
			cfg: &config.Config{
				ForceRun: true,
				Project:  "example-project",
				Push:     true,
			},
			clientFactory: func(ctx context.Context, opts ...option.ClientOption) (*cloudbuild.Client, error) {
				// Force WithoutAuthentication for these tests
				return cloudbuild.NewClient(ctx, option.WithoutAuthentication())
			},
		},
		{
			name: "create_cloud_build_client_failed",
			cfg: &config.Config{
				ForceRun: true,
				Project:  "example-project",
				Push:     true,
			},
			clientFactory: func(ctx context.Context, opts ...option.ClientOption) (*cloudbuild.Client, error) {
				// Force WithoutAuthentication for these tests
				return nil, errors.New("simulated error")
			},
			wantErr:    true,
			wantErrMsg: "error creating cloudbuild client",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Do not run in parallel because tests are changing the global package-level variable clientFactory.
			ctx := t.Context()
			clientFactory = test.clientFactory
			runner, err := newPublishRunner(ctx, test.cfg)
			if test.wantErr {
				if err == nil {
					t.Errorf("newPublishRunner() want error but got nil")
					return
				}

				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("want error message: %q, got %q", test.wantErrMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("newPublishRunner() got error: %v", err)
				return
			}

			if runner.cloudBuildClient == nil {
				t.Errorf("newPublishRunner() cloudBuildClient is not set")
			}
			if runner.ghClient == nil {
				t.Errorf("newPublishRunner() ghClient is not set")
			}
			if runner.forceRun != test.cfg.ForceRun {
				t.Errorf("newPublishRunner() forceRun is not set")
			}
			if runner.projectID != test.cfg.Project {
				t.Errorf("newPublishRunner() projectID is not set")
			}
			if runner.push != test.cfg.Push {
				t.Errorf("newPublishRunner() push is not set")
			}
		})
	}
}

func TestPublishRunnerRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		push               bool
		forceRun           bool
		repositoriesConfig *RepositoriesConfig
		want               string
		runError           error
		wantErr            bool
		buildTriggers      []*cloudbuildpb.BuildTrigger
		ghPRs              []*github.PullRequest
		ghError            error
		wantTriggersRun    []string
	}{
		{
			name:     "runs publish-release trigger",
			push:     true,
			forceRun: true,
			repositoriesConfig: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						SupportedCommands: []string{"publish-release"},
						Name:              "librarian",
					},
				},
			},
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "publish-release",
					Id:   "publish-release-trigger-id",
				},
			},
			ghPRs:           []*github.PullRequest{{HTMLURL: github.Ptr("https://github.com/googleapis/librarian/pull/1")}},
			wantTriggersRun: []string{"publish-release-trigger-id"},
		},
		{
			name:     "skips publish-release with no PRs",
			push:     true,
			forceRun: true,
			repositoriesConfig: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						SupportedCommands: []string{"publish-release"},
						Name:              "librarian",
					},
				},
			},
			ghPRs:           []*github.PullRequest{},
			wantTriggersRun: nil,
		},
		{
			name:     "error finding PRs for publish-release",
			push:     true,
			forceRun: true,
			repositoriesConfig: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						SupportedCommands: []string{"publish-release"},
						Name:              "librarian",
					},
				},
			},
			wantErr:         true,
			ghError:         fmt.Errorf("github error"),
			wantTriggersRun: nil,
		},
		{
			name:     "error finding repository",
			push:     true,
			forceRun: true,
			repositoriesConfig: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						SupportedCommands: []string{"publish-release"},
					},
				},
			},
			wantErr:         true,
			wantTriggersRun: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			cloudBuildClient := &mockCloudBuildClient{
				runError:      test.runError,
				buildTriggers: test.buildTriggers,
			}
			ghClient := &mockGitHubClient{
				prs: test.ghPRs,
				err: test.ghError,
			}
			runner := &publishRunner{
				cloudBuildClient: cloudBuildClient,
				ghClient:         ghClient,
				forceRun:         test.forceRun,
				repoConfig:       test.repositoriesConfig,
				projectID:        "some-project",
				push:             test.push,
			}
			err := runner.run(ctx)
			if test.wantErr && err == nil {
				t.Fatal("expected error, but did not return one")
			} else if !test.wantErr && err != nil {
				t.Errorf("did not expect error, but received one: %s", err)
			}
			if diff := cmp.Diff(test.wantTriggersRun, cloudBuildClient.triggersRun); diff != "" {
				t.Errorf("Run() triggersRun diff (-want, +got):\n%s", diff)
			}
		})
	}
}
