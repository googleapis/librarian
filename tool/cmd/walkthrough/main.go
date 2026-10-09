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

// walkthrough builds a self-contained interactive architecture map and
// step-by-step developer guide HTML artifact (or .zip package) directly from
// the checked-out repository, requiring no background HTTP server.
//
// Usage:
//
//	walkthrough [-out walkthrough.html] [-root .] [-extra .agents/skills/walkthrough/custom]
//	walkthrough -out walkthrough.zip
//	walkthrough -check
//	walkthrough -json
package main

import (
	"archive/zip"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultExtraDir = ".agents/skills/walkthrough/custom"

//go:embed template.html
var pageTemplate string

type options struct {
	out    string
	root   string
	extra  string
	sha    string
	gitRef string
	repo   string
	check  bool
	json   bool
}

func main() {
	var opts options
	flag.StringVar(&opts.out, "out", "walkthrough.html", "output path (.html self-contained file, .zip archive, or directory)")
	flag.StringVar(&opts.root, "root", ".", "repository root inspected to derive packages, LOC, imports and hooks")
	flag.StringVar(&opts.extra, "extra", defaultExtraDir, "optional directory containing custom JSON guide files")
	flag.StringVar(&opts.sha, "sha", "", "git commit SHA for source links (default: git rev-parse HEAD)")
	flag.StringVar(&opts.gitRef, "git-ref", "main", "upstream GitHub branch/ref for external GitHub fallback links")
	flag.StringVar(&opts.repo, "repo", "googleapis/librarian", "GitHub org/repo used in source links")
	flag.BoolVar(&opts.check, "check", false, "verify repository analysis and HTML generation without writing files")
	flag.BoolVar(&opts.json, "json", false, "print computed SiteData JSON to stdout and exit")
	flag.Parse()

	if err := run(context.Background(), opts); err != nil {
		log.Fatal(err)
	}
}

func resolveRepoRoot(root string) string {
	if root != "." && root != "" {
		return root
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}

func run(ctx context.Context, opts options) error {
	opts.root = resolveRepoRoot(opts.root)
	if opts.extra == defaultExtraDir && !filepath.IsAbs(opts.extra) {
		opts.extra = filepath.Join(opts.root, opts.extra)
	}
	if opts.sha == "" {
		if out, err := exec.CommandContext(ctx, "git", "-C", opts.root, "rev-parse", "HEAD").Output(); err == nil {
			opts.sha = strings.TrimSpace(string(out))
		} else {
			opts.sha = "HEAD"
		}
	}
	if opts.gitRef == "" {
		opts.gitRef = "main"
	}
	data, err := buildSiteData(opts)
	if err != nil {
		return err
	}
	if opts.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	if opts.check {
		fmt.Printf("verified %d packages, %d tour steps, %d guides, and %d source links at %s\n",
			len(data.Pkgs), len(data.Steps), len(data.Guides), len(data.Files), data.ShortSHA)
		return nil
	}
	htmlBytes, err := renderHTML(data)
	if err != nil {
		return err
	}
	if strings.HasSuffix(opts.out, ".zip") {
		if err := writeZip(opts.out, htmlBytes); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d HTML bytes zipped)\n", opts.out, len(htmlBytes))
		return nil
	}
	outFile := opts.out
	if !strings.HasSuffix(outFile, ".html") {
		outFile = filepath.Join(outFile, "index.html")
	}
	if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outFile, htmlBytes, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes)\n", outFile, len(htmlBytes))
	return nil
}

func writeZip(zipPath string, htmlBytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("walkthrough.html")
	if err != nil {
		return err
	}
	if _, err := w.Write(htmlBytes); err != nil {
		return err
	}
	return zw.Close()
}

func buildSiteData(opts options) (*SiteData, error) {
	stats, err := analyzeRepo(opts.root)
	if err != nil {
		return nil, err
	}
	guides, err := buildGuides(stats, opts.extra)
	if err != nil {
		return nil, err
	}
	short := opts.sha
	if len(short) > 8 {
		short = short[:8]
	}
	ref := opts.gitRef
	if ref == "" {
		ref = "main"
	}
	site := &SiteData{
		SHA:              opts.sha,
		ShortSHA:         short,
		GitRef:           ref,
		Repo:             opts.repo,
		GoVersion:        stats.goVersion,
		LibrarianVersion: stats.librarianVersion,
		TotalLOC:         stats.totalLOC,
		MaxEmbeddedLines: maxEmbeddedFileLines,
		Layers:           defaultLayers,
		Pkgs:             stats.pkgs,
		Steps:            buildSteps(stats),
		Flows:            buildFlows(),
		Langs:            stats.langs,
		ContractCols:     stats.contractCols,
		ContractHooks:    stats.contractHooks,
		Findings:         buildFindings(stats),
		Guides:           guides,
	}
	files, err := collectAndValidateSources(opts.root, site)
	if err != nil {
		return nil, err
	}
	site.Files = files
	return site, nil
}

func renderSiteHTML(opts options) ([]byte, error) {
	data, err := buildSiteData(opts)
	if err != nil {
		return nil, err
	}
	return renderHTML(data)
}

func renderHTML(data *SiteData) ([]byte, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return []byte(strings.Replace(pageTemplate, "{{.JSON}}", string(jsonBytes), 1)), nil
}
