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
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	"github.com/googleapis/librarian/internal/sidekick/rust_prost"
	"github.com/googleapis/librarian/internal/sources"
)

// IsMixedLibrary reports whether the library has handwritten code wrapping
// generated code.
//
// A library is a mixed library when it has Rust module configuration. A library
// with no APIs and an explicit output is a mixed library if its derived API
// path is not listed in sdk.yaml; libraries whose derived path appears in
// sdk.yaml are generated libraries whose APIs have not yet been populated
// (e.g. google-cloud-oslogin-common), not mixed libraries.
func IsMixedLibrary(lib *config.Library) bool {
	if lib.Rust != nil && len(lib.Rust.Modules) > 0 {
		return true
	}
	if len(lib.APIs) == 0 && lib.Output != "" {
		// If the derived API path is in sdk.yaml, this is a generated
		// library whose APIs have not yet been populated, not a mixed library.
		if serviceconfig.HasAPIPath(DeriveAPIPath(lib.Name), config.LanguageRust) {
			return false
		}
		return true
	}
	return false
}

// Generate generates a Rust client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, sources *sources.Sources) error {
	if IsMixedLibrary(library) {
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
	exists, err := crateExists(library.Output)
	if err != nil {
		return err
	}
	if !exists {
		if err := create(ctx, library.Output); err != nil {
			return err
		}
	}
	if err := sidekickrust.Generate(ctx, model, library.Output, modelConfig); err != nil {
		return err
	}
	if err := generateProstHybrid(ctx, model, library, library.Output, modelConfig); err != nil {
		return err
	}
	if needsRepoMetadata(model, library) {
		repoMetadata, err := createRepoMetadata(cfg, library, sources)
		if err != nil {
			return err
		}
		if err := repoMetadata.Write(library.Output); err != nil {
			return err
		}
	}
	if !exists {
		validate(ctx, library.Output)
	}
	return nil
}

func generateProstHybrid(ctx context.Context, model *api.API, library *config.Library, outdir string, modelConfig *parser.ModelConfig) error {
	if library.Rust == nil || !library.Rust.IncludeBidiStreamingMethods || library.Rust.TemplateOverride != "" {
		return nil
	}
	hasBidiStreaming := slices.ContainsFunc(model.Services, (*api.Service).HasBidiStreaming)
	if !hasBidiStreaming {
		return nil
	}

	hybridModel, err := filterModelToStreaming(model)
	if err != nil {
		return fmt.Errorf("filtering model for streaming: %w", err)
	}

	hybridConfig := *modelConfig
	hybridConfig.Codec = maps.Clone(modelConfig.Codec)
	if hybridConfig.Codec == nil {
		hybridConfig.Codec = make(map[string]string)
	}
	hybridConfig.Codec["convert-include-package"] = model.PackageName
	prostOutDir := filepath.Join(outdir, "src", "prost")
	if err := rust_prost.Generate(ctx, hybridModel, prostOutDir, "prost", &hybridConfig); err != nil {
		return fmt.Errorf("generating prost module: %w", err)
	}

	convertModelCfg := *modelConfig
	convertModelCfg.Codec = maps.Clone(modelConfig.Codec)
	convertModelCfg.Codec["template-override"] = "templates/convert-prost"
	convertOutDir := filepath.Join(outdir, "src")
	if err := sidekickrust.Generate(ctx, hybridModel, convertOutDir, &convertModelCfg); err != nil {
		return fmt.Errorf("generating convert.rs: %w", err)
	}
	return nil
}

// filterModelToStreaming constructs a hybrid api.API model containing only
// bidirectional streaming RPC types for prost conversion generation.
// Errors if Any is encountered.
func filterModelToStreaming(model *api.API) (*api.API, error) {
	type streamingTypeItem struct {
		id       string
		rpc      string
		methodID string
		path     string
	}

	streamingMsgs := make(map[string]bool)
	streamingEnums := make(map[string]bool)
	var queue []streamingTypeItem

	// Collect initial input/output message types from all bidirectional streaming RPCs.
	for _, s := range model.Services {
		for _, m := range s.Methods {
			if m.ClientSideStreaming && m.ServerSideStreaming {
				rpcName := s.Name + "." + m.Name
				if m.InputTypeID != "" {
					queue = append(queue, streamingTypeItem{
						id:       m.InputTypeID,
						rpc:      rpcName,
						methodID: m.ID,
						path:     m.InputTypeID,
					})
				}
				if m.OutputTypeID != "" {
					queue = append(queue, streamingTypeItem{
						id:       m.OutputTypeID,
						rpc:      rpcName,
						methodID: m.ID,
						path:     m.OutputTypeID,
					})
				}
			}
		}
	}

	// Discover all transitively reachable messages and enums.
	visited := make(map[string]bool)
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if visited[item.id] {
			continue
		}
		visited[item.id] = true

		anyError := func(path string) error {
			return fmt.Errorf("cannot generate prost conversion for streaming RPC %q: type google.protobuf.Any is unsupported (path: %s)\n"+
				"To resolve this, add the RPC method ID to skipped_ids in librarian.yaml (e.g. skipped_ids: [%s])",
				item.rpc, path, item.methodID)
		}

		if isAnyType(item.id) {
			return nil, anyError(item.path)
		}

		msg := model.Message(item.id)
		if msg != nil {
			streamingMsgs[msg.ID] = true
			for _, f := range msg.Fields {
				fieldPath := item.path + "." + f.Name
				if isAnyType(f.TypezID) {
					return nil, anyError(fieldPath)
				}
				if f.Typez == api.TypezMessage && f.TypezID != "" {
					queue = append(queue, streamingTypeItem{
						id:       f.TypezID,
						rpc:      item.rpc,
						methodID: item.methodID,
						path:     fieldPath,
					})
				}
				if f.Typez == api.TypezEnum && f.TypezID != "" {
					streamingEnums[f.TypezID] = true
				}
			}
			for _, o := range msg.OneOfs {
				for _, f := range o.Fields {
					fieldPath := item.path + "." + o.Name + "." + f.Name
					if isAnyType(f.TypezID) {
						return nil, anyError(fieldPath)
					}
					if f.Typez == api.TypezMessage && f.TypezID != "" {
						queue = append(queue, streamingTypeItem{
							id:       f.TypezID,
							rpc:      item.rpc,
							methodID: item.methodID,
							path:     fieldPath,
						})
					}
					if f.Typez == api.TypezEnum && f.TypezID != "" {
						streamingEnums[f.TypezID] = true
					}
				}
			}
		}

		enum := model.Enum(item.id)
		if enum != nil {
			streamingEnums[enum.ID] = true
		}
	}

	// Construct hybridModel for prost conversion code generation.
	hybridModel := api.API{
		Name:        model.Name,
		PackageName: model.PackageName,
		Title:       model.Title,
		Description: model.Description,
		Revision:    model.Revision,
		Services:    model.Services,
		// Messages and Enums slices are filtered to only streaming types so convert.rs
		// only contains conversion implementations for streaming RPCs.
		Messages:            language.FilterSlice(model.Messages, func(m *api.Message) bool { return streamingMsgs[m.ID] }),
		Enums:               language.FilterSlice(model.Enums, func(e *api.Enum) bool { return streamingEnums[e.ID] }),
		ResourceDefinitions: model.ResourceDefinitions,
		QuickstartService:   model.QuickstartService,
		Codec:               model.Codec,
	}
	for _, s := range hybridModel.Services {
		hybridModel.AddService(s)
		for _, m := range s.Methods {
			hybridModel.AddMethod(m)
		}
	}
	// Populate messageByID and enumByID with ALL package messages and enums (not just
	// streaming types). This avoids aliasing model.messageByID while allowing model
	// annotation to resolve field types across the entire package.
	for m := range model.AllMessages() {
		hybridModel.AddMessage(m)
	}
	for e := range model.AllEnums() {
		hybridModel.AddEnum(e)
	}
	for _, r := range hybridModel.ResourceDefinitions {
		hybridModel.AddResource(r)
	}
	return &hybridModel, nil
}

func isAnyType(id string) bool {
	return strings.TrimPrefix(id, ".") == "google.protobuf.Any"
}

// UpdateWorkspace updates dependencies for the entire Rust workspace.
func UpdateWorkspace(ctx context.Context) error {
	return command.Run(ctx, command.Cargo, "update", "--workspace")
}

// Format formats a generated Rust library. Must be called sequentially;
// parallel calls cause race conditions as cargo fmt runs cargo metadata,
// which competes for locks on the workspace Cargo.toml and Cargo.lock.
func Format(ctx context.Context, library *config.Library) error {
	if err := command.Run(ctx, "taplo", "fmt", filepath.Join(library.Output, "Cargo.toml")); err != nil {
		return err
	}
	if err := command.Run(ctx, command.Cargo, "fmt", "-p", library.Name); err != nil {
		return err
	}
	return nil
}

func generateVeneer(ctx context.Context, library *config.Library, sources *sources.Sources) error {
	if library.Rust == nil || len(library.Rust.Modules) == 0 {
		return nil
	}
	for _, module := range library.Rust.Modules {
		if module.Template == "storage" {
			return generateRustStorage(ctx, library, module.Output, sources)
		}
		if module.Template == "bigquery" {
			return generateRustBigQuery(ctx, library, module.Output, sources)
		}
		modelConfig, err := moduleToModelConfig(library, module, sources)
		if err != nil {
			return fmt.Errorf("moduleToModelConfig %q: %w", module.Output, err)
		}
		model, err := parser.CreateModel(modelConfig)
		if err != nil {
			return fmt.Errorf("CreateModel %q: %w", module.Output, err)
		}
		if module.Template == "prost" || module.Template == "tonic" {
			err = rust_prost.Generate(ctx, model, module.Output, module.Template, modelConfig)
		} else {
			err = sidekickrust.Generate(ctx, model, module.Output, modelConfig)
		}
		if err != nil {
			return fmt.Errorf("module %q: %w", module.Output, err)
		}
	}
	return nil
}

// Keep returns the list of files to preserve when cleaning the output directory.
func Keep(library *config.Library) ([]string, error) {
	if !IsMixedLibrary(library) {
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
func generateRustStorage(ctx context.Context, library *config.Library, moduleOutput string, sources *sources.Sources) error {
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

// generateRustBigQuery generates rust BigQuery query builder.
func generateRustBigQuery(ctx context.Context, library *config.Library, moduleOutput string, sources *sources.Sources) error {
	var bqModule *config.RustModule
	for _, module := range library.Rust.Modules {
		if module.Template == "bigquery" {
			bqModule = module
			break
		}
	}
	if bqModule == nil {
		return fmt.Errorf("module with template 'bigquery' not found in library %q", library.Name)
	}

	modelConfig, err := moduleToModelConfig(library, bqModule, sources)
	if err != nil {
		return fmt.Errorf("failed to create bigquery model config: %w", err)
	}
	model, err := parser.CreateModel(modelConfig)
	if err != nil {
		return fmt.Errorf("failed to create bigquery model: %w", err)
	}

	return sidekickrust.GenerateBigQueryBuilder(ctx, moduleOutput, model, modelConfig)
}

func findModuleByOutput(library *config.Library, output string) *config.RustModule {
	for _, module := range library.Rust.Modules {
		if module.Output == output {
			return module
		}
	}

	return nil
}
