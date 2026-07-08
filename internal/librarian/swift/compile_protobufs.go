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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
)

func compileProtobufs(ctx context.Context, library *config.Library, module *config.SwiftModule, src *sources.Sources) error {
	if err := os.MkdirAll(module.Output, 0755); err != nil {
		return err
	}

	sourceConfig := sources.NewSourceConfig(src, library.Roots)
	apiPathAbs := sourceConfig.ResolveDir(module.APIPath)

	var protoFiles []string
	if len(module.IncludeList) > 0 {
		for _, file := range module.IncludeList {
			protoFiles = append(protoFiles, filepath.Join(apiPathAbs, file))
		}
	} else {
		entries, err := os.ReadDir(apiPathAbs)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".proto") {
				protoFiles = append(protoFiles, filepath.Join(apiPathAbs, entry.Name()))
			}
		}
	}

	if len(protoFiles) == 0 {
		return fmt.Errorf("no proto files found in %s", apiPathAbs)
	}

	importsMap := make(map[string]bool)
	var protoImports []string

	addImport := func(path string) {
		if path != "" && !importsMap[path] {
			importsMap[path] = true
			protoImports = append(protoImports, "-I", path)
		}
	}

	for _, r := range sourceConfig.ActiveRoots {
		addImport(sourceConfig.Root(r))
	}
	addImport(src.Googleapis)
	addImport(src.ProtobufSrc)
	addImport(src.Showcase)
	addImport(src.Conformance)

	args := []string{
		"--swift_out=Visibility=Public:" + module.Output,
		"--grpc-swift_out=Visibility=Public:" + module.Output,
	}
	args = append(args, protoImports...)
	args = append(args, protoFiles...)

	return command.Run(ctx, "protoc", args...)
}
