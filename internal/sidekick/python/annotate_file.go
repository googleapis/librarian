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
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type externalImport struct {
	Package string // e.g. "google.protobuf.duration_pb2"
	Module  string // e.g. "duration_pb2"
}

type internalImport struct {
	Package string // e.g. "google.cloud.asset_v1.types"
	Module  string // e.g. "assets"
	Alias   string // e.g. "gca_assets" (empty if no alias)
}

// fileAnnotations decorates a .proto source file with Python-specific metadata.
type fileAnnotations struct {
	SourceFile         string
	Stem               string
	Package            string
	Messages           []*messageAnnotations
	Enums              []*enumAnnotations
	HasMessagesOrEnums bool
	Manifest           []string
	ExternalImports    []externalImport
	InternalImports    []internalImport
	HasImports         bool
	SortedMessages     []*messageAnnotations
	SortedEnums        []*enumAnnotations
	SortedSymbols      []string
	CopyrightYear      string
}

func (c *codec) annotateFile(sourceFile string, messages []*api.Message, enums []*api.Enum) *fileAnnotations {
	stem := strings.TrimSuffix(filepath.Base(sourceFile), ".proto")
	ann := &fileAnnotations{
		SourceFile:    sourceFile,
		Stem:          stem,
		Package:       c.Model.PackageName,
		CopyrightYear: c.GenerationYear,
	}

	for _, m := range messages {
		if mAnn, ok := m.Codec.(*messageAnnotations); ok {
			ann.Messages = append(ann.Messages, mAnn)
		}
	}
	for _, e := range enums {
		if eAnn, ok := e.Codec.(*enumAnnotations); ok {
			ann.Enums = append(ann.Enums, eAnn)
		}
	}

	ann.HasMessagesOrEnums = len(ann.Messages) > 0 || len(ann.Enums) > 0

	// Manifest: all top-level enum names in parsed order, followed by all top-level message names in parsed order.
	for _, e := range ann.Enums {
		ann.Manifest = append(ann.Manifest, e.Name)
	}
	for _, m := range ann.Messages {
		ann.Manifest = append(ann.Manifest, m.Name)
	}

	ann.SortedMessages = slices.Clone(ann.Messages)
	slices.SortFunc(ann.SortedMessages, func(a, b *messageAnnotations) int {
		return strings.Compare(a.Name, b.Name)
	})

	ann.SortedEnums = slices.Clone(ann.Enums)
	slices.SortFunc(ann.SortedEnums, func(a, b *enumAnnotations) int {
		return strings.Compare(a.Name, b.Name)
	})

	// SortedSymbols: top-level messages (alphabetically), then top-level enums (alphabetically).
	for _, m := range ann.SortedMessages {
		ann.SortedSymbols = append(ann.SortedSymbols, m.Name)
	}
	for _, e := range ann.SortedEnums {
		ann.SortedSymbols = append(ann.SortedSymbols, e.Name)
	}

	extSet := make(map[externalImport]bool)
	intSet := make(map[internalImport]bool)

	var scanFields func(fields []*api.Field)
	var scanMessage func(msg *api.Message)

	scanFields = func(fields []*api.Field) {
		for _, f := range fields {
			if f.Map {
				mapMsg := f.MessageType
				if mapMsg == nil && f.TypezID != "" && c.Model != nil {
					mapMsg = c.Model.Message(f.TypezID)
				}
				if mapMsg != nil {
					scanFields(mapMsg.Fields)
				}
				continue
			}
			switch f.Typez {
			case api.TypezMessage:
				target := f.MessageType
				if target == nil && f.TypezID != "" && c.Model != nil {
					target = c.Model.Message(f.TypezID)
				}
				if target != nil {
					targetFile := resolveProtoFile(target.SourceLocation, target.Package, target.Name)
					c.collectImport(sourceFile, stem, target.Package, targetFile, extSet, intSet)
				}
			case api.TypezEnum:
				target := f.EnumType
				if target == nil && f.TypezID != "" && c.Model != nil {
					target = c.Model.Enum(f.TypezID)
				}
				if target != nil {
					targetFile := resolveProtoFile(target.SourceLocation, target.Package, target.Name)
					c.collectImport(sourceFile, stem, target.Package, targetFile, extSet, intSet)
				}
			}
		}
	}

	scanMessage = func(msg *api.Message) {
		scanFields(msg.Fields)
		for _, nested := range msg.Messages {
			scanMessage(nested)
		}
	}

	for _, m := range messages {
		scanMessage(m)
	}

	for imp := range extSet {
		ann.ExternalImports = append(ann.ExternalImports, imp)
	}
	slices.SortFunc(ann.ExternalImports, func(a, b externalImport) int {
		if cmp := strings.Compare(a.Package, b.Package); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Module, b.Module)
	})

	for imp := range intSet {
		ann.InternalImports = append(ann.InternalImports, imp)
	}
	slices.SortFunc(ann.InternalImports, func(a, b internalImport) int {
		if cmp := strings.Compare(a.Package, b.Package); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.Module, b.Module); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Alias, b.Alias)
	})

	ann.HasImports = len(ann.ExternalImports) > 0 || len(ann.InternalImports) > 0
	return ann
}

func (c *codec) collectImport(currentFile, currentStem, targetPkg, targetFile string, extSet map[externalImport]bool, intSet map[internalImport]bool) {
	modelPkg := ""
	if c.Model != nil {
		modelPkg = c.Model.PackageName
	}
	if targetPkg == "" {
		targetPkg = modelPkg
	}
	targetStem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")

	if modelPkg != "" && targetPkg != modelPkg {
		if targetFile != "" {
			module := pythonModuleFromProto(targetFile)
			stem := strings.TrimSuffix(filepath.Base(targetFile), ".proto")
			extSet[externalImport{
				Package: module,
				Module:  stem + "_pb2",
			}] = true
		}
		return
	}

	if targetStem != "" && targetStem != currentStem {
		pyPkg := c.pythonPackage()
		alias := ""
		if c.fileNeedsAlias(currentFile, targetStem) {
			initials := packageInitials(pyPkg)
			if initials != "" {
				alias = initials + "_" + targetStem
			}
		}
		intSet[internalImport{
			Package: pyPkg + ".types",
			Module:  targetStem,
			Alias:   alias,
		}] = true
	}
}
