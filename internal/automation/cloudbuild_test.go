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
	"iter"
	"testing"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/gax-go/v2"
)

type mockCloudBuildClient struct {
	runError      error
	buildTriggers []*cloudbuildpb.BuildTrigger
}

func (c *mockCloudBuildClient) RunBuildTrigger(ctx context.Context, req *cloudbuildpb.RunBuildTriggerRequest, opts ...gax.CallOption) (*cloudbuild.RunBuildTriggerOperation, error) {
	if c.runError != nil {
		return nil, c.runError
	}
	return &cloudbuild.RunBuildTriggerOperation{}, nil
}

func (c *mockCloudBuildClient) ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) iter.Seq2[*cloudbuildpb.BuildTrigger, error] {
	return func(yield func(*cloudbuildpb.BuildTrigger, error) bool) {
		for _, v := range c.buildTriggers {
			if !yield(v, nil) {
				return // Stop iteration if yield returns false
			}
		}
	}
}

func TestRunCloudBuildTrigger(t *testing.T) {
	for _, test := range []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "pass",
			wantErr: false,
		},
		{
			name:    "pass",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			client := &mockCloudBuildClient{}
			err := runCloudBuildTrigger(ctx, client, "some-project", "some-location", "some-trigger-id", make(map[string]string))
			if diff := cmp.Diff(test.wantErr, err != nil); diff != "" {
				t.Errorf("runCloudBuildTrigger() error")
			}
		})
	}
}

func TestFindTriggerIdByName(t *testing.T) {
	t.Run("foo", func(t *testing.T) {
		if diff := cmp.Diff("foo", "bar"); diff != "" {
			t.Errorf("runCloudBuildTrigger() error")
		}
	})
}

func TestRunCloudBuildTriggerByName(t *testing.T) {
	t.Run("foo", func(t *testing.T) {
		if diff := cmp.Diff("foo", "bar"); diff != "" {
			t.Errorf("runCloudBuildTrigger() error")
		}
	})
}
