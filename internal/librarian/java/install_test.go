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
)

func TestInstall(t *testing.T) {
	tmpDir := t.TempDir()
	// 1. Setup temp HOME to isolate .m2/repository
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	// 2. Pre-create dummy downloaded artifacts in mock .m2 repository
	m2Repo := filepath.Join(tempHome, ".m2", "repository")
	// (a) Pre-create google-java-format JAR
	gjfDir := filepath.Join(m2Repo, "com", "google", "googlejavaformat", "google-java-format", "1.25.2")
	if err := os.MkdirAll(gjfDir, 0755); err != nil {
		t.Fatal(err)
	}
	gjfJarPath := filepath.Join(gjfDir, "google-java-format-1.25.2-all-deps.jar")
	if err := os.WriteFile(gjfJarPath, []byte("gjf jar content"), 0644); err != nil {
		t.Fatal(err)
	}
	// (b) Pre-create protoc-gen-grpc-java exe
	grpcDir := filepath.Join(m2Repo, "io", "grpc", "protoc-gen-grpc-java", "1.76.3")
	if err := os.MkdirAll(grpcDir, 0755); err != nil {
		t.Fatal(err)
	}
	grpcExePath := filepath.Join(grpcDir, "protoc-gen-grpc-java-1.76.3-linux-x86_64.exe")
	if err := os.WriteFile(grpcExePath, []byte("grpc exe content"), 0755); err != nil {
		t.Fatal(err)
	}
	// 3. Setup stub executables directory
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	// (a) Stub pip
	pipLogPath := filepath.Join(tmpDir, "pip_invocations.log")
	pipContent := fmt.Sprintf(`#!/bin/bash
echo "pip $@" >> %q
`, pipLogPath)
	if err := os.WriteFile(filepath.Join(stubDir, "pip"), []byte(pipContent), 0755); err != nil {
		t.Fatal(err)
	}
	// (b) Stub mvn
	mvnLogPath := filepath.Join(tmpDir, "mvn_invocations.log")
	mvnContent := fmt.Sprintf(`#!/bin/bash
echo "mvn $@" >> %q
`, mvnLogPath)
	if err := os.WriteFile(filepath.Join(stubDir, "mvn"), []byte(mvnContent), 0755); err != nil {
		t.Fatal(err)
	}
	// (c) Stub java
	if err := os.WriteFile(filepath.Join(stubDir, "java"), []byte("#!/bin/bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)
	// 4. Setup temp install dir
	installDir := filepath.Join(tmpDir, "java_tools", "bin")
	t.Setenv("LIBRARIAN_INSTALL_DIR", installDir)
	// 5. Execute Installer
	err := Install(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// 6. Assertions
	// (a) Verify pip calls
	pipData, err := os.ReadFile(pipLogPath)
	if err != nil {
		t.Fatal(err)
	}
	gotPip := strings.TrimSpace(string(pipData))
	wantPip := "pip install PyYAML==6.0.2 jinja2==3.1.6"
	if diff := cmp.Diff(wantPip, gotPip); diff != "" {
		t.Errorf("pip invocations mismatch (-want +got):\n%s", diff)
	}
	// (b) Verify mvn calls
	mvnData, err := os.ReadFile(mvnLogPath)
	if err != nil {
		t.Fatal(err)
	}
	gotMvn := strings.TrimSpace(string(mvnData))
	wantMvn := "mvn dependency:get -Dartifact=com.google.googlejavaformat:google-java-format:1.25.2:jar:all-deps\nmvn dependency:get -Dartifact=io.grpc:protoc-gen-grpc-java:1.76.3:exe:linux-x86_64"
	if diff := cmp.Diff(wantMvn, gotMvn); diff != "" {
		t.Errorf("mvn invocations mismatch (-want +got):\n%s", diff)
	}
	// (c) Verify files copied to libDir
	libDir := filepath.Join(tmpDir, "java_tools", "lib")
	gjfCopiedPath := filepath.Join(libDir, "google-java-format-1.25.2-all-deps.jar")
	gjfData, err := os.ReadFile(gjfCopiedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gjfData) != "gjf jar content" {
		t.Errorf("copied GJF jar contents mismatch: got %q", string(gjfData))
	}
	grpcCopiedPath := filepath.Join(libDir, "protoc-gen-grpc-java-1.76.3-linux-x86_64.exe")
	grpcData, err := os.ReadFile(grpcCopiedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(grpcData) != "grpc exe content" {
		t.Errorf("copied grpc exe contents mismatch: got %q", string(grpcData))
	}
	// (d) Verify wrappers are written to binDir pointing to libDir
	gjfWrapperPath := filepath.Join(installDir, "google-java-format")
	gjfWrapper, err := os.ReadFile(gjfWrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	wantGjfWrapper := fmt.Sprintf("#!/bin/sh\nexec java -jar %q \"$@\"\n", gjfCopiedPath)
	if diff := cmp.Diff(wantGjfWrapper, string(gjfWrapper)); diff != "" {
		t.Errorf("GJF wrapper contents mismatch (-want +got):\n%s", diff)
	}
	grpcWrapperPath := filepath.Join(installDir, "protoc-gen-java_grpc")
	grpcWrapper, err := os.ReadFile(grpcWrapperPath)
	if err != nil {
		t.Fatal(err)
	}
	wantGrpcWrapper := fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", grpcCopiedPath)
	if diff := cmp.Diff(wantGrpcWrapper, string(grpcWrapper)); diff != "" {
		t.Errorf("grpc wrapper contents mismatch (-want +got):\n%s", diff)
	}
}
