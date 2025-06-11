// Copyright 2024 Google LLC
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

package main_test

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var doubleSlashHeader = regexp.MustCompile(`^// Copyright 20\d\d Google LLC
//
// Licensed under the Apache License, Version 2\.0 \(the "License"\);`)

var shellHeader = regexp.MustCompile(`^#!/.*

# Copyright 20\d\d Google LLC
#
# Licensed under the Apache License, Version 2\.0 \(the "License"\);`)

var hashHeader = regexp.MustCompile(`^# Copyright 20\d\d Google LLC
#
# Licensed under the Apache License, Version 2\.0 \(the "License"\);`)

var noHeaderRequiredFiles = []string{".github/CODEOWNERS", "go.sum", "go.mod", ".gitignore", "LICENSE", "renovate.json"}

func TestHeaders(t *testing.T) {
	sfs := os.DirFS(".")
	fs.WalkDir(sfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}

		var requiredHeader *regexp.Regexp
		switch {
		case strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".proto"):
			requiredHeader = doubleSlashHeader
		case strings.HasSuffix(path, ".sh"):
			requiredHeader = shellHeader
		case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") || strings.HasPrefix(path, "Dockerfile"):
			requiredHeader = hashHeader
		case strings.HasSuffix(path, ".md"):
			return nil
		case slices.Contains(noHeaderRequiredFiles, path):
			return nil
		default:
			// Given the mixture of allow-lists and requirements, if there's a file which
			// isn't covered, we report an error.
			t.Errorf("%q: unknown header requirements", path)
			return nil
		}
		f, err := sfs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if !requiredHeader.MatchReader(bufio.NewReader(f)) {
			t.Errorf("%q: incorrect header", path)
		}
		return nil
	})
}

func TestStaticCheck(t *testing.T) {
	rungo(t, "run", "honnef.co/go/tools/cmd/staticcheck@latest", "./...")
}

func TestUnparam(t *testing.T) {
	rungo(t, "run", "mvdan.cc/unparam@latest", "./...")
}

func TestVet(t *testing.T) {
	rungo(t, "vet", "-all", "./...")
}

func TestGoModTidy(t *testing.T) {
	rungo(t, "mod", "tidy", "-diff")
}

func TestGovulncheck(t *testing.T) {
	rungo(t, "run", "golang.org/x/vuln/cmd/govulncheck@latest", "./...")
}

func rungo(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("go", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		if ee := (*exec.ExitError)(nil); errors.As(err, &ee) && len(ee.Stderr) > 0 {
			t.Fatalf("%v: %v\n%s", cmd, err, ee.Stderr)
		}
		t.Fatalf("%v: %v\n%s", cmd, err, output)
	}
}
