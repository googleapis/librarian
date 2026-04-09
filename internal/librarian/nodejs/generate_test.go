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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/sample"
	"github.com/googleapis/librarian/internal/sources"
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
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/access-approval",
				"--metadata",
				"--rest-numeric-enums",
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
				"--rest-numeric-enums",
				"--bundle-config", filepath.Join(absGoogleapisDir, "google/cloud/translate/v3/translate_gapic.yaml"),
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
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/apigeeconnect/v1/apigeeconnect_1.yaml"),
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
				"--common-proto-path=" + absGoogleapisDir,
				"-I", absGoogleapisDir,
				"--output-dir", "staging",
				"--package-name", "@google-cloud/fakefoo",
				"--metadata",
				"--rest-numeric-enums",
			},
		},
		{
			name: "metadata in extra params is skipped",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name: "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{
					ExtraProtocParameters: []string{"metadata", "some-other-param"},
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
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
				"--rest-numeric-enums",
				"--some-other-param",
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

func TestRunPostProcessor_Owlbot(t *testing.T) {
	testhelper.RequireCommand(t, "python3")

	repoRoot := t.TempDir()
	library := &config.Library{Name: "google-cloud-test"}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	owlbotScript := filepath.Join(outDir, "owlbot.py")
	if err := os.WriteFile(owlbotScript, []byte("import pathlib\npathlib.Path('owlbot-ran.txt').write_text('yes')\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	if err := runPostProcessor(t.Context(), cfg, library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "owlbot-ran.txt")); err != nil {
		t.Errorf("expected owlbot.py to run and create owlbot-ran.txt: %v", err)
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
	library := &config.Library{Name: "google-cloud-secretmanager"}
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
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected owl-bot-staging to be removed after post-processing")
	}
}

func TestRunPostProcessor_RemovesOwlBotYaml(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{Name: "google-cloud-test"}
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

func TestRunPostProcessor_CustomScripts(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")
	testhelper.RequireCommand(t, "node")
	testhelper.RequireCommand(t, "npx")

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-secretmanager",
		Keep: []string{"librarian.js", ".readme-partials.yaml", "README.md"},
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

	readmePath := filepath.Join(outDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("Some Title\n[//]: # \"partials.introduction\"\n[//]: # \"partials.body\"\nFooter"), 0644); err != nil {
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
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected owl-bot-staging to be removed after post-processing")
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "librarian-ran.txt")); err != nil {
		t.Errorf("expected librarian.js to run and create librarian-ran.txt in repoRoot: %v", err)
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

func TestRunPostProcessor_PreservesFiles(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{
		Name: "google-cloud-test",
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

func TestUpdateSnippetMetadataVersion(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		input   string
		want    string
	}{
		{
			name:    "updates version",
			version: "6.1.0",
			input: `{
  "clientLibrary": {
    "name": "nodejs-secretmanager",
    "version": "0.1.0",
    "language": "TYPESCRIPT",
    "apis": [{"id": "google.cloud.secretmanager.v1"}]
  },
  "snippets": [{"name": "CreateSecret"}]
}
`,
			want: `{
  "clientLibrary": {
    "name": "nodejs-secretmanager",
    "version": "6.1.0",
    "language": "TYPESCRIPT",
    "apis": [
      {
        "id": "google.cloud.secretmanager.v1"
      }
    ]
  },
  "snippets": [
    {
      "name": "CreateSecret"
    }
  ]
}
`,
		},
		{
			name:    "empty version is no-op",
			version: "",
			input: `{
  "clientLibrary": {
    "name": "nodejs-secretmanager",
    "version": "0.1.0",
    "language": "TYPESCRIPT",
    "apis": []
  },
  "snippets": []
}
`,
			want: `{
  "clientLibrary": {
    "name": "nodejs-secretmanager",
    "version": "0.1.0",
    "language": "TYPESCRIPT",
    "apis": []
  },
  "snippets": []
}
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			metadataDir := filepath.Join(outDir, "samples", "generated", "v1")
			if err := os.MkdirAll(metadataDir, 0755); err != nil {
				t.Fatal(err)
			}
			metadataFile := filepath.Join(metadataDir, "snippet_metadata.google.cloud.secretmanager.v1.json")
			if err := os.WriteFile(metadataFile, []byte(test.input), 0644); err != nil {
				t.Fatal(err)
			}
			if err := updateSnippetMetadataVersion(outDir, test.version); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(metadataFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateSnippetMetadataVersion_NoFiles(t *testing.T) {
	outDir := t.TempDir()
	if err := updateSnippetMetadataVersion(outDir, "1.0.0"); err != nil {
		t.Fatal(err)
	}
}

func TestWriteRepoMetadata(t *testing.T) {
	absGoogleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	cfg := &config.Config{
		Language: config.LanguageNodejs,
		Repo:     "googleapis/google-cloud-node",
	}
	library := &config.Library{
		Name: "google-cloud-secretmanager",
		APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
	}
	if err := writeRepoMetadata(cfg, library, absGoogleapisDir, outDir); err != nil {
		t.Fatal(err)
	}
	got, err := repometadata.Read(outDir)
	if err != nil {
		t.Fatal(err)
	}
	want := sample.RepoMetadata()
	want.DistributionName = "@google-cloud/secretmanager"
	want.Language = cfg.Language
	want.Repo = cfg.Repo
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
				AdditionalProtos: []string{commonProtos},
			},
		},
		{
			name: "found in config",
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
				AdditionalProtos: []string{"other.proto"},
			},
		},
		{
			name: "found in config, empty additional protos defaults to commonProtos",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path: "google/cloud/secretmanager/v1",
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:             "google/cloud/secretmanager/v1",
				AdditionalProtos: []string{commonProtos},
			},
		},
		{
			name: "found in config, explicitly skip defaults",
			library: &config.Library{
				Nodejs: &config.NodejsPackage{
					NodejsAPIs: []*config.NodejsAPI{
						{
							Path:                        "google/cloud/secretmanager/v1",
							SkipDefaultAdditionalProtos: true,
						},
					},
				},
			},
			api: &config.API{Path: "google/cloud/secretmanager/v1"},
			want: &config.NodejsAPI{
				Path:                        "google/cloud/secretmanager/v1",
				SkipDefaultAdditionalProtos: true,
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
