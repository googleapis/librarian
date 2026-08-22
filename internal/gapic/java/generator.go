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

// Package java provides functionality for generating Java client libraries from Protocol Buffers.
package java

import (
	"github.com/googleapis/librarian/internal/gapic/java/composer"
	"github.com/googleapis/librarian/internal/gapic/java/protoparser"
	"github.com/googleapis/librarian/internal/gapic/java/protowriter"
	"google.golang.org/protobuf/types/pluginpb"
)

// GenerateGapic generates Java GAPIC client files from a CodeGeneratorRequest.
func GenerateGapic(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	ctx, err := protoparser.Parse(req)
	if err != nil {
		return nil, err
	}

	classes := composer.ComposeServiceClasses(ctx)
	pkgInfo := composer.ComposePackageInfo(ctx)
	reflectConfigs := composer.ComposeNativeReflectConfig(ctx)

	return protowriter.Write(ctx, classes, pkgInfo, reflectConfigs), nil
}
