package gcloud

import (
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// goldenTestComparer compares generated files in testDir with golden files in goldenDir.
func goldenTestComparer(t *testing.T, testDir, goldenDir string) {
	t.Helper()

	walkErr := filepath.Walk(goldenDir, func(goldenPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(goldenDir, goldenPath)
		if err != nil {
			return err
		}

		generatedPath := filepath.Join(testDir, relPath)

		if _, err := os.Stat(generatedPath); os.IsNotExist(err) {
			t.Errorf("Generated file not found for golden file: %s", relPath)
			return nil
		}

		goldenContent, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("Failed to read golden file %s: %v", goldenPath, err)
		}

		generatedContent, err := os.ReadFile(generatedPath)
		if err != nil {
			t.Fatalf("Failed to read generated file %s: %v", generatedPath, err)
		}

		if filepath.Ext(goldenPath) == ".yaml" {
			var goldenYAML interface{}
			var generatedYAML interface{}

			if err := yaml.Unmarshal(goldenContent, &goldenYAML); err != nil {
				t.Fatalf("Failed to unmarshal golden YAML %s: %v", goldenPath, err)
			}
			if err := yaml.Unmarshal(generatedContent, &generatedYAML); err != nil {
				t.Fatalf("Failed to unmarshal generated YAML %s: %v", generatedPath, err)
			}

			if diff := cmp.Diff(goldenYAML, generatedYAML); diff != "" {
				t.Errorf("YAML content mismatch for %s (-want +got):", relPath)
				t.Logf("%q", diff)
			}
		} else {
			// For non-YAML files, do a direct string comparison
			if diff := cmp.Diff(string(goldenContent), string(generatedContent)); diff != "" {
				t.Errorf("Content mismatch for %s (-want +got):", relPath)
				t.Logf("%q", diff)
			}
		}

		return nil
	})

	if walkErr != nil {
		t.Fatalf("Error walking golden directory: %v", walkErr)
	}
}
func TestResourceStandardGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/resource_standard_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "resource_standard/v1/standard_resource.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v\nOutput:\n%s", err, string(output))
	}

	// Log the generated directory structure
	generatedDir := filepath.Join(tmpDir, "resource-standard", "surface", "resource-standard")
	t.Logf("Generated directory structure in: %s", generatedDir)
	filepath.Walk(generatedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Logf("Error accessing path %s: %v", path, err)
			return nil
		}
		relPath, _ := filepath.Rel(generatedDir, path)
		t.Logf("  - %s (IsDir: %v)", relPath, info.IsDir())
		return nil
	})

	// Log the ENTIRE tmpDir content for debugging
	t.Logf("Contents of tmpDir: %s", tmpDir)
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Just log error and continue
			t.Logf("  Error accessing path %s: %v", path, err)
			return nil
		}
		relPath, _ := filepath.Rel(tmpDir, path)
		t.Logf("  - %s (IsDir: %v)", relPath, info.IsDir())
		return nil
	})

	// t.Logf("Surfer output:\n%s", string(output)) // Optional: Log surfer output

	// Define paths for comparison
	goldenDir := "testdata/resource_standard_gen_sfc_goldens"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}
