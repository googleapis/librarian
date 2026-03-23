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
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

var (
	runAutogenComparison = flag.Bool("run-with-autogen-comparison", false, "if true, run integration tests that compare generated output with autogen golden files")
	updateGolden         = flag.Bool("update", false, "update surfer golden files")
)

func TestGolden(t *testing.T) {
	testhelper.RequireCommand(t, "protoc")

	var coreGoogleapisPath string
	// Locate core googleapis. Support SURFER_GOOGLEAPIS fallback.
	if env := os.Getenv("SURFER_GOOGLEAPIS"); env != "" {
		coreGoogleapisPath = env
	} else {
		// Try relative path from this directory.
		relPath := "../../testdata/googleapis"
		if _, err := os.Stat(relPath); err == nil {
			abs, _ := filepath.Abs(relPath)
			coreGoogleapisPath = abs
		}
	}

	if coreGoogleapisPath == "" {
		t.Fatal("core googleapis not found via repo layout or SURFER_GOOGLEAPIS env var")
	}

	for _, test := range []struct {
		name string
		skip string // Reason for skipping.
	}{
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
		{name: "method_custom", skip: "generator fails for this scenario"},
		{name: "method_minimal_list"},
		{name: "method_operations"},
		{name: "method_output_format"},
		{name: "multi_service", skip: "generator fails for this scenario"},
		{name: "multi_version_multi_track", skip: "generator fails for this scenario"},
		{name: "regional_endpoints/global_only"},
		{name: "regional_endpoints/regional_required"},
		{name: "regional_endpoints/regional_supported"},
		{name: "resource_multitype"},
		{name: "resource_non_standard"},
		{name: "resource_reference", skip: "generator fails for this scenario"},
		{name: "resource_standard"},
		{name: "update_mask"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.skip != "" {
				t.Skip(test.skip)
			}

			scenarioPath := filepath.Join("testdata", test.name)
			configFile := findGcloudConfig(scenarioPath)
			if configFile == "" {
				t.Fatal("gcloud configuration file not found in scenario directory")
			}

			// The expected output will be validated inside the subtests.

			// Set a timeout per scenario.
			ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
			defer cancel()

			tmpDir := t.TempDir()
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
			copyProtos(t, scenarioPath, protoRoot)
			if parent := filepath.Dir(scenarioPath); parent != "testdata" {
				copyProtos(t, parent, protoRoot)
			}

			protoFiles := findProtos(t, protoRoot)
			if len(protoFiles) == 0 {
				t.Fatal("no proto files found for scenario")
			}

			args := []string{
				"surfer",
				"generate",
				configFile,
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

			t.Run("surfer", func(t *testing.T) {
				surferExpectedRoot := filepath.Join(scenarioPath, "expected", "surfer", "surface")
				expectedServiceDir := findMatchingExpectedServiceDir(surferExpectedRoot, gotServiceName)
				if expectedServiceDir == "" {
					expectedServiceDir = filepath.Join(surferExpectedRoot, gotServiceName)
				}

				if *updateGolden {
					if err := os.RemoveAll(surferExpectedRoot); err != nil && !os.IsNotExist(err) {
						t.Fatal(err)
					}
					if err := os.MkdirAll(expectedServiceDir, 0755); err != nil {
						t.Fatal(err)
					}
					if err := updateGoldenDir(expectedServiceDir, gotServiceDir); err != nil {
						t.Fatal(err)
					}
				} else {
					if _, err := os.Stat(surferExpectedRoot); os.IsNotExist(err) {
						t.Fatalf("expected surfer output directory not found in scenario directory: %s", surferExpectedRoot)
					}
					// Find matching again in case it's actually in root
					actualExpectedServiceDir := findMatchingExpectedServiceDir(surferExpectedRoot, gotServiceName)
					if actualExpectedServiceDir == "" {
						actualExpectedServiceDir = surferExpectedRoot
					}
					if !compareDirectories(t, actualExpectedServiceDir, gotServiceDir) {
						t.Logf("Generated directory tree for %s:\n%s", test.name, getDirTree(gotServiceDir))
					}
				}
			})

			t.Run("autogen", func(t *testing.T) {
				if !*runAutogenComparison {
					t.Skip("skipping autogen comparison; use --run-with-autogen-comparison to enable")
				}
				autogenExpectedRoot := filepath.Join(scenarioPath, "expected", "autogen", "surface")
				if _, err := os.Stat(autogenExpectedRoot); os.IsNotExist(err) {
					t.Fatalf("expected autogen output directory not found in scenario directory: %s", autogenExpectedRoot)
				}
				expectedServiceDir := findMatchingExpectedServiceDir(autogenExpectedRoot, gotServiceName)
				if expectedServiceDir == "" {
					expectedServiceDir = autogenExpectedRoot
				}

				if !compareDirectories(t, expectedServiceDir, gotServiceDir) {
					t.Logf("Generated directory tree for %s:\n%s", test.name, getDirTree(gotServiceDir))
				}
			})
		})
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

func findGcloudConfig(dir string) string {
	for _, name := range []string{"gcloud.yaml", "gcloud_config.yaml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func copyProtos(t *testing.T, src, dst string) {
	t.Helper()
	absSrc, _ := filepath.Abs(src)
	entries, err := os.ReadDir(src)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() != "expected" && entry.Name() != "tests" && entry.Name() != "google" {
				copyProtos(t, filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()))
			}
			continue
		}
		if filepath.Ext(entry.Name()) == ".proto" {
			target := filepath.Join(dst, entry.Name())
			os.MkdirAll(filepath.Dir(target), 0755)
			if _, err := os.Stat(target); os.IsNotExist(err) {
				os.Symlink(filepath.Join(absSrc, entry.Name()), target)
			}
		}
	}
}

func findProtos(_ *testing.T, root string) []string {
	var protos []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "google" {
			return filepath.SkipDir
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

func findMatchingExpectedServiceDir(root, targetName string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	normalizedTarget := normalize(targetName)
	for _, entry := range entries {
		if entry.IsDir() && normalize(entry.Name()) == normalizedTarget {
			return filepath.Join(root, entry.Name())
		}
	}
	return ""
}

func normalize(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "")
}

func compareDirectories(t *testing.T, expectedDir, gotDir string) bool {
	t.Helper()
	allPass := true
	filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(expectedDir, path)
		basename := filepath.Base(relPath)
		if basename == "__init__.py" || basename == "_init_extensions.py" {
			return nil
		}

		gotPath := filepath.Join(gotDir, relPath)
		if _, err := os.Stat(gotPath); os.IsNotExist(err) {
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
	return allPass
}

func compareFiles(t *testing.T, expected, got, rel string) bool {
	t.Helper()
	wantContent, _ := os.ReadFile(expected)
	gotContent, _ := os.ReadFile(got)

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
		if string(wantContent) != string(gotContent) {
			t.Errorf("%s content mismatch", rel)
			return false
		}
	}
	return true
}
