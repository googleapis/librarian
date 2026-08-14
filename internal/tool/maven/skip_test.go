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

package maven

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

// mavenSkipHarness sets up a stub mvn that logs its invocations and a fake local
// source tree + prebuilt m2 artifact, returning the tools, dirs, and log path.
type mavenSkipHarness struct {
	tools      []*config.MavenTool
	binDir     string
	libDir     string
	mvnLogPath string
	localDir   string
}

func setupSkipHarness(t *testing.T) mavenSkipHarness {
	t.Helper()
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	localDir := filepath.Join(tmpDir, "sdk-platform-java", "gapic-generator-java")
	if err := os.MkdirAll(filepath.Join(localDir, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	pom := `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <parent><groupId>com.google.api.generator</groupId><version>2.28.0-SNAPSHOT</version></parent>
  <artifactId>gapic-generator-java</artifactId>
</project>`
	if err := os.WriteFile(filepath.Join(localDir, "pom.xml"), []byte(pom), 0o644); err != nil {
		t.Fatal(err)
	}
	// The stub mvn does not build, so pre-create the expected target artifact.
	if err := os.WriteFile(filepath.Join(localDir, "target", "gapic-generator-java-2.28.0-SNAPSHOT.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Prebuilt external artifact in the fake ~/.m2.
	gjfDir := filepath.Join(tempHome, ".m2", "repository", "com", "google", "googlejavaformat", "google-java-format", "1.25.2")
	if err := os.MkdirAll(gjfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gjfDir, "google-java-format-1.25.2-all-deps.jar"), []byte("gjf"), 0o644); err != nil {
		t.Fatal(err)
	}

	stubDir := filepath.Join(tmpDir, "stub")
	if err := os.MkdirAll(stubDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mvnLogPath := filepath.Join(tmpDir, "mvn.log")
	mvn := "#!/bin/sh\necho mvn \"$@\" >> " + shellQuote(mvnLogPath) + "\n"
	if err := os.WriteFile(filepath.Join(stubDir, "mvn"), []byte(mvn), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stubDir, "java"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)

	binDir := filepath.Join(tmpDir, "java_tools", "bin")
	libDir := filepath.Join(tmpDir, "java_tools", "lib")
	for _, d := range []string{binDir, libDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return mavenSkipHarness{
		tools: []*config.MavenTool{
			{Name: "google-java-format", GroupID: "com.google.googlejavaformat", ArtifactID: "google-java-format", Version: "1.25.2", Classifier: "all-deps", Packaging: "jar"},
			{Name: "protoc-gen-java_gapic", LocalPath: "sdk-platform-java/gapic-generator-java", MainClass: "com.google.api.generator.Main", Packaging: "jar"},
		},
		binDir:     binDir,
		libDir:     libDir,
		mvnLogPath: mvnLogPath,
		localDir:   localDir,
	}
}

func shellQuote(s string) string { return "\"" + s + "\"" }

func mvnInvocations(t *testing.T, logPath string) int {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(strings.TrimSpace(string(data)), "\n") + 1
}

func TestInstallSkipsWhenUpToDate(t *testing.T) {
	h := setupSkipHarness(t)

	if err := Install(t.Context(), h.tools, h.binDir, h.libDir); err != nil {
		t.Fatalf("first install: %v", err)
	}
	first := mvnInvocations(t, h.mvnLogPath)
	if first == 0 {
		t.Fatal("expected the first install to invoke mvn")
	}

	// Second install with unchanged source/coordinates must not re-invoke mvn.
	if err := Install(t.Context(), h.tools, h.binDir, h.libDir); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if second := mvnInvocations(t, h.mvnLogPath); second != first {
		t.Errorf("second install re-ran mvn: invocations went %d -> %d, want unchanged", first, second)
	}
}

func TestInstallRebuildsWhenSourceChanges(t *testing.T) {
	h := setupSkipHarness(t)

	if err := Install(t.Context(), h.tools, h.binDir, h.libDir); err != nil {
		t.Fatalf("first install: %v", err)
	}
	first := mvnInvocations(t, h.mvnLogPath)

	// Change the local source: the fingerprint must invalidate and rebuild.
	if err := os.WriteFile(filepath.Join(h.localDir, "pom.xml"), []byte("<project><artifactId>gapic-generator-java</artifactId><version>2.28.0-SNAPSHOT</version></project>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Install(t.Context(), h.tools, h.binDir, h.libDir); err != nil {
		t.Fatalf("second install: %v", err)
	}
	second := mvnInvocations(t, h.mvnLogPath)
	if second <= first {
		t.Errorf("changed source did not trigger a rebuild: invocations %d -> %d", first, second)
	}
}
