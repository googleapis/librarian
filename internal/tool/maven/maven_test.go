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
	"errors"
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
	localMvnDir := filepath.Join(tmpDir, "sdk-platform-java", "gapic-generator-java")
	if err := os.MkdirAll(filepath.Join(localMvnDir, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	mockPOM := `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <parent>
    <groupId>com.google.api.generator</groupId>
    <version>2.28.0-SNAPSHOT</version>
  </parent>
  <artifactId>gapic-generator-java</artifactId>
</project>`
	if err := os.WriteFile(filepath.Join(localMvnDir, "pom.xml"), []byte(mockPOM), 0o644); err != nil {
		t.Fatal(err)
	}
	mockJarPath := filepath.Join(localMvnDir, "target", "gapic-generator-java-2.28.0-SNAPSHOT.jar")
	if err := os.WriteFile(mockJarPath, []byte("local gapic jar content"), 0o644); err != nil {
		t.Fatal(err)
	}
	m2Repo := filepath.Join(tempHome, ".m2", "repository")
	gjfDir := filepath.Join(m2Repo, "com", "google", "googlejavaformat", "google-java-format", "1.25.2")
	if err := os.MkdirAll(gjfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gjfJarPath := filepath.Join(gjfDir, "google-java-format-1.25.2-all-deps.jar")
	if err := os.WriteFile(gjfJarPath, []byte("gjf jar content"), 0o644); err != nil {
		t.Fatal(err)
	}
	grpcDir := filepath.Join(m2Repo, "io", "grpc", "protoc-gen-grpc-java", "1.81.0")
	if err := os.MkdirAll(grpcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	grpcExePath := filepath.Join(grpcDir, "protoc-gen-grpc-java-1.81.0-linux-x86_64.exe")
	if err := os.WriteFile(grpcExePath, []byte("grpc exe content"), 0o755); err != nil {
		t.Fatal(err)
	}
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mvnLogPath := filepath.Join(tmpDir, "mvn_invocations.log")
	mvnContent := fmt.Sprintf("#!/bin/sh\necho mvn \"$@\" >> %q\n", mvnLogPath)
	if err := os.WriteFile(filepath.Join(stubDir, "mvn"), []byte(mvnContent), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stubDir, "java"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)
	tools := []*config.MavenTool{
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
	}
	binDir := filepath.Join(tmpDir, "java_tools", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(tmpDir, "java_tools", "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LIBRARIAN_BIN", tmpDir)
	if err := Install(t.Context(), tools, binDir, libDir); err != nil {
		t.Fatal(err)
	}
	mvnData, err := os.ReadFile(mvnLogPath)
	if err != nil {
		t.Fatal(err)
	}
	gotMvn := strings.TrimSpace(string(mvnData))
	wantMvn := "mvn dependency:get -Dartifact=com.google.googlejavaformat:google-java-format:1.25.2:jar:all-deps\n" +
		"mvn dependency:get -Dartifact=io.grpc:protoc-gen-grpc-java:1.81.0:exe:linux-x86_64\n" +
		"mvn package -B -ntp -T 1.5C -DskipTests -Dcheckstyle.skip -Dclirr.skip -Denforcer.skip -Dfmt.skip " +
		"-pl sdk-platform-java/gapic-generator-java --also-make"
	if diff := cmp.Diff(wantMvn, gotMvn); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	for _, test := range []struct {
		name        string
		filename    string
		wantContent string
		wrapperName string
		wantFormat  string
	}{
		{
			name:        "google-java-format",
			filename:    "google-java-format-1.25.2-all-deps.jar",
			wantContent: "gjf jar content",
			wrapperName: "google-java-format",
			wantFormat:  "#!/bin/sh\nexec java -jar %q \"$@\"\n",
		},
		{
			name:        "protoc-gen-java_grpc",
			filename:    "protoc-gen-grpc-java-1.81.0-linux-x86_64.exe",
			wantContent: "grpc exe content",
			wrapperName: "protoc-gen-java_grpc",
			wantFormat:  "#!/bin/sh\nexec %q \"$@\"\n",
		},
		{
			name:        "protoc-gen-java_gapic",
			filename:    "gapic-generator-java-2.28.0-SNAPSHOT.jar",
			wantContent: "local gapic jar content",
			wrapperName: "protoc-gen-java_gapic",
			wantFormat:  "#!/bin/sh\nexec java -cp %q \"com.google.api.generator.Main\" \"$@\"\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			copiedPath := filepath.Join(libDir, test.filename)
			data, err := os.ReadFile(copiedPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantContent, string(data)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			wrapperPath := filepath.Join(binDir, test.wrapperName)
			wrapper, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatal(err)
			}
			wantWrapper := fmt.Sprintf(test.wantFormat, copiedPath)
			if diff := cmp.Diff(wantWrapper, string(wrapper)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParsePOM(t *testing.T) {
	tmpDir := t.TempDir()
	mockPOM := `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <parent>
    <groupId>com.google.api.generator</groupId>
    <version>2.28.0-SNAPSHOT</version>
  </parent>
  <artifactId>gapic-generator-java</artifactId>
</project>`
	pomPath := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(mockPOM), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := parsePOM(pomPath)
	if err != nil {
		t.Fatal(err)
	}
	want := &pomProject{
		ArtifactID: "gapic-generator-java",
		Version:    "2.28.0-SNAPSHOT",
	}
	if diff := cmp.Diff(want.ArtifactID, got.ArtifactID); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.Version, got.Version); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestParsePOM_Error(t *testing.T) {
	tmpDir := t.TempDir()
	for _, test := range []struct {
		name    string
		pomPath string
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name:    "missing file error",
			pomPath: filepath.Join(tmpDir, "nonexistent.xml"),
			wantErr: errReadPOM,
		},
		{
			name:    "invalid XML syntax",
			pomPath: filepath.Join(tmpDir, "invalid.xml"),
			setup: func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(tmpDir, "invalid.xml"), []byte("<project><invalid"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errParsePOM,
		},
		{
			name:    "missing artifactId or version",
			pomPath: filepath.Join(tmpDir, "empty.xml"),
			setup: func(t *testing.T) {
				if err := os.WriteFile(filepath.Join(tmpDir, "empty.xml"), []byte("<project></project>"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: errInvalidPOM,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			_, err := parsePOM(test.pomPath)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("parsePOM() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}
