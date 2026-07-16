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

func TestGenerateService_Telemetry(t *testing.T) {
	outDir := t.TempDir()

	req := api.NewTestMessage("GetSecretRequest").
		WithPackage("google.cloud.test.v1").
		WithFields(
			&api.Field{
				Name:  "project",
				Typez: api.TypezString,
			},
			&api.Field{
				Name:  "secret",
				Typez: api.TypezString,
			})
	res := api.NewTestMessage("Secret").WithPackage("google.cloud.test.v1")

	method := api.NewTestMethod("GetSecret").
		WithInput(req).
		WithOutput(res).
		WithVerb("GET").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v1").
				WithLiteral("projects").
				WithVariableNamed("project").
				WithLiteral("secrets").
				WithVariableNamed("secret"))

	service := api.NewTestService("SecretManager").
		WithPackage("google.cloud.test.v1").
		WithMethods(method)

	model := api.NewTestAPI([]*api.Message{req, res}, []*api.Enum{}, []*api.Service{service})
	model.PackageName = "google.cloud.test.v1"
	cfg := &parser.ModelConfig{
		Codec: map[string]string{
			"version": "1.2.3-preview",
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg, swiftConfig(t, nil)); err != nil {
		t.Fatal(err)
	}
	checkClientsContents(t, outDir)
	checkStubContents(t, outDir)
}

func checkClientsContents(t *testing.T, outDir string) {
	t.Helper()
	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "Clients.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)
	got := extractBlock(t, contentStr, "_gapicApiClientHeader(", "\n")
	want := "_gapicApiClientHeader(packageVersion: \"1.2.3-preview\")\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func checkStubContents(t *testing.T, outDir string) {
	t.Helper()
	filename := filepath.Join(outDir, "Sources", "GoogleCloudTestV1", "SecretManager+Stub.swift")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)
	got := extractBlock(t, contentStr, "req.setValue(Clients.clientHeader,", "\n")
	want := "req.setValue(Clients.clientHeader, forHTTPHeaderField: \"X-Goog-Api-Client\")\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
