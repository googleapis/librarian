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
	t.Chdir(tmpDir)
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	localPkgDir := filepath.Join(tmpDir, "sdk-platform-java", "hermetic_build", "library_generation")
	if err := os.MkdirAll(localPkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	m2Repo := filepath.Join(tempHome, ".m2", "repository")
	gjfDir := filepath.Join(m2Repo, "com", "google", "googlejavaformat", "google-java-format", "1.25.2")
	if err := os.MkdirAll(gjfDir, 0755); err != nil {
		t.Fatal(err)
	}
	gjfJarPath := filepath.Join(gjfDir, "google-java-format-1.25.2-all-deps.jar")
	if err := os.WriteFile(gjfJarPath, []byte("gjf jar content"), 0644); err != nil {
		t.Fatal(err)
	}
	grpcDir := filepath.Join(m2Repo, "io", "grpc", "protoc-gen-grpc-java", "1.76.3")
	if err := os.MkdirAll(grpcDir, 0755); err != nil {
		t.Fatal(err)
	}
	grpcExePath := filepath.Join(grpcDir, "protoc-gen-grpc-java-1.76.3-linux-x86_64.exe")
	if err := os.WriteFile(grpcExePath, []byte("grpc exe content"), 0755); err != nil {
		t.Fatal(err)
	}
	stubs := []struct {
		name        string
		logFilename string
		wantArgs    string
	}{
		{
			name:        "pip",
			logFilename: "pip_invocations.log",
			wantArgs:    "pip install PyYAML==6.0.2 jinja2==3.1.6 " + localPkgDir,
		},
		{
			name:        "mvn",
			logFilename: "mvn_invocations.log",
			wantArgs:    "mvn dependency:get -Dartifact=com.google.googlejavaformat:google-java-format:1.25.2:jar:all-deps\nmvn dependency:get -Dartifact=io.grpc:protoc-gen-grpc-java:1.76.3:exe:linux-x86_64",
		},
	}
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, s := range stubs {
		logPath := filepath.Join(tmpDir, s.logFilename)
		content := fmt.Sprintf("#!/bin/sh\necho %q \"$@\" >> %q\n", s.name, logPath)
		if err := os.WriteFile(filepath.Join(stubDir, s.name), []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(stubDir, "java"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)
	installDir := filepath.Join(tmpDir, "java_tools", "bin")
	t.Setenv("LIBRARIAN_INSTALL_DIR", installDir)
	err := Install(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range stubs {
		logPath := filepath.Join(tmpDir, s.logFilename)
		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatal(err)
		}
		got := strings.TrimSpace(string(data))
		if diff := cmp.Diff(s.wantArgs, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
	libDir := filepath.Join(tmpDir, "java_tools", "lib")
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
			filename:    "protoc-gen-grpc-java-1.76.3-linux-x86_64.exe",
			wantContent: "grpc exe content",
			wrapperName: "protoc-gen-java_grpc",
			wantFormat:  "#!/bin/sh\nexec %q \"$@\"\n",
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
			wrapperPath := filepath.Join(installDir, test.wrapperName)
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
