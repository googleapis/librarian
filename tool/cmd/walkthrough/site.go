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
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/yaml"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

// site is the fully resolved input for the HTML templates.
type site struct {
	Title    string
	Repo     string
	SHA      string
	Pages    []*page
	Static   []string
	srcDir   string
	rootDir  string
	excerpts int
}

// frontMatter is the YAML header of a walkthrough source file.
type frontMatter struct {
	Title    string `yaml:"title"`
	Summary  string `yaml:"summary"`
	Audience string `yaml:"audience"`
	Order    int    `yaml:"order"`
	// Nav is an optional short label for the header navigation.
	Nav string `yaml:"nav"`
}

// NavTitle returns the short navigation label, falling back to the title.
func (f frontMatter) NavTitle() string {
	if f.Nav != "" {
		return f.Nav
	}
	return f.Title
}

// page is one walkthrough: an intro followed by steps.
type page struct {
	frontMatter
	Slug   string
	Source string
	Intro  template.HTML
	Steps  []*step
}

// step is one level-two section of a page.
type step struct {
	Index int
	Title string
	ID    string
	Runs  string
	Code  string
	Body  template.HTML
}

var (
	stepMetaRE = regexp.MustCompile(`^<!--\s*step\s+(.*?)\s*-->\s*$`)
	attrRE     = regexp.MustCompile(`(\w+)="([^"]*)"`)
	fenceRE    = regexp.MustCompile("^(```+|~~~+)\\s*(\\S*)")
)

func loadSite(opts options) (*site, error) {
	s := &site{Title: "Librarian walkthroughs", Repo: opts.repo, SHA: opts.sha, srcDir: opts.src, rootDir: opts.root}
	entries, err := os.ReadDir(opts.src)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p, err := s.loadPage(filepath.Join(opts.src, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if p != nil {
			s.Pages = append(s.Pages, p)
		}
	}
	if len(s.Pages) == 0 {
		return nil, fmt.Errorf("no pages with front matter found in %s", opts.src)
	}
	sort.SliceStable(s.Pages, func(i, j int) bool { return s.Pages[i].Order < s.Pages[j].Order })
	static, err := os.ReadDir(filepath.Join(opts.src, "static"))
	if err == nil {
		for _, e := range static {
			if !e.IsDir() {
				s.Static = append(s.Static, e.Name())
			}
		}
	}
	return s, nil
}

// loadPage parses one markdown file. Files without front matter (such as
// README.md) are skipped and return nil.
func (s *site) loadPage(path string) (*page, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fm, body, ok := splitFrontMatter(string(data))
	if !ok {
		return nil, nil
	}
	meta, err := yaml.Unmarshal[frontMatter]([]byte(fm))
	if err != nil {
		return nil, fmt.Errorf("front matter: %w", err)
	}
	if meta.Title == "" {
		return nil, fmt.Errorf("front matter: title is required")
	}
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	p := &page{frontMatter: *meta, Slug: base, Source: filepath.ToSlash(filepath.Join(filepath.Base(s.srcDir), filepath.Base(path)))}
	intro, sections := splitSteps(body)
	if p.Intro, err = s.render(intro); err != nil {
		return nil, err
	}
	for i, sec := range sections {
		st := &step{Index: i + 1, Title: sec.title, ID: fmt.Sprintf("step-%d", i+1)}
		content := sec.body
		if runs, code, rest, ok := stepMeta(content); ok {
			st.Runs, st.Code, content = runs, code, rest
		}
		if st.Body, err = s.render(content); err != nil {
			return nil, fmt.Errorf("step %q: %w", sec.title, err)
		}
		p.Steps = append(p.Steps, st)
	}
	return p, nil
}

func splitFrontMatter(text string) (fm, body string, ok bool) {
	if !strings.HasPrefix(text, "---\n") {
		return "", text, false
	}
	fm, body, ok = strings.Cut(text[4:], "\n---\n")
	if !ok {
		return "", text, false
	}
	return fm, body, true
}

type section struct {
	title string
	body  string
}

// splitSteps splits markdown on level-two headings, ignoring headings inside
// fenced code blocks.
func splitSteps(body string) (intro string, sections []section) {
	var cur *section
	var buf strings.Builder
	flush := func() {
		if cur == nil {
			intro = buf.String()
		} else {
			cur.body = buf.String()
			sections = append(sections, *cur)
		}
		buf.Reset()
	}
	var fence string
	for line := range strings.SplitSeq(body, "\n") {
		if m := fenceRE.FindStringSubmatch(line); m != nil {
			switch {
			case fence == "":
				fence = m[1]
			case strings.HasPrefix(line, fence):
				fence = ""
			}
		}
		if fence == "" && strings.HasPrefix(line, "## ") {
			flush()
			cur = &section{title: strings.TrimSpace(line[3:])}
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	flush()
	return intro, sections
}

// stepMeta extracts the optional `<!-- step runs="..." code="..." -->` line
// that may follow a step heading.
func stepMeta(content string) (runs, code, rest string, ok bool) {
	trimmed := strings.TrimLeft(content, "\n")
	first, after, _ := strings.Cut(trimmed, "\n")
	m := stepMetaRE.FindStringSubmatch(first)
	if m == nil {
		return "", "", content, false
	}
	for _, kv := range attrRE.FindAllStringSubmatch(m[1], -1) {
		switch kv[1] {
		case "runs":
			runs = kv[2]
		case "code":
			code = kv[2]
		}
	}
	return runs, code, after, true
}

var markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
)

// render converts markdown to HTML after replacing the special fenced blocks
// (excerpt, flow, mermaid) with pre-rendered HTML.
func (s *site) render(md string) (template.HTML, error) {
	pre, err := s.expandBlocks(md)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := markdown.Convert([]byte(pre), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func (s *site) expandBlocks(md string) (string, error) {
	var out strings.Builder
	var fence, kind string
	var block []string
	for line := range strings.SplitSeq(md, "\n") {
		switch {
		case fence != "" && strings.HasPrefix(line, fence):
			if kind != "" {
				rendered, err := s.renderBlock(kind, strings.Join(block, "\n"))
				if err != nil {
					return "", err
				}
				out.WriteString(rendered)
				out.WriteString("\n")
			} else {
				out.WriteString(line)
				out.WriteString("\n")
			}
			fence, kind, block = "", "", nil
			continue
		case fence != "" && kind != "":
			block = append(block, line)
			continue
		case fence == "":
			if m := fenceRE.FindStringSubmatch(line); m != nil {
				fence = m[1]
				switch m[2] {
				case "excerpt", "flow", "mermaid":
					kind = m[2]
					continue
				}
			}
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
	if kind != "" {
		return "", fmt.Errorf("unterminated %s block", kind)
	}
	return out.String(), nil
}

func (s *site) renderBlock(kind, body string) (string, error) {
	switch kind {
	case "excerpt":
		s.excerpts++
		return s.renderExcerpt(body)
	case "flow":
		return renderFlow(body)
	case "mermaid":
		return `<div class="mermaid-wrap"><pre class="mermaid">` + html.EscapeString(body) + `</pre></div>`, nil
	}
	return "", fmt.Errorf("unknown block kind %q", kind)
}

func (s *site) stepCount() int {
	n := 0
	for _, p := range s.Pages {
		n += len(p.Steps)
	}
	return n
}

func (s *site) excerptCount() int { return s.excerpts }
