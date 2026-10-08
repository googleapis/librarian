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
	"fmt"
	"html"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/yaml"
)

// excerptSpec is the YAML body of an ```excerpt block.
//
// Local excerpts name a file in this repository and a unique start fragment;
// the lines are read at build time. External excerpts (repo set) carry their
// code inline, as a snapshot taken at ref.
type excerptSpec struct {
	File      string `yaml:"file"`
	Start     string `yaml:"start"`
	End       string `yaml:"end"`
	Highlight string `yaml:"highlight"`
	Caption   string `yaml:"caption"`
	Repo      string `yaml:"repo"`
	Ref       string `yaml:"ref"`
	Code      string `yaml:"code"`
	Line      int    `yaml:"line"`
}

// maxExcerptLines bounds excerpts whose end is inferred from braces.
const maxExcerptLines = 60

func (s *site) renderExcerpt(body string) (string, error) {
	spec, err := yaml.Unmarshal[excerptSpec]([]byte(body))
	if err != nil {
		return "", fmt.Errorf("excerpt: %w", err)
	}
	if spec.File == "" {
		return "", fmt.Errorf("excerpt: file is required")
	}
	repo, ref := s.Repo, s.SHA
	var lines []string
	first := spec.Line
	if spec.Repo != "" {
		repo, ref = spec.Repo, spec.Ref
		if spec.Code == "" || ref == "" {
			return "", fmt.Errorf("excerpt %s: external excerpts need code and ref", spec.File)
		}
		lines = strings.Split(strings.TrimRight(spec.Code, "\n"), "\n")
		if first == 0 {
			first = 1
		}
	} else {
		if lines, first, err = extract(filepath.Join(s.rootDir, spec.File), spec); err != nil {
			return "", fmt.Errorf("excerpt %s: %w", spec.File, err)
		}
	}
	last := first + len(lines) - 1
	url := fmt.Sprintf("https://github.com/%s/blob/%s/%s#L%d-L%d", repo, ref, spec.File, first, last)
	var b strings.Builder
	b.WriteString(`<figure class="excerpt">`)
	fmt.Fprintf(&b, `<figcaption><code class="path">%s</code><span class="lines">L%d–L%d</span>`, html.EscapeString(spec.File), first, last)
	if spec.Repo != "" {
		fmt.Fprintf(&b, `<span class="repo">%s @ %s</span>`, html.EscapeString(repo), html.EscapeString(shortRef(ref)))
	}
	fmt.Fprintf(&b, `<a href="%s" target="_blank" rel="noopener">open on GitHub ↗</a></figcaption>`, url)
	b.WriteString(`<pre><code class="language-` + languageOf(spec.File) + `">`)
	for i, line := range dedent(lines) {
		cls := "line"
		if spec.Highlight != "" && strings.Contains(line, spec.Highlight) {
			cls += " hl"
		}
		fmt.Fprintf(&b, `<span class="%s"><span class="ln">%d</span><span class="lc">%s</span></span>`, cls, first+i, highlight(line, languageOf(spec.File)))
	}
	b.WriteString(`</code></pre>`)
	if spec.Caption != "" {
		b.WriteString(`<p class="caption">` + html.EscapeString(spec.Caption) + `</p>`)
	}
	b.WriteString(`</figure>`)
	return b.String(), nil
}

// extract reads the excerpt lines from path. It returns the lines and the
// 1-based number of the first line.
func extract(path string, spec *excerptSpec) ([]string, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	all := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start, err := findUnique(all, spec.Start)
	if err != nil {
		return nil, 0, err
	}
	end, err := findEnd(all, start, spec.End)
	if err != nil {
		return nil, 0, err
	}
	return all[start : end+1], start + 1, nil
}

func findUnique(lines []string, fragment string) (int, error) {
	if fragment == "" {
		return 0, fmt.Errorf("start is required")
	}
	idx := -1
	for i, l := range lines {
		if strings.Contains(l, fragment) {
			if idx >= 0 {
				return 0, fmt.Errorf("start %q matches more than one line (%d and %d)", fragment, idx+1, i+1)
			}
			idx = i
		}
	}
	if idx < 0 {
		return 0, fmt.Errorf("start %q not found", fragment)
	}
	return idx, nil
}

// findEnd resolves the end spec: "+N" for N lines, a fragment searched after
// the start line, or, when empty, the closing brace of the block opened on
// the start line (falling back to ten lines).
func findEnd(lines []string, start int, end string) (int, error) {
	switch {
	case strings.HasPrefix(end, "+"):
		n, err := strconv.Atoi(end[1:])
		if err != nil || n < 1 {
			return 0, fmt.Errorf("invalid end %q", end)
		}
		return min(start+n-1, len(lines)-1), nil
	case end != "":
		for i := start + 1; i < len(lines); i++ {
			if strings.Contains(lines[i], end) {
				return i, nil
			}
		}
		return 0, fmt.Errorf("end %q not found after line %d", end, start+1)
	}
	if strings.HasSuffix(strings.TrimSpace(lines[start]), "{") {
		indent := lines[start][:len(lines[start])-len(strings.TrimLeft(lines[start], " \t"))]
		for i := start + 1; i < len(lines) && i-start < maxExcerptLines; i++ {
			if strings.TrimRight(lines[i], " \t") == indent+"}" || strings.TrimRight(lines[i], " \t") == indent+"})" {
				return i, nil
			}
		}
	}
	return min(start+9, len(lines)-1), nil
}

func dedent(lines []string) []string {
	common := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if common < 0 || n < common {
			common = n
		}
	}
	if common <= 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if len(l) >= common {
			out[i] = l[common:]
		} else {
			out[i] = strings.TrimLeft(l, " \t")
		}
	}
	return out
}

func languageOf(file string) string {
	switch ext := path.Ext(file); ext {
	case ".go":
		return "go"
	case ".yaml", ".yml":
		return "yaml"
	case ".gotmpl", ".mustache":
		return "template"
	case ".md":
		return "markdown"
	case ".sh":
		return "shell"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}

func shortRef(ref string) string {
	if len(ref) == 40 {
		return ref[:8]
	}
	return ref
}

var (
	goKeywords = regexp.MustCompile(`\b(break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|import|interface|map|package|range|return|select|struct|switch|type|var|nil|true|false|error|string|int|bool|byte|any)\b`)
	goTokens   = regexp.MustCompile("//.*$|`[^`]*`|\"(?:[^\"\\\\]|\\\\.)*\"")
	yamlTokens = regexp.MustCompile(`#.*$|^(\s*-?\s*)([A-Za-z_][\w.-]*)(:)`)
)

// highlight returns HTML for one line with light syntax coloring. Every
// fragment is escaped, so the result is safe to embed.
func highlight(line, lang string) string {
	switch lang {
	case "go":
		return highlightGo(line)
	case "yaml":
		return highlightYAML(line)
	default:
		return html.EscapeString(line)
	}
}

func highlightGo(line string) string {
	var b strings.Builder
	last := 0
	for _, loc := range goTokens.FindAllStringIndex(line, -1) {
		b.WriteString(keywords(line[last:loc[0]]))
		tok := line[loc[0]:loc[1]]
		cls := "str"
		if strings.HasPrefix(tok, "//") {
			cls = "cmt"
		}
		fmt.Fprintf(&b, `<span class="%s">%s</span>`, cls, html.EscapeString(tok))
		last = loc[1]
	}
	b.WriteString(keywords(line[last:]))
	return b.String()
}

func keywords(text string) string {
	var b strings.Builder
	last := 0
	for _, loc := range goKeywords.FindAllStringIndex(text, -1) {
		b.WriteString(html.EscapeString(text[last:loc[0]]))
		fmt.Fprintf(&b, `<span class="kw">%s</span>`, text[loc[0]:loc[1]])
		last = loc[1]
	}
	b.WriteString(html.EscapeString(text[last:]))
	return b.String()
}

func highlightYAML(line string) string {
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return `<span class="cmt">` + html.EscapeString(line) + `</span>`
	}
	if i := strings.Index(line, " #"); i >= 0 {
		return html.EscapeString(line[:i]) + `<span class="cmt">` + html.EscapeString(line[i:]) + `</span>`
	}
	if m := yamlTokens.FindStringSubmatchIndex(line); m != nil && m[4] >= 0 {
		return html.EscapeString(line[:m[4]]) + `<span class="kw">` + html.EscapeString(line[m[4]:m[5]]) + `</span>` + html.EscapeString(line[m[5]:])
	}
	return html.EscapeString(line)
}
