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
)

func TestInstall_Success(t *testing.T) {
	// We must run from the root of a google-cloud-java clone.
	// We will simulate this by changing directory to a temp dir
	// and creating the required pom.xml.
	t.Chdir(t.TempDir())

	// Create fake pom.xml for prerequisite check
	pomDir := "sdk-platform-java/gapic-generator-java"
	if err := os.MkdirAll(pomDir, 0755); err != nil {
		t.Fatal(err)
	}
	pomPath := filepath.Join(pomDir, "pom.xml")
	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.google.api</groupId>
  <artifactId>gapic-generator-java</artifactId>
  <version>2.71.0</version>
</project>
`
	if err := os.WriteFile(pomPath, []byte(pomContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Setup temp install dir
	installDir := t.TempDir()
	t.Setenv("LIBRARIAN_INSTALL_DIR", installDir)

	// Setup temp HOME to isolate .m2/repository
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	m2Repo := filepath.Join(tempHome, ".m2", "repository")

	// Setup temp bin for stubs
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	testLogFile := filepath.Join(t.TempDir(), "invocations.log")

	// 1. Stub mvn
	mvnStub := fmt.Sprintf(`#!/bin/bash
echo "mvn $@" >> %q

# Handle dependency:get
if [[ "$*" == *"dependency:get"* ]]; then
    if [[ "$*" == *"com.google.googlejavaformat"* ]]; then
        mkdir -p %q/com/google/googlejavaformat/google-java-format/1.25.2
        touch %q/com/google/googlejavaformat/google-java-format/1.25.2/google-java-format-1.25.2-all-deps.jar
    fi
    if [[ "$*" == *"io.grpc"* ]]; then
        mkdir -p %q/io/grpc/protoc-gen-grpc-java/1.76.3
        touch %q/io/grpc/protoc-gen-grpc-java/1.76.3/protoc-gen-grpc-java-1.76.3-linux-x86_64.exe
    fi
fi

# Handle package
if [[ "$*" == *"package"* ]]; then
    mkdir -p sdk-platform-java/gapic-generator-java/target
    touch sdk-platform-java/gapic-generator-java/target/gapic-generator-java-2.71.0.jar
fi
`, testLogFile, m2Repo, m2Repo, m2Repo, m2Repo)

	if err := os.WriteFile(filepath.Join(binDir, "mvn"), []byte(mvnStub), 0755); err != nil {
		t.Fatal(err)
	}

	// 2. Stub pip
	pipStub := fmt.Sprintf(`#!/bin/bash
echo "pip $@" >> %q

# Mock local synthtool install check (simulate success, we don't actually need to create files for pip)
`, testLogFile)
	// But wait, pip install local_path will check if the path exists!
	// The path is sdk-platform-java/hermetic_build/library_generation.
	// We must create this directory so pip check doesn't fail (if install.go checks it, and it does: os.Stat(absPath))
	localPipPath := "sdk-platform-java/hermetic_build/library_generation"
	if err := os.MkdirAll(localPipPath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(binDir, "pip"), []byte(pipStub), 0755); err != nil {
		t.Fatal(err)
	}

	// 3. Stub java
	javaStub := `#!/bin/bash
# no-op
`
	if err := os.WriteFile(filepath.Join(binDir, "java"), []byte(javaStub), 0755); err != nil {
		t.Fatal(err)
	}

	// Run Install
	if err := Install(t.Context()); err != nil {
		t.Fatal(err)
	}

	// Verify log file content (invocations)
	logData, err := os.ReadFile(testLogFile)
	if err != nil {
		t.Fatal(err)
	}
	logContent := string(logData)

	// Verify mvn package was called
	expectedMvnBuild := "mvn package -B -ntp -T 1.5C -DskipTests -Dcheckstyle.skip -Dclirr.skip -Denforcer.skip -Dfmt.skip -pl sdk-platform-java/gapic-generator-java --also-make"
	if !strings.Contains(logContent, expectedMvnBuild) {
		t.Errorf("expected mvn build command not found in log:\n%s\nwant:\n%s", logContent, expectedMvnBuild)
	}

	// Verify mvn dependency:get was called for google-java-format
	expectedFormatGet := "mvn dependency:get -Dartifact=com.google.googlejavaformat:google-java-format:1.25.2:jar:all-deps"
	if !strings.Contains(logContent, expectedFormatGet) {
		t.Errorf("expected google-java-format get command not found in log:\n%s\nwant:\n%s", logContent, expectedFormatGet)
	}

	// Verify mvn dependency:get was called for protoc-gen-grpc-java
	expectedGrpcGet := "mvn dependency:get -Dartifact=io.grpc:protoc-gen-grpc-java:1.76.3:exe:linux-x86_64"
	if !strings.Contains(logContent, expectedGrpcGet) {
		t.Errorf("expected grpc plugin get command not found in log:\n%s\nwant:\n%s", logContent, expectedGrpcGet)
	}

	// Verify pip install was called
	// Target 1: absolute path of local synthtool
	absLocalPipPath, err := filepath.Abs(localPipPath)
	if err != nil {
		t.Fatal(err)
	}
	expectedPip := "pip install --no-build-isolation " + absLocalPipPath + " PyYAML==6.0.2 jinja2==3.1.6"
	if !strings.Contains(logContent, expectedPip) {
		t.Errorf("expected pip install command not found in log:\n%s\nwant:\n%s", logContent, expectedPip)
	}

	// Verify wrappers
	// 1. protoc-gen-java_gapic
	gapicWrapper := filepath.Join(installDir, "protoc-gen-java_gapic")
	content, err := os.ReadFile(gapicWrapper)
	if err != nil {
		t.Fatal(err)
	}
	// Wait, we changed to t.TempDir() at the beginning, so working directory is a temp dir.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	expectedGapicContent := `exec java -cp "` + filepath.Join(wd, "sdk-platform-java/gapic-generator-java/target/gapic-generator-java-2.71.0.jar") + `" com.google.api.generator.Main`
	if !strings.Contains(string(content), expectedGapicContent) {
		t.Errorf("unexpected gapic wrapper content:\n%s\nwant it to contain:\n%s", string(content), expectedGapicContent)
	}

	// 2. google-java-format
	formatWrapper := filepath.Join(installDir, "google-java-format")
	content, err = os.ReadFile(formatWrapper)
	if err != nil {
		t.Fatal(err)
	}
	expectedFormatContent := `exec java -jar "` + filepath.Join(m2Repo, "com/google/googlejavaformat/google-java-format/1.25.2/google-java-format-1.25.2-all-deps.jar") + `"`
	if !strings.Contains(string(content), expectedFormatContent) {
		t.Errorf("unexpected format wrapper content:\n%s\nwant it to contain:\n%s", string(content), expectedFormatContent)
	}

	// 3. protoc-gen-java_grpc
	grpcWrapper := filepath.Join(installDir, "protoc-gen-java_grpc")
	content, err = os.ReadFile(grpcWrapper)
	if err != nil {
		t.Fatal(err)
	}
	expectedGrpcContent := `exec "` + filepath.Join(m2Repo, "io/grpc/protoc-gen-grpc-java/1.76.3/protoc-gen-grpc-java-1.76.3-linux-x86_64.exe") + `"`
	if !strings.Contains(string(content), expectedGrpcContent) {
		t.Errorf("unexpected grpc wrapper content:\n%s\nwant it to contain:\n%s", string(content), expectedGrpcContent)
	}
}

func TestInstall_PrerequisiteFailed_NotRepoRoot(t *testing.T) {
	// Run from a clean temp dir (no pom.xml)
	t.Chdir(t.TempDir())

	err := Install(t.Context())
	if err == nil {
		t.Fatal("expected error when not run from google-cloud-java root, got nil")
	}
	if !strings.Contains(err.Error(), "must be run from the root of a google-cloud-java repository clone") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInstall_PrerequisiteFailed_MissingTools(t *testing.T) {
	t.Chdir(t.TempDir())
	// Create pom.xml to pass the first check
	pomDir := "sdk-platform-java/gapic-generator-java"
	if err := os.MkdirAll(pomDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pomDir, "pom.xml"), []byte("<project></project>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Empty PATH to simulate missing java, mvn, pip
	t.Setenv("PATH", "")

	err := Install(t.Context())
	if err == nil {
		t.Fatal("expected error when tools are missing, got nil")
	}
	if !strings.Contains(err.Error(), "is not installed or not in PATH") {
		t.Errorf("unexpected error message: %v", err)
	}
}
