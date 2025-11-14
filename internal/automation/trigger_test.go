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
	"fmt"
	"testing"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v69/github"
)

type mockGitHubClient struct {
	prs []*github.PullRequest
	err error
}

func (m *mockGitHubClient) FindMergedPullRequestsWithPendingReleaseLabel(ctx context.Context, owner, repo string) ([]*github.PullRequest, error) {
	return m.prs, m.err
}

func TestRunCommandWithClient(t *testing.T) {
	for _, test := range []struct {
		name            string
		command         string
		push            bool
		build           bool
		want            string
		runError        error
		wantErr         bool
		buildTriggers   []*cloudbuildpb.BuildTrigger
		ghPRs           []*github.PullRequest
		ghError         error
		wantTriggersRun []string
	}{
		{
			name:    "runs generate trigger",
			command: "generate",
			push:    true,
			build:   false,
			wantErr: false,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "generate",
					Id:   "generate-trigger-id",
				},
				{
					Name: "prepare-release",
					Id:   "prepare-release-trigger-id",
				},
			},
			wantTriggersRun: []string{"generate-trigger-id"},
		},
		{
			name:    "runs prepare-release trigger",
			command: "stage-release",
			push:    true,
			build:   false,
			wantErr: false,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "generate",
					Id:   "generate-trigger-id",
				},
				{
					Name: "stage-release",
					Id:   "stage-release-trigger-id",
				},
			},
			wantTriggersRun: []string{"stage-release-trigger-id"},
		},
		{
			name:    "invalid command",
			command: "invalid-command",
			push:    true,
			build:   false,
			wantErr: true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "generate",
					Id:   "generate-trigger-id",
				},
				{
					Name: "stage-release",
					Id:   "stage-release-trigger-id",
				},
			},
			wantTriggersRun: nil,
		},
		{
			name:     "error triggering",
			command:  "generate",
			push:     true,
			build:    false,
			runError: fmt.Errorf("some-error"),
			wantErr:  true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "generate",
					Id:   "generate-trigger-id",
				},
				{
					Name: "stage-release",
					Id:   "stage-release-trigger-id",
				},
			},
			wantTriggersRun: nil,
		},
		{
			name:    "runs publish-release trigger",
			command: "publish-release",
			push:    true,
			build:   false,
			wantErr: false,
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
			name:    "skips publish-release with no PRs",
			command: "publish-release",
			push:    true,
			build:   false,
			wantErr: false,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "publish-release",
					Id:   "publish-release-trigger-id",
				},
			},
			ghPRs:           []*github.PullRequest{},
			wantTriggersRun: nil,
		},
		{
			name:    "error finding PRs for publish-release",
			command: "publish-release",
			push:    true,
			build:   false,
			wantErr: true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "publish-release",
					Id:   "publish-release-trigger-id",
				},
			},
			ghError:         fmt.Errorf("github error"),
			wantTriggersRun: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			client := &mockCloudBuildClient{
				runError:      test.runError,
				buildTriggers: test.buildTriggers,
			}
			ghClient := &mockGitHubClient{
				prs: test.ghPRs,
				err: test.ghError,
			}
			err := runCommandWithClient(ctx, client, ghClient, test.command, "some-project", test.push, test.build)
			if test.wantErr && err == nil {
				t.Fatal("expected error, but did not return one")
			} else if !test.wantErr && err != nil {
				t.Errorf("did not expect error, but received one: %s", err)
			}
			if diff := cmp.Diff(test.wantTriggersRun, client.triggersRun); diff != "" {
				t.Errorf("runCommandWithClient() triggersRun diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestRunCommandWithConfig(t *testing.T) {
	var buildTriggers = []*cloudbuildpb.BuildTrigger{
		{
			Name: "generate",
			Id:   "generate-trigger-id",
		},
		{
			Name: "stage-release",
			Id:   "stage-release-trigger-id",
		},
		{
			Name: "publish-release",
			Id:   "publish-release-trigger-id",
		},
		{
			Name: "update-image",
			Id:   "update-image-trigger-id",
		},
		{
			Name: "prepare-release",
			Id:   "prepare-release-trigger-id",
		},
	}
	for _, test := range []struct {
		name              string
		command           string
		config            *RepositoriesConfig
		want              string
		runError          error
		wantErr           bool
		ghPRs             []*github.PullRequest
		ghError           error
		wantTriggersRun   []string
		wantSubstitutions []map[string]string
	}{
		{
			name:    "runs generate trigger with name",
			command: "generate",
			config: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"generate"},
						SecretName:        "foo",
					},
				},
			},
			wantErr:         false,
			wantTriggersRun: []string{"generate-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "foo",
				"_PUSH":                     "true",
				"_BUILD":                    "true",
			}},
		},
		{
			name:    "runs generate trigger with full name",
			command: "generate",
			config: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						FullName:          "https://github.com/googleapis/google-cloud-python",
						SupportedCommands: []string{"generate"},
						SecretName:        "bar",
					},
				},
			},
			wantErr:         false,
			wantTriggersRun: []string{"generate-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "bar",
				"_PUSH":                     "true",
				"_BUILD":                    "true",
			}},
		},
		{
			name:    "runs generate trigger without name",
			command: "generate",
			config: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						SupportedCommands: []string{"generate"},
					},
				},
			},
			wantErr:         true,
			wantTriggersRun: nil,
		},
		{
			name:    "runs stage-release trigger",
			command: "stage-release",
			config: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"stage-release"},
						SecretName:        "baz",
					},
				},
			},
			wantErr:         false,
			wantTriggersRun: []string{"stage-release-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "baz",
				"_PUSH":                     "true",
			}},
		},
		{
			name:    "runs publish-release trigger",
			command: "publish-release",
			config: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"publish-release"},
						SecretName:        "qux",
					},
				},
			},
			ghPRs:           []*github.PullRequest{{HTMLURL: github.Ptr("https://github.com/googleapis/google-cloud-python/pull/42")}},
			wantErr:         false,
			wantTriggersRun: []string{"publish-release-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "qux",
				"_PUSH":                     "true",
				"_PR":                       "https://github.com/googleapis/google-cloud-python/pull/42",
			}},
		},
		{
			name:    "runs update-image trigger",
			command: "update-image",
			config: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"update-image"},
						SecretName:        "quux",
					},
				},
			},
			wantErr:         false,
			wantTriggersRun: []string{"update-image-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "quux",
				"_PUSH":                     "true",
				"_BUILD":                    "true",
			}},
		},
		{
			name:    "runs generate trigger on branch",
			command: "generate",
			config: &RepositoriesConfig{
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"generate"},
						Branch:            "preview",
						SecretName:        "foo",
					},
				},
			},
			wantErr:         false,
			wantTriggersRun: []string{"generate-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "foo",
				"_PUSH":                     "true",
				"_BRANCH":                   "preview",
				"_BUILD":                    "true",
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			client := &mockCloudBuildClient{
				runError:      test.runError,
				buildTriggers: buildTriggers,
			}
			ghClient := &mockGitHubClient{
				prs: test.ghPRs,
				err: test.ghError,
			}
			err := runCommandWithConfig(ctx, client, ghClient, test.command, "some-project", true, true, test.config)
			if test.wantErr && err == nil {
				t.Fatal("expected error, but did not return one")
			} else if !test.wantErr && err != nil {
				t.Errorf("did not expect error, but received one: %s", err)
			}
			if diff := cmp.Diff(test.wantTriggersRun, client.triggersRun); diff != "" {
				t.Errorf("runCommandWithConfig() triggersRun diff (-want, +got):\n%s", diff)
			}
			if test.wantSubstitutions != nil {
				if diff := cmp.Diff(test.wantSubstitutions, client.substitutions); diff != "" {
					t.Errorf("runCommandWithConfig() substitutions diff (-want, +got):\n%s", diff)
				}
			}
		})
	}
}
