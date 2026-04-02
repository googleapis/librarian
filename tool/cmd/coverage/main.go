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
//	coverage <packages...>
//
// It runs "go test -race -coverprofile -covermode=atomic" on the given
// packages, then uses "go tool cover -func" to extract the total coverage
// percentage and compares it against the target for the component. The
// default target is 80%. Components with a different target are listed below.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/googleapis/librarian/internal/command"
)

const defaultTarget = 80

// targets lists components whose coverage target differs from defaultTarget.
// The key is a package path suffix that identifies the component.
var targets = map[string]float64{
	// TODO(https://github.com/googleapis/librarian/issues/4664): raise to 80.
	"internal/librarian/nodejs": 73,
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: coverage <packages...>")
	}
	pkgs := os.Args[1:]
	target := targetFor(pkgs)
	ctx := context.Background()
	if err := runTests(ctx, pkgs); err != nil {
		log.Fatalf("go test: %v", err)
	}
	coverage, err := totalCoverage(ctx)
	if err != nil {
		log.Fatalf("coverage: %v", err)
	}
	fmt.Printf("Coverage: %.1f%% (target: %.0f%%)\n", coverage, target)
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
