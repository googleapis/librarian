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

// walkthrough builds the interactive walkthrough site from the markdown
// sources in doc/walkthroughs.
//
// Usage:
//
//	walkthrough [-src doc/walkthroughs] [-out _site] [-root .] [-sha <commit>] [-check]
//
// Every markdown file with front matter becomes one page. Level-two headings
// split a page into steps. Fenced blocks whose info string is "excerpt",
// "flow" or "mermaid" are rendered specially: excerpts are extracted from the
// repository at build time, so the site always shows the code at the commit
// it was built from. See doc/walkthroughs/README.md for the authoring guide.
//
// With -check, the tool resolves every excerpt and exits without writing
// files. TestBuildDocs runs the same check, so a change that removes a line
// referenced by a walkthrough fails the test suite.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/googleapis/librarian/internal/command"
)

type options struct {
	src   string
	out   string
	root  string
	sha   string
	repo  string
	check bool
}

func main() {
	var opts options
	flag.StringVar(&opts.src, "src", "doc/walkthroughs", "directory with the markdown sources")
	flag.StringVar(&opts.out, "out", "_site", "output directory")
	flag.StringVar(&opts.root, "root", ".", "repository root used to resolve excerpts")
	flag.StringVar(&opts.sha, "sha", "", "commit used in source links (default: git rev-parse HEAD)")
	flag.StringVar(&opts.repo, "repo", "googleapis/librarian", "GitHub repository used in source links")
	flag.BoolVar(&opts.check, "check", false, "resolve all excerpts and exit without writing files")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: walkthrough [-src dir] [-out dir] [-root dir] [-sha commit] [-check]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := run(context.Background(), opts); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, opts options) error {
	if opts.sha == "" {
		out, err := command.OutputInDir(ctx, opts.root, "git", "rev-parse", "HEAD")
		if err != nil {
			return fmt.Errorf("determining commit: %w", err)
		}
		opts.sha = strings.TrimSpace(out)
	}
	s, err := loadSite(opts)
	if err != nil {
		return err
	}
	if opts.check {
		fmt.Printf("%d pages, %d steps, %d excerpts resolved at %s\n", len(s.Pages), s.stepCount(), s.excerptCount(), s.SHA)
		return nil
	}
	if err := writeSite(s, opts.out); err != nil {
		return err
	}
	fmt.Printf("wrote %d pages to %s\n", len(s.Pages)+1, opts.out)
	return nil
}
