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
	"path/filepath"

	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sidekick/protobuf"
)

func (codec *codec) annotateModel(model *api.API, cfg *parser.ModelConfig) error {
	rootSource := cfg.Source.Root(codec.RootName)
	files, err := protobuf.DetermineInputFiles(cfg.SpecificationSource, cfg.Source)
	if err != nil {
		return err
	}
	files, err = relativeFilenames(rootSource, files)
	if err != nil {
		return err
	}

	annotations := &modelAnnotations{
		CopyrightYear: codec.GenerationYear,
		BoilerPlate:   license.HeaderBulk(),
		PackageName:   PackageName(model, codec.PackageName),
		Files:         files,
	}
	model.Codec = annotations
	return nil
}

func relativeFilenames(rootSource string, files []string) ([]string, error) {
	for i, f := range files {
		rel, err := filepath.Rel(rootSource, f)
		if err != nil {
			return nil, err
		}
		files[i] = rel
	}
	return files, nil
}
