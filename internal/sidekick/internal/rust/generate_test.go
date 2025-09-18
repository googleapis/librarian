// Copyright 2025 Google LLC
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

package rust

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/parser"
)

var (
	testdataDir, _ = filepath.Abs("../../testdata")
)

func TestRustFromOpenAPI(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "openapi",
			ServiceConfig:       path.Join(testdataDir, "googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
			SpecificationSource: path.Join(testdataDir, "openapi/secretmanager_openapi_v1.json"),
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"README.md", "Cargo.toml", "src/lib.rs"} {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
}

func TestRustFromProtobuf(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "google/cloud/secretmanager/v1",
		},
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "googleapis"),
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"README.md", "Cargo.toml", "src/lib.rs"} {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
}

func TestRustModuleRpc(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
			ServiceConfig:       "google/rpc/rpc_publish.yaml",
			SpecificationSource: "google/rpc",
		},
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "googleapis"),
		},
		Codec: map[string]string{
			"copyright-year":    "2025",
			"template-override": "templates/mod",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, path.Join(outDir, "rpc"), cfg); err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"mod.rs"} {
		filename := path.Join(outDir, "rpc", expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
}

func TestRustBootstrapWkt(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &config.Config{
		General: config.GeneralConfig{
			SpecificationFormat: "protobuf",
			SpecificationSource: "google/protobuf",
		},
		Source: map[string]string{
			"protobuf-root": testdataDir,
			"include-list":  "source_context.proto",
		},
		Codec: map[string]string{
			"copyright-year":    "2025",
			"template-override": "templates/mod",
			"module-path":       "crate",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(model, path.Join(outDir, "wkt"), cfg); err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"mod.rs"} {
		filename := path.Join(outDir, "wkt", expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
}

func requireProtoc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
}
