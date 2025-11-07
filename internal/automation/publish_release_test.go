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
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v69/github"
)

func TestPublishRunnerRun(t *testing.T) {
	for _, test := range []struct {
		name            string
		command         string
		push            bool
		forceRun        bool
		want            string
		runError        error
		wantErr         bool
		buildTriggers   []*cloudbuildpb.BuildTrigger
		ghPRs           []*github.PullRequest
		ghError         error
		wantTriggersRun []string
	}{
		{
			name:     "runs publish-release trigger",
			command:  "publish-release",
			push:     true,
			forceRun: true,
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
			name:            "skips publish-release with no PRs",
			command:         "publish-release",
			push:            true,
			forceRun:        true,
			ghPRs:           []*github.PullRequest{},
			wantTriggersRun: nil,
		},
		{
			name:            "error finding PRs for publish-release",
			command:         "publish-release",
			push:            true,
			forceRun:        true,
			wantErr:         true,
			ghError:         fmt.Errorf("github error"),
			wantTriggersRun: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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
				repoConfig: &RepositoriesConfig{
					Repositories: []*RepositoryConfig{
						{
							SupportedCommands: []string{"publish-release"},
							Name:              "librarian",
						},
					},
				},
				projectID: "some-project",
				push:      test.push,
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

func TestShouldRun(t *testing.T) {
	for _, test := range []struct {
		name     string
		dateTime time.Time
		forceRun bool
		want     bool
	}{
		{
			name: "skip_in_even_week",
			// Jan 8, 2025 is in week 2 (even)
			dateTime: time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
			want:     true,
		},
		{
			name: "run_in_even_week",
			// Jan 8, 2025 is in week 2 (even)
			dateTime: time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
			forceRun: true,
		},
		{
			name: "run_in_odd_week",
			// Jan 1, 2025 is in week 1 (odd)
			dateTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "run_in_odd_week_force_run",
			// Jan 1, 2025 is in week 1 (odd)
			dateTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			forceRun: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := shouldSkip(test.dateTime, test.forceRun)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("shouldSkip() diff (-want, +got):\n%s", diff)
			}
		})
	}
}
