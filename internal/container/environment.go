// Copyright 2024 Google LLC
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

package container

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/librarian/internal/statepb"
	"github.com/googleapis/librarian/internal/utils"
)

// ContainerEnvironment represents configuration for environment
// variables for container invocations.
type ContainerEnvironment struct {
	// The context used for SecretManager requests
	ctx context.Context
	// The file used to store the environment variables for the duration of a docker run.
	tmpFile string
	// The client used to fetch secrets from Secret Manager, if any.
	secretManagerClient *secretmanager.Client
	// The project in which to look up secrets
	secretsProject string
	// A cache of secrets we've already fetched.
	secretCache map[string]string
	// The pipeline configuration, specifying which environment variables to obtain
	// for each command.
	pipelineConfig *statepb.PipelineConfig
}

func NewEnvironment(ctx context.Context, tmpRoot, secretsProject string, pipelineConfig *statepb.PipelineConfig) (*ContainerEnvironment, error) {
	var secretManagerClient *secretmanager.Client
	if secretsProject != "" {
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			return nil, err
		}
		secretManagerClient = client
	} else {
		secretManagerClient = nil
	}
	tmpFile := filepath.Join(tmpRoot, "docker-env.txt")
	return &ContainerEnvironment{
		ctx:                 ctx,
		tmpFile:             tmpFile,
		secretManagerClient: secretManagerClient,
		secretsProject:      secretsProject,
		secretCache:         make(map[string]string),
		pipelineConfig:      pipelineConfig,
	}, nil
}

func writeEnvironmentFile(containerEnv *ContainerEnvironment, commandName string) error {
	content, err := constructEnvironmentFileContent(containerEnv, commandName)
	if err != nil {
		return err
	}
	return utils.CreateAndWriteToFile(containerEnv.tmpFile, content)
}

func constructEnvironmentFileContent(containerEnv *ContainerEnvironment, commandName string) (string, error) {
	commandConfig := containerEnv.pipelineConfig.Commands[commandName]
	if commandConfig == nil {
		return "# No environment variables", nil
	}
	var builder strings.Builder
	for _, variable := range commandConfig.EnvironmentVariables {
		value, present := os.LookupEnv(variable.Name)
		if present {
			builder.WriteString(fmt.Sprintf("%s=%s\n", variable.Name, value))
			continue
		}
		if containerEnv.secretManagerClient == nil {
			builder.WriteString(fmt.Sprintf("# No value for %s\n", variable.Name))
			continue
		}
		value, present = containerEnv.secretCache[variable.SecretName]
		if present {
			builder.WriteString(fmt.Sprintf("%s=%s", variable.Name, value))
			continue
		}
		request := &secretmanagerpb.AccessSecretVersionRequest{
			Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", containerEnv.secretsProject, variable.SecretName),
		}
		secret, err := containerEnv.secretManagerClient.AccessSecretVersion(containerEnv.ctx, request)
		if err != nil {
			return "", err
		}
		// We assume the payload is valid UTF-8.
		value = string(secret.Payload.Data[:])
		builder.WriteString(fmt.Sprintf("%s=%s\n", variable.Name, value))
		containerEnv.secretCache[variable.SecretName] = value
	}
	return builder.String(), nil
}

func deleteEnvironmentFile(containerEnv *ContainerEnvironment) error {
	return os.Remove(containerEnv.tmpFile)
}
