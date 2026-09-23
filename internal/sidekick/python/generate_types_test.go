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

package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestGenerateTypes(t *testing.T) {
	outDir := t.TempDir()

	msgPayload := api.NewTestMessage("SecretPayload").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 1).
		WithFields(
			api.NewTestField("data").WithType(api.TypezBytes),
		)

	msgSecret := api.NewTestMessage("Secret").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 10).
		WithFields(
			api.NewTestField("name").WithType(api.TypezString),
			api.NewTestField("payload").
				WithType(api.TypezMessage).
				WithMessageType(msgPayload),
		)

	enumStatus := api.NewTestEnum("SecretStatus").
		WithPackage("google.cloud.secretmanager.v1").
		WithSourceLocation("google/cloud/secretmanager/v1/resources.proto", 20).
		WithValues(
			api.NewTestEnumValue("STATE_UNSPECIFIED", 0),
			api.NewTestEnumValue("ACTIVE", 1),
		)

	model := api.NewTestAPI([]*api.Message{msgPayload, msgSecret}, []*api.Enum{enumStatus}, nil)
	model.PackageName = "google.cloud.secretmanager.v1"
	model.Name = "google-cloud-secretmanager"

	library := &config.Library{
		Name:          "google-cloud-secretmanager",
		Version:       "1.0.0",
		CopyrightYear: "2026",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
		Python: &config.PythonPackage{
			DefaultVersion: "v1",
		},
	}

	if err := Generate(t.Context(), model, outDir, library); err != nil {
		t.Fatal(err)
	}

	typesInitPath := filepath.Join(outDir, "google", "cloud", "secretmanager_v1", "types", "__init__.py")
	initBytes, err := os.ReadFile(typesInitPath)
	if err != nil {
		t.Fatal(err)
	}
	initContent := string(initBytes)

	t.Run("types/__init__.py imports", func(t *testing.T) {
		gotImports := extractBlock(t, initContent, "from .resources import (", ")")
		wantImports := `from .resources import (
    Secret,
    SecretPayload,
    SecretStatus,
)`
		if diff := cmp.Diff(wantImports, gotImports); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("types/__init__.py __all__", func(t *testing.T) {
		gotAll := extractBlock(t, initContent, "__all__ = (", ")")
		wantAll := `__all__ = (
    'Secret',
    'SecretPayload',
    'SecretStatus',
)`
		if diff := cmp.Diff(wantAll, gotAll); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	protoPyPath := filepath.Join(outDir, "google", "cloud", "secretmanager_v1", "types", "resources.py")
	protoBytes, err := os.ReadFile(protoPyPath)
	if err != nil {
		t.Fatal(err)
	}
	protoContent := string(protoBytes)

	t.Run("types/resources.py manifest", func(t *testing.T) {
		gotManifest := extractBlock(t, protoContent, "__protobuf__ = proto.module(", ")")
		wantManifest := `__protobuf__ = proto.module(
    package='google.cloud.secretmanager.v1',
    manifest={
        'SecretStatus',
        'SecretPayload',
        'Secret',
    },
)`
		if diff := cmp.Diff(wantManifest, gotManifest); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("types/resources.py class definitions", func(t *testing.T) {
		gotClass := extractBlock(t, protoContent, "class Secret(proto.Message):", "    payload: 'SecretPayload' = proto.Field(")
		wantClass := `class Secret(proto.Message):
    r"""

    Attributes:
        name (str):

        payload (google.cloud.secretmanager_v1.types.SecretPayload):

    """

    name: str = proto.Field(
        proto.STRING,
        number=0,
    )
    payload: 'SecretPayload' = proto.Field(`
		if diff := cmp.Diff(wantClass, gotClass); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func extractBlock(t *testing.T, content, startStr, endStr string) string {
	t.Helper()
	start := strings.Index(content, startStr)
	if start == -1 {
		t.Fatalf("start marker %q not found in content:\n%s", startStr, content)
	}
	sub := content[start:]
	end := strings.Index(sub, endStr)
	if end == -1 {
		t.Fatalf("end marker %q not found in content after %q:\n%s", endStr, startStr, sub)
	}
	return sub[:end+len(endStr)]
}
