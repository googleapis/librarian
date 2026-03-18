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
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Log the generated directory structure
	generatedDir := filepath.Join(tmpDir, "resourcestandard")
	// t.Logf("Generated directory structure in: %s", generatedDir)
	// filepath.Walk(generatedDir, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		t.Logf("Error accessing path %s: %v", path, err)
	// 		return nil
	// 	}
	// 	relPath, _ := filepath.Rel(generatedDir, path)
	// 	t.Logf("  - %s (IsDir: %v)", relPath, info.IsDir())
	// 	return nil
	// })

	// Define paths for comparison
	goldenDir := "testdata/resource_standard_gen_sfc_goldens/resource_standard"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestFieldSimpleTypesGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/field_simple_types_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "field_simple_types/v1/field_simple_types.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
		if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
		}

		// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "fieldsimpletypes")
	goldenDir := "testdata/field_simple_types_gen_sfc_goldens/field_simple_types"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestFilteredCommandGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/filtered_command_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "filtered_command/v1/filtered_command.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "filteredcommand")
	goldenDir := "testdata/filtered_command_gen_sfc_goldens/filtered_command"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestHelpTextGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/help_text_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "help_text/v1/help_text.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "helptext")
	goldenDir := "testdata/help_text_gen_sfc_goldens/help_text"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestHiddenCommandGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/hidden_command_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "hidden_command/v1/hidden_command.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "hiddencommand")
	goldenDir := "testdata/hidden_command_gen_sfc_goldens/hidden_command"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestHiddenFeatureGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/hidden_feature_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "hidden_feature/v1/hidden_feature.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
		if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "hiddenfeature")
	goldenDir := "testdata/hidden_feature_gen_sfc_goldens/hidden_feature"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestMethodAsyncGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/method_async_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "method_async/v1/method_async.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
		if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "methodasync")
	goldenDir := "testdata/method_async_gen_sfc_goldens/method_async"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestMethodCustomGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/method_custom_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "method_custom/v1/method_custom.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "methodcustom")
	goldenDir := "testdata/method_custom_gen_sfc_goldens/method_custom"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestMethodMinimalListGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/method_minimal_list_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "method_minimal_list/v1/method_minimal_list.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "methodminimallist")
	goldenDir := "testdata/method_minimal_list_gen_sfc_goldens/method_minimal_list"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestMethodOperationsGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/method_operations_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "method_operations/v1/method_operations.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "methodoperations")
	goldenDir := "testdata/method_operations_gen_sfc_goldens/method_operations"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestMethodOutputFormatGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/method_output_format_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "method_output_format/v1/method_output_format.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
		if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "methodoutputformat")
	goldenDir := "testdata/method_output_format_gen_sfc_goldens/method_output_format"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestMultiServiceGA(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/multi_service_v1.yaml",
		"--googleapis", "./test_env",
		"--proto-files-include-list", "multi_service/v1/multi_service_first.proto,multi_service/v1/multi_service_second.proto",
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "multiservice")
	goldenDir := "testdata/multi_service_gen_sfc_goldens/multi_service"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}

func TestMultiVersionMultiTrack(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Get repo root
	repoRoot := "../../.."

	// Run Surfer command from repo root
	cmd := exec.Command(
		"./bin/surfer-dev",
		"generate",
				"./test_env/multi_version_multi_track.yaml",
		"--googleapis", "./test_env",
		// Surfer will glob all versions from proto_library_path
		"--out", tmpDir,
	)
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Surfer command failed: %v Output: %s", err, string(output))
	}

	// Define paths for comparison
	// NOTE: Surfer currently outputs to a dir matching the service name, not the override
	generatedDir := filepath.Join(tmpDir, "multiversionmultitrack")
	goldenDir := "testdata/multi_version_multi_track_gen_sfc_goldens/multi_version_multi_track"

	// Compare the generated files with the golden files
	goldenTestComparer(t, generatedDir, goldenDir)
}
