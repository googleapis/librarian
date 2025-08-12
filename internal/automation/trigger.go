package automation

import (
	"context"
	"fmt"
	"log/slog"
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

	repositories := config.RepositoriesForCommand(command)
	for _, repository := range repositories {
		slog.Debug("running command", slog.String("command", command), slog.String("repository", repository.Name))

		substitutions := map[string]string{
			"_REPOSITORY":               repository.Name,
			"_GITHUB_TOKEN_SECRET_NAME": repository.SecretName,
		}
		err = runCloudBuildTriggerByName(ctx, projectId, region, triggerName, substitutions)
		if err != nil {
			slog.Error("Error triggering cloudbuild", slog.Any("err", err))
		}
	}
	return nil
}
