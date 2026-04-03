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

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

type codec struct {
	GenerationYear string
	PackageName    string
	// Most libraries are generated from `googleapis`. Rarely, we use protobuf,
	// gapic-showcase, or a different root.
	RootName string
}

func newCodec(model *api.API, cfg *parser.ModelConfig) *codec {
	year, _, _ := time.Now().Date()
	result := &codec{
		GenerationYear: fmt.Sprintf("%04d", year),
		PackageName:    PackageName(model),
		RootName:       "googleapis",
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
