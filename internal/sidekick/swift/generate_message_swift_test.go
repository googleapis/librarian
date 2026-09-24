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
)

func swiftConfig(t *testing.T, extraDependencies []config.SwiftDependency) *config.SwiftPackage {
	t.Helper()
	deps := []config.SwiftDependency{
		{Name: "GoogleWKT", ApiPackage: wellKnownProtobufPackage},
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

	secret := api.NewTestMessage("Secret").WithPackage("google.cloud.test.v1")
	volume := api.NewTestMessage("Volume").WithPackage("google.cloud.test.v1")
	clash0 := api.NewTestMessage("HttpHealthCheck").WithPackage("google.cloud.test.v1")
	clash1 := api.NewTestMessage("HTTPHealthCheck").WithPackage("google.cloud.test.v1")
	clash2 := api.NewTestMessage("httpHealthCheck").WithPackage("google.cloud.test.v1")

	model := api.NewTestAPI([]*api.Message{secret, volume, clash0, clash1, clash2}, nil, nil)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	want := []string{
		"Secret.swift",
		"Volume.swift",
		"HttpHealthCheck.swift",
		"HTTPHealthCheck+000.swift",
		"httpHealthCheck+001.swift",
	}
	for _, expected := range want {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateMessage_WithNestedMessages(t *testing.T) {
	outDir := t.TempDir()

	nested1 := api.NewTestMessage("Nested1")
	nested2 := api.NewTestMessage("Nested2")
	withNested := api.NewTestMessage("WithNested").
		WithPackage("google.cloud.test.v1").
		WithMessages(nested1, nested2)

	model := api.NewTestAPI([]*api.Message{withNested}, nil, nil)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
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
	for _, p := range []string{"Codable", "Equatable", "GoogleWKT._AnyPackable", "Sendable"} {
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
	for _, p := range []string{"Codable", "Equatable", "GoogleWKT._AnyPackable", "Sendable"} {
		if !strings.Contains(decl2, p) {
			t.Errorf("expected %q in Nested2 declaration, got: %s", p, decl2)
		}
	}
}

func TestGenerateMessage_WithNestedEnum(t *testing.T) {
	outDir := t.TempDir()

	nestedEnum := api.NewTestEnum("NestedEnum").
		WithValues(api.NewTestEnumValue("NESTED_ENUM_UNSPECIFIED", 0))

	withNested := api.NewTestMessage("WithNestedEnum").
		WithPackage("google.cloud.test.v1").
		WithEnums(nestedEnum)

	model := api.NewTestAPI([]*api.Message{withNested}, nil, nil)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
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
	wantBlock := "public enum NestedEnum: Codable, Equatable, Hashable, Sendable {"
	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateMessage_WithExternalImports(t *testing.T) {
	outDir := t.TempDir()

	externalMessage := api.NewTestMessage("ExternalMessage").
		WithPackage("google.cloud.external.v1")

	message := api.NewTestMessage("LocalMessage").
		WithPackage("google.cloud.test.v1").
		WithFields(
			api.NewTestField("ext_field").WithMessageType(externalMessage),
		)

	model := api.NewTestAPI([]*api.Message{message}, nil, nil)
	model.AddMessage(externalMessage)

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

	library := &config.Library{
		Swift: swiftCfg,
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "LocalMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "public import GoogleCloudExternalV1") {
		t.Errorf("expected 'public import GoogleCloudExternalV1' in %s", filename)
	}
	if strings.Contains(contentStr, "import GoogleCloudUnusedV1") {
		t.Errorf("unexpected 'import GoogleCloudUnusedV1' in %s", filename)
	}
}

func TestGenerateMessage_WithRecursiveTypes(t *testing.T) {
	outDir := t.TempDir()

	nodeA := api.NewTestMessage("NodeA").WithPackage("google.cloud.test.v1")
	nodeB := api.NewTestMessage("NodeB").WithPackage("google.cloud.test.v1")

	fieldA := api.NewTestField("node_b").
		WithMessageType(nodeB).
		WithOptional()
	nodeA.WithFields(fieldA)

	fieldB := api.NewTestField("node_a").
		WithMessageType(nodeA).
		WithOptional()
	nodeB.WithFields(fieldB)

	model := api.NewTestAPI([]*api.Message{nodeA, nodeB}, nil, nil)

	// Run LabelRecursiveFields to mark recursive fields
	api.LabelRecursiveFields(model)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filenameA := filepath.Join(expectedDir, "NodeA.swift")
	contentA, err := os.ReadFile(filenameA)
	if err != nil {
		t.Fatal(err)
	}
	contentStrA := string(contentA)

	// Verify struct property uses WKTRecursive
	wantProp := "public var nodeB: GoogleWKT.WKTRecursive<NodeB>?"
	if !strings.Contains(contentStrA, wantProp) {
		t.Errorf("property definition mismatch: want %q; got:\n%s", wantProp, contentStrA)
	}

	// Verify initializer is the default initializer
	wantInit := "public init() {}"
	if !strings.Contains(contentStrA, wantInit) {
		t.Errorf("initializer mismatch: want %q; got:\n%s", wantInit, contentStrA)
	}
}

func TestGenerateMessage_SelfRecursive(t *testing.T) {
	outDir := t.TempDir()

	node := api.NewTestMessage("Node").WithPackage("google.cloud.test.v1")

	field := api.NewTestField("child").
		WithMessageType(node).
		WithOptional()
	node.WithFields(field)

	model := api.NewTestAPI([]*api.Message{node}, nil, nil)

	// Run LabelRecursiveFields to mark recursive fields
	api.LabelRecursiveFields(model)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "Node.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify struct property uses WKTRecursive
	wantProp := "public var child: GoogleWKT.WKTRecursive<Node>?"
	if !strings.Contains(contentStr, wantProp) {
		t.Errorf("property definition mismatch: want %q; got:\n%s", wantProp, contentStr)
	}

	// Verify initializer is the default initializer
	wantInit := "public init() {}"
	if !strings.Contains(contentStr, wantInit) {
		t.Errorf("initializer mismatch: want %q; got:\n%s", wantInit, contentStr)
	}
}

func TestGenerateMessage_RecursiveChain(t *testing.T) {
	outDir := t.TempDir()

	nodeA := api.NewTestMessage("NodeA").WithPackage("google.cloud.test.v1")
	nodeB := api.NewTestMessage("NodeB").WithPackage("google.cloud.test.v1")
	nodeC := api.NewTestMessage("NodeC").WithPackage("google.cloud.test.v1")

	fieldA := api.NewTestField("node_b").
		WithMessageType(nodeB).
		WithOptional()
	nodeA.WithFields(fieldA)

	fieldB := api.NewTestField("node_c").
		WithMessageType(nodeC).
		WithOptional()
	nodeB.WithFields(fieldB)

	fieldC := api.NewTestField("node_a").
		WithMessageType(nodeA).
		WithOptional()
	nodeC.WithFields(fieldC)

	model := api.NewTestAPI([]*api.Message{nodeA, nodeB, nodeC}, nil, nil)

	// Run LabelRecursiveFields to mark recursive fields
	api.LabelRecursiveFields(model)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
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
	wantPropA := "public var nodeB: GoogleWKT.WKTRecursive<NodeB>?"
	if !strings.Contains(contentStrA, wantPropA) {
		t.Errorf("nodeB property definition mismatch: want %q; got:\n%s", wantPropA, contentStrA)
	}
	wantInitA := "public init() {}"
	if !strings.Contains(contentStrA, wantInitA) {
		t.Errorf("nodeB initializer mismatch: want %q; got:\n%s", wantInitA, contentStrA)
	}

	// Verify NodeB contains wrapped NodeC
	filenameB := filepath.Join(expectedDir, "NodeB.swift")
	contentB, err := os.ReadFile(filenameB)
	if err != nil {
		t.Fatal(err)
	}
	contentStrB := string(contentB)
	wantPropB := "public var nodeC: GoogleWKT.WKTRecursive<NodeC>?"
	if !strings.Contains(contentStrB, wantPropB) {
		t.Errorf("nodeC property definition mismatch: want %q; got:\n%s", wantPropB, contentStrB)
	}
	wantInitB := "public init() {}"
	if !strings.Contains(contentStrB, wantInitB) {
		t.Errorf("nodeC initializer mismatch: want %q; got:\n%s", wantInitB, contentStrB)
	}

	// Verify NodeC contains wrapped NodeA
	filenameC := filepath.Join(expectedDir, "NodeC.swift")
	contentC, err := os.ReadFile(filenameC)
	if err != nil {
		t.Fatal(err)
	}
	contentStrC := string(contentC)
	wantPropC := "public var nodeA: GoogleWKT.WKTRecursive<NodeA>? = nil"
	if !strings.Contains(contentStrC, wantPropC) {
		t.Errorf("nodeA property definition mismatch: want %q; got:\n%s", wantPropC, contentStrC)
	}
	wantInitC := "public init() {}"
	if !strings.Contains(contentStrC, wantInitC) {
		t.Errorf("nodeA initializer mismatch: want %q; got:\n%s", wantInitC, contentStrC)
	}
}

func TestGenerateMessage_DocComments(t *testing.T) {
	outDir := t.TempDir()

	nested := api.NewTestMessage("NestedMessage").
		WithDocumentation("Documentation for NestedMessage.")

	msg := api.NewTestMessage("TestMessage").
		WithPackage("google.cloud.test.v1").
		WithDocumentation("Documentation for TestMessage.").
		WithMessages(nested)

	model := api.NewTestAPI([]*api.Message{msg}, nil, nil)

	library := &config.Library{
		Swift: swiftConfig(t, nil),
	}
	if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "TestMessage.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify top-level message documentation
	want := "/// Documentation for TestMessage.\npublic struct TestMessage"
	got := extractBlock(t, contentStr, "/// Documentation for TestMessage.", "public struct TestMessage")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify nested message documentation
	want = "  /// Documentation for NestedMessage.\n  public struct NestedMessage"
	got = extractBlock(t, contentStr, "  /// Documentation for NestedMessage.", "public struct NestedMessage")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateMessage_FoundationImport(t *testing.T) {
	for _, test := range []struct {
		name         string
		message      *api.Message
		wantImport   string
		unwantImport string
	}{
		{
			name: "without bytes field uses internal import Foundation",
			message: api.NewTestMessage("StringMessage").
				WithPackage("google.cloud.test.v1").
				WithFields(api.NewTestField("name").WithType(api.TypezString)),
			wantImport:   "\nimport Foundation\n",
			unwantImport: "\npublic import Foundation\n",
		},
		{
			name: "with bytes field uses public import Foundation",
			message: api.NewTestMessage("DataMessage").
				WithPackage("google.cloud.test.v1").
				WithFields(api.NewTestField("payload").WithType(api.TypezBytes)),
			wantImport:   "\npublic import Foundation\n",
			unwantImport: "\nimport Foundation\n",
		},
		{
			name: "with nested bytes message uses public import Foundation on parent",
			message: api.NewTestMessage("ParentMessage").
				WithPackage("google.cloud.test.v1").
				WithMessages(
					api.NewTestMessage("NestedData").
						WithPackage("google.cloud.test.v1").
						WithFields(api.NewTestField("payload").WithType(api.TypezBytes)),
				),
			wantImport:   "\npublic import Foundation\n",
			unwantImport: "\nimport Foundation\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			model := api.NewTestAPI([]*api.Message{test.message}, nil, nil)
			model.PackageName = "google.cloud.test.v1"
			library := &config.Library{
				Swift: swiftConfig(t, nil),
			}
			if err := Generate(t.Context(), model, outDir, library, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", test.message.Name+".swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			if !strings.Contains(contentStr, test.wantImport) {
				t.Errorf("expected %q in %s, got:\n%s", test.wantImport, filename, contentStr)
			}
			if test.unwantImport != "" && strings.Contains(contentStr, test.unwantImport) {
				t.Errorf("did not expect %q in %s, got:\n%s", test.unwantImport, filename, contentStr)
			}
		})
	}
}

func TestGenerateMessage_WKT(t *testing.T) {
	outDir := t.TempDir()

	kindEnum := api.NewTestEnum("Kind").
		WithPackage(wellKnownProtobufPackage).
		WithValues(api.NewTestEnumValue("TYPE_UNKNOWN", 0))
	fieldMsg := api.NewTestMessage("Field").
		WithPackage(wellKnownProtobufPackage).
		WithEnums(kindEnum)
	typeMsg := api.NewTestMessage("Type").
		WithPackage(wellKnownProtobufPackage).
		WithFields(api.NewTestField("fields").WithMessageType(fieldMsg).WithRepeated())
	syntaxEnum := api.NewTestEnum("Syntax").
		WithPackage(wellKnownProtobufPackage).
		WithValues(api.NewTestEnumValue("SYNTAX_PROTO2", 0))

	model := api.NewTestAPI([]*api.Message{fieldMsg, typeMsg}, []*api.Enum{syntaxEnum}, nil).
		WithPackageName(wellKnownProtobufPackage)
	library := &config.Library{
		Swift: &config.SwiftPackage{
			LibraryNameOverride: wellKnownSwiftPackage,
		},
	}
	module := &config.SwiftModule{
		ModuleType: "default",
	}
	if err := Generate(t.Context(), model, outDir, library, module); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		filename string
		wantDecl string
	}{
		{filename: "WKTType.swift", wantDecl: "public struct WKTType:"},
		{filename: "WKTType.swift", wantDecl: "public var fields: [WKTField] = []"},
		{filename: "WKTField.swift", wantDecl: "public struct WKTField:"},
		{filename: "WKTField.swift", wantDecl: "public enum Kind:"},
		{filename: "WKTSyntax.swift", wantDecl: "public enum WKTSyntax:"},
	} {
		content, err := os.ReadFile(filepath.Join(outDir, test.filename))
		if err != nil {
			t.Fatalf("expected generated file %s: %v", test.filename, err)
		}
		if !strings.Contains(string(content), test.wantDecl) {
			t.Errorf("expected %s to contain %q, got:\n%s", test.filename, test.wantDecl, string(content))
		}
	}
}
