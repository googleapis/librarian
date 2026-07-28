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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	sidekickrust "github.com/googleapis/librarian/internal/sidekick/rust"
	"github.com/googleapis/librarian/internal/sidekick/rust_prost"
)

func generateProstHybrid(ctx context.Context, model *api.API, library *config.Library, outdir string, modelConfig *parser.ModelConfig) error {
	if library.Rust == nil || !library.Rust.IncludeBidiStreamingMethods || library.Rust.TemplateOverride != "" {
		return nil
	}
	hasBidiStreaming := slices.ContainsFunc(model.Services, (*api.Service).HasBidiStreaming)
	if !hasBidiStreaming {
		return nil
	}

	hybridModel, unusedTypes, err := filterModelToStreaming(model)
	if err != nil {
		return err
	}

	hybridConfig := *modelConfig
	hybridConfig.Codec = maps.Clone(modelConfig.Codec)
	if hybridConfig.Codec == nil {
		hybridConfig.Codec = make(map[string]string)
	}
	hybridConfig.Codec["convert-include-package"] = model.PackageName
	if len(unusedTypes) > 0 {
		hybridConfig.Codec["unused-types"] = strings.Join(unusedTypes, "\n")
	}
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
// bidirectional streaming RPC types for prost conversion generation. It also returns
// a sorted slice of all non-WKT unused type IDs to exclude via prost_build extern_path.
// Errors if Any is encountered in the streaming reachability path.
func filterModelToStreaming(model *api.API) (*api.API, []string, error) {
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
			return nil, nil, anyError(item.path)
		}

		msg := model.Message(item.id)
		if msg != nil {
			streamingMsgs[msg.ID] = true
			for _, f := range msg.Fields {
				fieldPath := item.path + "." + f.Name
				if isAnyType(f.TypezID) {
					return nil, nil, anyError(fieldPath)
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
						return nil, nil, anyError(fieldPath)
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
		// Services, Messages and Enums slices are filtered to only streaming types so convert.rs
		// only contains conversion implementations for streaming RPCs.
		Services:            language.FilterSlice(model.Services, (*api.Service).HasBidiStreaming),
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

	var unusedTypes []string
	for m := range model.AllMessages() {
		if m.ID != "" && !isWKT(m.ID) && !streamingMsgs[m.ID] {
			unusedTypes = append(unusedTypes, m.ID)
		}
	}
	for e := range model.AllEnums() {
		if e.ID != "" && !isWKT(e.ID) && !streamingEnums[e.ID] {
			unusedTypes = append(unusedTypes, e.ID)
		}
	}
	slices.Sort(unusedTypes)

	return &hybridModel, slices.Compact(unusedTypes), nil
}

func isAnyType(id string) bool {
	return strings.TrimPrefix(id, ".") == "google.protobuf.Any"
}

func isWKT(id string) bool {
	return strings.HasPrefix(strings.TrimPrefix(id, "."), "google.protobuf.")
}
