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

package protowriter

import (
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/gapic/java/engine/ast"
	"github.com/googleapis/librarian/internal/gapic/java/model"
)

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestWriteResponse(t *testing.T) {
	ctx := &model.GapicContext{
		Services: []*model.Service{
			{
				Name:                "Echo",
				PackageName:         "com.google.example.v1",
				OriginalJavaPackage: "com.google.example.v1",
				Methods: []*model.Method{
					{Name: "Echo"},
				},
			},
		},
		HasMetadata: true,
	}

	classes := []*model.GapicClass{
		{
			Kind: model.KindMain,
			ClassDefinition: &ast.ClassDefinition{
				PackageName: "com.google.example.v1",
				Scope:       ast.Public,
				Name:        "EchoClient",
			},
		},
	}

	pkgInfo := &model.GapicPackageInfo{
		PackageName: "com.google.example.v1",
		Description: "A client to Echo",
	}

	reflectConfigs := []*model.ReflectConfig{
		{Name: "com.google.example.v1.EchoClient"},
	}

	resp := Write(ctx, classes, pkgInfo, reflectConfigs)
	if resp == nil {
		t.Fatalf("expected non-nil response")
	}

	fileMap := make(map[string]string)
	for _, f := range resp.GetFile() {
		fileMap[f.GetName()] = f.GetContent()
	}

	if _, ok := fileMap["com/google/example/v1/EchoClient.java"]; !ok {
		t.Errorf("missing EchoClient.java in response files")
	}
	if _, ok := fileMap["com/google/example/v1/package-info.java"]; !ok {
		t.Errorf("missing package-info.java in response files")
	}
	if _, ok := fileMap["gapic_metadata.json"]; !ok {
		t.Errorf("missing gapic_metadata.json in response files")
	}

	metaContent := fileMap["gapic_metadata.json"]
	if !strings.Contains(metaContent, `"libraryClient": "EchoClient"`) {
		t.Errorf("unexpected gapic_metadata.json content: %s", metaContent)
	}
}
