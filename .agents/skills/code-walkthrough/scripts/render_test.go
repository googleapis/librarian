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
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// testFacts builds facts for a fake module rooted at a temp dir containing
// the given files, without running go list.
func testFacts(t *testing.T, files map[string]string, pkgs ...string) *facts {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f := &facts{root: root, module: "github.com/org/repo", sha: "0123456789abcdef", pkgs: map[string]*Pkg{}, docs: map[string]string{}}
	for _, id := range pkgs {
		f.pkgs[id] = &Pkg{ID: id, LOC: 10, Imports: []string{}, Files: []string{"x.go"}}
		f.docs[id] = "Package " + id + " does things."
		f.order = append(f.order, id)
	}
	return f
}

func baseSite() *Site {
	return &Site{
		Title:  "t",
		Focus:  "x", // silence completeness warnings unless a test wants them
		Layers: []Layer{{ID: "a", Title: "A"}},
		Pkgs:   []Pkg{{ID: "p", Layer: "a"}, {ID: "q", Layer: "a"}},
	}
}

func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old, oldN := warnOut, warnings
	warnOut, warnings = &buf, 0
	t.Cleanup(func() { warnOut, warnings = old, oldN })
	return &buf
}

func TestValidateErrors(t *testing.T) {
	f := testFacts(t, map[string]string{"ok.go": "package x\n", "dir/f.go": "package d\n"}, "p", "q")
	for _, tc := range []struct {
		name string
		mut  func(*Site)
		want string
	}{
		{"missing title", func(s *Site) { s.Title = " " }, "title is required"},
		{"bad repo", func(s *Site) { s.Repo = "not a repo" }, `repo "not a repo" must look like org/name`},
		{"bad gitRef", func(s *Site) { s.GitRef = "../x" }, "gitRef"},
		{"layer without title", func(s *Site) { s.Layers = append(s.Layers, Layer{ID: "b"}) }, "needs id and title"},
		{"duplicate layer", func(s *Site) { s.Layers = append(s.Layers, Layer{ID: "a", Title: "A2"}) }, `duplicate layer id "a"`},
		{"no layers", func(s *Site) { s.Layers = nil }, "at least one layer is required"},
		{"duplicate package", func(s *Site) { s.Pkgs = append(s.Pkgs, Pkg{ID: "p", Layer: "a"}) }, `duplicate package "p"`},
		{"unknown package", func(s *Site) { s.Pkgs = append(s.Pkgs, Pkg{ID: "zz", Layer: "a"}) }, `package "zz" is not in go list`},
		{"unknown layer", func(s *Site) { s.Pkgs[0].Layer = "nope" }, `uses unknown layer "nope"`},
		{"defaultPkg unknown", func(s *Site) { s.DefaultPkg = "zz" }, "defaultPkg references unknown package"},
		{"step without body", func(s *Site) { s.Steps = []Step{{Title: "t"}} }, "steps[0] needs t and body"},
		{"step pkg unknown", func(s *Site) { s.Steps = []Step{{Title: "t", Body: "b", Pkg: "zz"}} }, "steps[0].pkg references unknown package"},
		{"look missing", func(s *Site) { s.Steps = []Step{{Title: "t", Body: "b", Look: []string{"nope.go"}}} }, `cites "nope.go" which does not exist`},
		{"flow incomplete", func(s *Site) { s.Flows = []Flow{{ID: "f"}} }, "flows[0] needs id, title, cmd and at least one stage"},
		{"flow duplicate id", func(s *Site) {
			fl := Flow{ID: "f", Title: "t", Cmd: "c", File: "ok.go", Stages: []FlowStage{{Title: "s", Body: "b"}}}
			s.Flows = []Flow{fl, fl}
		}, `duplicate flow id "f"`},
		{"stage without body", func(s *Site) {
			s.Flows = []Flow{{ID: "f", Title: "t", Cmd: "c", File: "ok.go", Stages: []FlowStage{{Title: "s"}}}}
		}, "flows[0].stages[0] needs title and body"},
		{"guide incomplete", func(s *Site) { s.Guides = []Guide{{ID: "g", Title: "t"}} }, "guides[0] needs id, title and at least one step"},
		{"guide duplicate id", func(s *Site) {
			g := Guide{ID: "g", Title: "t", Steps: []GuideStep{{Title: "s", Body: "b"}}}
			s.Guides = []Guide{g, g}
		}, `duplicate guide id "g"`},
		{"guide step without body", func(s *Site) { s.Guides = []Guide{{ID: "g", Title: "t", Steps: []GuideStep{{Title: "s"}}}} }, "guides[0].steps[0] needs t and body"},
		{"finding without body", func(s *Site) { s.Findings = []Finding{{Title: "t", File: "ok.go"}} }, "findings[0] needs t and b"},
		{"finding file missing", func(s *Site) { s.Findings = []Finding{{Title: "t", Body: "b", File: "gone.go"}} }, `findings[0].f cites "gone.go" which does not exist`},
		{"langs without langCols", func(s *Site) { s.Langs = []Lang{{Name: "Go", Pkg: "p"}} }, "langs requires langCols"},
		{"langCol without key", func(s *Site) { s.LangCols = []LangCol{{Title: "x"}} }, "langCols[0] needs key and title"},
		{"lang without name", func(s *Site) { s.LangCols = []LangCol{{Key: "k", Title: "K"}}; s.Langs = []Lang{{Pkg: "p"}} }, "langs[0] needs n"},
		{"undeclared cell key", func(s *Site) {
			s.LangCols = []LangCol{{Key: "k", Title: "K"}}
			s.Langs = []Lang{{Name: "Go", Pkg: "p", Cells: map[string]string{"other": "x"}}}
		}, `cells has key "other" not declared in langCols`},
		{"hook row width", func(s *Site) {
			s.ContractCols = []string{"a", "b"}
			s.ContractHooks = []ContractRow{{Hook: "H", Cells: []string{"x"}}}
		}, "contractHooks[0] has 1 cells, want 2"},
		{"absolute path", func(s *Site) { s.Steps = []Step{{Title: "t", Body: "b", Look: []string{"/etc/passwd"}}} }, "paths must be relative"},
		{"parent traversal", func(s *Site) { s.Steps = []Step{{Title: "t", Body: "b", Look: []string{"../x"}}} }, "paths must be relative"},
		{"file with trailing slash", func(s *Site) { s.Steps = []Step{{Title: "t", Body: "b", Look: []string{"ok.go/"}}} }, "with a trailing slash but it is a file"},
		{"dir without trailing slash", func(s *Site) { s.Steps = []Step{{Title: "t", Body: "b", Look: []string{"dir"}}} }, "without a trailing slash"},
		{"empty data-src", func(s *Site) { s.Steps = []Step{{Title: "t", Body: `<a data-src="">x</a>`}} }, "data-src with an empty path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := baseSite()
			tc.mut(s)
			merge(s, f)
			errs := validate(s, f)
			if !containsSubstring(errs, tc.want) {
				t.Errorf("want an error containing %q, got %q", tc.want, errs)
			}
		})
	}
}

func TestValidateDataSrcVariants(t *testing.T) {
	f := testFacts(t, map[string]string{"ok.go": "package x\n"}, "p", "q")
	for _, body := range []string{
		`<a data-src="missing.go">x</a>`,
		`<a data-src='missing.go'>x</a>`,
		`<a data-src=missing.go>x</a>`,
		`<a DATA-SRC="missing.go">x</a>`,
		`<a data-src = "missing.go">x</a>`,
		`<a class="x" data-src="missing.go" title="y">x</a>`,
		`<span data-src="missing.go">x</span>`,
	} {
		s := baseSite()
		s.Steps = []Step{{Title: "t", Body: body}}
		merge(s, f)
		if errs := validate(s, f); !containsSubstring(errs, `cites "missing.go" which does not exist`) {
			t.Errorf("%s: not rejected, got %q", body, errs)
		}
	}
	// A valid citation in any form passes and is recorded for the page.
	s := baseSite()
	s.Steps = []Step{{Title: "t", Body: `<a data-src='ok.go'>x</a>`}}
	merge(s, f)
	if errs := validate(s, f); len(errs) != 0 {
		t.Fatalf("unexpected errors: %q", errs)
	}
	if len(s.Cited) != 1 || s.Cited[0] != "ok.go" {
		t.Errorf("cited = %q, want [ok.go]", s.Cited)
	}
}

func TestValidateNumberedTitles(t *testing.T) {
	f := testFacts(t, map[string]string{"ok.go": "package x\n"}, "p", "q")
	for _, title := range []string{"1. Add", "2) Build", "3: Wire", "a. First", "Step 4: Test", "step 5. Ship"} {
		s := baseSite()
		s.Steps = []Step{{Title: title, Body: "b"}}
		s.Guides = []Guide{{ID: "g", Title: "G", Steps: []GuideStep{{Title: title, Body: "b"}}}}
		s.Flows = []Flow{{ID: "f", Title: "F", Cmd: "c", File: "ok.go", Stages: []FlowStage{{Title: title, Body: "b"}}}}
		s.Findings = []Finding{{Title: title, Body: "b", File: "ok.go"}}
		merge(s, f)
		errs := validate(s, f)
		n := 0
		for _, e := range errs {
			if strings.Contains(e, "starts with a number") {
				n++
			}
		}
		if n != 4 {
			t.Errorf("%q: want 4 numbered-title errors (step, guide step, stage, finding), got %d: %q", title, n, errs)
		}
	}
	s := baseSite()
	s.Steps = []Step{{Title: "Version 2 of the API", Body: "b"}, {Title: "1,000 lines", Body: "b"}}
	merge(s, f)
	if errs := validate(s, f); len(errs) != 0 {
		t.Errorf("titles that merely contain digits were rejected: %q", errs)
	}
}

func TestMergeUnclassifiedAndWarnings(t *testing.T) {
	f := testFacts(t, nil, "p", "q", "r")
	buf := captureWarnings(t)
	s := &Site{Title: "t", Layers: []Layer{{ID: "a", Title: "A"}}, Pkgs: []Pkg{{ID: "p", Layer: "a"}}}
	merge(s, f)
	if len(s.Pkgs) != 3 {
		t.Fatalf("merge dropped packages: %+v", s.Pkgs)
	}
	for _, p := range s.Pkgs[1:] {
		if p.Layer != otherLayer || p.Desc == "" || p.LOC == 0 || p.Imports == nil || len(p.Files) == 0 {
			t.Errorf("omitted package not filled in: %+v", p)
		}
	}
	if s.Layers[len(s.Layers)-1].ID != otherLayer {
		t.Errorf("other layer not declared: %+v", s.Layers)
	}
	if errs := validate(s, f); len(errs) != 0 {
		t.Errorf("merged site should validate, got %q", errs)
	}
	out := buf.String()
	for _, want := range []string{"q has no layer", "r has no layer", "no flows", "no guides", "no findings"} {
		if !strings.Contains(out, want) {
			t.Errorf("warnings missing %q:\n%s", want, out)
		}
	}
	if warnings != 5 {
		t.Errorf("warnings = %d, want 5", warnings)
	}
	if s.GitRef != f.sha {
		t.Errorf("gitRef = %q, want the commit %q", s.GitRef, f.sha)
	}
	if s.ShortSHA != "01234567" || s.Repo != "org/repo" || s.DefaultPkg != "p" || s.TotalLOC != 30 {
		t.Errorf("computed fields: short=%q repo=%q default=%q loc=%d", s.ShortSHA, s.Repo, s.DefaultPkg, s.TotalLOC)
	}
}

func TestMergeFocusAndExplicitGitRef(t *testing.T) {
	f := testFacts(t, nil, "p")
	buf := captureWarnings(t)
	s := &Site{Title: "t", Focus: "p", GitRef: "release-1", Layers: []Layer{{ID: "a", Title: "A"}}, Pkgs: []Pkg{{ID: "p", Layer: "a"}}}
	merge(s, f)
	if buf.Len() != 0 {
		t.Errorf("focus set but warnings emitted:\n%s", buf.String())
	}
	if s.GitRef != "release-1" {
		t.Errorf("explicit gitRef overwritten: %q", s.GitRef)
	}
	f.sha = ""
	s = &Site{Title: "t", Focus: "p", Layers: []Layer{{ID: "a", Title: "A"}}, Pkgs: []Pkg{{ID: "p", Layer: "a"}}}
	merge(s, f)
	if s.GitRef != "main" || !strings.Contains(buf.String(), "links point at main") {
		t.Errorf("no sha: gitRef=%q warnings=%q", s.GitRef, buf.String())
	}
}

func TestPinScripts(t *testing.T) {
	in := []byte(`<meta content="default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'">` +
		`<script id="walkthrough-data" type="application/json">{"a":1}</script><script>console.log(1)</script>`)
	out := string(pinScripts(in))
	if strings.Contains(out, "script-src 'unsafe-inline'") {
		t.Fatalf("unsafe-inline left in place: %s", out)
	}
	// sha256("console.log(1)") base64.
	if !strings.Contains(out, "script-src 'sha256-") || strings.Count(out, "'sha256-") != 1 {
		t.Errorf("want exactly one script hash (data block skipped), got %s", out)
	}
	if !strings.Contains(out, "style-src 'unsafe-inline'") {
		t.Errorf("style-src must be untouched: %s", out)
	}
	plain := []byte("<p>no csp</p>")
	if !bytes.Equal(pinScripts(plain), plain) {
		t.Errorf("template without the token must be unchanged")
	}
}

// tinyModule writes a two-package Go module to a temp dir and returns its root.
func tinyModule(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":      "module github.com/org/tiny\n\ngo 1.22\n",
		"a/a.go":      "// Package a is the leaf.\npackage a\n\nfunc A() int { return 1 }\n",
		"b/b.go":      "// Package b uses a.\npackage b\n\nimport \"github.com/org/tiny/a\"\n\nfunc B() int { return a.A() }\n",
		"b/b_test.go": "package b\n",
	} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// writeSite writes an authored, warning-free walkthrough JSON to path.
func writeSite(t *testing.T, path, title, focus string) {
	t.Helper()
	s := Site{
		Title:  title,
		Focus:  focus,
		Layers: []Layer{{ID: "core", Title: "Core"}},
		Pkgs:   []Pkg{{ID: "a", Layer: "core"}, {ID: "b", Layer: "core"}},
		Steps:  []Step{{Title: "Leaf", Body: "<p>leaf</p>", Pkg: "a"}},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInitCheckRoundTrip(t *testing.T) {
	root := tinyModule(t)
	ctx := context.Background()
	skel := filepath.Join(root, outDir, "architecture.json")
	if err := run(ctx, "", skel, root, "", true, false, false); err != nil {
		t.Fatalf("-init: %v", err)
	}
	if err := run(ctx, "", skel, root, "", true, false, false); err == nil {
		t.Fatal("-init overwrote an existing file")
	}
	buf := captureWarnings(t)
	// The unedited skeleton renders (warnings only) but -check refuses it.
	if err := run(ctx, skel, "", root, "", false, true, false); err == nil || !strings.Contains(err.Error(), "warning(s)") {
		t.Fatalf("-check on the raw skeleton should fail on warnings, got %v", err)
	}
	if !strings.Contains(buf.String(), "a has no layer") || !strings.Contains(buf.String(), "no flows") {
		t.Errorf("expected layer and completeness warnings, got:\n%s", buf.String())
	}
	// Author it minimally: assign layers, set focus, and -check passes clean.
	b, err := os.ReadFile(skel)
	if err != nil {
		t.Fatal(err)
	}
	var s Site
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	s.Focus = "a"
	for i := range s.Pkgs {
		s.Pkgs[i].Layer = "core"
	}
	s.Steps = []Step{{Title: "Leaf first — ✓", Body: `<p>See <a data-src="a/a.go">a.go</a> → done.</p>`, Look: []string{"a/a.go"}, Pkg: "a"}}
	authored, _ := json.Marshal(s)
	if err := os.WriteFile(skel, authored, 0o644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	warnings = 0
	if err := run(ctx, skel, "", root, "", false, true, false); err != nil {
		t.Fatalf("-check on authored JSON: %v\n%s", err, buf.String())
	}
	// With no -out, the page lands beside its JSON in the output directory.
	if err := run(ctx, skel, "", root, "", false, false, false); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := filepath.Join(root, outDir, "architecture.html")
	html, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range html {
		if c > 0x7f {
			t.Fatalf("rendered page has non-ASCII byte 0x%02x at offset %d", c, i)
		}
	}
	for _, want := range []string{`"module":"github.com/org/tiny"`, `"imports":["a"]`, `"files":["b.go"]`, `"cited":["a/a.go"]`, `\u2014 \u2713`, "script-src 'sha256-"} {
		if !bytes.Contains(html, []byte(want)) {
			t.Errorf("rendered page lacks %s", want)
		}
	}
	if bytes.Contains(html, []byte("script-src 'unsafe-inline'")) {
		t.Error("rendered page still allows unsafe-inline scripts")
	}
	if bytes.Contains(html, []byte("location.replace")) {
		t.Error("rendered page uses location.replace fallback which breaks about:srcdoc iframes")
	}
}

func TestRunRejectsOutputOutsideOutDir(t *testing.T) {
	root := tinyModule(t)
	ctx := context.Background()
	in := filepath.Join(root, outDir, "architecture.json")
	writeSite(t, in, "Arch", "a")
	for _, tc := range []struct {
		name string
		init bool
		out  string
	}{
		{"init at module root", true, filepath.Join(root, "walkthrough.json")},
		{"render at module root", false, filepath.Join(root, "walkthrough.html")},
		{"render in a subdirectory", false, filepath.Join(root, outDir, "sub", "x.html")},
		{"render in another module", false, filepath.Join(t.TempDir(), outDir, "x.html")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := in
			if tc.init {
				src = ""
			}
			err := run(ctx, src, tc.out, root, "", tc.init, false, false)
			if err == nil || !strings.Contains(err.Error(), "is outside") {
				t.Fatalf("want an outside-directory error, got %v", err)
			}
			if _, err := os.Stat(tc.out); err == nil {
				t.Errorf("%s was written", tc.out)
			}
		})
	}
	if err := run(ctx, in, filepath.Join(root, outDir, "bad name.html"), root, "", false, false, false); err == nil || !strings.Contains(err.Error(), "must end in .html") {
		t.Errorf("want a bad-name error, got %v", err)
	}
}

func TestWalkthroughsLinkEachOther(t *testing.T) {
	root := tinyModule(t)
	ctx := context.Background()
	dir := filepath.Join(root, outDir)
	writeSite(t, filepath.Join(dir, "rust-bump.json"), "Rust bump", "rust bump")
	writeSite(t, filepath.Join(dir, "architecture.json"), "Architecture", "")
	writeSite(t, filepath.Join(dir, "unrendered.json"), "Never rendered", "x")
	captureWarnings(t) // architecture.json lacks flows, guides and findings
	if err := run(ctx, filepath.Join(dir, "rust-bump.json"), "", root, "", false, false, false); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "rust-bump.html"))
	if err != nil {
		t.Fatal(err)
	}
	if got := pageLinks(t, first); len(got) != 1 || !got[0].Current {
		t.Fatalf("a lone page should list only itself, got %+v", got)
	}
	if err := run(ctx, filepath.Join(dir, "architecture.json"), "", root, "", false, false, false); err != nil {
		t.Fatal(err)
	}
	want := []Walkthrough{
		{Title: "Architecture", Href: "architecture.html", Current: true},
		{Title: "Rust bump", Focus: "rust bump", Href: "rust-bump.html"},
	}
	second, err := os.ReadFile(filepath.Join(dir, "architecture.html"))
	if err != nil {
		t.Fatal(err)
	}
	if got := pageLinks(t, second); !slices.Equal(got, want) {
		t.Errorf("new page links = %+v, want %+v", got, want)
	}
	// The earlier page is relinked with itself as current.
	relinked, err := os.ReadFile(filepath.Join(dir, "rust-bump.html"))
	if err != nil {
		t.Fatal(err)
	}
	want[0].Current, want[1].Current = false, true
	if got := pageLinks(t, relinked); !slices.Equal(got, want) {
		t.Errorf("relinked page links = %+v, want %+v", got, want)
	}
	// Only the data block changed, so the pinned script hashes still match.
	strip := func(b []byte) []byte { return dataBlockRe.ReplaceAll(b, nil) }
	if !bytes.Equal(strip(first), strip(relinked)) {
		t.Error("relinking changed the page outside its data block")
	}
	for i, c := range relinked {
		if c > 0x7f {
			t.Fatalf("relinked page has non-ASCII byte 0x%02x at offset %d", c, i)
		}
	}
}

// pageLinks decodes the walkthrough list embedded in a rendered page.
func pageLinks(t *testing.T, html []byte) []Walkthrough {
	t.Helper()
	m := dataBlockRe.FindSubmatch(html)
	if m == nil {
		t.Fatal("page has no data block")
	}
	var s struct{ Walkthroughs []Walkthrough }
	if err := json.Unmarshal(m[2], &s); err != nil {
		t.Fatal(err)
	}
	return s.Walkthroughs
}

func containsSubstring(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
