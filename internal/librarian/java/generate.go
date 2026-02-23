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

// Package java provides Java specific functionality for librarian.
package java

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// GenerateLibraries generates all the given libraries in sequence.
func GenerateLibraries(ctx context.Context, libraries []*config.Library, googleapisDir string) error {
	for _, library := range libraries {
		if err := generate(ctx, library, googleapisDir); err != nil {
			return err
		}
	}
	return nil
}

// generate generates a Java client library.
func generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("no apis configured for library %q", library.Name)
	}
	fmt.Printf("to be implemented with: %v, %v, %v", ctx, library.Name, googleapisDir)
	return nil
}

// Format formats a Java client library using google-java-format.
func Format(ctx context.Context, library *config.Library) error {
	files, err := collectJavaFiles(library.Output)
	if err != nil {
		return fmt.Errorf("failed to find java files for formatting: %w", err)
	}
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"--replace"}, files...)
	cmd := exec.CommandContext(ctx, "google-java-format", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}
	return nil
}

func collectJavaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".java" {
			return nil
		}
		// exclude samples/snippets/generated
		if strings.Contains(path, filepath.Join("samples", "snippets", "generated")) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// Clean removes files in the library's output directory that are not in the keep list.
// It targets patterns like proto-*, grpc-*, and the main GAPIC module.
func Clean(library *config.Library) error {
	return nil
}
