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

package container

import (
	"context"

	"github.com/googleapis/librarian/internal/statepb"
)

type ContainerConfig struct {
	// The Docker image to run.
	Image string

	// The provider for environment variables, if any.
	envProvider *EnvironmentProvider
}

func NewContainerConfig(ctx context.Context, workRoot, image, secretsProject string, pipelineConfig *statepb.PipelineConfig) (*ContainerConfig, error) {
	envProvider, err := newEnvironmentProvider(ctx, workRoot, secretsProject, pipelineConfig)
	if err != nil {
		return nil, err
	}
	return &ContainerConfig{
		Image:       image,
		envProvider: envProvider,
	}, nil
}
