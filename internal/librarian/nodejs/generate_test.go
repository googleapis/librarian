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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

const googleapisDir = "../../testdata/googleapis"

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
			got := DerivePackageName(test.lib)
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
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
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
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/access-approval",
				"--metadata",
			},
		},
		{
			name: "default transport not passed",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secretmanager",
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
			},
		},
		{
			name: "with bundle config and extra params",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-translate",
				Nodejs: &config.NodejsPackage{
					BundleConfig:          "google/cloud/translate/v3/translate_gapic.yaml",
					ExtraProtocParameters: []string{"auto-populate-field-oauth-scope"},
					HandwrittenLayer:      true,
					MainService:           "translate",
					Mixins:                "none",
				},
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/translate",
				"--metadata",
				"--bundle-config", filepath.Join(absGoogleapisDir, "google/cloud/translate/v3/translate_gapic.yaml"),
				"--auto-populate-field-oauth-scope",
				"--handwritten-layer",
				"--main-service", "translate",
				"--mixins", "none",
			},
		},
		{
			name: "grpc+rest transport is default and not passed",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name:      "google-cloud-secretmanager",
				Transport: "grpc+rest",
			},
			want: []string{
				"gapic-generator-typescript",
				"--protoc=" + protocPath,
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildGeneratorArgs(test.api, test.library, absGoogleapisDir, "staging")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateLibrary_NoAPIs(t *testing.T) {
	repoRoot := t.TempDir()
	library := &config.Library{
		Name:   "no-apis",
		Output: filepath.Join(repoRoot, "packages", "will-not-be-created"),
	}
	if err := generateLibrary(t.Context(), library, googleapisDir); err != nil {
		t.Fatal(err)
	}
	_, gotErr := os.Stat(library.Output)
	if !os.IsNotExist(gotErr) {
		t.Errorf("expected output directory to not exist, got err: %v", gotErr)
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

	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", "google-cloud-secretmanager")
	if _, err := os.Stat(stagingDir); err != nil {
		t.Errorf("expected staging directory to exist: %v", err)
	}
}

func TestRunPostProcessor(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{Name: "google-cloud-secretmanager"}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create staging structure matching gapic-generator-typescript output.
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

	if err := runPostProcessor(t.Context(), library, repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !os.IsNotExist(err) {
		t.Error("expected owl-bot-staging to be removed after post-processing")
	}
}

func TestRunPostProcessor_CustomScripts(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")
	testhelper.RequireCommand(t, "node")
	testhelper.RequireCommand(t, "npx")

	repoRoot := t.TempDir()
	library := &config.Library{Name: "google-cloud-secretmanager"}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create staging structure matching gapic-generator-typescript output.
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

	readmePath := filepath.Join(outDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("Some Title\n[//]: # \"partials.introduction\"\n[//]: # \"partials.body\"\nFooter"), 0644); err != nil {
		t.Fatal(err)
	}

	readmePartials := filepath.Join(outDir, ".readme-partials.yaml")
	if err := os.WriteFile(readmePartials, []byte("introduction: 'intro text'\nbody: 'body text'"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runPostProcessor(t.Context(), library, repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !os.IsNotExist(err) {
		t.Error("expected owl-bot-staging to be removed after post-processing")
	}

	if _, err := os.Stat(filepath.Join(outDir, "librarian-ran.txt")); err != nil {
		t.Errorf("expected librarian.js to run and create librarian-ran.txt: %v", err)
	}

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

	if err := Generate(t.Context(), libraries, absGoogleapisDir); err != nil {
		t.Fatal(err)
	}

	for _, library := range libraries {
		if _, err := os.Stat(library.Output); err != nil {
			t.Errorf("expected output directory for %q to exist: %v", library.Name, err)
		}
	}
}
