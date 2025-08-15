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
)

func TestRunCommandWithClient(t *testing.T) {
	for _, test := range []struct {
		name          string
		command       string
		push          bool
		want          string
		runError      error
		wantErr       bool
		buildTriggers []*cloudbuildpb.BuildTrigger
	}{
		{
			name:    "runs generate trigger",
			command: "generate",
			push:    true,
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
		},
		{
			name:    "runs prepare-release trigger",
			command: "stage-release",
			push:    true,
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
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			client := &mockCloudBuildClient{
				runError:      test.runError,
				buildTriggers: test.buildTriggers,
			}
			err := runCommandWithClient(ctx, client, test.command, "some-project", test.push)
			if test.wantErr && err == nil {
				t.Errorf("expected error, but did not return one")
			} else if !test.wantErr && err != nil {
				t.Errorf("did not expect error, but received one: %s", err)
			}
		})
	}
}
