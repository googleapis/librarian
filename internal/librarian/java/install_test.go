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

package java

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestInstall(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	localPkgDir := filepath.Join(tmpDir, "sdk-platform-java", "hermetic_build", "library_generation")
	if err := os.MkdirAll(localPkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	localMvnDir := filepath.Join(tmpDir, "sdk-platform-java", "gapic-generator-java")
	mockPOM := `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <parent>
    <groupId>com.google.api.generator</groupId>
    <version>2.28.0-SNAPSHOT</version>
  </parent>
  <artifactId>gapic-generator-java</artifactId>
</project>`
	writeTestFile(t, filepath.Join(localMvnDir, "pom.xml"), mockPOM, 0644)
	writeTestFile(t, filepath.Join(localMvnDir, "target", "gapic-generator-java-2.28.0-SNAPSHOT.jar"), "local gapic jar content", 0644)
	m2Repo := filepath.Join(tempHome, ".m2", "repository")
	writeTestFile(t, filepath.Join(m2Repo, "com/google/googlejavaformat/google-java-format/1.25.2/google-java-format-1.25.2-all-deps.jar"), "gjf jar content", 0644)
	writeTestFile(t, filepath.Join(m2Repo, "io/grpc/protoc-gen-grpc-java/1.81.0/protoc-gen-grpc-java-1.81.0-linux-x86_64.exe"), "grpc exe content", 0755)
	stubDir := filepath.Join(tmpDir, "bin")
	pipLog := filepath.Join(tmpDir, "pip_invocations.log")
	mvnLog := filepath.Join(tmpDir, "mvn_invocations.log")
	writeStub(t, stubDir, "pip", pipLog)
	writeStub(t, stubDir, "mvn", mvnLog)
	writeStub(t, stubDir, "java", "")
	t.Setenv("PATH", stubDir)
	tools := &config.Tools{
		Maven: []*config.MavenTool{
			{
				Name:       "google-java-format",
				GroupID:    "com.google.googlejavaformat",
				ArtifactID: "google-java-format",
				Version:    "1.25.2",
				Classifier: "all-deps",
				Packaging:  "jar",
			},
			{
				Name:       "protoc-gen-java_grpc",
				GroupID:    "io.grpc",
				ArtifactID: "protoc-gen-grpc-java",
				Version:    "1.81.0",
				Classifier: "linux-x86_64",
				Packaging:  "exe",
			},
			{
				Name:      "protoc-gen-java_gapic",
				LocalPath: "sdk-platform-java/gapic-generator-java",
				MainClass: "com.google.api.generator.Main",
				Packaging: "jar",
			},
		},
		Pip: []*config.PipTool{
			{
				Name:    "PyYAML",
				Version: "6.0.2",
			},
			{
				Name:    "jinja2",
				Version: "3.1.6",
			},
			{
				Name:      "synthtool",
				LocalPath: "sdk-platform-java/hermetic_build/library_generation",
			},
		},
	}
	installDir := filepath.Join(tmpDir, "java_tools", "bin")
	t.Setenv("LIBRARIAN_INSTALL_DIR", installDir)
	if err := Install(t.Context(), tools); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(pipLog)
	if err != nil {
		t.Fatal(err)
	}
	wantPip := "pip install PyYAML==6.0.2 jinja2==3.1.6 " + localPkgDir
	if diff := cmp.Diff(wantPip, strings.TrimSpace(string(data))); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	data, err = os.ReadFile(mvnLog)
	if err != nil {
		t.Fatal(err)
	}
	wantMvn := "mvn dependency:get -Dartifact=com.google.googlejavaformat:google-java-format:1.25.2:jar:all-deps\n" +
		"mvn dependency:get -Dartifact=io.grpc:protoc-gen-grpc-java:1.81.0:exe:linux-x86_64\n" +
		"mvn package -B -ntp -T 1.5C -DskipTests -Dcheckstyle.skip -Dclirr.skip -Denforcer.skip -Dfmt.skip -pl sdk-platform-java/gapic-generator-java --also-make"
	if diff := cmp.Diff(wantMvn, strings.TrimSpace(string(data))); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	libDir := filepath.Join(tmpDir, "java_tools", "lib")
	t.Run("google-java-format", func(t *testing.T) {
		verifyInstalledTool(t, libDir, installDir, "google-java-format-1.25.2-all-deps.jar", "gjf jar content", "google-java-format", "#!/bin/sh\nexec java -jar %q \"$@\"\n")
	})
	t.Run("protoc-gen-java_grpc", func(t *testing.T) {
		verifyInstalledTool(t, libDir, installDir, "protoc-gen-grpc-java-1.81.0-linux-x86_64.exe", "grpc exe content", "protoc-gen-java_grpc", "#!/bin/sh\nexec %q \"$@\"\n")
	})
	t.Run("protoc-gen-java_gapic", func(t *testing.T) {
		verifyInstalledTool(t, libDir, installDir, "gapic-generator-java-2.28.0-SNAPSHOT.jar", "local gapic jar content", "protoc-gen-java_gapic", "#!/bin/sh\nexec java -cp %q \"com.google.api.generator.Main\" \"$@\"\n")
	})
}

func writeTestFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
}

func writeStub(t *testing.T, dir, name, logFile string) {
	t.Helper()
	content := "#!/bin/sh\nexit 0\n"
	if logFile != "" {
		content = fmt.Sprintf("#!/bin/sh\necho %q \"$@\" >> %q\n", name, logFile)
	}
	writeTestFile(t, filepath.Join(dir, name), content, 0755)
}

func verifyInstalledTool(t *testing.T, libDir, installDir, filename, wantContent, wrapperName, wantFormat string) {
	t.Helper()
	copiedPath := filepath.Join(libDir, filename)
	data, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantContent, string(data)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	wrapperPath := filepath.Join(installDir, wrapperName)
	wrapper, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	wantWrapper := fmt.Sprintf(wantFormat, copiedPath)
	if diff := cmp.Diff(wantWrapper, string(wrapper)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
