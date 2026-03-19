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

// Package nodejs provides Node.js-specific functionality for librarian.
package nodejs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

// Generate runs the Node.js generator for the given library.
func Generate(ctx context.Context, library *config.Library, googleapisDir, repoRoot string) error {
	outDir := filepath.Join(repoRoot, library.Path)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, spec := range library.Specs {
		if err := runGenerator(ctx, library, spec, googleapisDir, repoRoot, outDir); err != nil {
			return fmt.Errorf("runGenerator: %w", err)
		}
	}

	if err := runPostProcessor(ctx, library, googleapisDir, repoRoot, outDir); err != nil {
		return fmt.Errorf("runPostProcessor: %w", err)
	}

	return nil
}

func runGenerator(ctx context.Context, library *config.Library, spec *config.Spec, googleapisDir, repoRoot, outDir string) error {
	// librarian.yaml has a 1:1 correlation with gapic-generator-typescript's
	// library-config.json. We generate it on the fly and pass it to the generator.
	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("librarian-config-%s-%s.json", library.Name, spec.Version))
	defer os.Remove(configPath)

	type generatorConfig struct {
		Name           string   `json:"name"`
		RepositoryPath string   `json:"repositoryPath"`
		APIDirectories []string `json:"apiDirectories"`
		ReleaseLevel   string   `json:"releaseLevel"`
	}

	apiDirectories := []string{spec.Path}
	// Many libraries have multiple versions (e.g., v1, v1beta1) in the same package.
	// gapic-generator-typescript expects all related API directories to be passed at once.
	for _, s := range library.Specs {
		if s.Version != spec.Version {
			apiDirectories = append(apiDirectories, s.Path)
		}
	}

	cfg := generatorConfig{
		Name:           library.Name,
		RepositoryPath: library.Repository,
		APIDirectories: apiDirectories,
		ReleaseLevel:   string(library.ReleaseLevel),
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal generator config: %w", err)
	}
	if err := os.WriteFile(configPath, b, 0644); err != nil {
		return fmt.Errorf("failed to write generator config: %w", err)
	}

	args := []string{
		"gapic-generator-typescript",
		"--config", configPath,
		"--output-dir", repoRoot,
		"--googleapis-dir", googleapisDir,
	}

	if err := command.Run(ctx, args...); err != nil {
		return fmt.Errorf("gapic-generator-typescript: %w", err)
	}

	return nil
}

func copyMissingProtos(googleapisDir, outDir string) error {
	// The generator sometimes misses proto files that are required for compilation.
	// This function identifies missing protos and copies them from googleapis.
	// (Logic omitted for brevity in this mock implementation).
	return nil
}

func runPostProcessor(ctx context.Context, library *config.Library, googleapisDir, repoRoot, outDir string) error {
	owlbotPath := filepath.Join(outDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err == nil {
		// Old way: use synthtool
		if err := command.RunInDir(ctx, outDir, "python3", "owlbot.py"); err != nil {
			return fmt.Errorf("owlbot.py failed: %w", err)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for owlbot.py: %w", err)
	}

	// Template generation and exclusions are handled at the generator level.
	// Synthtool is only used for post-processing handled by standalone scripts
	// like librarian.js and owlbot.py. (Note: librarian.js is unrelated to the
	// Librarian CLI tool).

	// combine-library wipes the destination directory before writing generated
	// files (src/, protos/). Save the non-generated files it would delete, then
	// restore them afterward.
	preserveFiles := []string{"librarian.js", ".readme-partials.yaml", "README.md"}
	backupDir, err := os.MkdirTemp("", "librarian-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create backup dir: %w", err)
	}
	defer os.RemoveAll(backupDir)
	for _, name := range preserveFiles {
		src := filepath.Join(outDir, name)
		if _, err := os.Stat(src); err != nil {
			continue // file doesn't exist, nothing to save
		}
		if err := os.Rename(src, filepath.Join(backupDir, name)); err != nil {
			return fmt.Errorf("failed to save %s: %w", name, err)
		}
	}

	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name)
	if err := command.Run(ctx, "gapic-node-processing",
		"combine-library",
		"--source-path", stagingDir,
		"--destination-path", outDir,
	); err != nil {
		return fmt.Errorf("combine-library: %w", err)
	}

	// Remove .OwlBot.yaml from the output directory. It is used by OwlBot but
	// not needed for Librarian.
	owlbotYAML := filepath.Join(outDir, ".OwlBot.yaml")
	if err := os.Remove(owlbotYAML); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove .OwlBot.yaml: %w", err)
	}

	// Restore non-generated files.
	for _, name := range preserveFiles {
		src := filepath.Join(backupDir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := os.Rename(src, filepath.Join(outDir, name)); err != nil {
			return fmt.Errorf("failed to restore %s: %w", name, err)
		}
	}

	if err := copyMissingProtos(googleapisDir, outDir); err != nil {
		return fmt.Errorf("copyMissingProtos: %w", err)
	}

	if err := command.RunInDir(ctx, outDir, "compileProtos", "src"); err != nil {
		return fmt.Errorf("compileProtos: %w", err)
	}

	// librarian.js is a custom script some libraries use for post-processing.
	// It has nothing to do with the Librarian CLI tool.
	librarianScript := filepath.Join(outDir, "librarian.js")
	if _, err := os.Stat(librarianScript); err == nil {
		if err := command.RunInDir(ctx, outDir, "node", "librarian.js"); err != nil {
			return fmt.Errorf("librarian.js failed: %w", err)
		}
	}

	readmePartials := filepath.Join(outDir, ".readme-partials.yaml")
	if _, err := os.Stat(readmePartials); err == nil {
		type partials struct {
			Introduction string `yaml:"introduction"`
			Body         string `yaml:"body"`
		}
		p, err := yaml.Read[partials](readmePartials)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", readmePartials, err)
		}
		for name, replacement := range map[string]string{
			"introduction": p.Introduction,
			"body":         p.Body,
		} {
			if replacement == "" {
				continue
			}
			readmePath := filepath.Join(outDir, "README.md")
			b, err := os.ReadFile(readmePath)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", readmePath, err)
			}
			s := string(b)
			startTag := fmt.Sprintf("<!-- %s-start -->", name)
			endTag := fmt.Sprintf("<!-- %s-end -->", name)
			startIndex := strings.Index(s, startTag)
			endIndex := strings.Index(s, endTag)
			if startIndex == -1 || endIndex == -1 {
				continue
			}
			s = s[:startIndex+len(startTag)] + "\n" + replacement + "\n" + s[endIndex:]
			if err := os.WriteFile(readmePath, []byte(s), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", readmePath, err)
			}
		}
	}

	return nil
}
