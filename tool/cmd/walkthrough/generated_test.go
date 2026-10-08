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

// writeRepo creates a fake module root with one package per generated kind
// and returns it.
func writeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/mod\n\ngo 1.24\n",
		"internal/config/config.go": `// Package config is pure data.
package config

const (
	LanguageAll  = "all"
	LanguageGo   = "go"
	LanguageRust = "rust"
)
`,
		"internal/app/app.go": `// Package app dispatches to languages.
//
// A second paragraph that the synopsis must not include.
package app

import (
	"example.com/mod/internal/config"
	"github.com/urfave/cli/v3"
)

type runner struct{}

func (r *runner) generate(lang string) {
	switch lang {
	case config.LanguageGo, config.LanguageAll:
	}
}

var tidiers = map[string]int{config.LanguageRust: 1}

func unrelated() {}

func rootCommand() *cli.Command {
	return &cli.Command{
		Name:  "app",
		Usage: "do things",
		Commands: []*cli.Command{
			{Name: "get", Usage: "get a value"},
			&cli.Command{Name: "set", Usage: "set a value"},
			subCommand(),
		},
	}
}

func subCommand() *cli.Command {
	return &cli.Command{Name: "sub", Usage: "a | pipe"}
}
`,
		"internal/app/app_test.go":    "package app\n",
		"internal/app/testdata/x.go":  "package ignored\n",
		"cmd/tool/main.go":            "package main\n",
		".github/workflows/ci.yaml":   "name: CI\non:\n  push:\n  pull_request:\njobs:\n  test:\n    runs-on: x\n  lint:\n    runs-on: x\n",
		".github/workflows/call.yml":  "name: Callable\non: workflow_call\njobs:\n  go:\n    runs-on: x\n",
		".github/workflows/list.yaml": "name: Listed\non: [push, schedule]\njobs:\n  a:\n    runs-on: x\n",
		".github/workflows/README.md": "ignored\n",
	}
	for name, data := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestPackagesTable(t *testing.T) {
	root := writeRepo(t)
	got, err := packagesTable(root, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "| Package | Description |\n|---|---|\n" +
		"| `cmd/tool` | Command. |\n" +
		"| `internal/app` | Package app dispatches to languages. |\n" +
		"| `internal/config` | Package config is pure data. |\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("packagesTable mismatch (-want +got):\n%s", diff)
	}
	got, err = packagesTable(root, "internal/config")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "internal/app") || !strings.Contains(got, "internal/config") {
		t.Errorf("prefix filter not applied:\n%s", got)
	}
	if _, err := packagesTable(t.TempDir(), ""); err == nil {
		t.Error("want error without go.mod")
	}
}

func TestDispatchTable(t *testing.T) {
	root := writeRepo(t)
	got, err := dispatchTable(root, "internal/app")
	if err != nil {
		t.Fatal(err)
	}
	want := "| Declaration | File | go | rust |\n|---|---|---|---|\n" +
		"| `runner.generate` | `app.go` | ✓ | · |\n" +
		"| `tidiers` | `app.go` | · | ✓ |\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dispatchTable mismatch (-want +got):\n%s", diff)
	}
	if _, err := dispatchTable(root, "internal/config"); err == nil {
		t.Error("want error when nothing dispatches")
	}
	if _, err := dispatchTable(root, "missing"); err == nil {
		t.Error("want error for missing dir")
	}
}

func TestCommandsTable(t *testing.T) {
	root := writeRepo(t)
	got, err := commandsTable(root, "internal/app")
	if err != nil {
		t.Fatal(err)
	}
	want := "| Command | Usage | Defined in |\n|---|---|---|\n" +
		"| `app` | do things | `internal/app/app.go` |\n" +
		"| `app get` | get a value | `internal/app/app.go` |\n" +
		"| `app set` | set a value | `internal/app/app.go` |\n" +
		"| `sub` | a \\| pipe | `internal/app/app.go` |\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("commandsTable mismatch (-want +got):\n%s", diff)
	}
	if _, err := commandsTable(root, "internal/config"); err == nil {
		t.Error("want error when there are no commands")
	}
}

func TestWorkflowsTable(t *testing.T) {
	root := writeRepo(t)
	got, err := workflowsTable(root, ".github/workflows")
	if err != nil {
		t.Fatal(err)
	}
	want := "| Workflow | File | Triggers | Jobs |\n|---|---|---|---|\n" +
		"| Callable | `call.yml` | workflow_call | go |\n" +
		"| CI | `ci.yaml` | pull_request, push | lint, test |\n" +
		"| Listed | `list.yaml` | push, schedule | a |\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("workflowsTable mismatch (-want +got):\n%s", diff)
	}
	if _, err := workflowsTable(root, "internal"); err == nil {
		t.Error("want error when there are no workflows")
	}
	if err := os.WriteFile(filepath.Join(root, ".github/workflows/bad.yaml"), []byte(": : :\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := workflowsTable(root, ".github/workflows"); err == nil {
		t.Error("want error for malformed workflow")
	}
}

func TestRenderGenerated(t *testing.T) {
	root := writeRepo(t)
	s := &site{rootDir: root}
	got, err := s.renderGenerated("kind: workflows\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "\n<div class=\"generated\">\n\n| Workflow |") || !strings.HasSuffix(got, "|\n\n</div>\n") {
		t.Errorf("table must be wrapped and start on its own line, got %q", got)
	}
	for _, test := range []struct{ body, want string }{
		{"kind: nope\n", `unknown kind "nope"`},
		{": : :\n", "generated:"},
		{"kind: packages\nprefix: nothing\n", ""},
		{"kind: dispatch\ndir: missing\n", "generated dispatch"},
	} {
		_, err := s.renderGenerated(test.body)
		switch {
		case test.want == "" && err != nil:
			t.Errorf("renderGenerated(%q): unexpected error %v", test.body, err)
		case test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)):
			t.Errorf("renderGenerated(%q): want error containing %q, got %v", test.body, test.want, err)
		}
	}
}

func TestFindSymbol(t *testing.T) {
	root := writeRepo(t)
	path := filepath.Join(root, "internal/app/app.go")
	for _, test := range []struct {
		symbol     string
		start, end int
	}{
		{"runner", 11, 11},
		{"runner.generate", 13, 17},
		{"tidiers", 19, 19},
		{"unrelated", 21, 21},
		{"subCommand", 35, 37},
	} {
		start, end, err := findSymbol(path, test.symbol)
		if err != nil {
			t.Errorf("findSymbol(%q): %v", test.symbol, err)
			continue
		}
		if start+1 != test.start || end+1 != test.end {
			t.Errorf("findSymbol(%q) = L%d-L%d, want L%d-L%d", test.symbol, start+1, end+1, test.start, test.end)
		}
	}
	for _, symbol := range []string{"missing", "runner.missing", "other.generate", "generate"} {
		if _, _, err := findSymbol(path, symbol); err == nil {
			t.Errorf("findSymbol(%q): want error", symbol)
		}
	}
	if _, _, err := findSymbol(filepath.Join(root, "go.mod"), "x"); err == nil {
		t.Error("want parse error for non-Go file")
	}
	// Grouped constants resolve to the one spec.
	start, end, err := findSymbol(filepath.Join(root, "internal/config/config.go"), "LanguageRust")
	if err != nil || start != end || start != 6 {
		t.Errorf("LanguageRust = L%d-L%d, %v; want L7-L7", start+1, end+1, err)
	}
}

func TestSymbolExcerpt(t *testing.T) {
	root := writeRepo(t)
	path := filepath.Join(root, "internal/app/app.go")
	lines, first, err := extract(path, &excerptSpec{Symbol: "runner.generate"})
	if err != nil {
		t.Fatal(err)
	}
	if first != 13 || len(lines) != 5 || !strings.HasPrefix(lines[0], "func (r *runner) generate") {
		t.Errorf("got first=%d lines=%q", first, lines)
	}
	lines, _, err = extract(path, &excerptSpec{Symbol: "runner.generate", End: "+2"})
	if err != nil || len(lines) != 2 {
		t.Errorf("end override: got %d lines, %v", len(lines), err)
	}
	if _, _, err := extract(path, &excerptSpec{Symbol: "runner.generate", Start: "func"}); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("want mutually exclusive error, got %v", err)
	}
	if _, _, err := extract(path, &excerptSpec{Symbol: "nope"}); err == nil {
		t.Error("want error for unknown symbol")
	}
	if _, _, err := extract(path, &excerptSpec{Symbol: "runner.generate", End: "nope"}); err == nil {
		t.Error("want error for bad end")
	}
}

// TestReproducible builds the real walkthroughs twice and requires identical
// output: the site must be a pure function of the checked-out commit.
func TestReproducible(t *testing.T) {
	root := "../../.."
	src := filepath.Join(root, "doc", "walkthroughs")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no walkthrough sources: %v", err)
	}
	var outs []string
	for range 2 {
		out := t.TempDir()
		opts := options{src: src, out: out, root: root, sha: "main", repo: "googleapis/librarian"}
		if err := run(t.Context(), opts); err != nil {
			t.Fatal(err)
		}
		outs = append(outs, out)
	}
	entries, err := os.ReadDir(outs[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		a, err := os.ReadFile(filepath.Join(outs[0], e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(outs[1], e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Errorf("%s differs between two builds of the same tree", e.Name())
		}
	}
}
