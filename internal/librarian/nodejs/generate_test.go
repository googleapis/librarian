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

package nodejs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

func TestIsMixedLibrary(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want bool
	}{
		{
			name: "mixed library case",
			lib: &config.Library{
				Output: "packages/typeless-sample-bot",
				APIs:   nil,
			},
			want: true,
		},
		{
			name: "standard gapic lib",
			lib: &config.Library{
				Output: "packages/gapic-lib",
				APIs:   []*config.API{{Path: "google/example/v1"}},
			},
			want: false,
		},
		{
			name: "no output set",
			lib: &config.Library{
				Output: "",
				APIs:   nil,
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsMixedLibrary(test.lib); got != test.want {
				t.Errorf("IsMixedLibrary() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDerivePackageName(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want string
	}{
		{
			name: "explicit package name",
			lib: &config.Library{
				Name: "google-cloud-accessapproval",
				Nodejs: &config.NodejsPackage{
					PackageName: "@google-cloud/access-approval",
				},
			},
			want: "@google-cloud/access-approval",
		},
		{
			name: "derived from library name",
			lib: &config.Library{
				Name: "google-cloud-batch",
			},
			want: "@google-cloud/batch",
		},
		{
			name: "derived with multi-segment suffix",
			lib: &config.Library{
				Name: "google-cloud-video-transcoder",
			},
			want: "@google-cloud/video-transcoder",
		},
		{
			name: "nil nodejs config",
			lib: &config.Library{
				Name: "google-cloud-speech",
			},
			want: "@google-cloud/speech",
		},
		{
			name: "empty package name in config",
			lib: &config.Library{
				Name:   "google-cloud-monitoring",
				Nodejs: &config.NodejsPackage{},
			},
			want: "@google-cloud/monitoring",
		},
		{
			name: "no second dash",
			lib: &config.Library{
				Name: "google",
			},
			want: "google",
		},
		{
			name: "only one dash",
			lib: &config.Library{
				Name: "google-cloud",
			},
			want: "google-cloud",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := derivePackageName(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name          string
		libName       string
		defaultOutput string
		want          string
	}{
		{
			name:          "standard",
			libName:       "google-cloud-batch",
			defaultOutput: "packages",
			want:          "packages/google-cloud-batch",
		},
		{
			name:          "empty default",
			libName:       "google-cloud-batch",
			defaultOutput: "",
			want:          "google-cloud-batch",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.libName, test.defaultOutput)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildGeneratorArgs(t *testing.T) {
	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		t.Skipf("skipping test: protoc not found in PATH")
	}

	for _, test := range []struct {
		name    string
		api     *config.API
		library *config.Library
		want    []string
	}{
		{
			name: "basic case",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secretmanager",
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--grpc-service-config", "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--service-yaml", "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
				"--rest-numeric-enums",
			},
		},
		{
			name: "with explicit package name",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-accessapproval",
				Nodejs: &config.NodejsPackage{
					PackageName: "@google-cloud/access-approval",
				},
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--grpc-service-config", "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--service-yaml", "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--package-name", "@google-cloud/access-approval",
				"--metadata",
				"--rest-numeric-enums",
			},
		},
		{
			name: "with bundle config and extra params",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Nodejs: &config.NodejsAPI{
					Mixins: "none",
				},
			},
			library: &config.Library{
				Name: "google-cloud-translate",
				Nodejs: &config.NodejsPackage{
					BundleConfig:          "google/cloud/translate/v3/translate_gapic.yaml",
					ExtraProtocParameters: []string{"auto-populate-field-oauth-scope"},
					HandwrittenLayer:      true,
					MainService:           "translate",
				},
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--grpc-service-config", "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--service-yaml", "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--package-name", "@google-cloud/translate",
				"--metadata",
				"--rest-numeric-enums",
				"--bundle-config", "google/cloud/translate/v3/translate_gapic.yaml",
				"--auto-populate-field-oauth-scope",
				"--handwritten-layer",
				"--main-service", "translate",
				"--mixins", "none",
			},
		},
		{
			name: "no grpc config",
			api:  &config.API{Path: "google/cloud/apigeeconnect/v1"},
			library: &config.Library{
				Name: "google-cloud-apigeeconnect",
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--service-yaml", "google/cloud/apigeeconnect/v1/apigeeconnect_1.yaml",
				"--package-name", "@google-cloud/apigeeconnect",
				"--metadata",
				"--rest-numeric-enums",
			},
		},
		{
			name: "no grpc config and no service config",
			api:  &config.API{Path: "google/cloud/fakefoo/v1"},
			library: &config.Library{
				Name: "google-cloud-fakefoo",
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--package-name", "@google-cloud/fakefoo",
				"--metadata",
				"--rest-numeric-enums",
			},
		},
		{
			name: "DIREGAPIC support",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:      "google/cloud/secretmanager/v1",
							DIREGAPIC: true,
						},
					},
				},
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--grpc-service-config", "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--service-yaml", "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
				"--rest-numeric-enums",
				"--diregapic",
			},
		},
		{
			name: "ESM support",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{
					ESM: true,
				},
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--grpc-service-config", "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--service-yaml", "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
				"--rest-numeric-enums",
				"--format=esm",
			},
		},
		{
			name: "API-level mixin override",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:   "google/cloud/secretmanager/v1",
							Mixins: "none",
						},
					},
				},
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--grpc-service-config", "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--service-yaml", "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
				"--rest-numeric-enums",
				"--mixins", "none",
			},
		},
		{
			name: "apis[].nodejs mixin override",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Nodejs: &config.NodejsAPI{
					Mixins: "none",
				},
			},
			library: &config.Library{
				Name:   "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{},
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=.",
				"-I", ".",
				"--output-dir", "staging",
				"--grpc-service-config", "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json",
				"--service-yaml", "google/cloud/secretmanager/v1/secretmanager_v1.yaml",
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
				"--rest-numeric-enums",
				"--mixins", "none",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			nodejsAPI := resolveNodejsAPI(test.library, test.api)
			got, err := buildGeneratorArgs(test.api, test.library, absGoogleapisDir, "staging", nodejsAPI)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: Node.js GAPIC code generation")
	}

	testhelper.RequireCommand(t, "gapic-generator-typescript")

	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	repoRoot := t.TempDir()
	outDir := filepath.Join(repoRoot, "packages", "google-cloud-secretmanager")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	err = generateAPI(
		t.Context(),
		&config.API{Path: "google/cloud/secretmanager/v1"},
		&config.Library{Name: "google-cloud-secretmanager", Output: outDir},
		absGoogleapisDir,
		repoRoot,
	)
	if err != nil {
		t.Fatal(err)
	}

	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", "google-cloud-secretmanager", "v1")
	if _, err := os.Stat(stagingDir); err != nil {
		t.Errorf("expected staging directory to exist: %v", err)
	}
}

func TestGenerateAPI_MultipleVersions(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: Node.js GAPIC code generation")
	}

	testhelper.RequireCommand(t, "gapic-generator-typescript")
	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-secretmanager",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
			{Path: "google/cloud/secretmanager/v1beta2"},
		},
	}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	library.Output = outDir

	for _, api := range library.APIs {
		t.Run(api.Path, func(t *testing.T) {
			if err := generateAPI(t.Context(), api, library, absGoogleapisDir, repoRoot); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, api := range library.APIs {
		version := filepath.Base(api.Path)
		stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name, version)
		if _, err := os.Stat(stagingDir); err != nil {
			t.Errorf("expected staging directory for %s to exist: %v", version, err)
		}
	}
}

func TestRunPostProcessor(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-secretmanager",
		APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
	}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	createStagingFixture(t, repoRoot, library.Name, []string{"v1", "v1beta1"})

	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	if err := runPostProcessor(t.Context(), cfg, library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	// Verify that the package staging directory is successfully cleaned up
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging", library.Name)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected package staging directory to be removed after post-processing")
	}
	// Verify that the top-level owl-bot-staging parent folder itself remains intact to support parallel executions
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); err != nil {
		t.Error("expected top-level owl-bot-staging directory to remain intact")
	}
}

func TestRunPostProcessor_RemovesOwlBotYaml(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-test",
		APIs: []*config.API{{Path: "google/cloud/test/v1"}},
	}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create staging structure with a .OwlBot.yaml file.
	stagingBase := filepath.Join(repoRoot, "owl-bot-staging", library.Name, "v1")
	srcDir := filepath.Join(stagingBase, "src", "v1")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte("export {};\n"), 0644); err != nil {
		t.Fatal(err)
	}
	protoDir := filepath.Join(stagingBase, "protos", "google", "cloud", "test", "v1")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "test.proto"), []byte("syntax = \"proto3\";\npackage google.cloud.test.v1;\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingBase, ".OwlBot.yaml"), []byte("deep-copy-regex:\n  - source: /owl-bot-staging\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Language: config.LanguageNodejs}
	if err := runPostProcessor(t.Context(), cfg, library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outDir, ".OwlBot.yaml")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected .OwlBot.yaml to be removed after post-processing")
	}
}

func TestRunPostProcessor_RemovesCloudCommonResourcesProto(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-test",
		APIs: []*config.API{{Path: "google/cloud/test/v1"}},
	}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create staging structure with a common_resources.proto file.
	stagingBase := filepath.Join(repoRoot, "owl-bot-staging", library.Name, "v1")
	srcDir := filepath.Join(stagingBase, "src", "v1")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte("export {};\n"), 0644); err != nil {
		t.Fatal(err)
	}
	protoDir := filepath.Join(stagingBase, "protos", "google", "cloud", "test", "v1")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "test.proto"), []byte("syntax = \"proto3\";\npackage google.cloud.test.v1;\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingBase, "protos", cloudCommonResourcesProto), []byte("syntax = \"proto3\";\npackage google.cloud;\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Language: config.LanguageNodejs}
	if err := runPostProcessor(t.Context(), cfg, library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "protos", cloudCommonResourcesProto)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected common_resources.proto to be removed after post-processing")
	}
}

func TestRunPostProcessor_CustomScripts(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")
	testhelper.RequireCommand(t, "node")
	testhelper.RequireCommand(t, "npx")

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-secretmanager",
		APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
		Keep: []string{"librarian.js", ".readme-partials.yaml"},
	}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	stagingBase := filepath.Join(repoRoot, "owl-bot-staging", library.Name, "v1")
	srcDir := filepath.Join(stagingBase, "src", "v1")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(srcDir, "index.ts"),
		[]byte("export {SecretManagerServiceClient} from './secret_manager_service_client';\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	protoDir := filepath.Join(stagingBase, "protos", "google", "cloud", "secretmanager", "v1")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatal(err)
	}
	protoContent := "syntax = \"proto3\";\npackage google.cloud.secretmanager.v1;\n"
	if err := os.WriteFile(filepath.Join(protoDir, "service.proto"), []byte(protoContent), 0644); err != nil {
		t.Fatal(err)
	}
	librarianJS := filepath.Join(outDir, "librarian.js")
	if err := os.WriteFile(librarianJS, []byte("const fs = require('fs');\nfs.writeFileSync('librarian-ran.txt', 'yes');\n"), 0644); err != nil {
		t.Fatal(err)
	}
	readmePartials := filepath.Join(outDir, ".readme-partials.yaml")
	if err := os.WriteFile(readmePartials, []byte("introduction: 'intro text'\nbody: 'body text'"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	if err := runPostProcessor(t.Context(), cfg, library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	// Verify package staging directory is cleaned up
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging", library.Name)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected package staging directory to be removed after post-processing")
	}
	// Verify parent folder remains intact
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); err != nil {
		t.Error("expected top-level owl-bot-staging directory to remain intact")
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "librarian-ran.txt")); err != nil {
		t.Errorf("expected librarian.js to run and create librarian-ran.txt in repoRoot: %v", err)
	}
	readmePath := filepath.Join(outDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "intro text") {
		t.Errorf("expected README.md to contain introduction, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "body text") {
		t.Errorf("expected README.md to contain body, got:\n%s", contentStr)
	}
}

func TestRunPostProcessor_PreservesFiles(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-test",
		APIs: []*config.API{{Path: "google/cloud/test/v1"}},
		Keep: []string{"README.md", ".readme-partials.yaml", "system-test/.eslintrc.yml"},
	}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	createStagingFixture(t, repoRoot, library.Name, []string{"v1"})

	readmeContent := "# Test README"
	if err := os.WriteFile(filepath.Join(outDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}
	partialsContent := "introduction: ''\nbody: ''"
	if err := os.WriteFile(filepath.Join(outDir, ".readme-partials.yaml"), []byte(partialsContent), 0644); err != nil {
		t.Fatal(err)
	}
	eslintContent := "extends: eslint:recommended"
	eslintDir := filepath.Join(outDir, "system-test")
	if err := os.MkdirAll(eslintDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(eslintDir, ".eslintrc.yml"), []byte(eslintContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	if err := runPostProcessor(t.Context(), cfg, library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != readmeContent {
		t.Errorf("README.md content = %q, want %q", string(got), readmeContent)
	}
	if _, err := os.Stat(filepath.Join(outDir, ".readme-partials.yaml")); err != nil {
		t.Errorf("expected .readme-partials.yaml to be preserved: %v", err)
	}
	gotEslint, err := os.ReadFile(filepath.Join(outDir, "system-test", ".eslintrc.yml"))
	if err != nil {
		t.Fatalf("expected system-test/.eslintrc.yml to be preserved: %v", err)
	}
	if string(gotEslint) != eslintContent {
		t.Errorf("system-test/.eslintrc.yml content = %q, want %q", string(gotEslint), eslintContent)
	}
}

func TestRestoreCopyrightYear(t *testing.T) {
	for _, test := range []struct {
		name  string
		dir   string
		year  string
		input string
		want  string
	}{
		{
			name:  "replaces year in src",
			dir:   "src",
			year:  "2020",
			input: "// Copyright 2026 Google LLC\n",
			want:  "// Copyright 2020 Google LLC\n",
		},
		{
			name:  "replaces year in test",
			dir:   "test",
			year:  "2019",
			input: "// Copyright 2026 Google LLC\n",
			want:  "// Copyright 2019 Google LLC\n",
		},
		{
			name:  "empty year is no-op",
			dir:   "src",
			year:  "",
			input: "// Copyright 2026 Google LLC\n",
			want:  "// Copyright 2026 Google LLC\n",
		},
		{
			name:  "no match is no-op",
			dir:   "src",
			year:  "2020",
			input: "// No copyright here\n",
			want:  "// No copyright here\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			dir := filepath.Join(outDir, test.dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(dir, "index.ts")
			if err := os.WriteFile(file, []byte(test.input), 0644); err != nil {
				t.Fatal(err)
			}
			if err := restoreCopyrightYear(outDir, test.year); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRestoreCopyrightYear_SkipsMissingDirs(t *testing.T) {
	outDir := t.TempDir()
	if err := restoreCopyrightYear(outDir, "2020"); err != nil {
		t.Fatal(err)
	}
}

func TestGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: Node.js code generation")
	}

	testhelper.RequireCommand(t, "gapic-generator-typescript")
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	repoRoot := t.TempDir()
	libraries := []*config.Library{
		{
			Name: "google-cloud-secretmanager",
			APIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
			},
		},
		{
			Name: "google-cloud-configdelivery",
			APIs: []*config.API{
				{Path: "google/cloud/configdelivery/v1"},
			},
		},
	}
	for _, library := range libraries {
		library.Output = filepath.Join(repoRoot, "packages", library.Name)
	}

	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	for _, library := range libraries {
		if err := Generate(t.Context(), cfg, library, &sources.Sources{Googleapis: absGoogleapisDir}); err != nil {
			t.Fatal(err)
		}
	}

	for _, library := range libraries {
		if _, err := os.Stat(library.Output); err != nil {
			t.Errorf("expected output directory for %q to exist: %v", library.Name, err)
		}
	}
}

func TestCopyMissingProtos(t *testing.T) {
	googleapisDir := t.TempDir()
	outDir := t.TempDir()

	srcProto := filepath.Join(googleapisDir, "google", "logging", "type", "log_severity.proto")
	if err := os.MkdirAll(filepath.Dir(srcProto), 0755); err != nil {
		t.Fatal(err)
	}
	srcContent := []byte("syntax = \"proto3\";\npackage google.logging.type;\n")
	if err := os.WriteFile(srcProto, srcContent, 0644); err != nil {
		t.Fatal(err)
	}

	listDir := filepath.Join(outDir, "src", "v1")
	if err := os.MkdirAll(listDir, 0755); err != nil {
		t.Fatal(err)
	}

	existingProto := filepath.Join(outDir, "protos", "google", "cloud", "foo", "v1", "existing.proto")
	if err := os.MkdirAll(filepath.Dir(existingProto), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingProto, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	entries := []string{
		// Already exists relative to listDir - should be skipped.
		"../../protos/google/cloud/foo/v1/existing.proto",
		// Missing proto with "protos/" prefix - should be copied.
		"../../protos/google/logging/type/log_severity.proto",
		// Entry without "protos/" prefix - should be skipped.
		"../../other/google/cloud/foo/v1/no_protos_prefix.proto",
	}
	listData, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	listPath := filepath.Join(listDir, "foo_proto_list.json")
	if err := os.WriteFile(listPath, listData, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyMissingProtos(googleapisDir, outDir); err != nil {
		t.Fatal(err)
	}

	copiedPath := filepath.Join(outDir, "protos", "google", "logging", "type", "log_severity.proto")
	got, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(srcContent), string(got)); diff != "" {
		t.Errorf("copied proto content mismatch (-want +got):\n%s", diff)
	}

	existingContent, err := os.ReadFile(existingProto)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff("existing", string(existingContent)); diff != "" {
		t.Errorf("existing proto should not be overwritten (-want +got):\n%s", diff)
	}
}

func TestCopySamplesFromStaging(t *testing.T) {
	stagingDir := t.TempDir()
	outDir := t.TempDir()

	for _, v := range []struct {
		version         string
		sampleContent   string
		metadataContent string
	}{
		{version: "v1", sampleContent: "console.log('v1');", metadataContent: `{"snippets":[]}`},
		{version: "v1beta1", metadataContent: `{"snippets":["beta"]}`},
	} {
		samplesDir := filepath.Join(stagingDir, v.version, "samples", "generated", v.version)
		if err := os.MkdirAll(samplesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if v.sampleContent != "" {
			if err := os.WriteFile(filepath.Join(samplesDir, "sample.js"), []byte(v.sampleContent), 0644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(samplesDir, "snippet_metadata_google.cloud.test."+v.version+".json"), []byte(v.metadataContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := copySamplesFromStaging(stagingDir, outDir); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "v1 sample file",
			path: filepath.Join(outDir, "samples", "generated", "v1", "sample.js"),
			want: "console.log('v1');",
		},
		{
			name: "v1 metadata",
			path: filepath.Join(outDir, "samples", "generated", "v1", "snippet_metadata_google.cloud.test.v1.json"),
			want: `{"snippets":[]}`,
		},
		{
			name: "v1beta1 metadata",
			path: filepath.Join(outDir, "samples", "generated", "v1beta1", "snippet_metadata_google.cloud.test.v1beta1.json"),
			want: `{"snippets":["beta"]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCopySamplesFromStaging_NonExistentDir(t *testing.T) {
	if err := copySamplesFromStaging(filepath.Join(t.TempDir(), "does-not-exist"), t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateAPI_NoProtos(t *testing.T) {
	googleapisDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create an API directory with no .proto files.
	apiPath := "google/cloud/emptyapi/v1"
	apiDir := filepath.Join(googleapisDir, apiPath)
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a non-proto file so the directory is not empty.
	if err := os.WriteFile(filepath.Join(apiDir, "BUILD.bazel"), []byte("# empty"), 0644); err != nil {
		t.Fatal(err)
	}

	library := &config.Library{
		Name:   "google-cloud-emptyapi",
		Output: filepath.Join(repoRoot, "packages", "google-cloud-emptyapi"),
	}
	if err := generateAPI(t.Context(), &config.API{Path: apiPath}, library, googleapisDir, repoRoot); err == nil {
		t.Fatal("expected error for API directory with no proto files")
	}
}

func createStagingFixture(t *testing.T, repoRoot, libName string, versions []string) {
	t.Helper()
	for _, v := range versions {
		stagingBase := filepath.Join(repoRoot, "owl-bot-staging", libName, v)
		srcDir := filepath.Join(stagingBase, "src", v)
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte("export {};\n"), 0644); err != nil {
			t.Fatal(err)
		}
		protoDir := filepath.Join(stagingBase, "protos", "google", "cloud", "test", v)
		if err := os.MkdirAll(protoDir, 0755); err != nil {
			t.Fatal(err)
		}
		protoContent := fmt.Sprintf("syntax = \"proto3\";\npackage google.cloud.test.%s;\n", v)
		if err := os.WriteFile(filepath.Join(protoDir, "service.proto"), []byte(protoContent), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWriteRepoMetadata(t *testing.T) {
	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	for _, test := range []struct {
		name    string
		library *config.Library
		want    func() *repometadata.RepoMetadata
	}{
		{
			name: "no overrides",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: func() *repometadata.RepoMetadata {
				w := sample.RepoMetadata()
				w.DistributionName = "@google-cloud/secretmanager"
				w.Language = cfg.Language
				w.Repo = cfg.Repo
				w.ClientDocumentation = "https://cloud.google.com/nodejs/docs/reference/secretmanager/latest"
				w.ProductDocumentation = "https://cloud.google.com/secret-manager/docs"
				return w
			},
		},
		{
			name: "client documentation override",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Nodejs: &config.NodejsPackage{
					ClientDocumentationOverride: "https://custom.docs.com/ref",
				},
			},
			want: func() *repometadata.RepoMetadata {
				w := sample.RepoMetadata()
				w.DistributionName = "@google-cloud/secretmanager"
				w.Language = cfg.Language
				w.Repo = cfg.Repo
				w.ClientDocumentation = "https://custom.docs.com/ref"
				w.ProductDocumentation = "https://cloud.google.com/secret-manager/docs"
				return w
			},
		},
		{
			name: "default version override",
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v1beta"},
				},
				Nodejs: &config.NodejsPackage{
					DefaultVersion: "v1beta",
				},
			},
			want: func() *repometadata.RepoMetadata {
				w := sample.RepoMetadata()
				w.DistributionName = "@google-cloud/secretmanager"
				w.Language = cfg.Language
				w.Repo = cfg.Repo
				w.ClientDocumentation = "https://cloud.google.com/nodejs/docs/reference/secretmanager/latest"
				w.ProductDocumentation = "https://cloud.google.com/secret-manager/docs"
				w.DefaultVersion = "v1beta"
				return w
			},
		},
		{
			name: "metadata name and name pretty overrides",
			library: &config.Library{
				Name: "google-cloud-dialogflow-cx",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Nodejs: &config.NodejsPackage{
					MetadataNameOverride: "dialogflow-cx",
					NamePrettyOverride:   "Dialogflow CX API",
				},
			},
			want: func() *repometadata.RepoMetadata {
				w := sample.RepoMetadata()
				w.DistributionName = "@google-cloud/dialogflow-cx"
				w.Language = cfg.Language
				w.Repo = cfg.Repo
				w.ClientDocumentation = "https://cloud.google.com/nodejs/docs/reference/dialogflow-cx/latest"
				w.ProductDocumentation = "https://cloud.google.com/secret-manager/docs"
				w.Name = "dialogflow-cx"
				w.NamePretty = "Dialogflow CX API"
				return w
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			if err := writeRepoMetadata(cfg, test.library, absGoogleapisDir, outDir); err != nil {
				t.Fatal(err)
			}
			got, err := repometadata.Read(outDir)
			if err != nil {
				t.Fatal(err)
			}
			want := test.want()
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteRepoMetadata_NoAPIs(t *testing.T) {
	cfg := &config.Config{Language: config.LanguageNodejs}
	library := &config.Library{Name: "google-cloud-test"}
	if err := writeRepoMetadata(cfg, library, "", t.TempDir()); err != nil {
		t.Errorf("expected nil error for library with no APIs, got: %v", err)
	}
}

func TestRunPostProcessor_CustomScripts_RootRelativePath(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")
	testhelper.RequireCommand(t, "node")
	testhelper.RequireCommand(t, "npx")
	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-secretmanager",
		APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
		Keep: []string{"librarian.js"},
	}

	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	// This script uses a path relative to the repository root.
	// This only works if the script is executed from repoRoot.
	relPath := filepath.Join("packages", library.Name, "output.txt")
	script := fmt.Sprintf("const fs = require('fs');\nfs.writeFileSync('%s', 'success');\n", relPath)

	librarianJS := filepath.Join(outDir, "librarian.js")
	if err := os.WriteFile(librarianJS, []byte(script), 0644); err != nil {
		t.Fatal(err)
	}
	createStagingFixture(t, repoRoot, library.Name, []string{"v1"})
	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}

	if err := runPostProcessor(t.Context(), cfg, library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	// The file should have been created at outDir/output.txt
	if _, err := os.Stat(filepath.Join(outDir, "output.txt")); err != nil {
		t.Errorf("expected librarian.js to create output.txt using root-relative path: %v", err)
	}
}

func TestResolveNodejsAPI(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		api     *config.API
		want    *config.NodejsAPI
	}{
		{
			name:    "not found, returns defaults",
			library: &config.Library{},
			api:     &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{cloudCommonResourcesProto},
			},
		},
		{
			name: "found in config, appends to defaults",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:             "google/cloud/secretmanager/v1",
							AdditionalProtos: []string{"other.proto"},
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{cloudCommonResourcesProto, "other.proto"},
			},
		},
		{
			name: "found in config, package and api level union",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					AdditionalProtos: []string{"pkg.proto"},
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:             "google/cloud/secretmanager/v1",
							AdditionalProtos: []string{"api.proto"},
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{cloudCommonResourcesProto, "pkg.proto", "api.proto"},
			},
		},
		{
			name: "deduplicates protos",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					AdditionalProtos: []string{cloudCommonResourcesProto, "other.proto"},
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:             "google/cloud/secretmanager/v1",
							AdditionalProtos: []string{"other.proto", "more.proto"},
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{cloudCommonResourcesProto, "other.proto", "more.proto"},
			},
		},
		{
			name: "DIREGAPIC support",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:      "google/cloud/secretmanager/v1",
							DIREGAPIC: true,
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{cloudCommonResourcesProto},
				DIREGAPIC:        true,
			},
		},
		{
			name: "omit common resources is true",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:                "google/api/cloudquotas/v1",
							OmitCommonResources: true,
						},
					},
				},
			},
			api: &config.API{Path: "google/api/cloudquotas/v1"},
			want: &config.NodejsAPI{
				Path:                "google/api/cloudquotas/v1",
				OmitCommonResources: true,
				AdditionalProtos:    nil,
			},
		},
		{
			name: "omit common resources is true, package-level additional protos preserved",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					AdditionalProtos: []string{"pkg.proto"},
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:                "google/cloud/secretmanager/v1",
							OmitCommonResources: true,
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:                "google/cloud/secretmanager/v1",
				OmitCommonResources: true,
				AdditionalProtos:    []string{"pkg.proto"},
			},
		},
		{
			name: "omit common resources is true, api-level additional protos preserved",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:                "google/cloud/secretmanager/v1",
							OmitCommonResources: true,
							AdditionalProtos:    []string{"api.proto"},
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:                "google/cloud/secretmanager/v1",
				OmitCommonResources: true,
				AdditionalProtos:    []string{"api.proto"},
			},
		},
		{
			name: "omit common resources is true, package-level and api-level protos combined",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					AdditionalProtos: []string{"pkg.proto"},
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:                "google/cloud/secretmanager/v1",
							OmitCommonResources: true,
							AdditionalProtos:    []string{"api.proto"},
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:                "google/cloud/secretmanager/v1",
				OmitCommonResources: true,
				AdditionalProtos:    []string{"pkg.proto", "api.proto"},
			},
		},
		{
			name: "omit common resources is false, all preserved",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					AdditionalProtos: []string{"pkg.proto"},
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:                "google/cloud/secretmanager/v1",
							OmitCommonResources: false,
							AdditionalProtos:    []string{"api.proto"},
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:                "google/cloud/secretmanager/v1",
				OmitCommonResources: false,
				AdditionalProtos:    []string{cloudCommonResourcesProto, "pkg.proto", "api.proto"},
			},
		},
		{
			name: "duplicated protos",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					AdditionalProtos: []string{"pkg.proto", "dup.proto"},
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:                "google/cloud/secretmanager/v1",
							OmitCommonResources: true,
							AdditionalProtos:    []string{"dup.proto", "api.proto"},
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:                "google/cloud/secretmanager/v1",
				OmitCommonResources: true,
				AdditionalProtos:    []string{"pkg.proto", "dup.proto", "api.proto"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := resolveNodejsAPI(test.library, test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInjectV1SmallExports(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "successfully injects",
			input: "import * as v1 from './v1';\nimport * as v1beta from './v1beta';\nexport {v1, v1beta};\nexport default {v1, v1beta};\n",
			want:  "import * as v1small from './v1small';\nimport * as v1 from './v1';\nimport * as v1beta from './v1beta';\nexport {v1small, v1, v1beta};\nexport default {v1small, v1, v1beta};\n",
		},
		{
			name:  "skips if already injected",
			input: "import * as v1small from './v1small';\nimport * as v1 from './v1';\nexport {v1small, v1};\n",
			want:  "import * as v1small from './v1small';\nimport * as v1 from './v1';\nexport {v1small, v1};\n",
		},
		{
			name:    "fails if v1 import missing",
			input:   "import * as v1beta from './v1beta';\nexport {v1, v1beta};\n",
			wantErr: true,
		},
		{
			name:    "fails if v1 export missing",
			input:   "import * as v1 from './v1';\nexport {v1beta};\n",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			srcDir := filepath.Join(outDir, "src")
			if err := os.MkdirAll(srcDir, 0755); err != nil {
				t.Fatal(err)
			}
			indexPath := filepath.Join(srcDir, "index.ts")
			if err := os.WriteFile(indexPath, []byte(test.input), 0644); err != nil {
				t.Fatal(err)
			}

			err := injectV1SmallExports(outDir)
			if (err != nil) != test.wantErr {
				t.Errorf("injectV1SmallExports() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if test.wantErr {
				return
			}

			got, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRemoveRedundantLinterFiles(t *testing.T) {
	for _, test := range []struct {
		name  string
		keep  []string
		files []string
		want  []string
	}{
		{
			name:  "removes all redundant linter files when keep is empty",
			keep:  []string{},
			files: []string{".eslintignore", ".eslintrc.json", ".prettierignore", ".prettierrc.js", ".prettierrc.cjs", "package.json", "README.md"},
			want:  []string{"README.md", "package.json"},
		},
		{
			name:  "preserves explicitly kept linter files",
			keep:  []string{".eslintignore", ".eslintrc.json"},
			files: []string{".eslintignore", ".eslintrc.json", ".prettierignore", ".prettierrc.js", "package.json", "README.md"},
			want:  []string{".eslintignore", ".eslintrc.json", "README.md", "package.json"},
		},
		{
			name:  "skips missing linter files without error",
			keep:  []string{},
			files: []string{"package.json", "README.md"},
			want:  []string{"README.md", "package.json"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			for _, f := range test.files {
				path := filepath.Join(outDir, f)
				if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			library := &config.Library{
				Keep: test.keep,
			}

			if err := removeRedundantLinterFiles(library, outDir); err != nil {
				t.Fatal(err)
			}

			var got []string
			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				got = append(got, entry.Name())
			}

			slices.Sort(got)
			slices.Sort(test.want)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveDefaultVersion(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want string
	}{
		{
			name: "default to first API path",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/test/v3"},
					{Path: "google/cloud/test/v4"},
				},
			},
			want: "v3",
		},
		{
			name: "no APIs or override",
			lib:  &config.Library{},
			want: "",
		},
		{
			name: "override",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/test/v3"},
					{Path: "google/cloud/test/v4"},
				},
				Nodejs: &config.NodejsPackage{
					DefaultVersion: "v4",
				},
			},
			want: "v4",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := resolveDefaultVersion(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMoveKeep(t *testing.T) {
	for _, test := range []struct {
		name        string
		setup       func(t *testing.T, srcDir string)
		filesToKeep []string
		wantFiles   []string
		unexpected  []string
	}{
		{
			name: "moves existing files successfully",
			setup: func(t *testing.T, srcDir string) {
				if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
					t.Fatal(err)
				}
				for _, file := range []string{"file1.txt", "subdir/file2.txt"} {
					if err := os.WriteFile(filepath.Join(srcDir, file), []byte("content"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			},
			filesToKeep: []string{"file1.txt", "subdir/file2.txt"},
			wantFiles:   []string{"file1.txt", "subdir/file2.txt"},
		},
		{
			name: "skips missing files without error",
			setup: func(t *testing.T, srcDir string) {
				if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			filesToKeep: []string{"file1.txt", "missing.txt"},
			wantFiles:   []string{"file1.txt"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			srcDir := t.TempDir()
			dstDir := t.TempDir()
			test.setup(t, srcDir)
			if err := moveKeep(test.filesToKeep, srcDir, dstDir); err != nil {
				t.Fatal(err)
			}
			for _, f := range test.wantFiles {
				path := filepath.Join(dstDir, f)
				if _, err := os.Stat(path); err != nil {
					t.Errorf("file %s does not exist in destination: %v", path, err)
				}
			}
		})
	}
}

func TestMoveKeep_Errors(t *testing.T) {
	for _, test := range []struct {
		name        string
		setup       func(t *testing.T, srcDir, dstDir string)
		filesToKeep []string
		wantErr     error
	}{
		{
			name: "mkdir failure when target parent is a regular file",
			setup: func(t *testing.T, srcDir, dstDir string) {
				if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file.txt"), []byte("content"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dstDir, "subdir"), []byte("not-a-dir"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			filesToKeep: []string{"subdir/file.txt"},
			wantErr:     syscall.ENOTDIR,
		},
		{
			name: "rename failure when target is an existing directory",
			setup: func(t *testing.T, srcDir, dstDir string) {
				if err := os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(dstDir, "file.txt"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			filesToKeep: []string{"file.txt"},
			wantErr:     os.ErrExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			srcDir := t.TempDir()
			dstDir := t.TempDir()
			test.setup(t, srcDir, dstDir)
			err := moveKeep(test.filesToKeep, srcDir, dstDir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("moveKeep() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
