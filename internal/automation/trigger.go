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
	"log/slog"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
)

var triggerNameByCommandName = map[string]string{
	"generate":        "generate",
	"stage-release":   "stage-release",
	"publish-release": "publish-release",
}

const region = "global"

// RunCommand triggers a command for each registered repository that supports it.
func RunCommand(ctx context.Context, command string, projectId string) error {
	// validate command is allowed
	if !availableCommands[command] {
		return fmt.Errorf("unsupported command: %s", command)
	}
	triggerName := triggerNameByCommandName[command]
	if triggerName == "" {
		return fmt.Errorf("could not find trigger name for command: %s", command)
	}

	config, err := loadRepositoriesConfig()
	if err != nil {
		slog.Error("error loading repositories config", slog.Any("err", err))
		return err
	}

	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating cloudbuild client: %w", err)
	}
	defer c.Close()

	repositories := config.RepositoriesForCommand(command)
	for _, repository := range repositories {
		slog.Debug("running command", slog.String("command", command), slog.String("repository", repository.Name))

		substitutions := map[string]string{
			"_REPOSITORY":               repository.Name,
			"_GITHUB_TOKEN_SECRET_NAME": repository.SecretName,
		}
		err = runCloudBuildTriggerByName(ctx, c, projectId, region, triggerName, substitutions)
		if err != nil {
			slog.Error("Error triggering cloudbuild", slog.Any("err", err))
		}
	}
	return nil
}
