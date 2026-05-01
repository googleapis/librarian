// Copyright 2026 Google LLC
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

package swift

import (
	"context"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickswift "github.com/googleapis/librarian/internal/sidekick/swift"
	"github.com/googleapis/librarian/internal/sources"
)

func generateModule(ctx context.Context, library *config.Library, src *sources.Sources) error {
	for _, module := range library.Swift.Modules {
		modelConfig := moduleToModelConfig(library, module, src)
		model, err := parser.CreateModel(modelConfig)
		if err != nil {
			return err
		}
		if err := sidekickswift.Generate(ctx, model, module.Output, modelConfig, library.Swift); err != nil {
			return err
		}
	}
	return nil
}

func moduleToModelConfig(library *config.Library, module *config.SwiftModule, src *sources.Sources) *parser.ModelConfig {
	sourceConfig := sources.NewSourceConfig(src, library.Roots)
	if library.Swift != nil && len(library.Swift.IncludeList) > 0 {
		sourceConfig.IncludeList = library.Swift.IncludeList
	}

	return &parser.ModelConfig{
		SpecificationFormat: config.SpecProtobuf,
		SpecificationSource: module.APIPath,
		Source:              sourceConfig,
		Codec: map[string]string{
			"copyright-year": library.CopyrightYear,
			"module":         "true",
		},
	}
}
