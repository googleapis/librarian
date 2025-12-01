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

package librarian

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var noHeaderRequiredFiles = []string{
	".gcloudignore",
	"CODEOWNERS",
	".gitignore",
	"Dockerfile",
	"LICENSE",
	"coverage.out",
	"go.mod",
	"go.sum",
	"librarian",
	"renovate.json",
}

var ignoredExts = map[string]bool{
	".excalidraw": true,
	".md":         true,
	".yml":        true,
	".yaml":       true,
	".txt":        true,
	".webp":       true,
	".sh":         true,
	".xml":        true,
	".tmpl":       true,
}

var ignoredDirs = []string{
	".git",
	".idea",
	".vscode",
	"infra",
	"testdata",
}

// expectedHeader defines the regex for the required copyright header.
const expectedHeader = `// Copyright 202\d Google LLC
//
// Licensed under the Apache License, Version 2.0 \(the "License"\);
// you may not use this file except in compliance with the License\.
// You may obtain a copy of the License at`

var (
	headerRegex          = regexp.MustCompile("(?s)" + expectedHeader)
	dockerGoVersionRegex = regexp.MustCompile(`golang:(?P<version>[^ \n]+)`)
	modGoVersionRegex    = regexp.MustCompile(`\ngo\s+(?P<version>[^ \n]+)`)
)

func TestHeaders(t *testing.T) {
	sfs := os.DirFS(".")
	err := fs.WalkDir(sfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored files and directories.
		if d.IsDir() {
			if slices.Contains(ignoredDirs, d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if slices.Contains(noHeaderRequiredFiles, filepath.Base(path)) || ignoredExts[filepath.Ext(path)] {
			return nil
		}

		f, err := sfs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		ok, err := hasValidHeader(path, f)
		if err != nil {
			return err
		}
		if !ok {
			t.Errorf("%q: invalid header", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func hasValidHeader(path string, r io.Reader) (bool, error) {
	allBytes, err := io.ReadAll(r)
	if err != nil {
		return false, err
	}

	// If the file is a shell script and starts with a shebang, skip that line.
	if strings.HasSuffix(path, ".sh") && bytes.HasPrefix(allBytes, []byte("#!")) {
		// Find the index of the first newline to get the rest of the content.
		if i := bytes.IndexByte(allBytes, '\n'); i > -1 {
			allBytes = allBytes[i+1:]
		}
	}

	// If the file is a mustache template, the license is expected to be
	// wrapped as:
	// {{!
	// Copyright 2024 Google LLC
	// ...
	// }}
	if strings.HasSuffix(path, ".mustache") {
		if !bytes.HasPrefix(allBytes, []byte("{{!")) {
			return false, nil
		}
		end := bytes.Index(allBytes, []byte("}}"))
		if end == -1 {
			return false, nil
		}
		headerContent := allBytes[len("{{!"):end]
		headerContent = bytes.TrimPrefix(headerContent, []byte("\n"))
		var builder strings.Builder
		lines := strings.Split(string(headerContent), "\n")
		for i, line := range lines {
			builder.WriteString("//")
			if len(line) > 0 {
				builder.WriteString(" ")
			}
			builder.WriteString(line)
			if i < len(lines)-1 {
				builder.WriteString("\n")
			}
		}
		return headerRegex.MatchString(builder.String()), nil
	}

	return headerRegex.Match(allBytes), nil
}

// TestConsistentGoVersions walks the directory tree and checks Dockerfiles and go.mod files for specified Go versions.
// It ensures that only one unique Go version is specified across all found files to maintain consistency. The test
// fails if multiple Go versions are detected.
// TODO(https://github.com/googleapis/librarian/issues/2739): remove this test once is resolved.
func TestConsistentGoVersions(t *testing.T) {
	goVersions := make(map[string][]string)
	sfs := os.DirFS(".")
	err := fs.WalkDir(sfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, "Dockerfile") {
			return recordGoVersion(path, sfs, dockerGoVersionRegex, goVersions)
		}

		if strings.HasSuffix(path, "go.mod") {
			return recordGoVersion(path, sfs, modGoVersionRegex, goVersions)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(goVersions) > 1 {
		for ver, paths := range goVersions {
			t.Logf("%s found in %s", ver, strings.Join(paths, "\n"))
		}
		t.Error("found multiple golang versions")
	}
}

// recordGoVersion reads the content of the file at the given path, finds all matches of the provided regular
// expression (re), and records the first capturing group (expected to be the version string) in the goVersions map
// along with the file path.
func recordGoVersion(path string, sfs fs.FS, re *regexp.Regexp, goVersions map[string][]string) error {
	f, err := sfs.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	allBytes, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	matches := re.FindAllStringSubmatch(string(allBytes), -1)
	for _, match := range matches {
		goVersions[match[1]] = append(goVersions[match[1]], path)
	}

	return nil
}

func TestGolangCILint(t *testing.T) {
	rungo(t, "tool", "golangci-lint", "run")
}

func TestGoImports(t *testing.T) {
	cmd := exec.Command("go", "tool", "goimports", "-d", ".")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("goimports failed to run: %v\nStdout:\n%s\nStderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stdout.Len() > 0 {
		t.Errorf("goimports found unformatted files:\n%s", stdout.String())
	}
}

func TestGoModTidy(t *testing.T) {
	rungo(t, "mod", "tidy", "-diff")
}

func TestYAMLFormat(t *testing.T) {
	cmd := exec.Command("go", "tool", "yamlfmt", "-lint", ".")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("yamlfmt failed to run: %v\nStdout:\n%s\nStderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stdout.Len() > 0 {
		t.Errorf("yamlfmt found unformatted files:\n%s", stdout.String())
	}
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
