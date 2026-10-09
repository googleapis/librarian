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

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeRealRepositoryAndReproducibility(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	opts := options{
		root:   repoRoot,
		sha:    "deadbeefcafebabe",
		gitRef: "main",
		repo:   "googleapis/librarian",
	}

	first, err := renderSiteHTML(opts)
	if err != nil {
		t.Fatalf("renderSiteHTML against real repository failed: %v", err)
	}
	second, err := renderSiteHTML(opts)
	if err != nil {
		t.Fatalf("second renderSiteHTML failed: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("renderSiteHTML output is not byte-identical across runs")
	}
	if strings.Contains(string(first), `"imports":null`) {
		t.Fatal("rendered HTML contains null imports slice instead of empty array []")
	}

	data, err := buildSiteData(opts)
	if err != nil {
		t.Fatalf("buildSiteData failed: %v", err)
	}
	if len(data.Pkgs) < 40 {
		t.Errorf("got %d packages, want at least 40", len(data.Pkgs))
	}
	if len(data.Langs) != 9 {
		t.Errorf("got %d languages, want 9", len(data.Langs))
	}
	if len(data.Guides) != 4 {
		t.Errorf("got %d default guides, want 4", len(data.Guides))
	}
	if len(data.Files) < 60 {
		t.Errorf("got %d embedded source entries, want at least 60", len(data.Files))
	}
	mainEntry, ok := data.Files["cmd/librarian/main.go"]
	if !ok || mainEntry.Content == "" {
		t.Errorf("expected embedded content for cmd/librarian/main.go, got %+v", mainEntry)
	}
}

func TestBrokenBuiltInPathFailsValidation(t *testing.T) {
	site := &SiteData{
		Steps: []TourStep{{
			Title: "Broken step",
			Look:  []string{"does/not/exist/anywhere.go"},
		}},
	}
	if _, err := collectAndValidateSources(".", site); err == nil {
		t.Fatal("expected error for non-existent referenced path, got nil")
	}
}

func TestCustomGuidesAndCLIModes(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	tmp := t.TempDir()
	extraDir := filepath.Join(tmp, "custom")
	if err := os.MkdirAll(extraDir, 0o755); err != nil {
		t.Fatal(err)
	}

	single := Guide{
		ID:       "rust-bump-deep-dive",
		Title:    "How Rust Bump Works",
		Subtitle: "Custom guide added by agent",
		Audience: "Custom walkthrough",
		Steps:    []GuideStep{{Title: "Inspect rust/bump.go", Body: "<p>Check bump logic.</p>"}},
	}
	b1, err := json.Marshal(single)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extraDir, "01-single.json"), b1, 0o644); err != nil {
		t.Fatal(err)
	}

	batch := []Guide{{
		ID:       "swift-publish-deep-dive",
		Title:    "How Swift Publish Splits Repos",
		Subtitle: "Batch guide added by agent",
		Audience: "Custom walkthrough",
		Steps:    []GuideStep{{Title: "Inspect swift/publish.go", Body: "<p>Check git split logic.</p>"}},
	}}
	b2, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extraDir, "02-batch.json"), b2, 0o644); err != nil {
		t.Fatal(err)
	}

	outHTML := filepath.Join(tmp, "walkthrough.html")
	opts := options{
		root:  repoRoot,
		extra: extraDir,
		out:   outHTML,
		sha:   "abcdef1234567890",
		repo:  "googleapis/librarian",
	}
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("run(-out html): %v", err)
	}
	html, err := os.ReadFile(outHTML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "How Rust Bump Works") || !strings.Contains(string(html), "How Swift Publish Splits Repos") {
		t.Errorf("generated HTML missing custom guide titles")
	}

	outZip := filepath.Join(tmp, "walkthrough.zip")
	opts.out = outZip
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("run(-out zip): %v", err)
	}
	zr, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatalf("opening output zip: %v", err)
	}
	defer zr.Close()
	if len(zr.File) != 1 || zr.File[0].Name != "walkthrough.html" {
		t.Errorf("unexpected zip entries: %v", zr.File)
	}

	dirOut := filepath.Join(tmp, "site_dir")
	opts.out = dirOut
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("run(-out dir): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirOut, "index.html")); err != nil {
		t.Errorf("expected index.html inside %s: %v", dirOut, err)
	}

	opts.check = true
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("run(-check): %v", err)
	}

	opts.check = false
	opts.json = true
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("run(-json): %v", err)
	}
}

func TestUnmappedLayerFailsBuild(t *testing.T) {
	tmp := t.TempDir()
	unknownDir := filepath.Join(tmp, "internal", "mysterypkg")
	if err := os.MkdirAll(unknownDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unknownDir, "mystery.go"), []byte("package mysterypkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := analyzeRepo(tmp); err == nil || !strings.Contains(err.Error(), "mysterypkg") {
		t.Fatalf("analyzeRepo succeeded or missed unmapped package error: %v", err)
	}
}
