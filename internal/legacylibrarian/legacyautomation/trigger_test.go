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

package legacyautomation

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v69/github"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygithub"
)

type mockGitHubClient struct {
	prs []*legacygithub.PullRequest
	err error
}

func (m *mockGitHubClient) FindMergedPullRequestsWithPendingReleaseLabel(ctx context.Context, owner, repo string) ([]*legacygithub.PullRequest, error) {
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
		ghPRs           []*legacygithub.PullRequest
		ghError         error
		wantTriggersRun []string
	}{
		{
			name:    "runs generate trigger",
			command: "generate",
			push:    true,
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
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "publish-release",
					Id:   "publish-release-trigger-id",
				},
			},
			ghPRs:           []*legacygithub.PullRequest{{HTMLURL: github.Ptr("https://github.com/googleapis/librarian/pull/1")}},
			wantTriggersRun: []string{"publish-release-trigger-id"},
		},
		{
			name:    "skips publish-release with no PRs",
			command: "publish-release",
			push:    true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "publish-release",
					Id:   "publish-release-trigger-id",
				},
			},
			ghPRs:           []*legacygithub.PullRequest{},
			wantTriggersRun: nil,
		},
		{
			name:    "error finding PRs for publish-release",
			command: "publish-release",
			push:    true,
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
		ghPRs             []*legacygithub.PullRequest
		ghError           error
		wantTriggersRun   []string
		wantSubstitutions []map[string]string
	}{
		{
			name:    "runs generate trigger with name",
			command: "generate",
			config: &RepositoriesConfig{
				ImageSHA: "test-sha",
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
				"_IMAGE_SHA":                "test-sha",
				"_PUSH":                     "true",
				"_BUILD":                    "true",
			}},
		},
		{
			name:    "runs generate trigger with full name",
			command: "generate",
			config: &RepositoriesConfig{
				ImageSHA: "test-sha",

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
				"_IMAGE_SHA":                "test-sha",
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
				ImageSHA: "test-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"stage-release"},
						SecretName:        "baz",
					},
				},
			},
			wantTriggersRun: []string{"stage-release-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "baz",
				"_IMAGE_SHA":                "test-sha",
				"_PUSH":                     "true",
			}},
		},
		{
			name:    "runs publish-release trigger",
			command: "publish-release",
			config: &RepositoriesConfig{
				ImageSHA: "test-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"publish-release"},
						SecretName:        "qux",
					},
				},
			},
			ghPRs:           []*legacygithub.PullRequest{{HTMLURL: github.Ptr("https://github.com/googleapis/google-cloud-python/pull/42")}},
			wantTriggersRun: []string{"publish-release-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "qux",
				"_IMAGE_SHA":                "test-sha",
				"_PUSH":                     "true",
				"_PR":                       "https://github.com/googleapis/google-cloud-python/pull/42",
			}},
		},
		{
			name:    "runs update-image trigger",
			command: "update-image",
			config: &RepositoriesConfig{
				ImageSHA: "test-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"update-image"},
						SecretName:        "quux",
					},
				},
			},
			wantTriggersRun: []string{"update-image-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "quux",
				"_IMAGE_SHA":                "test-sha",
				"_PUSH":                     "true",
				"_BUILD":                    "true",
			}},
		},
		{
			name:    "runs generate trigger on branch",
			command: "generate",
			config: &RepositoriesConfig{
				ImageSHA: "test-sha",
				Repositories: []*RepositoryConfig{
					{
						Name:              "google-cloud-python",
						SupportedCommands: []string{"generate"},
						Branch:            "preview",
						SecretName:        "foo",
					},
				},
			},
			wantTriggersRun: []string{"generate-trigger-id"},
			wantSubstitutions: []map[string]string{{
				"_REPOSITORY":               "google-cloud-python",
				"_FULL_REPOSITORY":          "https://github.com/googleapis/google-cloud-python",
				"_GITHUB_TOKEN_SECRET_NAME": "foo",
				"_PUSH":                     "true",
				"_IMAGE_SHA":                "test-sha",
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
			if diff := cmp.Diff(test.wantSubstitutions, client.substitutions); diff != "" {
				t.Errorf("runCommandWithConfig() substitutions diff (-want, +got):\n%s", diff)
			}
		})
	}
}
