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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/googleapis/librarian/internal/statepb"
	"github.com/googleapis/librarian/internal/utils"
	"google.golang.org/grpc/codes"
)

// EnvironmentProvider represents configuration for environment
// variables for container invocations.
type EnvironmentProvider struct {
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

func newEnvironmentProvider(ctx context.Context, workRoot, secretsProject string, pipelineConfig *statepb.PipelineConfig) (*EnvironmentProvider, error) {
	if pipelineConfig == nil {
		return nil, nil
	}
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
	tmpFile := filepath.Join(workRoot, "docker-env.txt")
	return &EnvironmentProvider{
		ctx:                 ctx,
		tmpFile:             tmpFile,
		secretManagerClient: secretManagerClient,
		secretsProject:      secretsProject,
		secretCache:         make(map[string]string),
		pipelineConfig:      pipelineConfig,
	}, nil
}

func writeEnvironmentFile(containerEnv *EnvironmentProvider, commandName string) error {
	content, err := constructEnvironmentFileContent(containerEnv, commandName)
	if err != nil {
		return err
	}
	return utils.CreateAndWriteToFile(containerEnv.tmpFile, content)
}

func constructEnvironmentFileContent(containerEnv *EnvironmentProvider, commandName string) (string, error) {
	commandConfig := containerEnv.pipelineConfig.Commands[commandName]
	if commandConfig == nil {
		return "# No environment variables", nil
	}
	var builder strings.Builder
	for _, variable := range commandConfig.EnvironmentVariables {
		var err error
		// First source: environment variables
		value, present := os.LookupEnv(variable.Name)
		// Second source: Secret Manager
		if !present {
			value, present, err = getSecretManagerValue(containerEnv, variable)
			if err != nil {
				return "", err
			}
		}
		// Final fallback: default value
		if !present && variable.DefaultValue != "" {
			value = variable.DefaultValue
			present = true
		}

		// Finally, write the value if we've got one
		if present {
			builder.WriteString(fmt.Sprintf("%s=%s\n", variable.Name, value))
		} else {
			builder.WriteString(fmt.Sprintf("# No value for %s\n", variable.Name))
		}
		continue
	}
	return builder.String(), nil
}

func getSecretManagerValue(containerEnv *EnvironmentProvider, variable *statepb.CommandEnvironmentVariable) (string, bool, error) {
	if variable.SecretName == "" || containerEnv.secretManagerClient == nil {
		return "", false, nil
	}
	value, present := containerEnv.secretCache[variable.SecretName]
	if present {
		return value, true, nil
	}
	request := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", containerEnv.secretsProject, variable.SecretName),
	}
	secret, err := containerEnv.secretManagerClient.AccessSecretVersion(containerEnv.ctx, request)
	if err != nil {
		// If the error is that the secret wasn't found, continue to the next source.
		// Any other error causes a real error to be returned.
		var ae *apierror.APIError
		if errors.As(err, &ae) && ae.GRPCStatus().Code() == codes.NotFound {
			return "", false, nil
		} else {
			return "", false, err
		}
	}
	// We assume the payload is valid UTF-8.
	value = string(secret.Payload.Data[:])
	containerEnv.secretCache[variable.SecretName] = value
	return value, true, nil
}

func deleteEnvironmentFile(containerEnv *EnvironmentProvider) error {
	return os.Remove(containerEnv.tmpFile)
}
