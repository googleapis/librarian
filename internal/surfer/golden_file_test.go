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

package surfer

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	runTargetComparison = flag.Bool("run-target", false, "if true, run integration tests that compare generated output with target (gen_sfc) golden files")
	updateGolden        = flag.Bool("update", false, "update surfer golden files")

	// Tests that are enabled by default because they are expected to pass against the target.
	enabledTargetTests = map[string]bool{
		"resource_standard": true,
	}
)

func TestGolden(t *testing.T) {
	coreGoogleapisPath := requireGoogleapisPath(t)

	for _, test := range []struct {
		name string
		skip string // Reason for skipping.
	}{
		{name: "apis/developerconnect"},
		{name: "apis/iam"},
		{name: "apis/parallelstore"},
		{name: "apis/seclm"},
		{name: "confirmation_prompt"},
		{name: "cyclic_messages", skip: "known infinite recursion/hang in surfer parser"},
		{name: "field_attributes"},
		{name: "field_complex_types"},
		{name: "field_flag_names"},
		{name: "field_oneof"},
		{name: "field_simple_types"},
		{name: "filtered_command"},
		{name: "help_text"},
		{name: "hidden_command"},
		{name: "hidden_feature"},
		{name: "method_async"},
		{name: "method_custom"},
		{name: "method_minimal_list"},
		{name: "method_operations"},
		{name: "method_output_format"},
		{name: "multi_service"},
		{name: "multi_version_multi_track", skip: "fails against autogen target"},
		{name: "regional_endpoints/global_only"},
		{name: "regional_endpoints/regional_required"},
		{name: "regional_endpoints/regional_supported"},
		{name: "resource_multitype"},
		{name: "resource_non_standard"},
		{name: "resource_reference"},
		{name: "resource_standard"},
		{name: "update_mask"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.skip != "" {
				t.Skip(test.skip)
			}

			scenarioPath := filepath.Join("testdata", test.name)

			// 1. Arrange: Build the complex virtual filesystem
			inputDir := filepath.Join(scenarioPath, "input")
			configFile := filepath.Join(inputDir, "gcloud.yaml")
			serviceFile := filepath.Join(inputDir, "service.yaml")
			if _, err := os.Stat(configFile); errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("gcloud.yaml not found in scenario input directory: %s", configFile)
			}

			// Set a timeout per scenario.
			ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
			defer cancel()

			tmpDir := t.TempDir()
			protoRoot, outDir := setupVirtualEnvironment(t, scenarioPath, coreGoogleapisPath, tmpDir)

			// 2. Act: Execute the CLI compiler
			gotServiceDir, gotServiceName := runSurferGenerator(ctx, t, configFile, serviceFile, protoRoot, outDir)

			// 3. Assert: Validate the outputs against the goldens
			t.Run("current", func(t *testing.T) {
				currentExpectedRoot := filepath.Join(scenarioPath, "expected", "current", "surface")
				if *updateGolden {
					updateGoldenOutputs(t, currentExpectedRoot, gotServiceDir, gotServiceName)
				} else {
					verifyGoldenOutputs(t, currentExpectedRoot, gotServiceDir, gotServiceName, test.name)
				}
			})

			t.Run("target", func(t *testing.T) {
				if !*runTargetComparison && !enabledTargetTests[test.name] {
					t.Skip("skipping target comparison; use --run-target to enable")
				}
				targetExpectedRoot := filepath.Join(scenarioPath, "expected", "target", "surface")
				verifyGoldenOutputs(t, targetExpectedRoot, gotServiceDir, gotServiceName, test.name)
			})
		})
	}
}

func setupVirtualEnvironment(t *testing.T, scenarioPath, coreGoogleapisPath, tmpDir string) (string, string) {
	t.Helper()
	outDir := filepath.Join(tmpDir, "out")
	protoRoot := filepath.Join(tmpDir, "proto_root")
	if err := os.MkdirAll(protoRoot, 0755); err != nil {
		t.Fatal(err)
	}

	// Symlink core googleapis
	if err := os.Symlink(filepath.Join(coreGoogleapisPath, "google"), filepath.Join(protoRoot, "google")); err != nil {
		t.Fatal(err)
	}

	// Symlink scenario protos
	inputDir := filepath.Join(scenarioPath, "input")
	copyProtos(t, inputDir, protoRoot)

	// Symlink parent protos if necessary (e.g., for regional_endpoints nested scenarios)
	if parent := filepath.Dir(scenarioPath); parent != "testdata" {
		parentInputDir := filepath.Join(parent, "input")
		if _, err := os.Stat(parentInputDir); err == nil {
			copyProtos(t, parentInputDir, protoRoot)
		}
	}

	return protoRoot, outDir
}

func runSurferGenerator(ctx context.Context, t *testing.T, configFile, serviceFile, protoRoot, outDir string) (string, string) {
	t.Helper()
	protoFiles := findProtos(protoRoot)
	if len(protoFiles) == 0 {
		t.Fatal("no proto files found for scenario")
	}

	args := []string{
		"surfer",
		"generate",
		configFile,
		"--service-config", serviceFile,
		"--googleapis", protoRoot,
		"--proto-files-include-list", strings.Join(protoFiles, ","),
		"--out", outDir,
	}

	if err := Run(ctx, args...); err != nil {
		t.Fatalf("surfer generation failed: %v", err)
	}

	// Find actual generated service directory.
	gotServiceDir, gotServiceName := findFirstSubdir(outDir)
	if gotServiceDir == "" {
		t.Fatalf("no output generated in %s", outDir)
	}
	return gotServiceDir, gotServiceName
}

func updateGoldenOutputs(t *testing.T, expectedRoot, gotServiceDir, gotServiceName string) {
	t.Helper()
	expectedServiceDir := filepath.Join(expectedRoot, gotServiceName)
	if err := os.RemoveAll(expectedRoot); err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	if err := os.MkdirAll(expectedServiceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := updateGoldenDir(expectedServiceDir, gotServiceDir); err != nil {
		t.Fatal(err)
	}
}

func verifyGoldenOutputs(t *testing.T, expectedRoot, gotServiceDir, gotServiceName, testName string) {
	t.Helper()
	if _, err := os.Stat(expectedRoot); errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected output directory not found in scenario directory: %s", expectedRoot)
	}
	expectedServiceDir := filepath.Join(expectedRoot, gotServiceName)
	if !compareDirectories(t, expectedServiceDir, gotServiceDir) {
		t.Logf("Generated directory tree for %s:\n%s", testName, getDirTree(gotServiceDir))
	}
}

func updateGoldenDir(dest string, src string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func copyProtos(t *testing.T, src, dst string) {
	t.Helper()
	absSrc, err := filepath.Abs(src)
	if err != nil {
		t.Fatalf("failed to get absolute path for %q: %v", src, err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to read directory %q: %v", src, err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".proto" {
			target := filepath.Join(dst, entry.Name())
			if _, err := os.Stat(target); errors.Is(err, fs.ErrNotExist) {
				if err := os.Symlink(filepath.Join(absSrc, entry.Name()), target); err != nil {
					t.Fatalf("failed to create symlink for %q: %v", target, err)
				}
			}
		}
	}
}

func findProtos(root string) []string {
	var protos []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".proto" {
			rel, _ := filepath.Rel(root, path)
			protos = append(protos, rel)
		}
		return nil
	})
	return protos
}

func getDirTree(root string) string {
	var sb strings.Builder
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		depth := strings.Count(rel, string(os.PathSeparator))
		sb.WriteString(strings.Repeat("  ", depth))
		if d.IsDir() {
			sb.WriteString(d.Name() + "/\n")
		} else {
			sb.WriteString(d.Name() + "\n")
		}
		return nil
	})
	return sb.String()
}

func findFirstSubdir(dir string) (string, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(dir, entry.Name()), entry.Name()
		}
	}
	return "", ""
}

func compareDirectories(t *testing.T, expectedDir, gotDir string) bool {
	t.Helper()
	allPass := true
	filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(expectedDir, path)

		gotPath := filepath.Join(gotDir, relPath)
		if _, err := os.Stat(gotPath); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: missing in output", relPath)
			allPass = false
			return nil
		}

		if !compareFiles(t, path, gotPath, relPath) {
			allPass = false
		} else {
			t.Logf("%s: MATCH", relPath)
		}
		return nil
	})

	filepath.WalkDir(gotDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(gotDir, path)

		expectedPath := filepath.Join(expectedDir, relPath)
		if _, err := os.Stat(expectedPath); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: extra file generated in output", relPath)
			allPass = false
		}
		return nil
	})

	return allPass
}

func compareFiles(t *testing.T, expected, got, rel string) bool {
	t.Helper()
	wantContent, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("%s: failed to read expected file: %v", rel, err)
	}
	gotContent, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("%s: failed to read generated file: %v", rel, err)
	}

	if filepath.Ext(expected) == ".yaml" {
		wantYAML, err := yaml.Unmarshal[any](wantContent)
		if err != nil {
			t.Errorf("%s: failed to unmarshal expected YAML: %v", rel, err)
			return false
		}
		gotYAML, err := yaml.Unmarshal[any](gotContent)
		if err != nil {
			t.Errorf("%s: failed to unmarshal generated YAML: %v", rel, err)
			return false
		}
		if diff := cmp.Diff(*wantYAML, *gotYAML, cmp.AllowUnexported()); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", rel, diff)
			return false
		}
	} else {
		wantStr := string(wantContent)
		gotStr := string(gotContent)

		// Ignore copyright year differences.
		re := regexp.MustCompile(`# Copyright \d{4} Google LLC`)
		wantStr = re.ReplaceAllString(wantStr, `# Copyright <YEAR> Google LLC`)
		gotStr = re.ReplaceAllString(gotStr, `# Copyright <YEAR> Google LLC`)

		if diff := cmp.Diff(wantStr, gotStr); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", rel, diff)
			return false
		}
	}
	return true
}

func requireGoogleapisPath(t *testing.T) string {
	t.Helper()
	testhelper.RequireCommand(t, "protoc")

	if env := os.Getenv("SURFER_GOOGLEAPIS"); env != "" {
		return env
	}

	relPath := "../testdata/googleapis"
	if _, err := os.Stat(relPath); err == nil {
		abs, err := filepath.Abs(relPath)
		if err != nil {
			t.Fatalf("failed to get absolute path for %q: %v", relPath, err)
		}
		return abs
	}

	t.Fatal("core googleapis not found via repo layout or SURFER_GOOGLEAPIS env var")
	return ""
}
