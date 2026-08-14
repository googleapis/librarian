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

package protoc

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestInstallDir(t *testing.T) {
	for _, test := range []struct {
		name         string
		version      string
		librarianBin string
		cacheDir     string
		want         string
	}{
		{
			name:         "valid version with LIBRARIAN_BIN",
			version:      "25.1",
			librarianBin: "/custom/bin",
			want:         filepath.FromSlash("/custom/bin/protoc/v25.1"),
		},
		{
			name:     "valid version with LIBRARIAN_CACHE fallback",
			version:  "26.0-rc1",
			cacheDir: "/custom/cache",
			want:     filepath.FromSlash("/custom/cache/bin/protoc/v26.0-rc1"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.librarianBin != "" {
				t.Setenv("LIBRARIAN_BIN", test.librarianBin)
			} else {
				t.Setenv("LIBRARIAN_BIN", "")
			}
			if test.cacheDir != "" {
				t.Setenv("LIBRARIAN_CACHE", test.cacheDir)
			} else {
				t.Setenv("LIBRARIAN_CACHE", "")
			}
			got, err := InstallDir(test.version)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBinaryPath(t *testing.T) {
	binaryName := protocBinaryName()
	for _, test := range []struct {
		name         string
		version      string
		librarianBin string
		cacheDir     string
		want         string
	}{
		{
			name:         "valid version with LIBRARIAN_BIN",
			version:      "25.1",
			librarianBin: "/custom/bin",
			want:         filepath.FromSlash("/custom/bin/protoc/v25.1/bin/" + binaryName),
		},
		{
			name:     "valid version with LIBRARIAN_CACHE fallback",
			version:  "26.0-rc1",
			cacheDir: "/custom/cache",
			want:     filepath.FromSlash("/custom/cache/bin/protoc/v26.0-rc1/bin/" + binaryName),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.librarianBin != "" {
				t.Setenv("LIBRARIAN_BIN", test.librarianBin)
			} else {
				t.Setenv("LIBRARIAN_BIN", "")
			}
			if test.cacheDir != "" {
				t.Setenv("LIBRARIAN_CACHE", test.cacheDir)
			} else {
				t.Setenv("LIBRARIAN_CACHE", "")
			}
			got, err := BinaryPath(test.version)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBinaryPath_Error(t *testing.T) {
	if _, err := BinaryPath(""); err == nil {
		t.Fatal("BinaryPath(\"\") expected error, got nil")
	}
}

func TestBinaryPathOrSystem(t *testing.T) {
	binaryName := protocBinaryName()

	for _, test := range []struct {
		name         string
		pc           *config.Protoc
		librarianBin string
		setupPATH    func(t *testing.T) string
		want         func(t *testing.T, installedPath string) string
	}{
		{
			name:         "configured protoc uses installed binary path",
			pc:           &config.Protoc{Version: "33.2"},
			librarianBin: "/custom/bin",
			want: func(t *testing.T, _ string) string {
				return filepath.FromSlash("/custom/bin/protoc/v33.2/bin/" + binaryName)
			},
		},
		{
			name: "nil config falls back to system PATH",
			pc:   nil,
			setupPATH: func(t *testing.T) string {
				return createFakeSystemExecutable(t, binaryName)
			},
			want: func(t *testing.T, installedPath string) string {
				return installedPath
			},
		},
		{
			name: "empty version falls back to system PATH",
			pc:   &config.Protoc{},
			setupPATH: func(t *testing.T) string {
				return createFakeSystemExecutable(t, binaryName)
			},
			want: func(t *testing.T, installedPath string) string {
				return installedPath
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.librarianBin != "" {
				t.Setenv("LIBRARIAN_BIN", test.librarianBin)
			}
			var installedPath string
			if test.setupPATH != nil {
				installedPath = test.setupPATH(t)
			}
			got, err := BinaryPathOrSystem(test.pc)
			if err != nil {
				t.Fatal(err)
			}
			want := test.want(t, installedPath)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBinaryPathOrSystem_Error(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := BinaryPathOrSystem(nil); err == nil {
		t.Fatal("BinaryPathOrSystem(nil) with empty PATH expected error, got nil")
	}
}

func TestRun(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping execution test on Windows")
	}
	binaryName := protocBinaryName()
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	version := "33.2"
	protocDir := filepath.Join(binDir, "protoc", "v"+version, "bin")
	if err := os.MkdirAll(protocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testhelper.WriteExecutable(t, filepath.Join(protocDir, binaryName), "#!/bin/sh\nexit 0\n")

	pc := &config.Protoc{Version: version}
	if err := Run(t.Context(), nil, pc, "--version"); err != nil {
		t.Fatal(err)
	}
}

func TestRunOrSystem(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("skipping execution test on Windows")
	}
	binaryName := protocBinaryName()
	createFakeSystemExecutable(t, binaryName)

	if err := RunOrSystem(t.Context(), nil, nil, "--version"); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadURL(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		os      string
		arch    string
		want    string
	}{
		{
			name:    "macos arm64",
			version: "25.1",
			os:      "darwin",
			arch:    "arm64",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-osx-aarch_64.zip",
		},
		{
			name:    "macos amd64",
			version: "25.1",
			os:      "darwin",
			arch:    "amd64",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-osx-x86_64.zip",
		},
		{
			name:    "linux amd64",
			version: "26.0-rc1",
			os:      "linux",
			arch:    "amd64",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v26.0-rc1/protoc-26.0-rc1-linux-x86_64.zip",
		},
		{
			name:    "linux arm64",
			version: "29.3",
			os:      "linux",
			arch:    "arm64",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protoc-29.3-linux-aarch_64.zip",
		},
		{
			name:    "windows amd64",
			version: "29.3",
			os:      "windows",
			arch:    "amd64",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protoc-29.3-win64.zip",
		},
		{
			name:    "windows 386",
			version: "29.3",
			os:      "windows",
			arch:    "386",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protoc-29.3-win32.zip",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := downloadURL(test.version, test.os, test.arch)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDownloadURL_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		os      string
		arch    string
	}{
		{
			name:    "unsupported os",
			version: "29.3",
			os:      "freebsd",
			arch:    "amd64",
		},
		{
			name:    "unsupported arch",
			version: "29.3",
			os:      "linux",
			arch:    "mips",
		},
		{
			name:    "unsupported windows arch",
			version: "29.3",
			os:      "windows",
			arch:    "mips",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := downloadURL(test.version, test.os, test.arch); err == nil {
				t.Errorf("downloadURL(%q, %q, %q) expected error, got nil", test.version, test.os, test.arch)
			}
		})
	}
}

func TestPlatformSuffix(t *testing.T) {
	for _, test := range []struct {
		name string
		os   string
		arch string
		want string
	}{
		{name: "darwin arm64", os: "darwin", arch: "arm64", want: "osx-aarch_64"},
		{name: "darwin amd64", os: "darwin", arch: "amd64", want: "osx-x86_64"},
		{name: "linux amd64", os: "linux", arch: "amd64", want: "linux-x86_64"},
		{name: "linux arm64", os: "linux", arch: "arm64", want: "linux-aarch_64"},
		{name: "windows amd64", os: "windows", arch: "amd64", want: "win64"},
		{name: "windows arm64", os: "windows", arch: "arm64", want: "win64"},
		{name: "windows 386", os: "windows", arch: "386", want: "win32"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := platformSuffix(test.os, test.arch)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSha256ForPlatform(t *testing.T) {
	pcWithMap := &config.Protoc{
		Version: "29.3",
		SHA256:  "legacy-fallback-sha",
		SHA256ByPlatform: map[string]string{
			"osx-aarch_64": "osx-arm-sha",
			"osx-x86_64":   "osx-intel-sha",
			"linux-x86_64": "linux-x86-sha",
			"win64":        "win64-sha",
		},
	}

	pcLegacyOnly := &config.Protoc{
		Version: "29.3",
		SHA256:  "legacy-fallback-sha",
	}

	for _, test := range []struct {
		name   string
		pc     *config.Protoc
		goos   string
		goarch string
		want   string
	}{
		{
			name:   "explicit map for osx-aarch_64",
			pc:     pcWithMap,
			goos:   "darwin",
			goarch: "arm64",
			want:   "osx-arm-sha",
		},
		{
			name:   "explicit map for osx-x86_64",
			pc:     pcWithMap,
			goos:   "darwin",
			goarch: "amd64",
			want:   "osx-intel-sha",
		},
		{
			name:   "explicit map for linux-x86_64",
			pc:     pcWithMap,
			goos:   "linux",
			goarch: "amd64",
			want:   "linux-x86-sha",
		},
		{
			name:   "explicit map for win64",
			pc:     pcWithMap,
			goos:   "windows",
			goarch: "amd64",
			want:   "win64-sha",
		},
		{
			name:   "legacy fallback for linux-x86_64",
			pc:     pcLegacyOnly,
			goos:   "linux",
			goarch: "amd64",
			want:   "legacy-fallback-sha",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := sha256ForPlatform(test.pc, test.goos, test.goarch)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSha256ForPlatform_Error(t *testing.T) {
	for _, test := range []struct {
		name   string
		pc     *config.Protoc
		goos   string
		goarch string
	}{
		{
			name: "missing macos checksum with legacy linux only",
			pc: &config.Protoc{
				Version: "29.3",
				SHA256:  "linux-only-sha",
			},
			goos:   "darwin",
			goarch: "arm64",
		},
		{
			name: "missing windows checksum with empty config",
			pc: &config.Protoc{
				Version: "29.3",
			},
			goos:   "windows",
			goarch: "amd64",
		},
		{
			name: "unsupported os",
			pc: &config.Protoc{
				Version: "29.3",
			},
			goos:   "plan9",
			goarch: "amd64",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := sha256ForPlatform(test.pc, test.goos, test.goarch); err == nil {
				t.Errorf("sha256ForPlatform(%+v, %q, %q) expected error, got nil", test.pc, test.goos, test.goarch)
			}
		})
	}
}

func TestDownloadAndExtract(t *testing.T) {
	mockZip, err := createMockZip(t)
	if err != nil {
		t.Fatal(err)
	}
	hasher := sha256.New()
	hasher.Write(mockZip)
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(mockZip)
	}))
	defer server.Close()
	dir := t.TempDir()
	if err := downloadAndExtract(t.Context(), server.URL, dir, checksum); err != nil {
		t.Fatal(err)
	}
	expectedFiles := []string{
		filepath.Join(dir, "bin", "protoc"),
		filepath.Join(dir, "include", "google", "protobuf", "any.proto"),
		filepath.Join(dir, "other_file.txt"),
	}
	for _, expected := range expectedFiles {
		if _, err := os.Stat(expected); err != nil {
			t.Errorf("expected file %q was not extracted: %v", expected, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "protoc.zip")); err == nil {
		t.Errorf("zip file was not cleaned up")
	}
}

func TestInstall_AlreadyInstalled(t *testing.T) {
	binaryName := protocBinaryName()
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	version := "29.3"
	protocDir := filepath.Join(binDir, "protoc", "v"+version, "bin")
	if err := os.MkdirAll(protocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testhelper.WriteExecutable(t, filepath.Join(protocDir, binaryName), "#!/bin/sh\nexit 0\n")

	pc := &config.Protoc{Version: version}
	if err := Install(t.Context(), pc); err != nil {
		t.Fatal(err)
	}
}

func TestInstall_MissingSHA256Error(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)
	pc := &config.Protoc{Version: "29.3-test-missing-sha"}
	if err := Install(t.Context(), pc); err == nil {
		t.Fatal("Install expected error for missing sha256, got nil")
	}
}

func protocBinaryName() string {
	if runtime.GOOS == osWindows {
		return "protoc.exe"
	}
	return "protoc"
}

func createFakeSystemExecutable(t *testing.T, binaryName string) string {
	t.Helper()
	tempDir := t.TempDir()
	fakePath := filepath.Join(tempDir, binaryName)
	testhelper.WriteExecutable(t, fakePath, "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", tempDir+string(filepath.ListSeparator)+os.Getenv("PATH"))
	return fakePath
}

func createMockZip(t *testing.T) ([]byte, error) {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	files := []struct {
		Name, Body string
	}{
		{"bin/protoc", "mock protoc binary"},
		{"include/google/protobuf/any.proto", "mock any proto"},
		{"other_file.txt", "should be included"},
	}
	for _, file := range files {
		f, err := w.Create(file.Name)
		if err != nil {
			return nil, err
		}
		_, err = f.Write([]byte(file.Body))
		if err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
