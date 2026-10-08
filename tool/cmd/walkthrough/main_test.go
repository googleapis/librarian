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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const sampleGo = `package sample

// Greet says hello.
func Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return "hello " + name
}

func other() {
	_ = Greet("x")
}
`

const samplePage = `---
title: Sample page
summary: A summary.
audience: Everyone
order: 2
---

Intro with a flow:

` + "```flow" + `
lanes:
  - id: a
    label: Lane A
    kind: ci
  - id: b
    label: Lane B
nodes:
  - id: n1
    lane: a
    label: start here
  - id: n2
    lane: b
    label: then this
    step: 1
  - id: n3
    lane: a
    label: back up
edges:
  - n1 -> n2 | go
  - n2 -> n3
  - n3 -> n1 | loop
` + "```" + `

## First step
<!-- step runs="CI" code="librarian" -->

Text with ` + "`code`" + `.

` + "```excerpt" + `
file: sample.go
start: "func Greet("
highlight: return
caption: The greeter.
` + "```" + `

## Second step

` + "```mermaid" + `
graph TD
  A --> B
` + "```" + `

` + "```excerpt" + `
file: config.yaml
repo: example/repo
ref: 0123456789abcdef0123456789abcdef01234567
line: 3
code: |
  language: go
  # comment
` + "```" + `

### Sub heading inside step two

` + "```go" + `
## not a heading
` + "```" + `

` + "````markdown" + `
` + "```excerpt" + `
file: not-resolved.go
start: nope
` + "```" + `
` + "````" + `
`

// writeFixture creates a source directory with one page and a fake repo
// root containing sample.go.
func writeFixture(t *testing.T, page string) (src, root string) {
	t.Helper()
	dir := t.TempDir()
	src = filepath.Join(dir, "doc", "walkthroughs")
	root = dir
	if err := os.MkdirAll(filepath.Join(src, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		filepath.Join(src, "sample.md"):            page,
		filepath.Join(src, "README.md"):            "# No front matter\n",
		filepath.Join(src, "static", "extra.html"): "<p>static</p>",
		filepath.Join(root, "sample.go"):           sampleGo,
	} {
		if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return src, root
}

func TestRun(t *testing.T) {
	src, root := writeFixture(t, samplePage)
	out := filepath.Join(t.TempDir(), "site")
	opts := options{src: src, out: out, root: root, sha: strings.Repeat("a", 40), repo: "example/librarian"}
	if err := run(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "sample.html", "style.css", "app.js", "extra.html", ".nojekyll"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("missing output %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(out, "sample.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	for _, want := range []string{
		`<title>Sample page · Librarian walkthroughs</title>`,
		`<span class="badge runs"`, `CI</span>`,
		`<span class="badge code"`, `librarian</span>`,
		`href="https://github.com/example/librarian/blob/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/sample.go#L4-L9"`,
		`<span class="ln">4</span>`,
		`class="line hl"`,
		`<p class="caption">The greeter.</p>`,
		`<span class="kw">func</span> Greet(`,
		`<span class="str">&#34;hello &#34;</span>`,
		`<pre class="mermaid">graph TD`,
		`example/repo @ 01234567`,
		`href="https://github.com/example/repo/blob/0123456789abcdef0123456789abcdef01234567/config.yaml#L3-L4"`,
		`<span class="cmt"># comment</span>`,
		`<h3 id="sub-heading-inside-step-two">`,
		`## not a heading`,
		`file: not-resolved.go`,
		`<a href="#step-1" class="node-link">`,
		`<rect class="lane lane-ci"`,
		`>go</text>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("sample.html does not contain %q", want)
		}
	}
	if got := strings.Count(html, `<section class="step"`); got != 2 {
		t.Errorf("want 2 step sections, got %d", got)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="sample.html"`, `2 steps`, `href="extra.html"`, `aaaaaaaa</a>`} {
		if !strings.Contains(string(index), want) {
			t.Errorf("index.html does not contain %q", want)
		}
	}
}

func TestRunCheck(t *testing.T) {
	src, root := writeFixture(t, samplePage)
	out := filepath.Join(t.TempDir(), "site")
	opts := options{src: src, out: out, root: root, sha: "abc", repo: "example/librarian", check: true}
	if err := run(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("check mode must not write output, got err=%v", err)
	}
}

func TestRunDefaultSHA(t *testing.T) {
	// The real repository root is a git checkout, so the default SHA comes
	// from git rev-parse.
	src, _ := writeFixture(t, "---\ntitle: T\n---\n\nhello\n")
	opts := options{src: src, out: t.TempDir(), root: "../../..", repo: "example/librarian", check: true}
	if err := run(t.Context(), opts); err != nil {
		t.Fatal(err)
	}
}

func TestRunErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		page string
		want string
	}{
		{"missing start", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nfile: sample.go\nstart: nope\n```\n", `start "nope" not found`},
		{"ambiguous start", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nfile: sample.go\nstart: Greet(\n```\n", "matches more than one line"},
		{"missing end", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nfile: sample.go\nstart: \"func Greet(\"\nend: nope\n```\n", `end "nope" not found`},
		{"bad end", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nfile: sample.go\nstart: \"func Greet(\"\nend: +x\n```\n", `invalid end "+x"`},
		{"no file", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nstart: x\n```\n", "file is required"},
		{"no start", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nfile: sample.go\n```\n", "start is required"},
		{"missing file", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nfile: missing.go\nstart: x\n```\n", "missing.go"},
		{"external without ref", "---\ntitle: T\n---\n\n## S\n\n```excerpt\nfile: a\nrepo: x/y\ncode: z\n```\n", "need code and ref"},
		{"bad yaml", "---\ntitle: T\n---\n\n## S\n\n```excerpt\n: : :\n```\n", "excerpt:"},
		{"unterminated", "---\ntitle: T\n---\n\n## S\n\n```flow\nlanes: []\n", "unterminated flow block"},
		{"flow unknown lane", "---\ntitle: T\n---\n\n```flow\nlanes:\n  - id: a\nnodes:\n  - id: n\n    lane: zz\n```\n", `unknown lane "zz"`},
		{"flow bad edge", "---\ntitle: T\n---\n\n```flow\nlanes:\n  - id: a\nnodes:\n  - id: n\n    lane: a\nedges:\n  - n => n\n```\n", "must look like"},
		{"flow unknown node", "---\ntitle: T\n---\n\n```flow\nlanes:\n  - id: a\nnodes:\n  - id: n\n    lane: a\nedges:\n  - n -> q\n```\n", "unknown node"},
		{"flow empty", "---\ntitle: T\n---\n\n```flow\nlanes: []\n```\n", "lanes and nodes are required"},
		{"flow bad yaml", "---\ntitle: T\n---\n\n```flow\n: : :\n```\n", "flow:"},
		{"no title", "---\nsummary: s\n---\n\nbody\n", "title is required"},
		{"bad front matter", "---\n: : :\n---\n\nbody\n", "front matter"},
	} {
		t.Run(test.name, func(t *testing.T) {
			src, root := writeFixture(t, test.page)
			opts := options{src: src, out: t.TempDir(), root: root, sha: "abc", repo: "x/y"}
			err := run(t.Context(), opts)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("want error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestRunNoPages(t *testing.T) {
	src, root := writeFixture(t, "no front matter\n")
	if err := os.Remove(filepath.Join(src, "sample.md")); err != nil {
		t.Fatal(err)
	}
	err := run(t.Context(), options{src: src, out: t.TempDir(), root: root, sha: "abc"})
	if err == nil || !strings.Contains(err.Error(), "no pages") {
		t.Fatalf("want no pages error, got %v", err)
	}
	err = run(t.Context(), options{src: filepath.Join(src, "missing"), out: t.TempDir(), root: root, sha: "abc"})
	if err == nil {
		t.Fatal("want error for missing source directory")
	}
	err = run(t.Context(), options{src: src, out: t.TempDir(), root: filepath.Join(root, "nogit")})
	if err == nil || !strings.Contains(err.Error(), "determining commit") {
		t.Fatalf("want git error, got %v", err)
	}
}

func TestFindEnd(t *testing.T) {
	lines := strings.Split(sampleGo, "\n")
	for _, test := range []struct {
		name  string
		start int
		end   string
		want  int
	}{
		{"brace", 3, "", 8},
		{"plus", 3, "+2", 4},
		{"fragment", 3, "return", 7},
		{"fallback", 2, "", 11},
		{"plus past eof", 3, "+100", len(lines) - 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := findEnd(lines, test.start, test.end)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("findEnd(%d, %q) = %d, want %d", test.start, test.end, got, test.want)
			}
		})
	}
}

func TestDedent(t *testing.T) {
	got := dedent([]string{"\t\ta", "", "\t\t\tb", "\tc"})
	want := []string{"\ta", "", "\t\tb", "c"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dedent mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"a", " b"}, dedent([]string{"a", " b"})); diff != "" {
		t.Errorf("dedent with no common indent (-want +got):\n%s", diff)
	}
}

func TestHighlight(t *testing.T) {
	for _, test := range []struct {
		line, lang, want string
	}{
		{`x := "a<b" // c`, "go", `x := <span class="str">&#34;a&lt;b&#34;</span> <span class="cmt">// c</span>`},
		{"return `raw`", "go", `<span class="kw">return</span> <span class="str">` + "`raw`" + `</span>`},
		{"  # only comment", "yaml", `<span class="cmt">  # only comment</span>`},
		{"key: value # note", "yaml", `key: value<span class="cmt"> # note</span>`},
		{"- name: x", "yaml", `- <span class="kw">name</span>: x`},
		{"plain <text>", "rust", `plain &lt;text&gt;`},
	} {
		if got := highlight(test.line, test.lang); got != test.want {
			t.Errorf("highlight(%q, %s)\n got %s\nwant %s", test.line, test.lang, got, test.want)
		}
	}
}

func TestLanguageOf(t *testing.T) {
	for file, want := range map[string]string{
		"a/b.go": "go", "x.yaml": "yaml", "x.yml": "yaml", "t.gotmpl": "template",
		"t.mustache": "template", "r.md": "markdown", "s.sh": "shell", "f.rs": "rs", "Makefile": "",
	} {
		if got := languageOf(file); got != want {
			t.Errorf("languageOf(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestSplitSteps(t *testing.T) {
	intro, sections := splitSteps("intro\n```\n## fenced\n```\n## One\nbody one\n~~~\n## also fenced\n~~~\n## Two\n")
	if !strings.Contains(intro, "## fenced") {
		t.Errorf("heading inside fence must stay in intro, got %q", intro)
	}
	if len(sections) != 2 || sections[0].title != "One" || sections[1].title != "Two" {
		t.Fatalf("unexpected sections: %+v", sections)
	}
	if !strings.Contains(sections[0].body, "## also fenced") {
		t.Errorf("tilde fence not respected: %q", sections[0].body)
	}
}

func TestStepMeta(t *testing.T) {
	runs, code, rest, ok := stepMeta("\n<!-- step runs=\"CI\" code=\"lib\" -->\nbody\n")
	if !ok || runs != "CI" || code != "lib" || rest != "body\n" {
		t.Errorf("stepMeta = %q %q %q %v", runs, code, rest, ok)
	}
	if _, _, rest, ok := stepMeta("body\n"); ok || rest != "body\n" {
		t.Errorf("stepMeta without comment = %q %v", rest, ok)
	}
}

func TestWrap(t *testing.T) {
	got := wrap("protoc plus the language generator plugin", 16)
	want := []string{"protoc plus the", "language", "generator plugin"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrap mismatch (-want +got):\n%s", diff)
	}
	if got := wrap("", 10); got != nil {
		t.Errorf("wrap(\"\") = %v, want nil", got)
	}
}

func TestShortRef(t *testing.T) {
	if got := shortRef(strings.Repeat("b", 40)); got != "bbbbbbbb" {
		t.Errorf("shortRef = %q", got)
	}
	if got := shortRef("main"); got != "main" {
		t.Errorf("shortRef(main) = %q", got)
	}
}

func TestAssetNames(t *testing.T) {
	names, err := assetNames()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"app.js", "index.html.tmpl", "page.html.tmpl", "style.css"}
	if diff := cmp.Diff(want, names); diff != "" {
		t.Errorf("assets mismatch (-want +got):\n%s", diff)
	}
}

// TestBuildDocs builds the real walkthroughs against the repository, so a
// code change that removes an excerpt anchor fails here.
func TestBuildDocs(t *testing.T) {
	root := "../../.."
	src := filepath.Join(root, "doc", "walkthroughs")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no walkthrough sources: %v", err)
	}
	opts := options{src: src, out: t.TempDir(), root: root, sha: "main", repo: "googleapis/librarian"}
	if err := run(t.Context(), opts); err != nil {
		t.Fatalf("doc/walkthroughs no longer builds; update the excerpt anchors: %v", err)
	}
}

func TestNavTitle(t *testing.T) {
	for _, test := range []struct {
		fm   frontMatter
		want string
	}{
		{frontMatter{Title: "Long title"}, "Long title"},
		{frontMatter{Title: "Long title", Nav: "Short"}, "Short"},
	} {
		if got := test.fm.NavTitle(); got != test.want {
			t.Errorf("NavTitle(%+v) = %q, want %q", test.fm, got, test.want)
		}
	}
}
