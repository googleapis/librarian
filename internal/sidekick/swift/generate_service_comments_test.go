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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateService_DocComments(t *testing.T) {
	outDir := t.TempDir()

	req := &api.Message{
		Name:    "GetSecretRequest",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.GetSecretRequest",
		Fields: []*api.Field{
			{
				Name:  "project",
				ID:    ".google.cloud.test.v1.GetSecretRequest.project",
				Typez: api.TypezString,
			},
			{
				Name:  "secret",
				ID:    ".google.cloud.test.v1.GetSecretRequest.secret",
				Typez: api.TypezString,
			},
		},
	}
	req.Fields[0].Parent = req
	req.Fields[1].Parent = req
	res := &api.Message{
		Name:    "Secret",
		Package: "google.cloud.test.v1",
		ID:      ".google.cloud.test.v1.Secret",
	}

	method := &api.Method{
		Name:          "GetSecret",
		Documentation: "Documentation for GetSecret method.",
		InputTypeID:   req.ID,
		OutputTypeID:  res.ID,
		InputType:     req,
		OutputType:    res,
		PathInfo: &api.PathInfo{
			Bindings: []*api.PathBinding{
				{
					Verb:         "GET",
					PathTemplate: (&api.PathTemplate{}).WithLiteral("v1").WithLiteral("projects").WithVariableNamed("project").WithLiteral("secrets").WithVariableNamed("secret"),
				},
			},
		},
	}

	service := &api.Service{
		Name:          "SecretManager",
		Package:       "google.cloud.test.v1",
		Documentation: "Documentation for SecretManager service.",
		Methods:       []*api.Method{method},
	}
	method.Service = service

	model := api.NewTestAPI([]*api.Message{req, res}, []*api.Enum{}, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"
	cfg := &parser.ModelConfig{}

	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "SecretManager.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// Verify service documentation
	want := "/// Documentation for SecretManager service.\n///\n/// @Snippet(path: \"SecretManagerQuickstart\")\npublic class SecretManagerClient"
	got := extractBlock(t, contentStr, "/// Documentation for SecretManager service.", "public class SecretManagerClient")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify method documentation
	want = "  /// Documentation for GetSecret method.\n  ///\n  /// @Snippet(path: \"SecretManager_GetSecret\")\n  public func getSecret"
	got = extractBlock(t, contentStr, "  /// Documentation for GetSecret method.", "public func getSecret")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
