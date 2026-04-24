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
				Typez:   api.TypezMessage,
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

func TestGenerateService_PathParameters(t *testing.T) {
	for _, test := range []struct {
		name      string
		path      *api.PathTemplate
		wantBlock string
	}{
		{
			name: "Nested",
			path: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("secret", "name"),
			wantBlock: `let path = try { () throws -> String in
      guard let pathVariable0 = request.secret.map({ $0.name }), !pathVariable0.isEmpty else {
        throw GoogleCloudGax.RequestError.binding("'request.secret.name' is not set or is empty")
      }
      return "/v1/\(pathVariable0)"
    }()`,
		},
		{
			name: "Plain",
			path: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("name"),
			wantBlock: `let path = try { () throws -> String in
      guard let pathVariable0 = request.name as String?, !pathVariable0.isEmpty else {
        throw GoogleCloudGax.RequestError.binding("'request.name' is not set or is empty")
      }
      return "/v1/\(pathVariable0)"
    }()`,
		},
		{
			name: "Multiple strings",
			path: api.NewPathTemplate().
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("locations").
				WithVariableNamed("location"),
			wantBlock: `let path = try { () throws -> String in
      guard let pathVariable0 = request.project as String?, !pathVariable0.isEmpty else {
        throw GoogleCloudGax.RequestError.binding("'request.project' is not set or is empty")
      }
      guard let pathVariable1 = request.location, !pathVariable1.isEmpty else {
        throw GoogleCloudGax.RequestError.binding("'request.location' is not set or is empty")
      }
      return "/v1/projects/\(pathVariable0)/locations/\(pathVariable1)"
    }()`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()

			secretMessage := &api.Message{
				Name:    "Secret",
				Package: "google.cloud.secretmanager.v1",
				ID:      ".google.cloud.secretmanager.v1.Secret",
				Fields: []*api.Field{
					{
						Name:  "name",
						Typez: api.TypezString,
					},
				},
			}

			requestMessage := &api.Message{
				Name:    "CreateSecretRequest",
				Package: "google.cloud.secretmanager.v1",
				ID:      ".google.cloud.secretmanager.v1.CreateSecretRequest",
				Fields: []*api.Field{
					{
						Name:  "name",
						Typez: api.TypezString,
					},
					{
						Name:     "secret",
						Typez:    api.TypezMessage,
						TypezID:  ".google.cloud.secretmanager.v1.Secret",
						Optional: true,
					},
					{
						Name:  "project",
						Typez: api.TypezString,
					},
					{
						Name:     "location",
						Typez:    api.TypezString,
						Optional: true,
					},
				},
			}

			iam := &api.Service{
				Name: "SecretManagerService",
				Methods: []*api.Method{
					{
						Name:        "CreateSecret",
						InputTypeID: requestMessage.ID,
						InputType:   requestMessage,
						PathInfo: &api.PathInfo{
							Bindings: []*api.PathBinding{{
								Verb:         "POST",
								PathTemplate: test.path,
							}},
						},
					},
				},
			}

			model := api.NewTestAPI([]*api.Message{requestMessage, secretMessage}, nil, []*api.Service{iam})
			model.PackageName = "google.cloud.secretmanager.v1"

			cfg := &parser.ModelConfig{
				Codec: map[string]string{
					"copyright-year": "2038",
				},
			}

			if err := Generate(t.Context(), model, outDir, cfg, nil); err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(outDir, "Sources", "GoogleCloudSecretmanagerV1", "SecretManagerService.swift")
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			contentStr := string(content)

			gotBlock := extractBlock(t, contentStr, "let path = try { () throws -> String in", "    }()")
			if diff := cmp.Diff(test.wantBlock, gotBlock); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
