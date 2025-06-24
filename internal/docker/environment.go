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

package docker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/grpc/codes"
)

// EnvironmentProvider represents configuration for environment
// variables for docker invocations.
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

func (e *EnvironmentProvider) writeEnvironmentFile(commandName string) error {
	content, err := e.constructEnvironmentFileContent(commandName)
	if err != nil {
		return err
	}
	return createAndWriteToFile(e.tmpFile, content)
}

func createAndWriteToFile(filePath string, content string) (err error) {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		err = cerr
	}()

	_, err = file.WriteString(content)
	return err
}

func (e *EnvironmentProvider) constructEnvironmentFileContent(commandName string) (string, error) {
	commandConfig := e.pipelineConfig.Commands[commandName]
	if commandConfig == nil {
		return "# No environment variables", nil
	}
	var builder strings.Builder
	for _, variable := range commandConfig.EnvironmentVariables {
		source := "host environment"
		var err error
		// First source: environment variables
		value, present := os.LookupEnv(variable.Name)
		// Second source: Secret Manager
		if !present {
			source = "Secret Manager"
			value, present, err = getSecretManagerValue(e, variable)
			if err != nil {
				return "", err
			}
		}
		// Final fallback: default value
		if !present && variable.DefaultValue != "" {
			source = "default value"
			value = variable.DefaultValue
			present = true
		}

		// Finally, write the value if we've got one
		if present {
			slog.Info(fmt.Sprintf("Using %s to provide value to docker for %s", source, variable.Name))
			builder.WriteString(fmt.Sprintf("%s=%s\n", variable.Name, value))
		} else {
			slog.Info(fmt.Sprintf("No value to provide to docker for '%s'", variable.Name))
			builder.WriteString(fmt.Sprintf("# No value for %s\n", variable.Name))
		}
		continue
	}
	return builder.String(), nil
}

func getSecretManagerValue(dockerEnv *EnvironmentProvider, variable *statepb.CommandEnvironmentVariable) (string, bool, error) {
	if variable.SecretName == "" || dockerEnv.secretManagerClient == nil {
		return "", false, nil
	}
	value, present := dockerEnv.secretCache[variable.SecretName]
	if present {
		return value, true, nil
	}
	request := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", dockerEnv.secretsProject, variable.SecretName),
	}
	secret, err := dockerEnv.secretManagerClient.AccessSecretVersion(dockerEnv.ctx, request)
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
	dockerEnv.secretCache[variable.SecretName] = value
	return value, true, nil
}

func deleteEnvironmentFile(dockerEnv *EnvironmentProvider) error {
	return os.Remove(dockerEnv.tmpFile)
}
