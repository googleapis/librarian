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

package librarian

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProtocInstallDir(t *testing.T) {
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
			got, err := protocInstallDir(test.version)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtocDownloadURL(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		os      string
		arch    string
		want    string
	}{
		{
			name:    "simple version",
			version: "25.1",
			os:      "darwin",
			arch:    "arm64",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-osx-aarch_64.zip",
		},
		{
			name:    "release candidate",
			version: "26.0-rc1",
			os:      "linux",
			arch:    "amd64",
			want:    "https://github.com/protocolbuffers/protobuf/releases/download/v26.0-rc1/protoc-26.0-rc1-linux-x86_64.zip",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := protocDownloadURL(test.version, test.os, test.arch)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInstallProtoc(t *testing.T) {
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
	if err := installProtoc(context.Background(), server.URL, dir, checksum); err != nil {
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
