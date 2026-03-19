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
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestRunPostProcessor_Owlbot(t *testing.T) {
	testhelper.RequireCommand(t, "python3")

	repoRoot := t.TempDir()
	library := &config.Library{Name: "google-cloud-secretmanager"}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create owlbot.py that creates a dummy file.
	owlbotContent := `
import os
with open("owlbot-ran.txt", "w") as f:
    f.write("Owlbot ran successfully\n")
`
	if err := os.WriteFile(filepath.Join(outDir, "owlbot.py"), []byte(owlbotContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runPostProcessor(t.Context(), library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "owlbot-ran.txt")); err != nil {
		t.Errorf("expected owlbot-ran.txt to exist: %v", err)
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

	// Create staging structure matching gapic-generator-typescript output for multiple versions.
	for _, v := range []string{"v1", "v1beta1"} {
		stagingBase := filepath.Join(repoRoot, "owl-bot-staging", library.Name, v)
		srcDir := filepath.Join(stagingBase, "src", v)
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
		protoDir := filepath.Join(stagingBase, "protos", "google", "cloud", "secretmanager", v)
		if err := os.MkdirAll(protoDir, 0755); err != nil {
			t.Fatal(err)
		}
		protoContent := fmt.Sprintf("syntax = \"proto3\";\npackage google.cloud.secretmanager.%s;\n", v)
		if err := os.WriteFile(filepath.Join(protoDir, "service.proto"), []byte(protoContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a dummy .OwlBot.yaml in staging that should be copied to outDir
	// by combine-library and then deleted by runPostProcessor.
	if err := os.WriteFile(filepath.Join(repoRoot, "owl-bot-staging", library.Name, "v1", ".OwlBot.yaml"), []byte("api-name: secretmanager\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runPostProcessor(t.Context(), library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "owl-bot-staging")); !os.IsNotExist(err) {
		t.Error("expected owl-bot-staging to be removed after post-processing")
	}
	if _, err := os.Stat(filepath.Join(outDir, ".OwlBot.yaml")); !os.IsNotExist(err) {
		t.Error("expected .OwlBot.yaml to be removed after post-processing")
	}
}

func TestRunPostProcessor_CustomScripts(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")
	testhelper.RequireCommand(t, "node")

	repoRoot := t.TempDir()
	library := &config.Library{Name: "google-cloud-secretmanager"}
	outDir := filepath.Join(repoRoot, "packages", library.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create staging structure.
	stagingBase := filepath.Join(repoRoot, "owl-bot-staging", library.Name, "v1")
	srcDir := filepath.Join(stagingBase, "src", "v1")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte("export {};\n"), 0644); err != nil {
		t.Fatal(err)
	}
	protoDir := filepath.Join(stagingBase, "protos", "google", "cloud", "secretmanager", "v1")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "service.proto"), []byte("syntax = \"proto3\";\npackage google.cloud.secretmanager.v1;\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create librarian.js script.
	librarianJS := `
const fs = require('fs');
fs.writeFileSync('librarian-ran.txt', 'librarian.js ran successfully\n');
`
	if err := os.WriteFile(filepath.Join(outDir, "librarian.js"), []byte(librarianJS), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runPostProcessor(t.Context(), library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "librarian-ran.txt")); err != nil {
		t.Errorf("expected librarian-ran.txt to exist: %v", err)
	}
}

func TestRunPostProcessor_PreservesFiles(t *testing.T) {
	testhelper.RequireCommand(t, "gapic-node-processing")
	testhelper.RequireCommand(t, "compileProtos")

	repoRoot := t.TempDir()
	library := &config.Library{Name: "google-cloud-test"}
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

	// Create files that should be preserved across combine-library.
	readmeContent := "# Test README"
	if err := os.WriteFile(filepath.Join(outDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}
	partialsContent := "introduction: ''\nbody: ''"
	if err := os.WriteFile(filepath.Join(outDir, ".readme-partials.yaml"), []byte(partialsContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runPostProcessor(t.Context(), library, "", repoRoot, outDir); err != nil {
		t.Fatal(err)
	}

	// Verify preserved files still exist.
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
}
