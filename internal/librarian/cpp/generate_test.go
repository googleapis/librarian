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

package cpp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestGenerate_ValidationErrors(t *testing.T) {
	ctx := t.Context()

	t.Run("nil library", func(t *testing.T) {
		err := Generate(ctx, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "missing cpp configuration") {
			t.Fatalf("expected missing cpp configuration error, got: %v", err)
		}
	})

	t.Run("nil cpp config", func(t *testing.T) {
		lib := &config.Library{Name: "test-lib"}
		err := Generate(ctx, nil, lib, nil)
		if err == nil || !strings.Contains(err.Error(), "library test-lib is missing cpp configuration") {
			t.Fatalf("expected missing cpp configuration error, got: %v", err)
		}
	})

	t.Run("no apis", func(t *testing.T) {
		lib := &config.Library{
			Name: "test-lib",
			Cpp:  &config.CppLibrary{},
		}
		err := Generate(ctx, nil, lib, nil)
		if err == nil || !strings.Contains(err.Error(), "library test-lib has no configured APIs") {
			t.Fatalf("expected no configured APIs error, got: %v", err)
		}
	})

	t.Run("missing googleapis source", func(t *testing.T) {
		lib := &config.Library{
			Name: "test-lib",
			Cpp:  &config.CppLibrary{},
			APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
		}
		err := Generate(ctx, nil, lib, nil)
		if err == nil || !strings.Contains(err.Error(), "missing googleapis source for library test-lib") {
			t.Fatalf("expected missing googleapis source error, got: %v", err)
		}
	})

	t.Run("missing service config", func(t *testing.T) {
		lib := &config.Library{
			Name: "test-lib",
			Cpp:  &config.CppLibrary{},
			APIs: []*config.API{{Path: "nonexistent/api/v1"}},
		}
		src := &sources.Sources{Googleapis: t.TempDir()}
		err := Generate(ctx, nil, lib, src)
		if err == nil {
			t.Fatal("expected error for nonexistent service config, got nil")
		}
	})

	t.Run("uses override service config yaml name", func(t *testing.T) {
		tempDir := t.TempDir()
		overrideYAML := "custom_service.yaml"
		lib := &config.Library{
			Name: "test-lib",
			Cpp: &config.CppLibrary{
				OverrideServiceConfigYAMLName: overrideYAML,
			},
			APIs: []*config.API{{Path: "nonexistent/api/v1"}},
		}
		src := &sources.Sources{Googleapis: tempDir}
		pc := &config.Protoc{}
		modelCfg, err := libraryToModelConfig(lib, lib.APIs[0], src, pc)
		if err != nil {
			t.Fatalf("expected nil error when OverrideServiceConfigYAMLName is provided, got: %v", err)
		}
		if modelCfg.ServiceConfig != overrideYAML {
			t.Errorf("ServiceConfig mismatch: got %q, want %q", modelCfg.ServiceConfig, overrideYAML)
		}
	})
}

func TestDefaultOutput(t *testing.T) {
	tests := []struct {
		name       string
		api        string
		defaultOut string
		want       string
	}{
		{
			name:       "empty defaultOut",
			api:        "google/cloud/secretmanager/v1",
			defaultOut: "",
			want:       "google/cloud/secretmanager/v1",
		},
		{
			name:       "with defaultOut",
			api:        "google/cloud/secretmanager/v1",
			defaultOut: "generated",
			want:       "generated/google/cloud/secretmanager/v1",
		},
		{
			name:       "absolute defaultOut",
			api:        "test/v1",
			defaultOut: "/var/tmp/output",
			want:       "/var/tmp/output/test/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultOutput(tt.api, tt.defaultOut)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("DefaultOutput mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerate_Success(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")

	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	library := &config.Library{
		Name:                "google-cloud-secretmanager-v1",
		Output:              outDir,
		APIs:                []*config.API{{Path: "google/cloud/secretmanager/v1"}},
		SpecificationFormat: config.SpecProtobuf,
		Cpp:                 &config.CppLibrary{},
	}
	src := &sources.Sources{Googleapis: googleapisDir}
	cfg := &config.Config{}

	if err := Generate(t.Context(), cfg, library, src); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	cmakeFile := filepath.Join(outDir, "CMakeLists.txt")
	content, err := os.ReadFile(cmakeFile)
	if err != nil {
		t.Fatalf("failed to read generated CMakeLists.txt: %v", err)
	}
	if !strings.Contains(string(content), "Generated by the Codegen C++ plugin.") {
		t.Errorf("expected generated CMakeLists.txt to contain prologue banner, got:\n%s", string(content))
	}
}
