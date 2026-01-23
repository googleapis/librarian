// Copyright 2024 Google LLC
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

package sidekick

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/sdk/internal/sidekick/config"
	"github.com/walle/targz"
)

func TestExistingDirectory(t *testing.T) {
	tmp := t.TempDir()
	rootConfig := config.Config{
		Source: map[string]string{
			"googleapis-root": tmp,
		},
	}
	root, err := makeSourceRoot(context.Background(), &rootConfig, "googleapis")
	if err != nil {
		t.Error(err)
	}
	if root != tmp {
		t.Errorf("mismatched root directory got=%s, want=%s", root, tmp)
	}
}

func TestValidateConfig(t *testing.T) {
	rootConfig := config.Config{
		Source: map[string]string{
			"googleapis-root": "https://unused",
		},
	}
	_, err := makeSourceRoot(context.Background(), &rootConfig, "googleapis")
	if err == nil {
		t.Errorf("expected error when missing `googleapis-sha256")
	}
}

func TestWithDownload(t *testing.T) {
	testDir := t.TempDir()

	simulatedSha := "2d08f07eab9bbe8300cd20b871d0811bbb693fab"
	simulatedSubdir := fmt.Sprintf("googleapis-%s", simulatedSha)
	simulatedPath := fmt.Sprintf("/archive/%s.tar.gz", simulatedSha)
	tarball, err := makeTestTarball(t, testDir, simulatedSubdir)
	if err != nil {
		t.Fatal(err)
	}

	// In this test we expect that a download is needed.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != simulatedPath {
			t.Errorf("Expected to request '%s', got: %s", simulatedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(tarball.Contents)
	}))
	defer server.Close()

	rootConfig := &config.Config{
		Source: map[string]string{
			"googleapis-root":   server.URL + simulatedPath,
			"googleapis-sha256": tarball.Sha256,
			"cachedir":          testDir,
		},
	}
	got, err := makeSourceRoot(context.Background(), rootConfig, "googleapis")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, tarball.Sha256) {
		t.Errorf("mismatched suffix in makeSourceRoot want=%s, got=%s", tarball.Sha256, got)
	}
	if err := os.RemoveAll(got); err != nil {
		t.Error(err)
	}
	if err := os.Remove(got + ".tar.gz"); err != nil {
		t.Error(err)
	}
}

func TestTargetExists(t *testing.T) {
	testDir := t.TempDir()

	sha256 := "eb853d49313f20a096607fea87dfc10bd6a1b917ad17ad5db8a205b457a940e1"
	rootConfig := &config.Config{
		Source: map[string]string{
			"googleapis-root":   "https://unused/path",
			"googleapis-sha256": sha256,
			"cachedir":          testDir,
		},
	}

	downloads, err := getCacheDir(rootConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(path.Join(downloads, sha256), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := makeSourceRoot(context.Background(), rootConfig, "googleapis")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, sha256) {
		t.Errorf("mismatched suffix in makeSourceRoot want=%s, got=%s", sha256, got)
	}
	if err := os.RemoveAll(got); err != nil {
		t.Error(err)
	}
}

type contents struct {
	Sha256   string
	Contents []byte
}

func makeTestTarball(t *testing.T, tempDir, subdir string) (*contents, error) {
	t.Helper()

	top := path.Join(tempDir, subdir)
	if err := os.MkdirAll(top, 0755); err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		name := fmt.Sprintf("file-%04d", i)
		err := os.WriteFile(path.Join(top, name), []byte(fmt.Sprintf("%08d the quick brown fox jumps over the lazy dog\n", i)), 0644)
		if err != nil {
			return nil, err
		}
	}

	tgz := path.Join(tempDir, "tarball.tgz")
	defer os.Remove(tgz)

	if err := targz.Compress(top, tgz); err != nil {
		return nil, err
	}

	hasher := sha256.New()
	data, err := os.ReadFile(tgz)
	if err != nil {
		return nil, err
	}
	hasher.Write(data)

	return &contents{
		Sha256:   fmt.Sprintf("%x", hasher.Sum(nil)),
		Contents: data,
	}, nil
}

func TestExtractedName(t *testing.T) {
	var rootConfig config.Config
	got := extractedName(&rootConfig, "https://github.com/googleapis/googleapis/archive/2d08f07eab9bbe8300cd20b871d0811bbb693fab.tar.gz", "googleapis")
	want := "googleapis-2d08f07eab9bbe8300cd20b871d0811bbb693fab"
	if got != want {
		t.Errorf("mismatched extractedName, got=%s, want=%s", got, want)
	}
}

func TestExtractedNameOverride(t *testing.T) {
	want := "override"
	rootConfig := config.Config{
		Source: map[string]string{
			"googleapis-extracted-name": want,
		},
	}
	got := extractedName(&rootConfig, "https://github.com/googleapis/googleapis/archive/2d08f07eab9bbe8300cd20b871d0811bbb693fab.tar.gz", "googleapis")
	if got != want {
		t.Errorf("mismatched extractedName, got=%s, want=%s", got, want)
	}
}

func TestDownloadsCacheDir(t *testing.T) {
	dir, err := getCacheDir(&config.Config{Source: map[string]string{"cachedir": "test-only"}})
	if err != nil {
		t.Fatal(err)
	}
	checkDownloadsCacheDir(t, dir, "test-only")

	user, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	dir, err = getCacheDir(&config.Config{Source: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	checkDownloadsCacheDir(t, dir, user)
}

func checkDownloadsCacheDir(t *testing.T, got, root string) {
	t.Helper()
	if !strings.HasPrefix(got, root) {
		t.Errorf("mismatched downloadsCacheDir, want=%s, got=%s", root, got)
	}
	if !strings.Contains(got, path.Join("sidekick", "downloads")) {
		t.Errorf("mismatched downloadsCacheDir, want=%s, got=%s", "sidekick", root)
	}
}

func TestMakeSourceRootErrors(t *testing.T) {
	t.Run("invalid-source-root", func(t *testing.T) {
		rootConfig := config.Config{
			Source: map[string]string{
				"googleapis-root": "this-is-not-a-valid-path-and-not-a-url",
			},
		}
		_, err := makeSourceRoot(context.Background(), &rootConfig, "googleapis")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "only directories and https URLs are supported") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("download-fails", func(t *testing.T) {
		// Temporarily replace the download function with a mock to simulate failure
		oldDownloadTarball := downloadTarball
		downloadTarball = func(ctx context.Context, target, url, expectedSha256 string) error {
			return fmt.Errorf("download failed after 3 attempts")
		}
		t.Cleanup(func() {
			downloadTarball = oldDownloadTarball
		})
		rootConfig := &config.Config{
			Source: map[string]string{
				"googleapis-root":   "https://some-url",
				"googleapis-sha256": "somesha",
				"cachedir":          t.TempDir(),
			},
		}
		_, err := makeSourceRoot(context.Background(), rootConfig, "googleapis")
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "download failed after 3 attempts") {
			t.Errorf("expected 'download failed after 3 attempts' in error, got %v", err)
		}
	})

	t.Run("extract-fails", func(t *testing.T) {
		testDir := t.TempDir()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("this is not a tarball"))
		}))
		defer server.Close()

		hasher := sha256.New()
		hasher.Write([]byte("this is not a tarball"))
		sha := fmt.Sprintf("%x", hasher.Sum(nil))

		rootConfig := &config.Config{
			Source: map[string]string{
				"googleapis-root":   server.URL,
				"googleapis-sha256": sha,
				"cachedir":          testDir,
			},
		}
		_, err := makeSourceRoot(context.Background(), rootConfig, "googleapis")
		if err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("rename-fails", func(t *testing.T) {
		testDir := t.TempDir()
		simulatedSha := "2d08f07eab9bbe8300cd20b871d0811bbb693fab"
		simulatedSubdir := fmt.Sprintf("googleapis-%s", simulatedSha)
		simulatedPath := fmt.Sprintf("/archive/%s.tar.gz", simulatedSha)
		tarball, err := makeTestTarball(t, testDir, simulatedSubdir)
		if err != nil {
			t.Fatal(err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(tarball.Contents)
		}))
		defer server.Close()

		rootConfig := &config.Config{
			Source: map[string]string{
				"googleapis-root":   server.URL + simulatedPath,
				"googleapis-sha256": tarball.Sha256,
				"cachedir":          testDir,
			},
		}
		cacheDir, err := getCacheDir(rootConfig)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			t.Fatal(err)
		}
		targetFile := path.Join(cacheDir, tarball.Sha256)
		if err := os.WriteFile(targetFile, []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err = makeSourceRoot(context.Background(), rootConfig, "googleapis")
		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestGetCacheDirFails(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	_, err := getCacheDir(&config.Config{Source: map[string]string{}})
	if err == nil {
		t.Fatal("expected an error when HOME and XDG_CACHE_HOME are not set")
	}
}

func TestIsDirectoryFails(t *testing.T) {
	if isDirectory("a-path-with\x00-null-byte") {
		t.Error("isDirectory returned true for a path with a null byte, expected false")
	}
}
