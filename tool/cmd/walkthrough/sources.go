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
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	maxEmbeddedFileLines = 80
	maxEmbeddedFileBytes = 2 << 20 // 2 MiB guard against large binaries
)

var dataSrcRe = regexp.MustCompile(`data-src="([^"]+)"`)

func collectAndValidateSources(root string, data *SiteData) (map[string]SourceEntry, error) {
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("opening repository root %s: %w", root, err)
	}
	defer rootFS.Close()

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

	refs[""] = true
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
		if l.CI != "" {
			addRef(".github/workflows/" + l.CI)
		}
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
	builtInRefs := maps.Clone(refs)
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
		entry, err := loadSourceEntry(rootFS, ref, pkgByID)
		if err != nil {
			if !builtInRefs[ref] {
				fmt.Fprintf(os.Stderr, "warning: ignoring invalid custom guide source path %q: %v\n", ref, err)
				continue
			}
			return nil, fmt.Errorf("referenced source path %q invalid in %s: %w", ref, root, err)
		}
		out[ref] = entry
	}
	return out, nil
}

func isGeneratedWalkthroughArtifact(name string, cleanRel string, isDir bool) bool {
	if name == "walkthrough.html" || name == "walkthrough.zip" {
		return true
	}
	if cleanRel == "" {
		return name == "_site" || (name == "walkthrough" && !isDir)
	}
	return false
}

func loadSourceEntry(rootFS *os.Root, ref string, pkgByID map[string]bool) (SourceEntry, error) {
	cleanRel := strings.TrimSuffix(ref, "/")
	fsPath := "."
	if cleanRel != "" {
		fsPath = filepath.FromSlash(cleanRel)
	}
	info, err := rootFS.Stat(fsPath)
	if err != nil {
		return SourceEntry{}, err
	}
	pkg := resolveOwningPkg(cleanRel, pkgByID)
	if info.IsDir() {
		dirFile, err := rootFS.Open(fsPath)
		if err != nil {
			return SourceEntry{}, err
		}
		defer dirFile.Close()
		dirEntries, err := dirFile.ReadDir(-1)
		if err != nil {
			return SourceEntry{}, err
		}
		var names []string
		totalLines := 0
		for _, e := range dirEntries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || isGeneratedWalkthroughArtifact(name, cleanRel, e.IsDir()) {
				continue
			}
			if e.IsDir() {
				names = append(names, name+"/")
				continue
			}
			if !e.Type().IsRegular() {
				continue
			}
			entryInfo, err := e.Info()
			if err != nil || entryInfo.Size() > maxEmbeddedFileBytes {
				names = append(names, name)
				continue
			}
			childPath := name
			if fsPath != "." {
				childPath = filepath.Join(fsPath, name)
			}
			if b, err := rootFS.ReadFile(childPath); err == nil && bytes.IndexByte(b, 0) < 0 {
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

	if !info.Mode().IsRegular() {
		return SourceEntry{}, fmt.Errorf("non-regular file mode %s", info.Mode())
	}
	if info.Size() > maxEmbeddedFileBytes {
		return SourceEntry{
			IsDir:     false,
			Truncated: true,
			Pkg:       pkg,
		}, nil
	}
	raw, err := rootFS.ReadFile(fsPath)
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
