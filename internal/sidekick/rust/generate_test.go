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
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/parser"
)

var (
	testdataDir, _      = filepath.Abs("../testdata")
	localTestdataDir, _ = filepath.Abs("testdata")
	expectedInNosvc     = []string{
		"README.md",
		"Cargo.toml",
		path.Join("src", "lib.rs"),
		path.Join("src", "model.rs"),
		path.Join("src", "model", "debug.rs"),
		path.Join("src", "model", "deserialize.rs"),
		path.Join("src", "model", "serialize.rs"),
	}
	expectedInCrate = append(expectedInNosvc,
		path.Join("src", "builder.rs"),
		path.Join("src", "client.rs"),
		path.Join("src", "tracing.rs"),
		path.Join("src", "transport.rs"),
		path.Join("src", "stub.rs"),
		path.Join("src", "stub", "dynamic.rs"),
	)
	expectedInClient = []string{
		path.Join("mod.rs"),
		path.Join("model.rs"),
		path.Join("model", "debug.rs"),
		path.Join("model", "deserialize.rs"),
		path.Join("model", "serialize.rs"),
		path.Join("builder.rs"),
		path.Join("client.rs"),
		path.Join("tracing.rs"),
		path.Join("transport.rs"),
		path.Join("stub.rs"),
		path.Join("stub", "dynamic.rs"),
	}
	unexpectedInClient = []string{
		"README.md",
		"Cargo.toml",
		path.Join("src", "lib.rs"),
	}
	expectedInModule = []string{
		path.Join("mod.rs"),
		path.Join("debug.rs"),
		path.Join("deserialize.rs"),
		path.Join("serialize.rs"),
	}
	// The test search for API Version comments in the 50 lines prior
	// to the first #[cfg(feature = "${thing}")].
	versionDistance = 50
)

func TestCodecError(t *testing.T) {
	outDir := t.TempDir()

	goodConfig := parser.ModelConfig{
		SpecificationFormat: "openapi",
		ServiceConfig:       path.Join(testdataDir, "../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "../../testdata/secretmanager_openapi_v1.json"),
		Codec: map[string]string{
			"package:wkt": "source=google.protobuf,package=google-cloud-wkt",
		},
	}

	errorConfig := parser.ModelConfig{
		SpecificationFormat: "openapi",
		ServiceConfig:       path.Join(testdataDir, "../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "../../testdata/secretmanager_openapi_v1.json"),
		Codec: map[string]string{
			"--invalid--": "--invalid--",
		},
	}
	model, err := parser.CreateModel(errorConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, errorConfig); err == nil {
		t.Errorf("expected an error with invalid Codec options")
	}

	if err := GenerateStorage(t.Context(), outDir, model, errorConfig, model, goodConfig); err == nil {
		t.Errorf("expected an error with invalid Codec options for storage")
	}
	if err := GenerateStorage(t.Context(), outDir, model, goodConfig, model, errorConfig); err == nil {
		t.Errorf("expected an error with invalid Codec options for control")
	}
}

func TestRustFromOpenAPI(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := parser.ModelConfig{
		SpecificationFormat: "openapi",
		ServiceConfig:       path.Join(testdataDir, "../../testdata/googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "../../testdata/secretmanager_openapi_v1.json"),
		Codec: map[string]string{
			"package:wkt": "source=google.protobuf,package=google-cloud-wkt",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range expectedInCrate {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
}

func TestRustFromDiscovery(t *testing.T) {
	outDir := t.TempDir()

	cfg := parser.ModelConfig{
		SpecificationFormat: "discovery",
		ServiceConfig:       path.Join(testdataDir, "../../testdata/googleapis/google/cloud/compute/v1/compute_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "../../testdata/discovery/compute.v1.json"),
		Codec: map[string]string{
			"package:wkt":          "source=google.protobuf,package=google-cloud-wkt",
			"per-service-features": "true",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range expectedInCrate {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
	checkApiVersionComments(t, outDir)
}

func TestRustFromProtobuf(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := parser.ModelConfig{
		SpecificationFormat: "protobuf",
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "../../testdata/googleapis"),
		},
		Codec: map[string]string{
			"package:wkt":                   "source=google.protobuf,package=google-cloud-wkt",
			"package:google-cloud-api":      "source=google.api,package=google-cloud-api",
			"package:google-cloud-location": "source=google.cloud.location,package=google-cloud-location",
			"package:google-cloud-iam-v1":   "source=google.iam.v1,package=google-cloud-iam-v1",
			"package:google-cloud-type":     "source=google.type,package=google-cloud-type",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range expectedInCrate {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
}

func TestRustClient(t *testing.T) {
	requireProtoc(t)
	for _, override := range []string{"http-client", "grpc-client"} {
		outDir := t.TempDir()

		cfg := parser.ModelConfig{
			SpecificationFormat: "protobuf",
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "google/cloud/secretmanager/v1",
			Source: map[string]string{
				"googleapis-root": path.Join(testdataDir, "../../testdata/googleapis"),
			},
			Codec: map[string]string{
				"copyright-year":                "2025",
				"template-override":             path.Join("templates", override),
				"package:wkt":                   "source=google.protobuf,package=google-cloud-wkt",
				"package:google-cloud-api":      "source=google.api,package=google-cloud-api",
				"package:google-cloud-location": "source=google.cloud.location,package=google-cloud-location",
				"package:google-cloud-iam-v1":   "source=google.iam.v1,package=google-cloud-iam-v1",
				"package:google-cloud-type":     "source=google.type,package=google-cloud-type",
			},
		}
		model, err := parser.CreateModel(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if err := Generate(t.Context(), model, outDir, cfg); err != nil {
			t.Fatal(err)
		}
		for _, expected := range expectedInClient {
			filename := path.Join(outDir, expected)
			stat, err := os.Stat(filename)
			if os.IsNotExist(err) {
				t.Errorf("missing %s: %s", filename, err)
			}
			if stat.Mode().Perm()|0666 != 0666 {
				t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
			}
		}
		for _, unexpected := range unexpectedInClient {
			filename := path.Join(outDir, unexpected)
			if stat, err := os.Stat(filename); err == nil {
				t.Errorf("did not expect file %s, got=%v", unexpected, stat)
			}
		}
		importsModelModules(t, path.Join(outDir, "model.rs"))
	}
}

func TestRustNosvc(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := parser.ModelConfig{
		SpecificationFormat: "protobuf",
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "../../testdata/googleapis"),
		},
		Codec: map[string]string{
			"copyright-year":                "2025",
			"template-override":             path.Join("templates", "nosvc"),
			"package:wkt":                   "source=google.protobuf,package=google-cloud-wkt",
			"package:google-cloud-api":      "source=google.api,package=google-cloud-api",
			"package:google-cloud-location": "source=google.cloud.location,package=google-cloud-location",
			"package:google-cloud-iam-v1":   "source=google.iam.v1,package=google-cloud-iam-v1",
			"package:google-cloud-type":     "source=google.type,package=google-cloud-type",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range expectedInNosvc {
		filename := path.Join(outDir, expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
}

func TestRustModuleRpc(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := parser.ModelConfig{
		SpecificationFormat: "protobuf",
		ServiceConfig:       "google/rpc/rpc_publish.yaml",
		SpecificationSource: "google/rpc",
		Source: map[string]string{
			"googleapis-root": path.Join(testdataDir, "../../testdata/googleapis"),
		},
		Codec: map[string]string{
			"copyright-year":    "2025",
			"template-override": "templates/mod",
			"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
		},
	}
	model, err := parser.CreateModel(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(t.Context(), model, path.Join(outDir, "rpc"), cfg); err != nil {
		t.Fatal(err)
	}

	for _, expected := range expectedInModule {
		filename := path.Join(outDir, "rpc", expected)
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0666 != 0666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "rpc", "mod.rs"))
}

func TestRustBootstrapWkt(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := parser.ModelConfig{
		SpecificationFormat: "protobuf",
		SpecificationSource: "google/protobuf",
		Source: map[string]string{
			"protobuf-root": localTestdataDir,
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
	if err := Generate(t.Context(), model, path.Join(outDir, "wkt"), cfg); err != nil {
		t.Fatal(err)
	}

	for _, expected := range expectedInModule {
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

func importsModelModules(t *testing.T, filename string) {
	t.Helper()
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n")
	for _, want := range []string{"mod debug;", "mod serialize;", "mod deserialize;"} {
		if !slices.Contains(lines, want) {
			t.Errorf("expected file %s to have a line matching %q, got:\n%s", filename, want, contents)
		}
	}
}

func checkApiVersionComments(t *testing.T, outDir string) {
	t.Helper()
	contents, err := os.ReadFile(path.Join(outDir, "src", "client.rs"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(contents), "\n")
	for _, test := range []struct {
		featureName string
		wantVersion string
	}{
		{
			featureName: "accelerator-types",
			wantVersion: "v1_20260131",
		},
		{
			featureName: "addresses",
			wantVersion: "v1_20260205",
		},
	} {
		t.Run(test.featureName, func(t *testing.T) {
			target := fmt.Sprintf(`#[cfg(feature = "%s")]`, test.featureName)
			anchor := slices.Index(lines, target)
			if anchor <= versionDistance {
				t.Fatalf("cannot find %s in generated client.rs file: \n%v", target, lines[0:versionDistance])
			}
			subHaystack := lines[anchor-versionDistance : anchor]
			found := slices.IndexFunc(subHaystack, func(line string) bool {
				return strings.Contains(line, test.wantVersion)
			})
			if found == -1 {
				t.Errorf("cannot find API version (%s) in client comments: %v", test.wantVersion, subHaystack)
			}
		})
	}
	checkNoCommentsWithoutApiVersion(t, lines)
}

func checkNoCommentsWithoutApiVersion(t *testing.T, lines []string) {
	target := `#[cfg(feature = "instances")]`
	anchor := slices.Index(lines, target)
	if anchor <= versionDistance {
		t.Fatalf("cannot find %s in generated client.rs file: \n%v", target, lines[0:versionDistance])
	}
	subHaystack := lines[anchor-versionDistance : anchor]
	found := slices.IndexFunc(subHaystack, func(line string) bool {
		return strings.Contains(line, " with API version ")
	})
	if found != -1 {
		t.Errorf("found unexpected API version line (%s) in client comments: %v", subHaystack[found], subHaystack)
	}
}

func requireProtoc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("skipping test because protoc is not installed")
	}
}
