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

package rust

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	"github.com/googleapis/librarian/internal/sidekick/rust_prost"
)

func generateProstHybrid(ctx context.Context, model *api.API, rootTypeIDs []string, library *config.Library, outdir string, modelConfig *parser.ModelConfig) error {
	if library.Rust == nil || library.Rust.TemplateOverride != "" || len(rootTypeIDs) == 0 {
		return nil
	}

	hybridModel, unusedTypes, hasGoogleRpcStatus, err := filterModelToTypes(model, rootTypeIDs, library.Rust.AllowGrpcAnyFields)
	if err != nil {
		return err
	}

	hybridConfig := *modelConfig
	hybridConfig.Codec = maps.Clone(modelConfig.Codec)
	if hybridConfig.Codec == nil {
		hybridConfig.Codec = make(map[string]string)
	}
	hybridConfig.Codec["include-file"] = "includes.rs"
	if len(unusedTypes) > 0 {
		hybridConfig.Codec["unused-types"] = strings.Join(unusedTypes, "\n")
	}
	prostOutDir := filepath.Join(outdir, "src", "prost")
	if err := rust_prost.Generate(ctx, hybridModel, prostOutDir, "prost", &hybridConfig); err != nil {
		return fmt.Errorf("generating prost module: %w", err)
	}

	convertModelCfg := *modelConfig
	convertModelCfg.Codec = maps.Clone(modelConfig.Codec)
	if hasGoogleRpcStatus {
		convertModelCfg.Codec["include-rpc-status-conversion"] = "true"
	}
	convertModelCfg.Codec["template-override"] = "templates/convert-prost"
	convertModelCfg.Codec["prost-path"] = "super::prost"
	convertOutDir := filepath.Join(outdir, "src")
	if err := sidekickrust.Generate(ctx, hybridModel, convertOutDir, &convertModelCfg); err != nil {
		return fmt.Errorf("generating convert.rs: %w", err)
	}
	return nil
}

// filterModelToTypes constructs an api.API model containing only types reachable
// from rootTypeIDs for prost conversion generation. It also returns a sorted slice
// of all non-WKT unused type IDs to exclude via prost_build extern_path, and a boolean
// indicating whether google.rpc.Status is referenced in the reachability path.
// Errors if Any is encountered in the reachability path unless explicitly allowed via allowGrpcAnyFields.
func filterModelToTypes(model *api.API, rootTypeIDs []string, allowGrpcAnyFields []string) (*api.API, []string, bool, error) {
	return api.FilterToTypes(model, rootTypeIDs, allowGrpcAnyFields)
}
