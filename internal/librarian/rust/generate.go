// Copyright 2025 Google LLC
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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	"github.com/googleapis/librarian/internal/sidekick/rust_prost"
	"github.com/googleapis/librarian/internal/sidekick/source"
)

// Generate generates a Rust client library.
func Generate(ctx context.Context, library *config.Library, sources *source.Sources) error {
	if library.Veneer {
		return generateVeneer(ctx, library, sources)
	}
	if len(library.APIs) != 1 {
		return fmt.Errorf("the Rust generator only supports a single api per library")
	}

	modelConfig, err := libraryToModelConfig(library, library.APIs[0], sources)
	if err != nil {
		return err
	}
	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		return err
	}
	exists := true
	if _, err := os.Stat(library.Output); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot access output directory %q: %w", library.Output, err)
		}
		exists = false
	}
	if !exists {
		if err := create(ctx, library.Output); err != nil {
			return err
		}
	}
	if err := sidekickrust.Generate(ctx, model, library.Output, modelConfig); err != nil {
		return err
	}
	if !exists {
		validate(ctx, library.Output)
	}
	return nil
}

// UpdateWorkspace updates dependencies for the entire Rust workspace.
func UpdateWorkspace(ctx context.Context) error {
	return command.Run(ctx, "cargo", "update", "--workspace")
}

// Format formats a generated Rust library. Must be called sequentially;
// parallel calls cause race conditions as cargo fmt runs cargo metadata,
// which competes for locks on the workspace Cargo.toml and Cargo.lock.
func Format(ctx context.Context, library *config.Library) error {
	if err := command.Run(ctx, "taplo", "fmt", filepath.Join(library.Output, "Cargo.toml")); err != nil {
		return err
	}
	if err := command.Run(ctx, "cargo", "fmt", "-p", library.Name); err != nil {
		return err
	}
	return nil
}

func generateVeneer(ctx context.Context, library *config.Library, sources *source.Sources) error {
	if library.Rust == nil || len(library.Rust.Modules) == 0 {
		return nil
	}
	for _, module := range library.Rust.Modules {
		modelConfig, err := moduleToModelConfig(library, module, sources)
		if err != nil {
			return fmt.Errorf("module %q: %w", module.Output, err)
		}
		model, err := parser.CreateModel(modelConfig)
		if err != nil {
			return fmt.Errorf("module %q: %w", module.Output, err)
		}
		switch modelConfig.Language {
		case "rust":
			err = sidekickrust.Generate(ctx, model, module.Output, modelConfig)
		case "rust_storage":
			return generateRustStorage(ctx, library, module.Output, sources)
		case "rust+prost":
			err = rust_prost.Generate(ctx, model, module.Output, modelConfig)
		default:
			err = fmt.Errorf("language %q not supported", modelConfig.Language)
		}
		if err != nil {
			return fmt.Errorf("module %q: %w", module.Output, err)
		}
	}
	return nil
}

// Keep returns the list of files to preserve when cleaning the output directory.
func Keep(library *config.Library) ([]string, error) {
	if !library.Veneer {
		return library.Keep, nil
	}
	// For veneers, keep all files outside module output directories. We walk
	// library.Output and keep files not under any module.Output.
	var keep []string
	moduleOutputs := make(map[string]bool)
	for _, m := range library.Rust.Modules {
		moduleOutputs[m.Output] = true
	}
	err := filepath.WalkDir(library.Output, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if moduleOutputs[path] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(library.Output, path)
		if err != nil {
			return err
		}
		keep = append(keep, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keep, nil
}

// DefaultLibraryName derives a library name from an api path.
// For example: google/cloud/secretmanager/v1 -> google-cloud-secretmanager-v1.
func DefaultLibraryName(api string) string {
	return strings.ReplaceAll(api, "/", "-")
}

// DeriveAPIPath derives an api path from a library name.
// For example: google-cloud-secretmanager-v1 -> google/cloud/secretmanager/v1.
func DeriveAPIPath(name string) string {
	return strings.ReplaceAll(name, "-", "/")
}

// DefaultOutput derives an output path from an api path and default output.
// For example: google/cloud/secretmanager/v1 with default src/generated/
// returns src/generated/cloud/secretmanager/v1.
func DefaultOutput(api, defaultOutput string) string {
	return filepath.Join(defaultOutput, strings.TrimPrefix(api, "google/"))
}

// generateRustStorage generates rust StorageControl client.
//
// The StorageControl client depends on multiple specification sources.
// We load them both here, and pass them along to `rust.GenerateStorage` which will merge them appropriately.
func generateRustStorage(ctx context.Context, library *config.Library, moduleOutput string, sources *source.Sources) error {
	output := "src/storage/src/generated/gapic"
	storageModule := findModuleByOutput(library, output)
	if storageModule == nil {
		return fmt.Errorf("module %q not found in library %q", output, library.Name)
	}
	storageConfig, err := moduleToModelConfig(library, storageModule, sources)
	if err != nil {
		return fmt.Errorf("failed to create storage model config: %w", err)
	}
	storageModel, err := parser.CreateModel(storageConfig)
	if err != nil {
		return fmt.Errorf("failed to create storage model: %w", err)
	}

	output = "src/storage/src/generated/gapic_control"
	controlModule := findModuleByOutput(library, "src/storage/src/generated/gapic_control")
	if controlModule == nil {
		return fmt.Errorf("module %q not found in library %q", output, library.Name)
	}
	controlConfig, err := moduleToModelConfig(library, controlModule, sources)
	if err != nil {
		return fmt.Errorf("failed to create control model config: %w", err)
	}
	controlModel, err := parser.CreateModel(controlConfig)
	if err != nil {
		return fmt.Errorf("failed to create control model: %w", err)
	}

	return sidekickrust.GenerateStorage(ctx, moduleOutput, storageModel, storageConfig, controlModel, controlConfig)
}

func findModuleByOutput(library *config.Library, output string) *config.RustModule {
	for _, module := range library.Rust.Modules {
		if module.Output == output {
			return module
		}
	}

	return nil
}
