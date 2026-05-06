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

package python

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func TestInstall(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "pip"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cache := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cache)
	cacheBin := filepath.Join(cache, "bin")
	if err := os.MkdirAll(cacheBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheBin, "pandoc"), []byte("#!/bin/sh\necho \"pandoc 3.8.2\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Install(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestInstallPandoc(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake pandoc is a shell script; not runnable on Windows")
	}

	const version = "3.8.2"
	archive, err := buildFakePandocTarGz(version)
	if err != nil {
		t.Fatal(err)
	}

	artifactName, err := pandocArtifact(version, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	expectedPath := "/" + version + "/" + artifactName

	var requests atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != expectedPath {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(archive)
	}))
	t.Cleanup(srv.Close)

	originalBaseURL := pandocBaseURL
	originalArtifact := pandocArtifactFn
	pandocBaseURL = srv.URL
	pandocArtifactFn = func(v, _, _ string) (string, error) {
		return pandocArtifact(v, "linux", "amd64")
	}
	t.Cleanup(func() {
		pandocBaseURL = originalBaseURL
		pandocArtifactFn = originalArtifact
	})

	cache := t.TempDir()
	t.Setenv("LIBRARIAN_CACHE", cache)

	if err := installPandoc(t.Context(), version); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(cache, "bin", "pandoc")
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o755 {
		t.Errorf("mode = %o, want 0755", mode)
	}
	out, err := exec.Command(target, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "pandoc "+version) {
		t.Errorf("--version output = %q, want to contain %q", string(out), "pandoc "+version)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("server requests = %d, want 1", got)
	}

	t.Run("idempotent", func(t *testing.T) {
		if err := installPandoc(t.Context(), version); err != nil {
			t.Fatal(err)
		}
		if got := requests.Load(); got != 1 {
			t.Errorf("server requests after second call = %d, want 1", got)
		}
	})
}

// buildFakePandocTarGz creates an in-memory tar.gz archive whose only entry
// is a tiny shell script at pandoc-<version>/bin/pandoc that prints the
// version string.
func buildFakePandocTarGz(version string) ([]byte, error) {
	script := []byte("#!/bin/sh\necho \"pandoc " + version + "\"\n")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name:     "pandoc-" + version + "/bin/pandoc",
		Mode:     0o755,
		Size:     int64(len(script)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(script); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
