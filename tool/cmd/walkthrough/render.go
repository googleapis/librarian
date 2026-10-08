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
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets
var assets embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"short": shortRef,
	"minutes": func(steps int) int {
		return max(2, steps*2)
	},
	"title": func(s string) string {
		return strings.ReplaceAll(strings.TrimSuffix(s, filepath.Ext(s)), "-", " ")
	},
}).ParseFS(assets, "assets/*.tmpl"))

type pageData struct {
	Site *site
	Page *page
}

func writeSite(s *site, out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := writeTemplate(filepath.Join(out, "index.html"), "index.html.tmpl", pageData{Site: s}); err != nil {
		return err
	}
	for _, p := range s.Pages {
		if err := writeTemplate(filepath.Join(out, p.Slug+".html"), "page.html.tmpl", pageData{Site: s, Page: p}); err != nil {
			return err
		}
	}
	for _, name := range []string{"style.css", "app.js"} {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, name), data, 0o644); err != nil {
			return err
		}
	}
	for _, name := range s.Static {
		data, err := os.ReadFile(filepath.Join(s.srcDir, "static", name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, name), data, 0o644); err != nil {
			return err
		}
	}
	// GitHub Pages must not run Jekyll on the output.
	return os.WriteFile(filepath.Join(out, ".nojekyll"), nil, 0o644)
}

func writeTemplate(path, name string, data pageData) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := templates.ExecuteTemplate(f, name, data); err != nil {
		return fmt.Errorf("rendering %s: %w", path, err)
	}
	return nil
}

// assetNames lists the embedded files; used by tests to make sure every
// asset referenced by the templates exists.
func assetNames() ([]string, error) {
	var names []string
	err := fs.WalkDir(assets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, filepath.Base(path))
		}
		return err
	})
	return names, err
}
