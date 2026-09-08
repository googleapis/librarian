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
	"errors"
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

// streamingRootTypeIDs returns the root message type IDs for all enabled bidirectional
// and server-side streaming RPCs in the library.
func streamingRootTypeIDs(model *api.API, library *config.Library) []string {
	if library.Rust == nil || library.Rust.TemplateOverride != "" {
		return nil
	}
	includeBidi := library.Rust.IncludeBidiStreamingMethods != nil && *library.Rust.IncludeBidiStreamingMethods
	includeServer := library.Rust.IncludeServerStreamingMethods != nil && *library.Rust.IncludeServerStreamingMethods
	if !includeBidi && !includeServer {
		return nil
	}
	var rootTypeIDs []string
	for _, s := range model.Services {
		for _, m := range s.Methods {
			isBidi := m.ClientSideStreaming && m.ServerSideStreaming && includeBidi
			isServer := !m.ClientSideStreaming && m.ServerSideStreaming && includeServer
			if isBidi || isServer {
				if m.InputTypeID != "" {
					rootTypeIDs = append(rootTypeIDs, m.InputTypeID)
				}
				if m.OutputTypeID != "" {
					rootTypeIDs = append(rootTypeIDs, m.OutputTypeID)
				}
			}
		}
	}
	slices.Sort(rootTypeIDs)
	return slices.Compact(rootTypeIDs)
}

func generateProstHybrid(ctx context.Context, model *api.API, rootTypeIDs []string, library *config.Library, outdir string, modelConfig *parser.ModelConfig) error {
	if library.Rust == nil || library.Rust.TemplateOverride != "" || len(rootTypeIDs) == 0 {
		return nil
	}

	hybridModel, unusedTypes, hasGoogleRpcStatus, err := filterModelToTypes(model, rootTypeIDs, library.Rust.AllowStreamingAnyTypes)
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

// filterModelToTypes constructs an api.API model containing only types reachable
// from rootTypeIDs for prost conversion generation. It also returns a sorted slice
// of all non-WKT unused type IDs to exclude via prost_build extern_path, and a boolean
// indicating whether google.rpc.Status is referenced in the reachability path.
// Errors if Any is encountered in the reachability path unless explicitly allowed via allowAnyTypes.
func filterModelToTypes(model *api.API, rootTypeIDs []string, allowAnyTypes []string) (*api.API, []string, bool, error) {
	type typeItem struct {
		id   string
		path string
	}

	reachableMsgs := make(map[string]bool)
	reachableEnums := make(map[string]bool)
	var queue []typeItem

	for _, id := range rootTypeIDs {
		if id != "" {
			queue = append(queue, typeItem{
				id:   id,
				path: id,
			})
		}
	}

	// Discover all transitively reachable messages and enums.
	visited := make(map[string]bool)
	var enqueueMsg func(m *api.Message, path string)
	var enqueueEnum func(e *api.Enum, path string)

	// enqueueMsg marks a message as used and enqueues it for field
	// traversal. It recursively enqueues all parent ancestor messages to ensure:
	// 1. Parent messages are not placed in unusedTypes (which would cause prost_build's
	//    prefix-matching extern_path to hijack nested types like Partition.TemporalPartition).
	// 2. Parent messages are enqueued for field traversal so their sibling field types
	//    (e.g., Partition.SpatialPartition) are also included in reachableMsgs rather than
	//    placed in unusedTypes, preventing missing type errors in convert.rs.
	enqueueMsg = func(m *api.Message, path string) {
		if m == nil {
			return
		}
		if !reachableMsgs[m.ID] {
			reachableMsgs[m.ID] = true
			queue = append(queue, typeItem{
				id:   m.ID,
				path: path,
			})
		}
		if m.Parent != nil {
			enqueueMsg(m.Parent, path)
		}
	}

	// enqueueEnum marks an enum as used. If the enum is nested inside a message,
	// it recursively enqueues parent ancestor messages to ensure they and their sibling field
	// types are not placed in unusedTypes.
	enqueueEnum = func(e *api.Enum, path string) {
		if e == nil {
			return
		}
		reachableEnums[e.ID] = true
		if e.Parent != nil {
			enqueueMsg(e.Parent, path)
		}
	}

	hasGoogleRpcStatus := false
	var unsupportedAnyFields []string

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.id] {
			continue
		}
		visited[item.id] = true
		if isAnyType(item.id) {
			unsupportedAnyFields = append(unsupportedAnyFields, item.id)
			continue
		}

		msg := model.Message(item.id)
		if msg != nil {
			reachableMsgs[msg.ID] = true

			// Shortcircuit search for google.rpc.Status to avoid inspecting its fields,
			// which would otherwise error on Status.details (google.protobuf.Any).
			if isGoogleRpcStatus(item.id) {
				hasGoogleRpcStatus = true
				continue
			}
			enqueueMsg(msg, item.path)
			for _, f := range msg.Fields {
				fieldPath := item.path + "." + f.Name
				if isAnyType(f.TypezID) {
					if matchesAllowedAnyField(f.ID, allowAnyTypes) || matchesAllowedAnyField(fieldPath, allowAnyTypes) {
						f.SkipProtoConversion = true
						continue
					}
					unsupportedAnyFields = append(unsupportedAnyFields, fieldPath)
					continue
				}
				if f.Typez == api.TypezMessage && f.TypezID != "" {
					queue = append(queue, typeItem{
						id:   f.TypezID,
						path: fieldPath,
					})
				}
				if f.Typez == api.TypezEnum && f.TypezID != "" {
					enqueueEnum(model.Enum(f.TypezID), fieldPath)
				}
			}
			for _, o := range msg.OneOfs {
				for _, f := range o.Fields {
					fieldPath := item.path + "." + f.Name
					if isAnyType(f.TypezID) {
						if matchesAllowedAnyField(f.ID, allowAnyTypes) || matchesAllowedAnyField(fieldPath, allowAnyTypes) {
							f.SkipProtoConversion = true
							continue
						}
						unsupportedAnyFields = append(unsupportedAnyFields, fieldPath)
						continue
					}
					if f.Typez == api.TypezMessage && f.TypezID != "" {
						queue = append(queue, typeItem{
							id:   f.TypezID,
							path: fieldPath,
						})
					}
					if f.Typez == api.TypezEnum && f.TypezID != "" {
						enqueueEnum(model.Enum(f.TypezID), fieldPath)
					}
				}
			}
		}

		enum := model.Enum(item.id)
		if enum != nil {
			enqueueEnum(enum, item.path)
		}
	}

	if len(unsupportedAnyFields) > 0 {
		slices.Sort(unsupportedAnyFields)
		unsupportedAnyFields = slices.Compact(unsupportedAnyFields)
		var sb strings.Builder
		sb.WriteString("cannot generate prost conversion: type google.protobuf.Any is unsupported:\n")
		for _, f := range unsupportedAnyFields {
			fmt.Fprintf(&sb, "  - %s\n", f)
		}
		sb.WriteString("\nTo resolve this, allow dropping Any fields in prost conversion by adding them to allow_streaming_any_types in librarian.yaml:\n")
		sb.WriteString("    rust:\n")
		sb.WriteString("      allow_streaming_any_types:\n")
		for _, f := range unsupportedAnyFields {
			fmt.Fprintf(&sb, "        - %s\n", f)
		}
		return nil, nil, false, errors.New(strings.TrimSuffix(sb.String(), "\n"))
	}

	// Collect external top-level messages (e.g. google.type.LatLng) referenced by roots.
	// We sort external types by ID because model.AllMessages() iterates over a map whose
	// iteration order is randomized by the Go runtime.
	var externalMessages []*api.Message
	for m := range model.AllMessages() {
		if m.ID != "" && reachableMsgs[m.ID] && !isWKT(m.ID) && m.ID != ".google.rpc.Status" && !m.IsMap && m.Parent == nil {
			if m.Package != model.PackageName && m.Package != api.ReservedPackageName {
				externalMessages = append(externalMessages, m)
			}
		}
	}
	slices.SortFunc(externalMessages, func(a, b *api.Message) int { return cmp.Compare(a.ID, b.ID) })

	// Collect external top-level enums referenced by roots.
	var externalEnums []*api.Enum
	for e := range model.AllEnums() {
		if e.ID != "" && reachableEnums[e.ID] && !isWKT(e.ID) && e.Parent == nil {
			if e.Package != model.PackageName && e.Package != api.ReservedPackageName {
				externalEnums = append(externalEnums, e)
			}
		}
	}
	slices.SortFunc(externalEnums, func(a, b *api.Enum) int { return cmp.Compare(a.ID, b.ID) })

	// Construct hybridModel for prost conversion code generation.
	hybridModel := api.API{
		Name:                model.Name,
		PackageName:         model.PackageName,
		Title:               model.Title,
		Description:         model.Description,
		Revision:            model.Revision,
		Services:            model.Services,
		Messages:            language.FilterSlice(model.Messages, func(m *api.Message) bool { return reachableMsgs[m.ID] }),
		ExternalMessages:    externalMessages,
		Enums:               language.FilterSlice(model.Enums, func(e *api.Enum) bool { return reachableEnums[e.ID] }),
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
	// Populate messageByID and enumByID with ALL package messages and enums.
	// This avoids aliasing model.messageByID while allowing model
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
		if m.ID != "" && !isWKT(m.ID) && !reachableMsgs[m.ID] {
			unusedTypes = append(unusedTypes, m.ID)
		}
	}
	for e := range model.AllEnums() {
		if e.ID != "" && !isWKT(e.ID) && !reachableEnums[e.ID] {
			unusedTypes = append(unusedTypes, e.ID)
		}
	}
	slices.Sort(unusedTypes)

	return &hybridModel, slices.Compact(unusedTypes), hasGoogleRpcStatus, nil
}

func matchesAllowedAnyField(id string, allowStreamingAnyTypes []string) bool {
	idTrimmed := strings.TrimPrefix(id, ".")
	for _, allowed := range allowStreamingAnyTypes {
		if strings.TrimPrefix(allowed, ".") == idTrimmed {
			return true
		}
	}
	return false
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
