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
	"iter"

	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"github.com/googleapis/gax-go/v2"
	"golang.org/x/exp/slog"
)

// CloudBuildClient is an interface for mocking calls to Cloud Build.
type CloudBuildClient interface {
	RunBuildTrigger(ctx context.Context, req *cloudbuildpb.RunBuildTriggerRequest, opts ...gax.CallOption) error
	ListBuildTriggers(ctx context.Context, req *cloudbuildpb.ListBuildTriggersRequest, opts ...gax.CallOption) iter.Seq2[*cloudbuildpb.BuildTrigger, error]
}

func runCloudBuildTriggerByName(ctx context.Context, c CloudBuildClient, projectId string, location string, triggerName string, substitutions map[string]string) error {
	triggerId, err := findTriggerIdByName(ctx, c, projectId, location, triggerName)
	if err != nil {
		return fmt.Errorf("error finding triggerid: %w", err)
	}
	slog.Info("found triggerId", slog.String("triggerId", triggerId))
	return runCloudBuildTrigger(ctx, c, projectId, location, triggerId, substitutions)
}

func findTriggerIdByName(ctx context.Context, c CloudBuildClient, projectId string, location string, triggerName string) (string, error) {
	slog.Info("looking for triggerId by name",
		slog.String("projectId", projectId),
		slog.String("location", location),
		slog.String("triggerName", triggerName),
	)
	req := &cloudbuildpb.ListBuildTriggersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location),
	}
	for resp, err := range c.ListBuildTriggers(ctx, req) {
		if err != nil {
			return "", fmt.Errorf("error running trigger %w", err)
		}
		if resp.Name == triggerName {
			return resp.Id, nil
		}
	}
	return "", fmt.Errorf("could not find trigger id")
}

func runCloudBuildTrigger(ctx context.Context, c CloudBuildClient, projectId string, location string, triggerId string, substitutions map[string]string) error {
	triggerName := fmt.Sprintf("projects/%s/locations/%s/triggers/%s", projectId, location, triggerId)
	req := &cloudbuildpb.RunBuildTriggerRequest{
		Name:      triggerName,
		ProjectId: projectId,
		TriggerId: triggerId,
		Source: &cloudbuildpb.RepoSource{
			Substitutions: substitutions,
		},
	}
	slog.Info("triggering", slog.String("triggerName", triggerName), slog.String("triggerId", triggerId))
	err := c.RunBuildTrigger(ctx, req)
	if err != nil {
		return fmt.Errorf("error running trigger %w", err)
	}
	return nil
}
