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
	"cmp"
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
	if library.Rust == nil || library.Rust.TemplateOverride != "" {
		return nil
	}
	includeBidi := library.Rust.IncludeBidiStreamingMethods != nil && *library.Rust.IncludeBidiStreamingMethods
	includeServer := library.Rust.IncludeServerStreamingMethods != nil && *library.Rust.IncludeServerStreamingMethods
	if !includeBidi && !includeServer {
		return nil
	}
	hasStreaming := slices.ContainsFunc(model.Services, func(s *api.Service) bool {
		return (includeBidi && s.HasBidiStreaming()) || (includeServer && s.HasServerSideStreaming())
	})
	if !hasStreaming {
		return nil
	}

	hybridModel, unusedTypes, hasGoogleRpcStatus, err := filterModelToStreaming(model, includeBidi, includeServer)
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
	if hasGoogleRpcStatus {
		convertModelCfg.Codec["include-rpc-status-conversion"] = "true"
	}
	convertModelCfg.Codec["template-override"] = "templates/convert-prost"
	convertOutDir := filepath.Join(outdir, "src")
	if err := sidekickrust.Generate(ctx, hybridModel, convertOutDir, &convertModelCfg); err != nil {
		return fmt.Errorf("generating convert.rs: %w", err)
	}
	return nil
}

// filterModelToStreaming constructs a hybrid api.API model containing only
// enabled bidirectional and server-side streaming RPC types for prost conversion generation. It also returns
// a sorted slice of all non-WKT unused type IDs to exclude via prost_build extern_path,
// and a boolean indicating whether google.rpc.Status is referenced in the streaming path.
// Errors if Any is encountered in the streaming reachability path.
func filterModelToStreaming(model *api.API, includeBidi bool, includeServer bool) (*api.API, []string, bool, error) {
	type streamingTypeItem struct {
		id       string
		rpc      string
		methodID string
		path     string
	}

	streamingMsgs := make(map[string]bool)
	streamingEnums := make(map[string]bool)
	var queue []streamingTypeItem

	// Collect initial input/output message types from all enabled bidirectional and server-side streaming RPCs.
	for _, s := range model.Services {
		for _, m := range s.Methods {
			isBidi := m.ClientSideStreaming && m.ServerSideStreaming && includeBidi
			isServer := !m.ClientSideStreaming && m.ServerSideStreaming && includeServer
			if isBidi || isServer {
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
	var enqueueMsg func(m *api.Message, rpc, methodID, path string)
	var enqueueEnum func(e *api.Enum, rpc, methodID, path string)

	// enqueueMsg marks a message as used in streaming and enqueues it for field
	// traversal. It recursively enqueues all parent ancestor messages to ensure:
	// 1. Parent messages are not placed in unusedTypes (which would cause prost_build's
	//    prefix-matching extern_path to hijack nested types like Partition.TemporalPartition).
	// 2. Parent messages are enqueued for field traversal so their sibling field types
	//    (e.g., Partition.SpatialPartition) are also included in streamingMsgs rather than
	//    placed in unusedTypes, preventing missing type errors in convert.rs.
	enqueueMsg = func(m *api.Message, rpc, methodID, path string) {
		if m == nil {
			return
		}
		if !streamingMsgs[m.ID] {
			streamingMsgs[m.ID] = true
			queue = append(queue, streamingTypeItem{
				id:       m.ID,
				rpc:      rpc,
				methodID: methodID,
				path:     path,
			})
		}
		if m.Parent != nil {
			enqueueMsg(m.Parent, rpc, methodID, path)
		}
	}

	// enqueueEnum marks an enum as used in streaming. If the enum is nested inside a message,
	// it recursively enqueues parent ancestor messages to ensure they and their sibling field
	// types are not placed in unusedTypes.
	enqueueEnum = func(e *api.Enum, rpc, methodID, path string) {
		if e == nil {
			return
		}
		streamingEnums[e.ID] = true
		if e.Parent != nil {
			enqueueMsg(e.Parent, rpc, methodID, path)
		}
	}

	hasGoogleRpcStatus := false

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
			return nil, nil, false, anyError(item.path)
		}

		msg := model.Message(item.id)
		if msg != nil {
			streamingMsgs[msg.ID] = true

			// Shortcircuit search for google.rpc.Status to avoid inspecting its fields,
			// which would otherwise error on Status.details (google.protobuf.Any).
			if isGoogleRpcStatus(item.id) {
				hasGoogleRpcStatus = true
				continue
			}
			enqueueMsg(msg, item.rpc, item.methodID, item.path)
			for _, f := range msg.Fields {
				fieldPath := item.path + "." + f.Name
				if isAnyType(f.TypezID) {
					return nil, nil, false, anyError(fieldPath)
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
					enqueueEnum(model.Enum(f.TypezID), item.rpc, item.methodID, fieldPath)
				}
			}
			for _, o := range msg.OneOfs {
				for _, f := range o.Fields {
					fieldPath := item.path + "." + o.Name + "." + f.Name
					if isAnyType(f.TypezID) {
						return nil, nil, false, anyError(fieldPath)
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
						enqueueEnum(model.Enum(f.TypezID), item.rpc, item.methodID, fieldPath)
					}
				}
			}
		}

		enum := model.Enum(item.id)
		if enum != nil {
			enqueueEnum(enum, item.rpc, item.methodID, item.path)
		}
	}

	// Collect external top-level messages (e.g. google.type.LatLng) referenced by streaming RPCs.
	// We sort external types by ID because model.AllMessages() iterates over a map whose
	// iteration order is randomized by the Go runtime.
	var externalMessages []*api.Message
	for m := range model.AllMessages() {
		if m.ID != "" && streamingMsgs[m.ID] && !isWKT(m.ID) && m.ID != ".google.rpc.Status" && !m.IsMap && m.Parent == nil {
			if m.Package != model.PackageName && m.Package != api.ReservedPackageName {
				externalMessages = append(externalMessages, m)
			}
		}
	}
	slices.SortFunc(externalMessages, func(a, b *api.Message) int { return cmp.Compare(a.ID, b.ID) })

	// Collect external top-level enums referenced by streaming RPCs.
	var externalEnums []*api.Enum
	for e := range model.AllEnums() {
		if e.ID != "" && streamingEnums[e.ID] && !isWKT(e.ID) && e.Parent == nil {
			if e.Package != model.PackageName && e.Package != api.ReservedPackageName {
				externalEnums = append(externalEnums, e)
			}
		}
	}
	slices.SortFunc(externalEnums, func(a, b *api.Enum) int { return cmp.Compare(a.ID, b.ID) })

	// Construct hybridModel for prost conversion code generation.
	hybridModel := api.API{
		Name:        model.Name,
		PackageName: model.PackageName,
		Title:       model.Title,
		Description: model.Description,
		Revision:    model.Revision,
		// Services, Messages, and Enums slices are filtered to only streaming types so convert.rs
		// only contains conversion implementations for streaming RPCs.
		// Filtering model.Messages and model.Enums preserves their original deterministic file order.
		Services: language.FilterSlice(model.Services, func(s *api.Service) bool {
			return (includeBidi && s.HasBidiStreaming()) || (includeServer && s.HasServerSideStreaming())
		}),
		Messages:            language.FilterSlice(model.Messages, func(m *api.Message) bool { return streamingMsgs[m.ID] }),
		ExternalMessages:    externalMessages,
		Enums:               language.FilterSlice(model.Enums, func(e *api.Enum) bool { return streamingEnums[e.ID] }),
		ExternalEnums:       externalEnums,
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

	return &hybridModel, slices.Compact(unusedTypes), hasGoogleRpcStatus, nil
}

func isAnyType(id string) bool {
	return strings.TrimPrefix(id, ".") == "google.protobuf.Any"
}

func isGoogleRpcStatus(id string) bool {
	return strings.TrimPrefix(id, ".") == "google.rpc.Status"
}

func isWKT(id string) bool {
	return strings.HasPrefix(strings.TrimPrefix(id, "."), "google.protobuf.")
}
