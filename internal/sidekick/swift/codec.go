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
	"fmt"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

// codec represents the configuration for a Swift sidekick Codec.
//
// A sideckick Codec is a package that generates libraries from an `api.API`
// model and some configuration. In the Swift codec, the `Generate()`
// function  creates a `codec` object for each `api.API` that needs to be
// generated. That lends naturally into a single object that carries all the
// information needed to generate the library.
type codec struct {
	GenerationYear string
	PackageName    string
	// Most libraries are generated from `googleapis`. Rarely, we use protobuf,
	// gapic-showcase, or a different root.
	RootName     string
	Model        *api.API
	Dependencies []config.SwiftDependency
}

func newCodec(model *api.API, cfg *parser.ModelConfig, swiftCfg *config.SwiftPackage) *codec {
	year, _, _ := time.Now().Date()
	result := &codec{
		GenerationYear: fmt.Sprintf("%04d", year),
		PackageName:    PackageName(model),
		RootName:       "googleapis",
		Model:          model,
	}
	if swiftCfg != nil {
		result.Dependencies = swiftCfg.Dependencies
	}
	for key, definition := range cfg.Codec {
		switch key {
		case "copyright-year":
			result.GenerationYear = definition
		case "package-name-override":
			result.PackageName = definition
		case "root-name":
			result.RootName = definition
		default:
			// Ignore other options.
		}
	}
	return result
}
