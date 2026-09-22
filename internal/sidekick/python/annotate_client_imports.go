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

// clientTypeImport represents an import of an internal type module.
type clientTypeImport struct {
	Stem string
}

// clientExternalImport represents an import of an external protobuf module.
type clientExternalImport struct {
	Module string
	Alias  string
}

func (c *codec) collectClientImports(service *api.Service) ([]*clientTypeImport, []*clientExternalImport) {
	typeStems := make(map[string]bool)
	extSet := make(map[clientExternalImport]bool)

	modelPkg := ""
	if c.Model != nil {
		modelPkg = c.Model.PackageName
	}

	recordMessageOrEnum := func(msg *api.Message, enum *api.Enum) {
		var loc *api.SourceLocation
		var pkg string
		var name string
		if msg != nil {
			loc = msg.SourceLocation
			pkg = msg.Package
			name = msg.Name
		} else if enum != nil {
			loc = enum.SourceLocation
			pkg = enum.Package
			name = enum.Name
		} else {
			return
		}

		if pkg == "google.cloud.location" || pkg == "google.longrunning" {
			return
		}

		protoFile := resolveProtoFile(loc, pkg, name)
		if modelPkg != "" && pkg != "" && pkg != modelPkg {
			if protoFile != "" {
				mod := pythonModuleFromProto(protoFile)
				stem := strings.TrimSuffix(filepath.Base(protoFile), ".proto")
				extSet[clientExternalImport{
					Module: mod,
					Alias:  stem + "_pb2",
				}] = true
			}
			return
		}
		if protoFile != "" {
			stem := strings.TrimSuffix(filepath.Base(protoFile), ".proto")
			typeStems[stem] = true
		}
	}

	recordFieldType := func(f *api.Field) {
		if f == nil {
			return
		}
		switch f.Typez {
		case api.TypezMessage:
			target := f.MessageType
			if target == nil && f.TypezID != "" && c.Model != nil {
				target = c.Model.Message(f.TypezID)
			}
			if target != nil {
				recordMessageOrEnum(target, nil)
			} else if f.TypezID != "" {
				if after, ok := strings.CutPrefix(f.TypezID, ".google.protobuf."); ok {
					stem := snakeCase(after)
					extSet[clientExternalImport{
						Module: "google.protobuf." + stem + "_pb2",
						Alias:  stem + "_pb2",
					}] = true
				} else if after, ok := strings.CutPrefix(f.TypezID, ".google.rpc."); ok {
					stem := snakeCase(after)
					extSet[clientExternalImport{
						Module: "google.rpc." + stem + "_pb2",
						Alias:  stem + "_pb2",
					}] = true
				}
			}
		case api.TypezEnum:
			target := f.EnumType
			if target == nil && f.TypezID != "" && c.Model != nil {
				target = c.Model.Enum(f.TypezID)
			}
			if target != nil {
				recordMessageOrEnum(nil, target)
			}
		}
	}

	for _, m := range service.Methods {
		if isMixin(m, service) {
			continue
		}

		// 1. Input message
		if iamMod, _, ok := isIAMType(m.InputTypeID); ok {
			extSet[clientExternalImport{
				Module: "google.iam.v1." + iamMod,
				Alias:  iamMod,
			}] = true
		} else {
			inMsg := m.InputType
			if inMsg == nil && m.InputTypeID != "" {
				inMsg = c.resolveMessageType(m.InputTypeID)
			}
			if inMsg != nil {
				recordMessageOrEnum(inMsg, nil)
			}
		}

		// 2. Flattened fields from signatures
		for _, sig := range m.Signatures {
			for _, f := range sig.Fields {
				recordFieldType(f)
			}
		}

		// 3. Output
		isLRO := m.OperationInfo != nil || m.IsLRO
		if isLRO {
			if m.OperationInfo != nil {
				if strings.TrimPrefix(m.OperationInfo.ResponseTypeID, ".") == "google.protobuf.Empty" {
					extSet[clientExternalImport{
						Module: "google.protobuf.empty_pb2",
						Alias:  "empty_pb2",
					}] = true
				} else if m.OperationInfo.ResponseTypeID != "" && c.Model != nil {
					resp := c.resolveMessageType(m.OperationInfo.ResponseTypeID)
					if resp != nil {
						recordMessageOrEnum(resp, nil)
					}
				}
				if m.OperationInfo.MetadataTypeID != "" && c.Model != nil {
					meta := c.resolveMessageType(m.OperationInfo.MetadataTypeID)
					if meta != nil {
						recordMessageOrEnum(meta, nil)
					}
				}
			}
		} else {
			returnsEmpty := m.ReturnsEmpty || strings.TrimPrefix(m.OutputTypeID, ".") == "google.protobuf.Empty" || (m.OutputType != nil && strings.TrimPrefix(m.OutputType.ID, ".") == "google.protobuf.Empty")
			if iamMod, _, ok := isIAMType(m.OutputTypeID); ok {
				extSet[clientExternalImport{
					Module: "google.iam.v1." + iamMod,
					Alias:  iamMod,
				}] = true
			} else if !returnsEmpty {
				outMsg := m.OutputType
				if outMsg == nil && m.OutputTypeID != "" {
					outMsg = c.resolveMessageType(m.OutputTypeID)
				}
				if outMsg != nil {
					recordMessageOrEnum(outMsg, nil)
					// Direct fields of output message
					for _, f := range outMsg.Fields {
						recordFieldType(f)
					}
				}
			}
		}
	}

	var typeImports []*clientTypeImport
	for stem := range typeStems {
		typeImports = append(typeImports, &clientTypeImport{Stem: stem})
	}
	slices.SortFunc(typeImports, func(a, b *clientTypeImport) int {
		return strings.Compare(a.Stem, b.Stem)
	})

	var externalImports []*clientExternalImport
	for ext := range extSet {
		item := ext
		externalImports = append(externalImports, &item)
	}
	slices.SortFunc(externalImports, func(a, b *clientExternalImport) int {
		if cmp := strings.Compare(a.Module, b.Module); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Alias, b.Alias)
	})

	return typeImports, externalImports
}
