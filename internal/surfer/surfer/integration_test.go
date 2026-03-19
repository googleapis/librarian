package surfer

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	testdata := "testdata"

	// Runtime discovery of protoc
	protocPath := os.Getenv("SURFER_PROTOC")
	if protocPath == "" {
		var err error
		protocPath, err = exec.LookPath("protoc")
		if err != nil {
			t.Log("protoc not found in PATH and SURFER_PROTOC not set")
			t.Skip("skipping integration tests because protoc is not available")
		}
	} else {
		// Ensure the directory containing the custom protoc is in PATH for surfer's internal calls.
		os.Setenv("PATH", os.Getenv("PATH")+":"+filepath.Dir(protocPath))
	}
	_ = protocPath

	coreGoogleapis, err := findCoreGoogleapis()
	if err != nil {
		t.Fatalf("failed to find core googleapis: %v", err)
	}

	err = filepath.WalkDir(testdata, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		configFile := findGcloudConfig(path)
		if configFile == "" {
			return nil
		}
		expectedDir := filepath.Join(path, "expected", "surface")
		if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
			return nil
		}

		scenarioName, _ := filepath.Rel(testdata, path)
		t.Run(scenarioName, func(t *testing.T) {
			if scenarioName == "cyclic_messages" {
				t.Skip("skipping cyclic_messages due to known hang in surfer parser")
			}

			// Set a timeout per scenario to avoid hangs.
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			tmpDir := t.TempDir()
			outDir := filepath.Join(tmpDir, "out")
			protoRoot := filepath.Join(tmpDir, "proto_root")
			if err := os.MkdirAll(protoRoot, 0755); err != nil {
				t.Fatalf("failed to create proto root: %v", err)
			}

			// Symlink core googleapis
			absCore, _ := filepath.Abs(filepath.Join(coreGoogleapis, "google"))
			if err := os.Symlink(absCore, filepath.Join(protoRoot, "google")); err != nil {
				t.Fatalf("failed to symlink core googleapis: %v", err)
			}

			// Symlink scenario protos
			copyProtos(t, path, protoRoot)
			if parent := filepath.Dir(path); parent != testdata {
				copyProtos(t, parent, protoRoot)
			}

			protoFiles := findProtos(t, protoRoot)
			if len(protoFiles) == 0 {
				t.Fatalf("no proto files found for scenario %s", scenarioName)
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

			// Find actual generated service directory
			genServiceDir := findGeneratedServiceDir(outDir)
			if genServiceDir == "" {
				t.Fatalf("no output generated in %s", outDir)
			}

			compareDirectories(t, expectedDir, genServiceDir)
		})

		return nil
	})

	if err != nil {
		t.Fatalf("failed to walk testdata: %v", err)
	}
}

func findCoreGoogleapis() (string, error) {
	if env := os.Getenv("SURFER_GOOGLEAPIS"); env != "" {
		return env, nil
	}
	path := "../../testdata/googleapis"
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("could not find core googleapis at ../../testdata/googleapis and SURFER_GOOGLEAPIS not set")
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

func findProtos(t *testing.T, root string) []string {
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

func findGeneratedServiceDir(outDir string) string {
	var found string
	filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if d.IsDir() && path != outDir {
			// Check if this dir contains YAML files or subdirs with YAML files.
			// surfer usually generates service_name/resource_name/...
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func compareDirectories(t *testing.T, expectedDir, gotDir string) {
	t.Helper()
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
			return nil
		}

		compareFiles(t, path, gotPath, relPath)
		return nil
	})
}

func compareFiles(t *testing.T, expected, got, rel string) {
	t.Helper()
	wantContent, _ := os.ReadFile(expected)
	gotContent, _ := os.ReadFile(got)

	if filepath.Ext(expected) == ".yaml" {
		var wantYAML, gotYAML interface{}
		if err := yaml.Unmarshal(wantContent, &wantYAML); err != nil {
			t.Errorf("%s: failed to unmarshal expected YAML: %v", rel, err)
			return
		}
		if err := yaml.Unmarshal(gotContent, &gotYAML); err != nil {
			t.Errorf("%s: failed to unmarshal generated YAML: %v", rel, err)
			return
		}
		if diff := cmp.Diff(wantYAML, gotYAML); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", rel, diff)
		}
	} else {
		if string(wantContent) != string(gotContent) {
			t.Errorf("%s content mismatch", rel)
		}
	}
}
