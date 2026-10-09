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

func TestSecurityPathTraversalAndScriptBreakout(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	tmp := t.TempDir()
	extraDir := filepath.Join(tmp, "custom")
	if err := os.MkdirAll(extraDir, 0o755); err != nil {
		t.Fatal(err)
	}

	malicious := Guide{
		ID:       "sec-traversal-test",
		Title:    "Traversal & Script Escape Test </SCRIPT><script>alert(1)</script>",
		Subtitle: "Testing path confinement",
		Audience: "Security test",
		Steps: []GuideStep{{
			Title: "Attempt escape",
			Body:  `<p>Escape attempt <a class="src" data-src="../../../../../../etc/passwd">passwd</a></p>`,
			Look:  []string{"../../../../../../etc/passwd"},
		}},
	}
	raw, err := json.Marshal(malicious)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extraDir, "sec.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	opts := options{
		root:   repoRoot,
		extra:  extraDir,
		sha:    "abcdef1234567890",
		gitRef: "main",
		repo:   "googleapis/librarian",
	}
	site, err := buildSiteData(opts)
	if err != nil {
		t.Fatalf("buildSiteData failed: %v", err)
	}
	if _, leaked := site.Files["../../../../../../etc/passwd"]; leaked {
		t.Fatal("path traversal outside repo root leaked into SiteData.Files")
	}
	for _, entry := range site.Files[""].Entries {
		if strings.HasPrefix(entry, ".") {
			t.Errorf("root directory listing exposed dotfile entry %q", entry)
		}
	}

	htmlBytes, err := renderHTML(site)
	if err != nil {
		t.Fatalf("renderHTML failed: %v", err)
	}
	if strings.Contains(string(htmlBytes), "</SCRIPT>") {
		t.Fatal("renderHTML left raw uppercase </SCRIPT> sequence unescaped in inline JSON")
	}

	// Custom guide referencing a missing built-in path must NOT silence built-in validation errors.
	brokenSite := &SiteData{
		Steps: []TourStep{{Title: "Broken built-in", Look: []string{"nonexistent/builtin.go"}}},
		Guides: []Guide{{
			ID:     "custom",
			Custom: true,
			Steps:  []GuideStep{{Title: "Custom", Look: []string{"nonexistent/builtin.go"}}},
		}},
	}
	if _, err := collectAndValidateSources(repoRoot, brokenSite); err == nil {
		t.Fatal("expected broken built-in ref to fail validation even when also cited by custom guide")
	}
}

func TestRootOutputIdempotencyAndNoGitFallback(t *testing.T) {
	tmp := t.TempDir()
	for _, dir := range []string{
		"cmd/librarian",
		"tool/cmd/walkthrough",
		"tool/cmd/coverage",
		"internal/librarian/dart", "internal/librarian/golang", "internal/librarian/java",
		"internal/librarian/nodejs", "internal/librarian/php", "internal/librarian/python",
		"internal/librarian/ruby", "internal/librarian/rust", "internal/librarian/swift",
		"internal/sidekick/rust_prost", "internal/sidekick/rust", "internal/sidekick/swift",
		"internal/sidekick/dart", "internal/sidekick/codec_sample", "internal/sidekick/language",
		"internal/sidekick/parser/discovery", "internal/sidekick/parser/httprule", "internal/sidekick/parser/svcconfig",
		"internal/sidekick/protobuf", "internal/sidekick/api/apitest",
		"internal/tool/protoc", "internal/tool/maven", "internal/tool/pip",
		"internal/tool/pnpm", "internal/tool/gem", "internal/tool/composer",
		"internal/config", "internal/serviceconfig", "internal/sources", "internal/repometadata",
		"internal/postprocessing", "internal/snippetmetadata", "internal/semver",
		"internal/proto", "internal/license", "internal/command", "internal/cache",
		"internal/fetch", "internal/filesystem", "internal/git", "internal/yaml",
		"internal/testhelper", "internal/sample",
	} {
		if err := os.MkdirAll(filepath.Join(tmp, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		pkgName := filepath.Base(dir)
		if err := os.WriteFile(filepath.Join(tmp, dir, pkgName+".go"), []byte("package "+pkgName+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{
		"cmd/librarian/main.go", "internal/librarian/librarian.go", "internal/librarian/config.go",
		"internal/config/config.go", "internal/config/language.go", "doc/config-schema.md",
		"internal/librarian/library.go", "internal/librarian/tidy.go", "internal/librarian/source.go",
		"internal/fetch/fetch.go", "internal/sources/sources.go", "internal/librarian/update.go",
		"internal/librarian/generate.go", "internal/librarian/clean.go", "internal/librarian/docindex.go",
		"internal/librarian/golang/generate.go", "internal/librarian/rust/generate.go",
		"internal/tool/protoc/protoc.go", "internal/librarian/install.go",
		"internal/sidekick/parser/parser.go", "internal/sidekick/parser/protobuf.go",
		"internal/sidekick/api/api.go", "internal/serviceconfig/sdk.yaml",
		"internal/sidekick/language/gotemplate.go", "internal/sidekick/language/walk_templates_dir.go",
		"internal/sidekick/swift/codec.go", "internal/postprocessing/fileops.go",
		"internal/repometadata/repometadata.go", "internal/librarian/python/generate.go",
		"internal/librarian/java/postgenerate.go", "internal/librarian/bump.go",
		"internal/librarian/publish.go", "internal/librarian/tag.go", "internal/librarian/release_please.go",
		"all_test.go", "dependency_test.go", ".github/workflows/ci.yaml", "action.yaml",
		"CONTRIBUTING.md", ".agents/skills/new-sidekick/SKILL.md", ".agents/skills/walkthrough/SKILL.md",
		"internal/librarian/add.go", "internal/sidekick/api/model.go",
		"internal/sidekick/swift/templates/pkg.gotmpl", "internal/sidekick/rust/templates/pkg.gotmpl",
		"internal/sidekick/dart/templates/pkg.gotmpl", "internal/sidekick/rust/annotate_model.go",
		"internal/sidekick/dart/annotate.go", "internal/sidekick/swift/generate_service_swift_test.go",
		".github/workflows/dart.yaml", ".github/workflows/go.yaml", ".github/workflows/java.yaml",
		".github/workflows/nodejs.yaml", ".github/workflows/php.yaml", ".github/workflows/python.yaml",
		".github/workflows/ruby.yaml", ".github/workflows/sidekick.yaml",
	} {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		content := []byte("# stub\n")
		if strings.HasSuffix(f, ".go") {
			pkg := filepath.Base(filepath.Dir(f))
			if pkg == "." {
				pkg = "librarian_test"
			}
			content = []byte("package " + pkg + "\n")
		}
		if err := os.WriteFile(filepath.Join(tmp, f), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	outInRoot := filepath.Join(tmp, "walkthrough.html")
	opts := options{
		root: tmp,
		out:  outInRoot,
		repo: "googleapis/librarian",
	}
	// Run 1 without git (.sha == "" falls back cleanly to "HEAD") writing walkthrough.html inside root:
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("first run without .git failed: %v", err)
	}
	firstHTML, err := os.ReadFile(outInRoot)
	if err != nil {
		t.Fatal(err)
	}
	// Run 2 writing walkthrough.html inside root must be byte-for-byte identical:
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("second run without .git failed: %v", err)
	}
	secondHTML, err := os.ReadFile(outInRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstHTML, secondHTML) {
		t.Fatal("consecutive runs writing walkthrough.html inside repo root were not byte-identical")
	}
}
