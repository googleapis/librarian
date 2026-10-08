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
//	walkthrough -serve 127.0.0.1:8080 [-ask "<command>"]
//
// Every markdown file with front matter becomes one page. Level-two headings
// split a page into steps. Fenced blocks whose info string is "excerpt",
// "generated", "flow" or "mermaid" are rendered specially: excerpts and
// generated tables are derived from the repository at build time, so the site
// always reflects the commit it was built from. See doc/walkthroughs/README.md
// for the authoring guide.
//
// With -check, the tool resolves every excerpt and exits without writing
// files. TestBuildDocs runs the same check, so a change that removes a line
// referenced by a walkthrough fails the test suite.
//
// With -serve, the tool serves the site on a loopback address and rebuilds
// it on every page load. With -ask, each step gets an "Ask" panel whose
// question is piped on stdin to the given shell command (for example an
// agent CLI); its stdout is shown in the page.
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
	serve string
	ask   string
	// Set by the server for each build.
	serveToken string
	askEnabled bool
}

func main() {
	var opts options
	flag.StringVar(&opts.src, "src", "doc/walkthroughs", "directory with the markdown sources")
	flag.StringVar(&opts.out, "out", "_site", "output directory")
	flag.StringVar(&opts.root, "root", ".", "repository root used to resolve excerpts")
	flag.StringVar(&opts.sha, "sha", "", "commit used in source links (default: git rev-parse HEAD)")
	flag.StringVar(&opts.repo, "repo", "googleapis/librarian", "GitHub repository used in source links")
	flag.BoolVar(&opts.check, "check", false, "resolve all excerpts and exit without writing files")
	flag.StringVar(&opts.serve, "serve", "", "serve the site on this loopback address, rebuilding on every page load (e.g. 127.0.0.1:8080)")
	flag.StringVar(&opts.ask, "ask", "", "with -serve: shell command that answers questions piped on stdin, e.g. an agent CLI")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: walkthrough [-src dir] [-out dir] [-root dir] [-sha commit] [-check]\n       walkthrough -serve addr [-ask command]\n")
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
	if opts.serve != "" {
		return serve(ctx, opts)
	}
	if opts.check {
		s, err := loadSite(opts)
		if err != nil {
			return err
		}
		fmt.Printf("%d pages, %d steps, %d excerpts and %d generated tables resolved at %s\n", len(s.Pages), s.stepCount(), s.excerptCount(), s.generatedCount(), s.SHA)
		return nil
	}
	if err := build(ctx, opts); err != nil {
		return err
	}
	fmt.Printf("wrote site to %s\n", opts.out)
	return nil
}

// build loads the sources and writes the site to opts.out.
func build(_ context.Context, opts options) error {
	s, err := loadSite(opts)
	if err != nil {
		return err
	}
	return writeSite(s, opts.out)
}
