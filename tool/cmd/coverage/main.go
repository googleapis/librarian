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

// coverage runs Go tests with coverage and checks that the total coverage
// percentage meets a per-component target.
//
// Usage:
//
//	coverage [-target=N] <packages...>
//
// It runs "go test -race -coverprofile -covermode=atomic" on the given
// packages, then uses "go tool cover -func" to extract the total coverage
// percentage and compares it against the target for the component. The
// default target is 80%. Components with a different target are listed below.
// The -target flag overrides both the default and per-component targets.
//
// When GITHUB_STEP_SUMMARY is set, as it is in GitHub Actions, a Markdown
// report with per-package statement coverage is appended to that file.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/command"
)

const defaultTarget = 80

// targets lists components whose coverage target differs from defaultTarget.
// The key is a package path suffix that identifies the component, and the value is the coverage target percentage.
// For example, "internal/librarian/foo": 70.
var targets = map[string]float64{}

func main() {
	targetFlag := flag.Float64("target", 0, "override coverage target (0 means use per-package default)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: coverage [-target=N] <packages...>\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	pkgs := flag.Args()
	if len(pkgs) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	target := *targetFlag
	if target == 0 {
		target = targetFor(pkgs)
	}
	ctx := context.Background()
	if err := runTests(ctx, pkgs); err != nil {
		log.Fatalf("go test: %v", err)
	}
	coverage, err := totalCoverage(ctx)
	if err != nil {
		log.Fatalf("coverage: %v", err)
	}
	fmt.Printf("Coverage: %.1f%% (target: %.0f%%)\n", coverage, target)
	if err := writeStepSummary(coverage, target); err != nil {
		log.Fatalf("step summary: %v", err)
	}
	if coverage < target {
		fmt.Printf("FAIL: coverage %.1f%% is below target %.0f%%\n", coverage, target)
		os.Exit(1)
	}
}

// cleanPkg strips Go package wildcards (./... suffix) and cleans the path.
func cleanPkg(pkg string) string {
	pkg = strings.TrimSuffix(pkg, "/...")
	return path.Clean(pkg)
}

// targetFor returns the coverage target for the given package list.
func targetFor(pkgs []string) float64 {
	for _, pkg := range pkgs {
		pkg = cleanPkg(pkg)
		for suffix, t := range targets {
			if strings.HasSuffix(pkg, suffix) {
				return t
			}
		}
	}
	return defaultTarget
}

// runTests runs "go test -race -coverprofile=coverage.out -covermode=atomic"
// on the given packages.
func runTests(ctx context.Context, pkgs []string) error {
	args := []string{"test", "-race", "-coverprofile=coverage.out", "-covermode=atomic"}
	args = append(args, pkgs...)
	return command.RunStreaming(ctx, "go", args...)
}

// totalCoverage runs "go tool cover -func" on coverage.out and returns the
// total coverage percentage.
func totalCoverage(ctx context.Context) (float64, error) {
	out, err := command.Output(ctx, "go", "tool", "cover", "-func=coverage.out")
	if err != nil {
		return 0, err
	}
	return parseTotalCoverage(out)
}

// parseTotalCoverage extracts the total coverage percentage from the output
// of "go tool cover -func".
func parseTotalCoverage(out string) (float64, error) {
	for line := range strings.SplitSeq(out, "\n") {
		if !strings.HasPrefix(line, "total:") {
			continue
		}
		pctStr := strings.TrimRightFunc(line, unicode.IsSpace)
		pctStr = strings.TrimSuffix(pctStr, "%")
		if i := strings.LastIndexFunc(pctStr, unicode.IsSpace); i >= 0 {
			pctStr = pctStr[i+1:]
		}
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse coverage percentage %q: %v", pctStr, err)
		}
		return pct, nil
	}
	return 0, fmt.Errorf("no total line found in coverage output")
}

// writeStepSummary appends a Markdown coverage report to the file named by
// GITHUB_STEP_SUMMARY. It does nothing if the variable is unset.
func writeStepSummary(total, target float64) error {
	name := os.Getenv("GITHUB_STEP_SUMMARY")
	if name == "" {
		return nil
	}
	profile, err := os.ReadFile("coverage.out")
	if err != nil {
		return err
	}
	pkgs, err := packageCoverage(string(profile))
	if err != nil {
		return err
	}
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(formatSummary(total, target, pkgs)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// pkgCoverage is the statement coverage of a single package.
type pkgCoverage struct {
	pkg     string
	covered int
	total   int
}

func (p pkgCoverage) percent() float64 {
	if p.total == 0 {
		return 0
	}
	return 100 * float64(p.covered) / float64(p.total)
}

// packageCoverage aggregates a Go coverage profile by package, sorted by
// package path.
func packageCoverage(profile string) ([]pkgCoverage, error) {
	byPkg := map[string]*pkgCoverage{}
	for line := range strings.SplitSeq(profile, "\n") {
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		// Each line has the form "pkg/file.go:start,end numStmts count".
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("malformed profile line %q", line)
		}
		file, _, ok := strings.Cut(fields[0], ":")
		if !ok {
			return nil, fmt.Errorf("malformed profile line %q", line)
		}
		stmts, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("malformed profile line %q: %w", line, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("malformed profile line %q: %w", line, err)
		}
		pkg := path.Dir(file)
		p, ok := byPkg[pkg]
		if !ok {
			p = &pkgCoverage{pkg: pkg}
			byPkg[pkg] = p
		}
		p.total += stmts
		if count > 0 {
			p.covered += stmts
		}
	}
	var out []pkgCoverage
	for _, p := range byPkg {
		out = append(out, *p)
	}
	slices.SortFunc(out, func(a, b pkgCoverage) int { return strings.Compare(a.pkg, b.pkg) })
	return out, nil
}

// formatSummary renders a Markdown coverage report.
func formatSummary(total, target float64, pkgs []pkgCoverage) string {
	status := "✅ PASS"
	if total < target {
		status = "❌ FAIL"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "### Coverage: %.1f%% (target: %.0f%%) %s\n\n", total, target, status)
	b.WriteString("<details><summary>Per-package statement coverage</summary>\n\n")
	b.WriteString("| Package | Coverage | Statements |\n|---|---:|---:|\n")
	for _, p := range pkgs {
		fmt.Fprintf(&b, "| `%s` | %.1f%% | %d/%d |\n", p.pkg, p.percent(), p.covered, p.total)
	}
	b.WriteString("\n</details>\n\n")
	return b.String()
}
