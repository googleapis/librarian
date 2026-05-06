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
	"archive/zip"
	"compress/gzip"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

//go:embed librarian.yaml
var librarianYAML []byte

// pandocBaseURL is the base URL for downloading pandoc release archives. It is
// a variable so tests can override it.
var pandocBaseURL = "https://github.com/jgm/pandoc/releases/download"

// pandocArtifactFn resolves the archive file name for the given version,
// runtime.GOOS, and runtime.GOARCH. It is a variable so tests can override
// platform detection.
var pandocArtifactFn = pandocArtifact

// Install installs Python pip tool dependencies.
func Install(ctx context.Context) error {
	cfg, err := yaml.Unmarshal[config.Config](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}
	if cfg.Tools == nil {
		return nil
	}
	if len(cfg.Tools.Pip) > 0 {
		args := []string{"install"}
		for _, tool := range cfg.Tools.Pip {
			pkg := tool.Package
			if pkg == "" {
				pkg = fmt.Sprintf("%s==%s", tool.Name, tool.Version)
			}
			args = append(args, pkg)
		}
		if err := command.RunStreaming(ctx, "pip", args...); err != nil {
			return err
		}
	}
	if cfg.Tools.Pandoc != "" {
		if err := installPandoc(ctx, cfg.Tools.Pandoc); err != nil {
			return err
		}
	}
	return nil
}

// installPandoc downloads pandoc at the given version into the librarian
// cache bin directory. It is a no-op if the right version is already present.
func installPandoc(ctx context.Context, version string) error {
	cache, err := librarianCacheDir()
	if err != nil {
		return err
	}
	binDir := filepath.Join(cache, "bin")
	target := filepath.Join(binDir, "pandoc")

	if installed, err := pandocVersion(ctx, target); err == nil && installed == version {
		return nil
	}

	artifact, err := pandocArtifactFn(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%s", pandocBaseURL, version, artifact)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("creating %q: %w", binDir, err)
	}
	tmp, err := os.CreateTemp("", "pandoc-*"+filepath.Ext(artifact))
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := downloadPandoc(ctx, url, tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", tmpPath, err)
	}

	if strings.HasSuffix(artifact, ".tar.gz") {
		return extractPandocTarGz(tmpPath, target)
	}
	return extractPandocZip(tmpPath, target)
}

func librarianCacheDir() (string, error) {
	if cache := os.Getenv("LIBRARIAN_CACHE"); cache != "" {
		return cache, nil
	}
	home, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache directory: %w", err)
	}
	return filepath.Join(home, "librarian"), nil
}

func pandocVersion(ctx context.Context, bin string) (string, error) {
	if _, err := os.Stat(bin); err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	if len(fields) < 2 || fields[0] != "pandoc" {
		return "", fmt.Errorf("unexpected pandoc --version output: %q", string(out))
	}
	return fields[1], nil
}

func pandocArtifact(version, goos, goarch string) (string, error) {
	switch goos + "/" + goarch {
	case "linux/amd64":
		return fmt.Sprintf("pandoc-%s-linux-amd64.tar.gz", version), nil
	case "linux/arm64":
		return fmt.Sprintf("pandoc-%s-linux-arm64.tar.gz", version), nil
	case "darwin/amd64":
		return fmt.Sprintf("pandoc-%s-x86_64-macOS.zip", version), nil
	case "darwin/arm64":
		return fmt.Sprintf("pandoc-%s-arm64-macOS.zip", version), nil
	}
	return "", fmt.Errorf("unsupported platform for pandoc install: %s/%s", goos, goarch)
}

func downloadPandoc(ctx context.Context, url string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: status %s", url, resp.Status)
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("reading response from %s: %w", url, err)
	}
	return nil
}

func extractPandocTarGz(archive, target string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("reading gzip %q: %w", archive, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("pandoc binary not found in %q", archive)
		}
		if err != nil {
			return fmt.Errorf("reading tar %q: %w", archive, err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if path.Base(hdr.Name) != "pandoc" {
			continue
		}
		return writePandoc(target, tr)
	}
}

func extractPandocZip(archive, target string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("reading zip %q: %w", archive, err)
	}
	defer zr.Close()
	for _, entry := range zr.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if path.Base(entry.Name) != "pandoc" {
			continue
		}
		rc, err := entry.Open()
		if err != nil {
			return fmt.Errorf("opening %q in %q: %w", entry.Name, archive, err)
		}
		err = writePandoc(target, rc)
		rc.Close()
		return err
	}
	return fmt.Errorf("pandoc binary not found in %q", archive)
}

func writePandoc(target string, src io.Reader) error {
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("creating %q: %w", target, err)
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		return fmt.Errorf("writing %q: %w", target, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", target, err)
	}
	if err := os.Chmod(target, 0755); err != nil {
		return fmt.Errorf("chmod %q: %w", target, err)
	}
	return nil
}
