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

//go:build integration

package cpp

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestAssuredWorkloads(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "clang-format")
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "tar")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cppRepo := os.Getenv("GOOGLE_CLOUD_CPP_DIR")
	if cppRepo == "" {
		candidates := []string{
			filepath.Clean(filepath.Join(wd, "../../../../../google-cloud-cpp/cpp-migration")),
			filepath.Clean(filepath.Join(wd, "../../../../../google-cloud-cpp")),
			filepath.Clean(filepath.Join(wd, "../../../../../google-cloud-cpp/main")),
			filepath.Clean(filepath.Join(wd, "../../../../google-cloud-cpp/cpp-migration")),
			filepath.Clean(filepath.Join(wd, "../../../../google-cloud-cpp")),
		}
		for _, c := range candidates {
			if fi, err := os.Stat(c); err == nil && fi.IsDir() {
				cppRepo = c
				break
			}
		}
	}
	if cppRepo == "" {
		t.Skip("google-cloud-cpp repo not found")
	}
	if fi, err := os.Stat(cppRepo); err != nil || !fi.IsDir() {
		t.Skip("google-cloud-cpp repo not found")
	}

	bzlBytes, err := os.ReadFile(filepath.Join(cppRepo, "bazel", "workspace0.bzl"))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`github\.com/googleapis/googleapis/archive/([0-9a-f]{40})\.tar\.gz`)
	matches := re.FindSubmatch(bzlBytes)
	if len(matches) < 2 {
		t.Fatal("googleapis commit SHA not found in workspace0.bzl")
	}
	commitSHA := string(matches[1])

	googleapisRepo := os.Getenv("GOOGLEAPIS_DIR")
	if googleapisRepo == "" {
		candidates := []string{
			filepath.Clean(filepath.Join(wd, "../../../../../googleapis/main")),
			filepath.Clean(filepath.Join(wd, "../../../../../googleapis")),
			filepath.Clean(filepath.Join(wd, "../../../../googleapis/main")),
			filepath.Clean(filepath.Join(wd, "../../../../googleapis")),
		}
		for _, c := range candidates {
			if fi, err := os.Stat(c); err == nil && fi.IsDir() {
				googleapisRepo = c
				break
			}
		}
	}
	if googleapisRepo == "" {
		t.Skip("googleapis repo not found")
	}
	if fi, err := os.Stat(googleapisRepo); err != nil || !fi.IsDir() {
		t.Skip("googleapis repo not found")
	}

	tmpGoogleapis := t.TempDir()
	gitCmd := exec.CommandContext(t.Context(), "git", "-C", googleapisRepo, "archive", commitSHA, "google")
	tarCmd := exec.CommandContext(t.Context(), "tar", "-x", "-C", tmpGoogleapis)
	pipe, err := gitCmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	tarCmd.Stdin = pipe
	var tarErr bytes.Buffer
	tarCmd.Stderr = &tarErr
	var gitErr bytes.Buffer
	gitCmd.Stderr = &gitErr

	if err := tarCmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := gitCmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := gitCmd.Wait(); err != nil {
		t.Fatalf("git archive failed: %v: %s", err, gitErr.String())
	}
	if err := tarCmd.Wait(); err != nil {
		t.Fatalf("tar extract failed: %v: %s", err, tarErr.String())
	}

	outDir := t.TempDir()
	library := &config.Library{
		Name:                "assuredworkloads",
		Output:              outDir,
		APIs:                []*config.API{{Path: "google/cloud/assuredworkloads/v1"}},
		SpecificationFormat: config.SpecProtobuf,
		Cpp: &config.CppLibrary{
			ProductPath:           "google/cloud/assuredworkloads/v1",
			ForwardingProductPath: "google/cloud/assuredworkloads",
			InitialCopyrightYear:  "2022",
			RetryableStatusCodes:  []string{"kUnavailable"},
		},
	}
	src := &sources.Sources{Googleapis: tmpGoogleapis}
	cfg := &config.Config{
		Tools: &config.Tools{
			ClangFormat: &config.ClangFormat{},
		},
	}

	if err := Generate(t.Context(), cfg, library, src); err != nil {
		t.Fatal(err)
	}

	clangFormatContent, err := os.ReadFile(filepath.Join(cppRepo, ".clang-format"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, ".clang-format"), clangFormatContent, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Format(t.Context(), cfg, library); err != nil {
		t.Fatal(err)
	}

	isIgnored := func(path string) bool {
		norm := filepath.ToSlash(path)
		base := filepath.Base(norm)
		if base == "CMakeLists.txt" || base == ".clang-format" ||
			base == ".repo-metadata.json" || base == "BUILD.bazel" || base == "README.md" {
			return true
		}
		for part := range strings.SplitSeq(norm, "/") {
			if part == "quickstart" || part == "samples" || part == "doc" {
				return true
			}
		}
		return false
	}

	matchedCount := 0
	fileCount := 0
	err = filepath.WalkDir(outDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(outDir, p)
		if err != nil {
			return err
		}
		if isIgnored(relPath) {
			return nil
		}
		fileCount++

		t.Run(relPath, func(t *testing.T) {
			got, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) == 0 {
				t.Errorf("generated file %s is empty", relPath)
			}

			refPath := filepath.Join(cppRepo, relPath)
			want, err := os.ReadFile(refPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(want) == 0 {
				t.Errorf("reference file %s is empty", relPath)
			}

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
				return
			}
			matchedCount++
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify all expected reference files exist in the generated output.
	refBase := filepath.Join(cppRepo, "google", "cloud", "assuredworkloads")
	err = filepath.WalkDir(refBase, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(cppRepo, p)
		if err != nil {
			return err
		}
		if isIgnored(relPath) {
			return nil
		}
		genPath := filepath.Join(outDir, relPath)
		if _, err := os.Stat(genPath); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("reference file %s was not generated", relPath)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// The 33 expected files comprise public headers and sources (1 client,
	// 1 connection, 1 connection_idempotency_policy, 1 mock_connection, 1 options),
	// their internal stubs, decorators, and sources, plus forwarding headers:
	// - 5 forwarding headers under google/cloud/assuredworkloads/
	// - 7 public v1 headers and sources under google/cloud/assuredworkloads/v1/
	// - 1 mock connection header under google/cloud/assuredworkloads/v1/mocks/
	// - 20 internal implementation files under google/cloud/assuredworkloads/v1/internal/
	const expectedFiles = 33
	if fileCount != expectedFiles {
		t.Errorf("expected %d generated files, got %d", expectedFiles, fileCount)
	}
	t.Logf("%d/%d generated files matched reference (%d mismatches)", matchedCount, expectedFiles, fileCount-matchedCount)
}
