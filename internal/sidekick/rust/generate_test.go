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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sources"
)

var (
	testdataDir, _      = filepath.Abs("../../testdata")
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

	goodConfig := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecOpenAPI,
		ServiceConfig:       path.Join(testdataDir, "googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "secretmanager_openapi_v1.json"),
		Codec: map[string]string{
			"package:wkt": "source=google.protobuf,package=google-cloud-wkt",
		},
	}

	errorConfig := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecOpenAPI,
		ServiceConfig:       path.Join(testdataDir, "googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "secretmanager_openapi_v1.json"),
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

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecOpenAPI,
		ServiceConfig:       path.Join(testdataDir, "googleapis/google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "secretmanager_openapi_v1.json"),
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
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
}

func TestRustFromDiscovery(t *testing.T) {
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecDiscovery,
		ServiceConfig:       path.Join(testdataDir, "googleapis/google/cloud/compute/v1/small-compute_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "discovery/small-compute.v1.json"),
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
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
	checkApiVersionComments(t, outDir)
}

func TestRustFromDiscoveryWithLro(t *testing.T) {
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecDiscovery,
		ServiceConfig:       path.Join(testdataDir, "googleapis/google/cloud/compute/v1/compute_v1.yaml"),
		SpecificationSource: path.Join(testdataDir, "discovery/small-compute.v1.json"),
		Codec: map[string]string{
			"package:wkt":          "source=google.protobuf,package=google-cloud-wkt",
			"per-service-features": "true",
		},
		Discovery: &api.Discovery{
			OperationID: ".google.cloud.compute.v1.Operation",
			Pollers: []*api.Poller{
				{
					Prefix:   "compute/v1/projects/{project}/zones/{zone}",
					MethodID: ".google.cloud.compute.v1.zoneOperations.get",
				},
				{
					Prefix:   "compute/v1/projects/{project}/regions/{region}",
					MethodID: ".google.cloud.compute.v1.regionOperations.get",
				},
				{
					Prefix:   "compute/v1/projects/{project}",
					MethodID: ".google.cloud.compute.v1.globalOperations.get",
				},
				{
					Prefix:   "compute/v1/",
					MethodID: ".google.cloud.compute.v1.globalOrganizationOperations.get",
				},
			},
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
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	contentBytes, err := os.ReadFile(path.Join(outDir, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	contents := string(contentBytes)
	got := extractBlock(t, contents, fmt.Sprintf("\n%s", lroOperationFeature), " = []\n")
	want := fmt.Sprintf("\n%s = []\n", lroOperationFeature)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s\n====\n%s", diff, contents)
	}
	got = extractBlock(t, contents, "\nautoscalers = ", "]\n")
	want = fmt.Sprintf("\nautoscalers = [\"%s\", ]\n", lroOperationFeature)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s\n====\n%s", diff, contents)
	}
}

func TestRustFromProtobuf(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: path.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
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
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
}

func TestRustClient(t *testing.T) {
	requireProtoc(t)
	for _, override := range []string{"http-client", "grpc-client"} {
		outDir := t.TempDir()

		cfg := &parser.ModelConfig{
			SpecificationFormat: libconfig.SpecProtobuf,
			ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
			SpecificationSource: "google/cloud/secretmanager/v1",
			Source: &sources.SourceConfig{
				Sources: &sources.Sources{
					Googleapis: path.Join(testdataDir, "googleapis"),
				},
				ActiveRoots: []string{"googleapis"},
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
			if errors.Is(err, fs.ErrNotExist) {
				t.Errorf("missing %s: %s", filename, err)
			}
			if stat.Mode().Perm()|0o666 != 0o666 {
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

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		ServiceConfig:       "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
		SpecificationSource: "google/cloud/secretmanager/v1",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: path.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
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
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "src", "model.rs"))
}

func TestRustModuleRpc(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		ServiceConfig:       "google/rpc/rpc_publish.yaml",
		SpecificationSource: "google/rpc",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				Googleapis: path.Join(testdataDir, "googleapis"),
			},
			ActiveRoots: []string{"googleapis"},
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
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
			t.Errorf("generated files should not be executable %s: %o", filename, stat.Mode())
		}
	}
	importsModelModules(t, path.Join(outDir, "rpc", "mod.rs"))
}

func TestRustBootstrapWkt(t *testing.T) {
	requireProtoc(t)
	outDir := t.TempDir()

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		SpecificationSource: "google/protobuf",
		Source: &sources.SourceConfig{
			Sources: &sources.Sources{
				ProtobufSrc: localTestdataDir,
			},
			ActiveRoots: []string{"protobuf-src"},
			IncludeList: []string{"source_context.proto"},
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
		if errors.Is(err, fs.ErrNotExist) {
			t.Errorf("missing %s: %s", filename, err)
		}
		if stat.Mode().Perm()|0o666 != 0o666 {
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
	lines := strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n")
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

func TestValidateTemplates(t *testing.T) {
	if err := parsedTemplates.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateExtensibleBuilders(t *testing.T) {
	msg := api.NewTestMessage("Request").WithPackage("test.v1")
	method1 := api.NewTestMethod("GetResource").
		WithInput(msg).
		WithOutput(msg).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())
	method2 := api.NewTestMethod("ListResources").
		WithInput(msg).
		WithOutput(msg).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())
	svc1 := api.NewTestService("ExtensibleService").
		WithPackage("test.v1").
		WithMethods(method1, method2)

	method3 := api.NewTestMethod("OtherAction").
		WithInput(msg).
		WithOutput(msg).
		WithVerb("POST").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())
	svc2 := api.NewTestService("PrivateService").
		WithPackage("test.v1").
		WithMethods(method3)

	model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{svc1, svc2}).
		WithPackageName("test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"package:wkt":         "source=google.protobuf,package=google-cloud-wkt",
			"extensible-builders": ".test.v1.ExtensibleService.GetResource",
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	builderBytes, err := os.ReadFile(filepath.Join(outDir, "src", "builder.rs"))
	if err != nil {
		t.Fatal(err)
	}
	builderRs := string(builderBytes)

	wantSnippets := []string{
		"pub(crate) stub: std::sync::Arc<dyn super::super::stub::dynamic::ExtensibleService>,",
		"pub struct GetResource(pub(crate) RequestBuilder<crate::model::Request>);",
		"pub struct ListResources(RequestBuilder<crate::model::Request>);",
		"stub: std::sync::Arc<dyn super::super::stub::dynamic::PrivateService>,",
		"pub struct OtherAction(RequestBuilder<crate::model::Request>);",
	}
	for _, want := range wantSnippets {
		if !strings.Contains(builderRs, want) {
			t.Errorf("builder.rs missing %q in:\n%s", want, builderRs)
		}
	}
}

func TestGenerateRedactedDebugFields(t *testing.T) {
	nested := api.NewTestMessage("Nested").
		WithFields(
			api.NewTestField("algorithm").WithType(api.TypezString),
			api.NewTestField("raw_key").WithType(api.TypezBytes),
		)

	req := api.NewTestMessage("Request").
		WithPackage("test.v1").
		WithMessages(nested)

	secretMap := api.NewTestMap("$SecretMap", api.TypezString, api.TypezBytes)

	credPublic := api.NewTestField("public_id").WithType(api.TypezString).WithOneOf()
	credPublicNested := api.NewTestField("public_nested").WithMessageType(nested).WithOneOf()
	credSecret := api.NewTestField("secret_token").WithType(api.TypezString).WithOneOf()
	credNested := api.NewTestField("secret_nested").WithMessageType(nested).WithOneOf()
	credDeprecated := api.NewTestField("deprecated_token").WithType(api.TypezString).WithDeprecated(true).WithOneOf()
	credentialOneOf := api.NewTestOneOf("credential").WithFields(
		credPublic,
		credPublicNested,
		credSecret,
		credNested,
		credDeprecated,
	)

	wholeVariant := api.NewTestField("secret_payload").WithType(api.TypezBytes).WithOneOf()
	wholeSecretOneOf := api.NewTestOneOf("whole_secret").WithFields(wholeVariant)

	normalVariant := api.NewTestField("label").WithType(api.TypezString).WithOneOf()
	normalOneOf := api.NewTestOneOf("normal_choice").WithFields(normalVariant)

	req.WithFields(
		api.NewTestField("public_name").WithType(api.TypezString),
		api.NewTestField("encryption_key_bytes").WithType(api.TypezBytes),
		api.NewTestField("encryption_key_sha256_bytes").WithType(api.TypezBytes),
		api.NewTestField("optional_secret").WithType(api.TypezString).WithOptional(),
		api.NewTestField("repeated_secrets").WithType(api.TypezBytes).WithRepeated(),
		api.NewTestField("secret_headers").WithMapType(secretMap),
		api.NewTestField("nested").WithMessageType(nested).WithOptional(),
	).WithOneOfs(credentialOneOf, wholeSecretOneOf, normalOneOf)

	model := api.NewTestAPI([]*api.Message{req}, nil, nil).
		WithPackageName("test.v1").
		WithMaps(secretMap)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"package:wkt":    "source=google.protobuf,package=google-cloud-wkt",
			"name-overrides": ".test.v1.Request.credential=CredentialOneOf",
			"redacted-debug-fields": strings.Join([]string{
				".test.v1.Request.encryption_key_bytes",
				".test.v1.Request.optional_secret",
				".test.v1.Request.repeated_secrets",
				".test.v1.Request.secret_headers",
				".test.v1.Request.Nested.raw_key",
				".test.v1.Request.secret_token",
				".test.v1.Request.secret_nested",
				".test.v1.Request.whole_secret",
			}, ","),
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	debugBytes, err := os.ReadFile(filepath.Join(outDir, "src", "model", "debug.rs"))
	if err != nil {
		t.Fatal(err)
	}
	debugRs := string(debugBytes)

	wantDebugSnippets := []string{
		`debug_struct.field("public_name", &self.public_name);`,
		`debug_struct.field("encryption_key_bytes", &"[censored]");`,
		`debug_struct.field("encryption_key_sha256_bytes", &self.encryption_key_sha256_bytes);`,
		`debug_struct.field("optional_secret", &"[censored]");`,
		`debug_struct.field("repeated_secrets", &"[censored]");`,
		`debug_struct.field("secret_headers", &"[censored]");`,
		`debug_struct.field("nested", &self.nested);`,
		`debug_struct.field("credential", &self.credential);`,
		`debug_struct.field("whole_secret", &"[censored]");`,
		`debug_struct.field("normal_choice", &self.normal_choice);`,
		`debug_struct.field("algorithm", &self.algorithm);`,
		`debug_struct.field("raw_key", &"[censored]");`,
		`impl std::fmt::Debug for super::request::CredentialOneOf {`,
		`Self::PublicId(v) => f.debug_tuple("PublicId").field(v).finish(),`,
		`Self::PublicNested(v) => f.debug_tuple("PublicNested").field(v).finish(),`,
		`Self::SecretToken(_) => f.debug_tuple("SecretToken").field(&"[censored]").finish(),`,
		`Self::SecretNested(_) => f.debug_tuple("SecretNested").field(&"[censored]").finish(),`,
		"#[allow(deprecated)]\n            Self::DeprecatedToken(v) => f.debug_tuple(\"DeprecatedToken\").field(v).finish(),",
		`impl std::fmt::Debug for super::request::WholeSecret {`,
		`Self::SecretPayload(_) => f.debug_tuple("SecretPayload").field(&"[censored]").finish(),`,
	}
	for _, want := range wantDebugSnippets {
		if !strings.Contains(debugRs, want) {
			t.Errorf("debug.rs missing %q in:\n%s", want, debugRs)
		}
	}
	nestedImpl := extractBlock(t, debugRs, "impl std::fmt::Debug for super::request::Nested {", "\n}")
	if !strings.Contains(nestedImpl, `debug_struct.field("raw_key", &"[censored]");`) {
		t.Errorf("Nested Debug impl missing redacted raw_key in:\n%s", nestedImpl)
	}
	if !strings.Contains(nestedImpl, `debug_struct.field("algorithm", &self.algorithm);`) {
		t.Errorf("Nested Debug impl missing uncensored algorithm in:\n%s", nestedImpl)
	}
	for _, unwanted := range []string{
		`&self.encryption_key_bytes`,
		`&self.optional_secret`,
		`&self.repeated_secrets`,
		`&self.secret_headers`,
		`&self.raw_key`,
		`&self.whole_secret`,
		`impl std::fmt::Debug for super::request::NormalChoice`,
	} {
		if strings.Contains(debugRs, unwanted) {
			t.Errorf("debug.rs unexpectedly contained %q in:\n%s", unwanted, debugRs)
		}
	}

	modelBytes, err := os.ReadFile(filepath.Join(outDir, "src", "model.rs"))
	if err != nil {
		t.Fatal(err)
	}
	modelRs := string(modelBytes)
	wantCustomOneOf := `#[derive(Clone, PartialEq)]
    #[non_exhaustive]
    pub enum CredentialOneOf`
	if !strings.Contains(modelRs, wantCustomOneOf) {
		t.Errorf("model.rs expected CredentialOneOf without derived Debug in:\n%s", modelRs)
	}
	wantDerivedOneOf := `#[derive(Clone, Debug, PartialEq)]
    #[non_exhaustive]
    pub enum NormalChoice`
	if !strings.Contains(modelRs, wantDerivedOneOf) {
		t.Errorf("model.rs expected NormalChoice with derived Debug in:\n%s", modelRs)
	}
}

func TestGenerateRedactedDebugFieldsPerServiceFeatures(t *testing.T) {
	reqA := api.NewTestMessage("ReqA").WithPackage("test.v1")
	reqB := api.NewTestMessage("ReqB").WithPackage("test.v1")

	secretBranch := api.NewTestField("secret_key").WithType(api.TypezBytes).WithOneOf()
	secretChoice := api.NewTestOneOf("secret_choice").WithFields(secretBranch)
	orphan := api.NewTestMessage("Orphan").
		WithPackage("test.v1").
		WithOneOfs(secretChoice)

	methodA := api.NewTestMethod("GetA").
		WithInput(reqA).
		WithOutput(reqA).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())
	svcA := api.NewTestService("ServiceA").
		WithPackage("test.v1").
		WithMethods(methodA)

	methodB := api.NewTestMethod("GetB").
		WithInput(reqB).
		WithOutput(reqB).
		WithVerb("GET").
		WithPathTemplate(api.NewPathTemplateBuilder().Build())
	svcB := api.NewTestService("ServiceB").
		WithPackage("test.v1").
		WithMethods(methodB)

	model := api.NewTestAPI([]*api.Message{reqA, reqB, orphan}, nil, []*api.Service{svcA, svcB}).
		WithPackageName("test.v1")
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"package:wkt":           "source=google.protobuf,package=google-cloud-wkt",
			"per-service-features":  "true",
			"redacted-debug-fields": ".test.v1.Orphan.secret_key",
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	debugBytes, err := os.ReadFile(filepath.Join(outDir, "src", "model", "debug.rs"))
	if err != nil {
		t.Fatal(err)
	}
	debugRs := string(debugBytes)
	wantGatedOneOf := "#[cfg(all (feature = \"service-a\",feature = \"service-b\", ) )]\nimpl std::fmt::Debug for super::orphan::SecretChoice {"
	if !strings.Contains(debugRs, wantGatedOneOf) {
		t.Errorf("debug.rs missing feature-gated oneof Debug impl %q in:\n%s", wantGatedOneOf, debugRs)
	}

	modelBytes, err := os.ReadFile(filepath.Join(outDir, "src", "model.rs"))
	if err != nil {
		t.Fatal(err)
	}
	orphanEnumBlock := extractBlock(t, string(modelBytes), "pub mod orphan {", "pub enum SecretChoice")
	if strings.Contains(orphanEnumBlock, "#[cfg(") {
		t.Errorf("model.rs unexpectedly added redundant #[cfg(...)] before pub enum SecretChoice in:\n%s", orphanEnumBlock)
	}
	if !strings.Contains(orphanEnumBlock, "#[derive(Clone, PartialEq)]") {
		t.Errorf("model.rs expected SecretChoice without derived Debug in:\n%s", orphanEnumBlock)
	}
}

func TestGenerateRedactedDebugFieldsNotFound(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
	}{
		{
			name:  "unknown field in same package",
			field: ".test.v1.Request.nonexistent_field",
		},
		{
			name:  "package or version typo",
			field: ".test.v2.Request.name",
		},
		{
			name:  "external message field not generated in debug.rs",
			field: ".ext.v1.ExternalMsg.secret",
		},
		{
			name:  "service placeholder field not generated in debug.rs",
			field: ".test.v1.ServicePlaceholder.secret",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			extMsg := api.NewTestMessage("ExternalMsg").
				WithPackage("ext.v1").
				WithFields(api.NewTestField("secret").WithType(api.TypezBytes))
			req := api.NewTestMessage("Request").
				WithPackage("test.v1").
				WithFields(api.NewTestField("name").WithType(api.TypezString))
			placeholder := api.NewTestMessage("ServicePlaceholder").
				WithPackage("test.v1").
				WithServicePlaceholder().
				WithFields(api.NewTestField("secret").WithType(api.TypezBytes))
			model := api.NewTestAPI([]*api.Message{req, placeholder}, nil, nil).
				WithPackageName("test.v1")
			model.ExternalMessages = []*api.Message{extMsg}
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}

			cfg := &parser.ModelConfig{
				SpecificationFormat: libconfig.SpecProtobuf,
				Codec: map[string]string{
					"package:wkt":           "source=google.protobuf,package=google-cloud-wkt",
					"package:ext":           "source=ext.v1,package=google-cloud-ext-v1",
					"redacted-debug-fields": test.field,
				},
			}
			err := Generate(t.Context(), model, t.TempDir(), cfg)
			if !errors.Is(err, ErrRedactedDebugFieldNotFound) {
				t.Errorf("expected ErrRedactedDebugFieldNotFound, got %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), `"test.v1"`) {
				t.Errorf("expected error to name proto package %q, got %v", "test.v1", err)
			}
		})
	}
}

func TestGenerateRedactedDebugFieldsTemplates(t *testing.T) {
	newModel := func() *api.API {
		req := api.NewTestMessage("Request").
			WithPackage("test.v1").
			WithFields(api.NewTestField("name").WithType(api.TypezString))
		model := api.NewTestAPI([]*api.Message{req}, nil, nil).
			WithPackageName("test.v1")
		if err := api.CrossReference(model); err != nil {
			t.Fatal(err)
		}
		return model
	}

	t.Run("exempt template ignores unknown redacted_debug_fields", func(t *testing.T) {
		cfg := &parser.ModelConfig{
			SpecificationFormat: libconfig.SpecProtobuf,
			Codec: map[string]string{
				"package:wkt":           "source=google.protobuf,package=google-cloud-wkt",
				"template-override":     "templates/convert-prost",
				"redacted-debug-fields": ".test.v1.Request.nonexistent_field",
			},
		}
		if err := Generate(t.Context(), newModel(), t.TempDir(), cfg); err != nil {
			t.Fatalf("expected convert-prost to ignore redacted-debug-fields, got %v", err)
		}
	})

	t.Run("bigquery template-override rejects redacted_debug_fields", func(t *testing.T) {
		cfg := &parser.ModelConfig{
			SpecificationFormat: libconfig.SpecProtobuf,
			Codec: map[string]string{
				"package:wkt":           "source=google.protobuf,package=google-cloud-wkt",
				"template-override":     "templates/bigquery",
				"redacted-debug-fields": ".test.v1.Request.name",
			},
		}
		err := Generate(t.Context(), newModel(), t.TempDir(), cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "redacted_debug_fields is not supported for template") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("GenerateBigQueryBuilder rejects redacted_debug_fields", func(t *testing.T) {
		cfg := &parser.ModelConfig{
			SpecificationFormat: libconfig.SpecProtobuf,
			Codec: map[string]string{
				"package:wkt":           "source=google.protobuf,package=google-cloud-wkt",
				"redacted-debug-fields": ".test.v1.Request.name",
			},
		}
		err := GenerateBigQueryBuilder(t.Context(), t.TempDir(), newModel(), cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "redacted_debug_fields is not supported for template") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestGenerateTracingTarget(t *testing.T) {
	for _, test := range []struct {
		name     string
		codec    map[string]string
		want     string
		unwanted string
	}{
		{
			name: "default target omitted",
			codec: map[string]string{
				"package:wkt": "source=google.protobuf,package=google-cloud-wkt",
			},
			want:     `#[tracing::instrument(level = tracing::Level::DEBUG, ret)]`,
			unwanted: `target =`,
		},
		{
			name: "custom target override",
			codec: map[string]string{
				"package:wkt":    "source=google.protobuf,package=google-cloud-wkt",
				"tracing-target": "google_cloud_storage::tracing",
			},
			want:     `#[tracing::instrument(target = "google_cloud_storage::tracing", level = tracing::Level::DEBUG, ret)]`,
			unwanted: `#[tracing::instrument(level = tracing::Level::DEBUG, ret)]`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			msg := api.NewTestMessage("Request").WithPackage("test.v1")
			method := api.NewTestMethod("GetResource").
				WithInput(msg).
				WithOutput(msg).
				WithVerb("GET").
				WithPathTemplate(api.NewPathTemplateBuilder().Build())
			svc := api.NewTestService("TestService").
				WithPackage("test.v1").
				WithMethods(method)
			model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{svc}).
				WithPackageName("test.v1")
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}

			outDir := t.TempDir()
			cfg := &parser.ModelConfig{
				SpecificationFormat: libconfig.SpecProtobuf,
				Codec:               test.codec,
			}
			if err := Generate(t.Context(), model, outDir, cfg); err != nil {
				t.Fatal(err)
			}

			tracingBytes, err := os.ReadFile(filepath.Join(outDir, "src", "tracing.rs"))
			if err != nil {
				t.Fatal(err)
			}
			tracingRs := string(tracingBytes)
			if !strings.Contains(tracingRs, test.want) {
				t.Errorf("tracing.rs missing %q in:\n%s", test.want, tracingRs)
			}
			if strings.Contains(tracingRs, test.unwanted) {
				t.Errorf("tracing.rs unexpectedly contained %q in:\n%s", test.unwanted, tracingRs)
			}
		})
	}
}
