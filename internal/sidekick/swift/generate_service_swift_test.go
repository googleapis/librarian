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

func TestGenerateService_Files(t *testing.T) {
	outDir := t.TempDir()

	iam := &api.Service{Name: "IAM"}
	secretManager := &api.Service{Name: "SecretManagerService"}

	model := api.NewTestAPI(nil, nil, []*api.Service{iam, secretManager})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	for _, expected := range []string{"IAM.swift", "SecretManagerService.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateServiceSwift_SnippetReference(t *testing.T) {
	outDir := t.TempDir()

	// "Protocol" is a reserved word that gets mangled to "Protocol_"
	service := &api.Service{Name: "Protocol"}

	model := api.NewTestAPI(nil, nil, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	// The file name uses the unmangled name
	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Protocol.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	gotBlock := extractBlock(t, contentStr, "/// @Snippet", "public class Protocol_ {")
	wantBlock := `/// @Snippet(id: "ProtocolQuickstart")
public class Protocol_ {`

	if diff := cmp.Diff(wantBlock, gotBlock); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateService_SnippetFiles(t *testing.T) {
	outDir := t.TempDir()

	iam := &api.Service{Name: "IAM"}
	secretManager := &api.Service{Name: "SecretManagerService"}

	model := api.NewTestAPI(nil, nil, []*api.Service{iam, secretManager})
	model.PackageName = "google.cloud.test.v1"

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Snippets")
	for _, expected := range []string{"IAMQuickstart.swift", "SecretManagerServiceQuickstart.swift"} {
		filename := filepath.Join(expectedDir, expected)
		if _, err := os.Stat(filename); err != nil {
			t.Error(err)
		}
	}
}

func TestGenerateService_WithImports(t *testing.T) {
	outDir := t.TempDir()

	externalMessage := &api.Message{
		Name:    "ExternalMessage",
		Package: "google.cloud.external.v1",
		ID:      ".google.cloud.external.v1.ExternalMessage",
	}

	inputMessage := &api.Message{
		Name:    "LocalMessage",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.LocalMessage",
		Fields: []*api.Field{
			{
				Name:    "ext_field",
				Typez:   api.MESSAGE_TYPE,
				TypezID: ".google.cloud.external.v1.ExternalMessage",
			},
		},
	}

	iam := &api.Service{
		Name: "IAM",
		Methods: []*api.Method{
			{
				Name:      "TestMethod",
				InputType: inputMessage,
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{{Verb: "POST", PathTemplate: &api.PathTemplate{}}},
				},
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{inputMessage}, nil, []*api.Service{iam})
	model.PackageName = "google.cloud.test.v1"
	model.State = &api.APIState{
		MessageByID: map[string]*api.Message{
			".google.cloud.external.v1.ExternalMessage": externalMessage,
		},
	}

	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"copyright-year": "2038",
		},
	}

	swiftCfg := &config.SwiftPackage{
		SwiftDefault: config.SwiftDefault{
			Dependencies: []config.SwiftDependency{
				{
					Name:               "GoogleCloudGax",
					RequiredByServices: true,
				},
				{
					Name:               "GoogleCloudAuth",
					RequiredByServices: true,
				},
				{
					ApiPackage: "google.cloud.external.v1",
					Name:       "GoogleCloudExternalV1",
				},
			},
		},
	}

	if err := Generate(t.Context(), model, outDir, cfg, swiftCfg); err != nil {
		t.Fatal(err)
	}

	expectedDir := filepath.Join(outDir, "Sources", "GoogleCloudTestV1")
	filename := filepath.Join(expectedDir, "IAM.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	expectedImports := `import GoogleCloudAuth
import GoogleCloudGax

import GoogleCloudExternalV1`

	if !strings.Contains(contentStr, expectedImports) {
		t.Errorf("expected imports block not found in %s. Got content:\n%s", filename, contentStr)
	}
}
