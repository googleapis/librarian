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

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"golang.org/x/exp/slog"
)

func runCloudBuildTriggerByName(ctx context.Context, projectId string, location string, triggerName string, substitutions map[string]string) error {
	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating cloudbuild client: %w", err)
	}
	defer c.Close()
	triggerId, err := findTriggerIdByName(ctx, c, projectId, location, triggerName)
	if err != nil {
		return fmt.Errorf("error finding triggerid: %w", err)
	}
	slog.Info("found triggerId", slog.String("triggerId", triggerId))
	return runCloudBuildTrigger(ctx, c, projectId, location, triggerId, substitutions)
}

func findTriggerIdByName(ctx context.Context, c *cloudbuild.Client, projectId string, location string, triggerName string) (string, error) {
	slog.Info("looking for triggerId by name",
		slog.String("projectId", projectId),
		slog.String("location", location),
		slog.String("triggerName", triggerName),
	)
	req := &cloudbuildpb.ListBuildTriggersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location),
	}
	for resp, err := range c.ListBuildTriggers(ctx, req).All() {
		if err != nil {
			return "", fmt.Errorf("error running trigger %w", err)
		}
		if resp.Name == triggerName {
			return resp.Id, nil
		}
	}
	return "", fmt.Errorf("could not find trigger id")
}

func runCloudBuildTrigger(ctx context.Context, c *cloudbuild.Client, projectId string, location string, triggerId string, substitutions map[string]string) error {
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
	resp, err := c.RunBuildTrigger(ctx, req)
	if err != nil {
		return fmt.Errorf("error running trigger %w", err)
	}

	slog.Info("triggered", slog.String("LRO Name", resp.Name()))
	return nil
}
