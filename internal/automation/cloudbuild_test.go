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
	"iter"
	"testing"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/gax-go/v2"
	"golang.org/x/exp/slog"
)

type mockCloudBuildClient struct {
	runError      error
	buildTriggers []*cloudbuildpb.BuildTrigger
	triggersRun   []string
	// Only really useful for testing triggers with a single repository.
	substitutions map[string]string
}

func (c *mockCloudBuildClient) RunBuildTrigger(ctx context.Context, req *cloudbuildpb.RunBuildTriggerRequest, opts ...gax.CallOption) error {
	slog.Info("running fake RunBuildTrigger")
	if c.runError != nil {
		return c.runError
	}
	for _, t := range c.triggersRun {
		if t == req.TriggerId {
			return nil
		}
	}
	c.triggersRun = append(c.triggersRun, req.TriggerId)
	c.substitutions = req.GetSource().GetSubstitutions()
	return nil
}

func (c *mockCloudBuildClient) ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) iter.Seq2[*cloudbuildpb.BuildTrigger, error] {
	return func(yield func(*cloudbuildpb.BuildTrigger, error) bool) {
		for _, v := range c.buildTriggers {
			var err error
			if c.runError != nil {
				v = nil
				err = c.runError
			}
			if !yield(v, err) {
				return // Stop iteration if yield returns false
			}
		}
	}
}

func TestRunCloudBuildTrigger(t *testing.T) {
	for _, test := range []struct {
		name     string
		runError error
		wantErr  bool
	}{
		{
			name:    "pass",
			wantErr: false,
		},
		{
			name:     "error",
			runError: fmt.Errorf("some-error"),
			wantErr:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			client := &mockCloudBuildClient{
				runError:      test.runError,
				buildTriggers: make([]*cloudbuildpb.BuildTrigger, 0),
			}
			substitutions := make(map[string]string)
			err := runCloudBuildTrigger(ctx, client, "some-project", "some-location", "some-trigger-id", substitutions)
			if diff := cmp.Diff(test.wantErr, err != nil); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindTriggerIdByName(t *testing.T) {
	for _, test := range []struct {
		name          string
		want          string
		runError      error
		wantErr       bool
		buildTriggers []*cloudbuildpb.BuildTrigger
	}{
		{
			name:    "finds trigger",
			want:    "some-trigger-id",
			wantErr: false,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "different-trigger",
					Id:   "different-trigger-id",
				},
				{
					Name: "some-trigger-name",
					Id:   "some-trigger-id",
				},
			},
		},
		{
			name:    "not found",
			want:    "",
			wantErr: true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "different-trigger",
					Id:   "different-trigger-id",
				},
			},
		},
		{
			name:     "runtime error",
			want:     "",
			runError: fmt.Errorf("some-error"),
			wantErr:  true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "different-trigger",
					Id:   "different-trigger-id",
				},
				{
					Name: "some-trigger-name",
					Id:   "some-trigger-id",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			client := &mockCloudBuildClient{
				runError:      test.runError,
				buildTriggers: test.buildTriggers,
			}
			triggerId, err := findTriggerIdByName(ctx, client, "some-project", "some-location", "some-trigger-name")
			if diff := cmp.Diff(test.wantErr, err != nil); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.want, triggerId); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunCloudBuildTriggerByName(t *testing.T) {
	for _, test := range []struct {
		name          string
		want          string
		runError      error
		wantErr       bool
		buildTriggers []*cloudbuildpb.BuildTrigger
	}{
		{
			name:    "finds trigger",
			wantErr: false,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "different-trigger",
					Id:   "different-trigger-id",
				},
				{
					Name: "some-trigger-name",
					Id:   "some-trigger-id",
				},
			},
		},
		{
			name:    "not found",
			wantErr: true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "different-trigger",
					Id:   "different-trigger-id",
				},
			},
		},
		{
			name:     "runtime error",
			runError: fmt.Errorf("some-error"),
			wantErr:  true,
			buildTriggers: []*cloudbuildpb.BuildTrigger{
				{
					Name: "different-trigger",
					Id:   "different-trigger-id",
				},
				{
					Name: "some-trigger-name",
					Id:   "some-trigger-id",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			client := &mockCloudBuildClient{
				runError:      test.runError,
				buildTriggers: test.buildTriggers,
			}
			err := runCloudBuildTriggerByName(ctx, client, "some-project", "some-location", "some-trigger-name", make(map[string]string))
			if diff := cmp.Diff(test.wantErr, err != nil); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
