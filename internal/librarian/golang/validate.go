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

package golang

import (
	"errors"
	"fmt"
	"go/token"
	"path"
	"regexp"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"golang.org/x/mod/module"
)

var (
	errOpaqueCopyExtraProto   = errors.New("invalid opaque copy extra proto")
	errOpaqueCopyImportPath   = errors.New("invalid opaque copy import path")
	errOpaqueCopyProtoPackage = errors.New("invalid opaque copy proto package")
	protoPackagePattern       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

// Validate checks Go-specific configuration.
func Validate(cfg *config.Config) error {
	for _, library := range cfg.Libraries {
		for _, api := range library.APIs {
			if api.Go == nil || api.Go.OpaqueCopy == nil {
				continue
			}
			if err := validateOpaqueCopy(api.Go.OpaqueCopy); err != nil {
				return fmt.Errorf("library %q API %q: %w", library.Name, api.Path, err)
			}
		}
	}
	return nil
}

func validateOpaqueCopy(copy *config.GoOpaqueCopy) error {
	if copy.ImportPath == "" || path.IsAbs(copy.ImportPath) || path.Clean(copy.ImportPath) != copy.ImportPath || strings.HasPrefix(copy.ImportPath, "../") {
		return fmt.Errorf("%w: %q", errOpaqueCopyImportPath, copy.ImportPath)
	}
	if err := module.CheckImportPath("cloud.google.com/go/" + copy.ImportPath); err != nil || !token.IsIdentifier(path.Base(copy.ImportPath)) {
		return fmt.Errorf("%w: %q", errOpaqueCopyImportPath, copy.ImportPath)
	}
	if copy.ProtoPackage != "" && !protoPackagePattern.MatchString(copy.ProtoPackage) {
		return fmt.Errorf("%w: %q", errOpaqueCopyProtoPackage, copy.ProtoPackage)
	}
	for _, proto := range copy.ExtraProtos {
		if proto == "" || path.IsAbs(proto) || path.Clean(proto) != proto || strings.HasPrefix(proto, "../") || path.Ext(proto) != ".proto" {
			return fmt.Errorf("%w: %q", errOpaqueCopyExtraProto, proto)
		}
	}
	return nil
}
