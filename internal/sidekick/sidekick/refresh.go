// Copyright 2024 Google LLC
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

package sidekick

import (
	"context"
	"fmt"
	"path"

	"cloud.google.com/go/sdk/internal/sidekick/api"
	"cloud.google.com/go/sdk/internal/sidekick/codec_sample"
	"cloud.google.com/go/sdk/internal/sidekick/config"
	"cloud.google.com/go/sdk/internal/sidekick/dart"
	"cloud.google.com/go/sdk/internal/sidekick/parser"
	"cloud.google.com/go/sdk/internal/sidekick/rust"
	"cloud.google.com/go/sdk/internal/sidekick/rust_prost"
)

func init() {
	newCommand(
		"sidekick refresh",
		"Reruns the generator for a single client library.",
		`
Reruns the generator for a single client library, using the configuration parameters saved in the .sidekick.toml file.
`,
		cmdSidekick,
		refresh,
	)
}

// refresh reruns the generator in one directory, using the configuration
// parameters saved in its `.sidekick.toml` file.
func refresh(ctx context.Context, rootConfig *config.Config, cmdLine *CommandLine) error {
	override, err := overrideSources(ctx, rootConfig)
	if err != nil {
		return err
	}
	return refreshDir(ctx, override, cmdLine, cmdLine.Output)
}
func loadDir(rootConfig *config.Config, output string) (*api.API, *config.Config, error) {
	config, err := config.MergeConfigAndFile(rootConfig, path.Join(output, ".sidekick.toml"))
	if err != nil {
		return nil, nil, err
	}
	if config.General.SpecificationFormat == "" {
		return nil, nil, fmt.Errorf("must provide general.specification-format")
	}
	if config.General.SpecificationSource == "" {
		return nil, nil, fmt.Errorf("must provide general.specification-source")
	}
	model, err := parser.CreateModel(config)
	if err != nil {
		return nil, nil, err
	}
	return model, config, nil
}

func refreshDir(ctx context.Context, rootConfig *config.Config, cmdLine *CommandLine, output string) error {
	model, config, err := loadDir(rootConfig, output)
	if err != nil {
		return err
	}
	if cmdLine.DryRun {
		return nil
	}

	switch config.General.Language {
	case "rust":
		return rust.Generate(ctx, model, output, config)
	case "rust_storage":
		// The StorageControl client depends on multiple specification sources.
		// We load them both here manually, and pass them along to
		// `rust.GenerateStorage` which will merge them appropriately.
		storageModel, storageConfig, err := loadDir(rootConfig, "src/storage/src/generated/gapic")
		if err != nil {
			return err
		}
		controlModel, controlConfig, err := loadDir(rootConfig, "src/storage/src/generated/gapic_control")
		if err != nil {
			return err
		}
		return rust.GenerateStorage(ctx, output, storageModel, storageConfig, controlModel, controlConfig)
	case "rust+prost":
		return rust_prost.Generate(ctx, model, output, config)
	case "dart":
		return dart.Generate(ctx, model, output, config)
	case "sample":
		return codec_sample.Generate(ctx, model, output, config)
	default:
		return fmt.Errorf("unknown language: %s", config.General.Language)
	}
}
