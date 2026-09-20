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

package python

import (
	"errors"
	"fmt"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// ErrNilModel is returned when a nil api.API model is supplied to newCodec or Generate.
var ErrNilModel = errors.New("model cannot be nil")

// codec holds the configuration and context for the Python sidekick generator.
type codec struct {
	Model          *api.API
	Library        *config.Library
	OutDir         string
	GenerationYear string
	PackageName    string
	PackageVersion string
}

// newCodec constructs a new Python codec instance from model, library config, and outdir.
func newCodec(model *api.API, library *config.Library, outdir string) (*codec, error) {
	if model == nil {
		return nil, ErrNilModel
	}

	year := fmt.Sprintf("%04d", time.Now().Year())
	if library != nil && library.CopyrightYear != "" {
		year = library.CopyrightYear
	}

	version := "0.0.0"
	if library != nil && library.Version != "" {
		version = library.Version
	}

	packageName := model.PackageName
	if model.Name != "" {
		packageName = model.Name
	}
	if library != nil && library.Name != "" {
		packageName = library.Name
	}

	return &codec{
		Model:          model,
		Library:        library,
		OutDir:         outdir,
		GenerationYear: year,
		PackageName:    packageName,
		PackageVersion: version,
	}, nil
}
