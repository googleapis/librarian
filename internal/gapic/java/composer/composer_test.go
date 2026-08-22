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

package composer

import (
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/gapic/java/engine/ast"
	"github.com/googleapis/librarian/internal/gapic/java/engine/writer"
	"github.com/googleapis/librarian/internal/gapic/java/model"
)

func createTestContext() *model.GapicContext {
	svc := &model.Service{
		Name:        "Echo",
		PackageName: "com.google.example.v1",
		HostName:    "echo.googleapis.com",
		DefaultPort: "443",
		Methods: []*model.Method{
			{
				Name:       "Echo",
				InputType:  ast.ObjectType("EchoRequest", "com.google.example.v1"),
				OutputType: ast.ObjectType("EchoResponse", "com.google.example.v1"),
				MethodSignatures: [][]string{
					{"content"},
				},
			},
		},
	}

	return &model.GapicContext{
		Services: []*model.Service{svc},
		Messages: map[string]*model.Message{
			"EchoRequest": {
				Name:        "EchoRequest",
				PackageName: "com.google.example.v1",
				Fields: map[string]*model.Field{
					"content": {
						Name: "content",
						Type: ast.TypeString,
					},
				},
			},
		},
		HelperResourceNames: map[string]*model.ResourceName{
			"example.googleapis.com/Echo": {
				Type:        "example.googleapis.com/Echo",
				Patterns:    []string{"projects/{project}/echoes/{echo}"},
				PackageName: "com.google.example.v1",
				ClassName:   "EchoName",
			},
		},
		Transport:              model.TransportGRPCRest,
		HasGenerateVersionJava: true,
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestComposeServiceClasses(t *testing.T) {
	ctx := createTestContext()
	classes := ComposeServiceClasses(ctx)

	if len(classes) == 0 {
		t.Fatalf("expected composed classes, got 0")
	}

	// Verify each class renders to valid Java code without panic
	classNames := make(map[string]bool)
	classCode := make(map[string]string)
	for _, c := range classes {
		code := writer.WriteClass(c.ClassDefinition)
		if len(code) == 0 {
			t.Errorf("empty code generated for class %s", c.ClassDefinition.Name)
		}
		classNames[c.ClassDefinition.Name] = true
		classCode[c.ClassDefinition.Name] = code
	}

	expectedClasses := []string{
		"EchoClient",
		"EchoSettings",
		"EchoStub",
		"EchoStubSettings",
		"GrpcEchoStub",
		"GrpcEchoCallableFactory",
		"HttpJsonEchoStub",
		"HttpJsonEchoCallableFactory",
		"EchoName",
		"Version",
	}

	for _, exp := range expectedClasses {
		if !classNames[exp] {
			t.Errorf("missing expected class %s", exp)
		}
	}

	// Verify that signature overload populates request builder fields
	clientCode := classCode["EchoClient"]
	if !strings.Contains(clientCode, "EchoRequest.newBuilder().setContent(content).build()") {
		t.Errorf("EchoClient missing builder setter invocation in signature overload, got:\n%s", clientCode)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestComposePackageInfo(t *testing.T) {
	ctx := createTestContext()
	pkgInfo := ComposePackageInfo(ctx)

	if pkgInfo == nil {
		t.Fatalf("expected package info, got nil")
	}
	if pkgInfo.PackageName != "com.google.example.v1" {
		t.Errorf("unexpected package name: %s", pkgInfo.PackageName)
	}
}

// Ported from gapic-generator-java:
// https://github.com/googleapis/google-cloud-java/commits/2a27c2c39/sdk-platform-java/gapic-generator-java
func TestComposeNativeReflectConfig(t *testing.T) {
	ctx := createTestContext()
	reflectConfigs := ComposeNativeReflectConfig(ctx)

	if len(reflectConfigs) < 2 {
		t.Errorf("expected at least 2 reflect configs, got %d", len(reflectConfigs))
	}
}
