package release

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/container/java/message"
)

func TestReleaseInit(t *testing.T) {
	tests := []struct {
		name        string
		libraryID   string
		version     string
		expected    string
		expectError bool
	}{
		{
			name:      "happy path",
			libraryID: "google-cloud-java",
			version:   "2.0.0",
							expected:  "    <version>2.0.0<!-- {x-version-update:google-cloud-java:current} --> </version>",		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			// Copy the testdata pom.xml to the temporary directory.
			inputPath := filepath.Join("testdata", "pom.xml")
			outputPath := filepath.Join(tmpDir, "pom.xml")
			input, err := ioutil.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("failed to read input file: %v", err)
			}
			if err := ioutil.WriteFile(outputPath, input, 0644); err != nil {
				t.Fatalf("failed to write output file: %v", err)
			}

			request := &message.Request{
				ReleaseInit: &message.ReleaseInitRequest{
					LibraryID: test.libraryID,
					Version:   test.version,
				},
			}
			response := &message.Response{}

			// Change the current working directory to the temporary directory.
			// This is important because UpdateVersions walks the current directory.
			originalDir, err := filepath.Abs(".")
			if err != nil {
				t.Fatalf("failed to get current directory: %v", err)
			}
			// The filepath.Walk function is not suitable for changing the current working directory.
			// Instead, we should change the directory once to the temporary directory.
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change directory to %s: %v", tmpDir, err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Fatalf("failed to change back to original directory: %v", err)
				}
			}()

			ReleaseInit(request, response)

			if test.expectError {
				if response.Error == "" {
					t.Errorf("expected error, got success")
				}
			} else {
				if response.Error != "" {
					t.Errorf("expected success, got error: %s", response.Error)
				}
				content, err := ioutil.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("failed to read output file: %v", err)
				}
				if !strings.Contains(string(content), test.expected) {
					t.Errorf("expected file to contain %q, got %q", test.expected, string(content))
				}
			}
		})
	}
}