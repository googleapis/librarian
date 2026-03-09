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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
				"-I", absGoogleapisDir,
				"--output_dir", "staging",
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
				"-I", absGoogleapisDir,
				"--output_dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/access-approval",
				"--metadata",
			},
		},
		{
			name: "with transport grpc",
			api:  &config.API{Path: "google/cloud/secretmanager/v1"},
			library: &config.Library{
				Name:      "google-cloud-secretmanager",
				Transport: "grpc",
			},
			want: []string{
				"gapic-generator-typescript",
				"-I", absGoogleapisDir,
				"--output_dir", "staging",
				"--grpc-service-config", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"--service-yaml", filepath.Join(absGoogleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"--package-name", "@google-cloud/secretmanager",
				"--metadata",
				"--transport", "grpc",
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
				"-I", absGoogleapisDir,
				"--output_dir", "staging",
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
				"-I", absGoogleapisDir,
				"--output_dir", "staging",
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
	testhelper.RequireCommand(t, "python3")
	requireSynthtool(t)

	repoRoot := t.TempDir()
	outDir := filepath.Join(repoRoot, "packages", "google-cloud-secretmanager")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(outDir, ".repo-metadata.json"), []byte(`{"default_version":"v1"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create staging structure that synthtool expects: owl-bot-staging/<lib>/v1/src/v1/index.ts
	stagingVersionDir := filepath.Join(repoRoot, "owl-bot-staging", "google-cloud-secretmanager", "v1", "src", "v1")
	if err := os.MkdirAll(stagingVersionDir, 0755); err != nil {
		t.Fatal(err)
	}
	indexContent := `export {SecretManagerServiceClient} from './secret_manager_service_client';`
	if err := os.WriteFile(filepath.Join(stagingVersionDir, "index.ts"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runPostProcessor(t.Context(), repoRoot, outDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !os.IsNotExist(err) {
		t.Error("expected owl-bot-staging to be removed after post-processing")
	}
}

func TestGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("slow test: Node.js code generation")
	}

	testhelper.RequireCommand(t, "gapic-generator-typescript")
	testhelper.RequireCommand(t, "python3")
	requireSynthtool(t)

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

func requireSynthtool(t *testing.T) {
	t.Helper()
	cmd := exec.Command("python3", "-c", fmt.Sprintf("import %s", "synthtool"))
	if err := cmd.Run(); err != nil {
		t.Skipf("skipping test because Python module synthtool is not installed")
	}
}
