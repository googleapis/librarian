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
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const maxEmbeddedFileLines = 80

var dataSrcRe = regexp.MustCompile(`data-src="([^"]+)"`)

func collectAndValidateSources(root string, data *SiteData) (map[string]SourceEntry, error) {
	pkgByID := make(map[string]bool, len(data.Pkgs))
	for _, p := range data.Pkgs {
		pkgByID[p.ID] = true
	}

	refs := make(map[string]bool)
	addRef := func(ref string) {
		if ref != "" {
			refs[ref] = true
		}
	}
	extractHTMLRefs := func(html string) {
		for _, m := range dataSrcRe.FindAllStringSubmatch(html, -1) {
			addRef(m[1])
		}
	}

	addRef("")
	for _, p := range data.Pkgs {
		if p.Path != "" {
			addRef(p.Path)
		} else {
			addRef(p.ID + "/")
		}
		extractHTMLRefs(p.Desc)
	}
	for _, s := range data.Steps {
		extractHTMLRefs(s.Body)
		for _, l := range s.Look {
			addRef(l)
		}
	}
	for _, f := range data.Flows {
		addRef(f.File)
		for _, st := range f.Stages {
			extractHTMLRefs(st.Where)
			extractHTMLRefs(st.Body)
		}
	}
	for _, l := range data.Langs {
		addRef(l.Pkg + "/")
		extractHTMLRefs(l.Detail)
	}
	for _, fn := range data.Findings {
		addRef(fn.File)
		extractHTMLRefs(fn.Body)
	}

	customRefs := make(map[string]bool)
	for _, g := range data.Guides {
		for _, s := range g.Steps {
			if g.Custom {
				for _, m := range dataSrcRe.FindAllStringSubmatch(s.Body, -1) {
					customRefs[m[1]] = true
				}
				for _, l := range s.Look {
					customRefs[l] = true
				}
				continue
			}
			extractHTMLRefs(s.Body)
			for _, l := range s.Look {
				addRef(l)
			}
		}
	}
	for k := range customRefs {
		refs[k] = true
	}

	out := make(map[string]SourceEntry, len(refs))
	keys := make([]string, 0, len(refs))
	for k := range refs {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, ref := range keys {
		entry, err := loadSourceEntry(root, ref, pkgByID)
		if err != nil {
			if customRefs[ref] && !hasBuiltInRef(data, ref) {
				continue
			}
			return nil, fmt.Errorf("referenced source path %q invalid in %s: %w", ref, root, err)
		}
		out[ref] = entry
	}
	return out, nil
}

func hasBuiltInRef(data *SiteData, target string) bool {
	for _, g := range data.Guides {
		if g.Custom {
			continue
		}
		for _, s := range g.Steps {
			if slices.Contains(s.Look, target) || strings.Contains(s.Body, `data-src="`+target+`"`) {
				return true
			}
		}
	}
	return false
}

func loadSourceEntry(root, ref string, pkgByID map[string]bool) (SourceEntry, error) {
	cleanRel := strings.TrimSuffix(ref, "/")
	fullPath := root
	if cleanRel != "" {
		fullPath = filepath.Join(root, filepath.FromSlash(cleanRel))
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return SourceEntry{}, err
	}
	pkg := resolveOwningPkg(cleanRel, pkgByID)
	if info.IsDir() {
		dirEntries, err := os.ReadDir(fullPath)
		if err != nil {
			return SourceEntry{}, err
		}
		var names []string
		totalLines := 0
		for _, e := range dirEntries {
			name := e.Name()
			if strings.HasPrefix(name, ".") && cleanRel != "" {
				continue
			}
			if e.IsDir() {
				names = append(names, name+"/")
				continue
			}
			if b, err := os.ReadFile(filepath.Join(fullPath, name)); err == nil {
				nLines := bytes.Count(b, []byte{'\n'})
				if !strings.HasSuffix(name, "_test.go") {
					totalLines += nLines
				}
				names = append(names, fmt.Sprintf("%s (%d lines)", name, nLines))
			} else {
				names = append(names, name)
			}
		}
		slices.Sort(names)
		return SourceEntry{
			IsDir:   true,
			Lines:   totalLines,
			Pkg:     pkg,
			Entries: names,
		}, nil
	}

	raw, err := os.ReadFile(fullPath)
	if err != nil {
		return SourceEntry{}, err
	}
	snippet, totalLines, trunc := truncateLines(string(raw), maxEmbeddedFileLines)
	return SourceEntry{
		IsDir:     false,
		Lines:     totalLines,
		Truncated: trunc,
		Pkg:       pkg,
		Content:   snippet,
	}, nil
}

func truncateLines(s string, maxLines int) (string, int, bool) {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	total := len(lines)
	if total <= maxLines {
		return strings.Join(lines, "\n"), total, false
	}
	return strings.Join(lines[:maxLines], "\n"), total, true
}

func resolveOwningPkg(cleanRel string, pkgByID map[string]bool) string {
	dir := cleanRel
	for dir != "" && dir != "." {
		id, _ := normalizePkgID(dir)
		if pkgByID[id] {
			return id
		}
		next := filepath.ToSlash(filepath.Dir(dir))
		if next == dir {
			break
		}
		dir = next
	}
	return ""
}
