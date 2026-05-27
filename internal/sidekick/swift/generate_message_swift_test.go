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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func swiftConfig(t *testing.T, extraDependencies []config.SwiftDependency) *config.SwiftPackage {
	t.Helper()
	deps := []config.SwiftDependency{
		{Name: "GoogleCloudWkt", ApiPackage: wellKnownProtobufPackage},
	}
	deps = append(deps, extraDependencies...)
	return &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: deps,
		},
	}
}

func TestGenerateMessage_Files(t *testing.T) {
	outDir := t.TempDir()

	secret := &api.Message{Name: "Secret", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Secret"}
	volume := &api.Message{Name: "Volume", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.Volume"}

	model := api.NewTestAPI([]*api.Message{secret, volume}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	for _, expected := range []string{"Secret.swift", "Volume.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateMessage_WithNestedMessages(t *testing.T) {
	outDir := t.TempDir()

	nested1 := &api.Message{Name: "Nested1", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.WithNested.Nested1"}
	nested2 := &api.Message{Name: "Nested2", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.WithNested.Nested2"}
	withNested := &api.Message{
		Name:     "WithNested",
		Package:  "google.cloud.test.v1",
		ID:       ".google.cloud.test.v1.WithNested",
		Messages: []*api.Message{nested1, nested2},
	}

	model := api.NewTestAPI([]*api.Message{withNested}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "WithNested.swift")
	for _, unexpected := range []string{"Nested1.swift", "Nested2.swift"} {
		unexpectedFilename := filepath.Join(expectedDir, unexpected)
		if _, err := os.Stat(unexpectedFilename); err == nil {
			t.Errorf("unexpected file generated: %s", unexpectedFilename)
		}
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	startIdx1 := strings.Index(contentStr, "public struct Nested1")
	if startIdx1 == -1 {
		t.Fatal("missing public struct Nested1")
	}
	endIdx1 := strings.Index(contentStr[startIdx1:], "{")
	if endIdx1 == -1 {
		t.Fatal("missing { for Nested1")
	}
	decl1 := contentStr[startIdx1 : startIdx1+endIdx1]
	for _, p := range []string{"Codable", "Equatable", "GoogleCloudWkt._AnyPackable", "Sendable"} {
		if !strings.Contains(decl1, p) {
			t.Errorf("expected %q in Nested1 declaration, got: %s", p, decl1)
		}
	}

	startIdx2 := strings.Index(contentStr, "public struct Nested2")
	if startIdx2 == -1 {
		t.Fatal("missing public struct Nested2")
	}
	endIdx2 := strings.Index(contentStr[startIdx2:], "{")
	if endIdx2 == -1 {
		t.Fatal("missing { for Nested2")
	}
	decl2 := contentStr[startIdx2 : startIdx2+endIdx2]
	for _, p := range []string{"Codable", "Equatable", "GoogleCloudWkt._AnyPackable", "Sendable"} {
		if !strings.Contains(decl2, p) {
			t.Errorf("expected %q in Nested2 declaration, got: %s", p, decl2)
		}
	}
}

func TestGenerateMessage_WithNestedEnum(t *testing.T) {
	outDir := t.TempDir()

	nestedEnum := &api.Enum{Name: "NestedEnum", Package: "google.cloud.test.v1", ID: ".google.cloud.test.v1.WithNestedEnum.NestedEnum"}
	nestedEnum.Values = []*api.EnumValue{{Name: "NESTED_ENUM_UNSPECIFIED", Number: 0, Parent: nestedEnum}}
	nestedEnum.UniqueNumberValues = nestedEnum.Values

	withNested := &api.Message{
		Name:    "WithNestedEnum",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.WithNestedEnum",
		Enums:   []*api.Enum{nestedEnum},
	}

	model := api.NewTestAPI([]*api.Message{withNested}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "WithNestedEnum.swift")
	unexpectedFilename := filepath.Join(expectedDir, "NestedEnum.swift")
	if _, err := os.Stat(unexpectedFilename); err == nil {
		t.Errorf("unexpected file generated: %s", unexpectedFilename)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotBlock := extractBlock(t, contentStr, "public enum NestedEnum", ", Sendable {")
	wantBlock := "public enum NestedEnum: Codable, Equatable, Sendable {"
	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateMessage_WithExternalImports(t *testing.T) {
	outDir := t.TempDir()

	externalMessage := &api.Message{
		Name:    "ExternalMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalMessage",
	}

	message := &api.Message{
		Name:    "LocalMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.LocalMessage",
		Fields: []*api.Field{
			{
				Name:    "ext_field",
				Typez:   api.TypezMessage,
				TypezID: ".google.cloud.external.v1.ExternalMessage",
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{message}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"
	model.AddMessage(externalMessage)

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	swiftCfg := swiftConfig(t, []config.SwiftDependency{
		{
			ApiPackage: "google.cloud.external.v1",
			Name:       "GoogleCloudExternalV1",
		},
		{
			ApiPackage: "google.cloud.unused.v1",
			Name:       "GoogleCloudUnusedV1",
		},
	})

	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "LocalMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "import GoogleCloudExternalV1") {
		t.Errorf("expected 'import GoogleCloudExternalV1' in %s", filename)
	}
	if strings.Contains(contentStr, "import GoogleCloudUnusedV1") {
		t.Errorf("unexpected 'import GoogleCloudUnusedV1' in %s", filename)
	}
}

func TestGenerateMessage_WithRecursiveTypes(t *testing.T) {
	outDir := t.TempDir()

	nodeA := &api.Message{
		Name:    "NodeA",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.NodeA",
	}
	nodeB := &api.Message{
		Name:    "NodeB",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.NodeB",
	}

	fieldA := &api.Field{
		Name:     "node_b",
		Typez:    api.TypezMessage,
		TypezID:  ".google.cloud.test.v1.NodeB",
		Optional: true,
		Parent:   nodeA,
	}
	nodeA.Fields = []*api.Field{fieldA}

	fieldB := &api.Field{
		Name:     "node_a",
		Typez:    api.TypezMessage,
		TypezID:  ".google.cloud.test.v1.NodeA",
		Optional: true,
		Parent:   nodeB,
	}
	nodeB.Fields = []*api.Field{fieldB}

	// Set the MessageType fields correctly
	fieldA.MessageType = nodeB
	fieldB.MessageType = nodeA

	model := api.NewTestAPI([]*api.Message{nodeA, nodeB}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	// Run LabelRecursiveFields to mark recursive fields
	api.LabelRecursiveFields(model)

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filenameA := filepath.Join(expectedDir, "NodeA.swift")
	contentA, err := os.ReadFile(filenameA)
	if err != nil {
		t.Fatal(err)
	}
	contentStrA := string(contentA)

	// Verify struct property uses Recursive
	wantProp := "public var nodeB: GoogleCloudWkt.Recursive<NodeB>?"
	if !strings.Contains(contentStrA, wantProp) {
		t.Errorf("property definition mismatch: want %q; got:\n%s", wantProp, contentStrA)
	}

	// Verify initializer parameter uses the unwrapped type
	wantParam := "nodeB: NodeB? = nil"
	if !strings.Contains(contentStrA, wantParam) {
		t.Errorf("initializer parameter mismatch: want %q; got:\n%s", wantParam, contentStrA)
	}

	// Verify initializer maps the value
	wantMap := "self.nodeB = nodeB.map { GoogleCloudWkt.Recursive(value: $0) }"
	if !strings.Contains(contentStrA, wantMap) {
		t.Errorf("initializer body mapping mismatch: want %q; got:\n%s", wantMap, contentStrA)
	}
}

func TestGenerateMessage_SelfRecursive(t *testing.T) {
	outDir := t.TempDir()

	node := &api.Message{
		Name:    "Node",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.Node",
	}

	field := &api.Field{
		Name:     "child",
		Typez:    api.TypezMessage,
		TypezID:  ".google.cloud.test.v1.Node",
		Optional: true,
		Parent:   node,
	}
	node.Fields = []*api.Field{field}
	field.MessageType = node

	model := api.NewTestAPI([]*api.Message{node}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	// Run LabelRecursiveFields to mark recursive fields
	api.LabelRecursiveFields(model)

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "Node.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify struct property uses Recursive
	wantProp := "public var child: GoogleCloudWkt.Recursive<Node>?"
	if !strings.Contains(contentStr, wantProp) {
		t.Errorf("property definition mismatch: want %q; got:\n%s", wantProp, contentStr)
	}

	// Verify initializer parameter uses the unwrapped type
	wantParam := "child: Node? = nil"
	if !strings.Contains(contentStr, wantParam) {
		t.Errorf("initializer parameter mismatch: want %q; got:\n%s", wantParam, contentStr)
	}

	// Verify initializer maps the value
	wantMap := "self.child = child.map { GoogleCloudWkt.Recursive(value: $0) }"
	if !strings.Contains(contentStr, wantMap) {
		t.Errorf("initializer body mapping mismatch: want %q; got:\n%s", wantMap, contentStr)
	}
}

func TestGenerateMessage_RecursiveChain(t *testing.T) {
	outDir := t.TempDir()

	nodeA := &api.Message{
		Name:    "NodeA",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.NodeA",
	}
	nodeB := &api.Message{
		Name:    "NodeB",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.NodeB",
	}
	nodeC := &api.Message{
		Name:    "NodeC",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.NodeC",
	}

	fieldA := &api.Field{
		Name:     "node_b",
		Typez:    api.TypezMessage,
		TypezID:  ".google.cloud.test.v1.NodeB",
		Optional: true,
		Parent:   nodeA,
	}
	nodeA.Fields = []*api.Field{fieldA}

	fieldB := &api.Field{
		Name:     "node_c",
		Typez:    api.TypezMessage,
		TypezID:  ".google.cloud.test.v1.NodeC",
		Optional: true,
		Parent:   nodeB,
	}
	nodeB.Fields = []*api.Field{fieldB}

	fieldC := &api.Field{
		Name:     "node_a",
		Typez:    api.TypezMessage,
		TypezID:  ".google.cloud.test.v1.NodeA",
		Optional: true,
		Parent:   nodeC,
	}
	nodeC.Fields = []*api.Field{fieldC}

	fieldA.MessageType = nodeB
	fieldB.MessageType = nodeC
	fieldC.MessageType = nodeA

	model := api.NewTestAPI([]*api.Message{nodeA, nodeB, nodeC}, []*api.Enum{}, []*api.Service{})
	model.PackageName = "google.cloud.test.v1"

	// Run LabelRecursiveFields to mark recursive fields
	api.LabelRecursiveFields(model)

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")

	// Verify NodeA contains wrapped NodeB
	filenameA := filepath.Join(expectedDir, "NodeA.swift")
	contentA, err := os.ReadFile(filenameA)
	if err != nil {
		t.Fatal(err)
	}
	contentStrA := string(contentA)
	// Verify NodeA contains wrapped NodeB
	wantPropA := "public var nodeB: GoogleCloudWkt.Recursive<NodeB>?"
	if !strings.Contains(contentStrA, wantPropA) {
		t.Errorf("nodeB property definition mismatch: want %q; got:\n%s", wantPropA, contentStrA)
	}
	wantParamA := "nodeB: NodeB? = nil"
	if !strings.Contains(contentStrA, wantParamA) {
		t.Errorf("nodeB initializer parameter mismatch: want %q; got:\n%s", wantParamA, contentStrA)
	}

	// Verify NodeB contains wrapped NodeC
	filenameB := filepath.Join(expectedDir, "NodeB.swift")
	contentB, err := os.ReadFile(filenameB)
	if err != nil {
		t.Fatal(err)
	}
	contentStrB := string(contentB)
	wantPropB := "public var nodeC: GoogleCloudWkt.Recursive<NodeC>?"
	if !strings.Contains(contentStrB, wantPropB) {
		t.Errorf("nodeC property definition mismatch: want %q; got:\n%s", wantPropB, contentStrB)
	}
	wantParamB := "nodeC: NodeC? = nil"
	if !strings.Contains(contentStrB, wantParamB) {
		t.Errorf("nodeC initializer parameter mismatch: want %q; got:\n%s", wantParamB, contentStrB)
	}

	// Verify NodeC contains wrapped NodeA
	filenameC := filepath.Join(expectedDir, "NodeC.swift")
	contentC, err := os.ReadFile(filenameC)
	if err != nil {
		t.Fatal(err)
	}
	contentStrC := string(contentC)
	wantPropC := "public var nodeA: GoogleCloudWkt.Recursive<NodeA>?"
	if !strings.Contains(contentStrC, wantPropC) {
		t.Errorf("nodeA property definition mismatch: want %q; got:\n%s", wantPropC, contentStrC)
	}
	wantParamC := "nodeA: NodeA? = nil"
	if !strings.Contains(contentStrC, wantParamC) {
		t.Errorf("nodeA initializer parameter mismatch: want %q; got:\n%s", wantParamC, contentStrC)
	}
}
