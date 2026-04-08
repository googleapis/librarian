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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	commonProtos = "google/cloud/common_resources.proto"
)

// Generate generates a Node.js client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, srcs *sources.Sources) error {
	googleapisDir := srcs.Googleapis
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory path: %w", err)
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	repoRoot := filepath.Dir(filepath.Dir(outdir))
	for _, api := range library.APIs {
		if err := generateAPI(ctx, api, library, googleapisDir, repoRoot); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}
	if err := runPostProcessor(ctx, cfg, library, googleapisDir, repoRoot, outdir); err != nil {
		return fmt.Errorf("failed to run post processor: %w", err)
	}
	return nil
}

func generateAPI(ctx context.Context, api *config.API, library *config.Library, googleapisDir, repoRoot string) error {
	version := filepath.Base(api.Path)
	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name, version)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return err
	}

	nodejsAPI := resolveNodejsAPI(library, api)

	googleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		return fmt.Errorf("failed to resolve googleapis directory path: %w", err)
	}

	apiDir := filepath.Join(googleapisDir, api.Path)
	protos, err := filepath.Glob(apiDir + "/*.proto")
	if err != nil {
		return fmt.Errorf("failed to find protos: %w", err)
	}
	if len(protos) == 0 {
		return fmt.Errorf("no protos found in api %q", api.Path)
	}

	// Add extra protos from configuration.
	for _, extra := range nodejsAPI.ExtraProtos {
		protos = append(protos, filepath.Join(googleapisDir, extra))
	}

	args, err := buildGeneratorArgs(api, library, googleapisDir, stagingDir)
	if err != nil {
		return err
	}
	cmdArgs := append(args[1:], protos...)
	return command.Run(ctx, args[0], cmdArgs...)
}

// resolveNodejsAPI returns the Node.js-specific configuration for the given API,
// applying default values if no explicit configuration is found in the library.
func resolveNodejsAPI(library *config.Library, api *config.API) *config.NodejsAPI {
	res := &config.NodejsAPI{
		Path:        api.Path,
		ExtraProtos: []string{commonProtos},
	}
	if library.Nodejs == nil {
		return res
	}
	for _, nodejsAPI := range library.Nodejs.NodejsAPIs {
		if nodejsAPI.Path != api.Path {
			continue
		}
		// Copy explicit config
		res.Path = nodejsAPI.Path
		res.ExtraProtos = nodejsAPI.ExtraProtos
		if len(res.ExtraProtos) == 0 {
			res.ExtraProtos = []string{commonProtos}
		}
		return res
	}
	return res
}

// buildGeneratorArgs constructs the gapic-generator-typescript arguments,
// excluding proto files.
func buildGeneratorArgs(api *config.API, library *config.Library, googleapisDir, stagingDir string) ([]string, error) {
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return nil, fmt.Errorf("failed to find protoc: %w", err)
	}

	args := []string{
		"gapic-generator-typescript",
		"--protoc=" + protocPath,
		"--common-proto-path=" + googleapisDir,
		"-I", googleapisDir,
		"--output-dir", stagingDir,
	}

	grpcConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	if grpcConfigPath != "" {
		args = append(args, "--grpc-service-config", filepath.Join(googleapisDir, grpcConfigPath))
	}

	apiMetadata, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageNodejs)
	if err != nil {
		return nil, err
	}
	if apiMetadata != nil && apiMetadata.ServiceConfig != "" {
		args = append(args, "--service-yaml", filepath.Join(googleapisDir, apiMetadata.ServiceConfig))
	}

	args = append(args, "--package-name", DerivePackageName(library))
	args = append(args, "--metadata")

	// Only pass --transport for non-default values (default is grpc+rest).
	transport := serviceconfig.GRPCRest
	if apiMetadata != nil {
		transport = apiMetadata.Transport(config.LanguageNodejs)
	}
	if transport != serviceconfig.GRPCRest {
		args = append(args, "--transport", string(transport))
	}
	if apiMetadata != nil && apiMetadata.HasRESTNumericEnums(config.LanguageNodejs) {
		args = append(args, "--rest-numeric-enums")
	}
	if library.Nodejs != nil {
		if library.Nodejs.BundleConfig != "" {
			args = append(args, "--bundle-config", filepath.Join(googleapisDir, library.Nodejs.BundleConfig))
		}
		for _, param := range library.Nodejs.ExtraProtocParameters {
			if param == "metadata" {
				continue
			}
			args = append(args, "--"+param)
		}
		if library.Nodejs.HandwrittenLayer {
			args = append(args, "--handwritten-layer")
		}
		if library.Nodejs.MainService != "" {
			args = append(args, "--main-service", library.Nodejs.MainService)
		}
		if library.Nodejs.Mixins != "" {
			args = append(args, "--mixins", library.Nodejs.Mixins)
		}
	}
	return args, nil
}

// runPostProcessor combines versioned API outputs from owl-bot-staging/ into
// the output directory using gapic-node-processing, then compiles protos.
func runPostProcessor(ctx context.Context, cfg *config.Config, library *config.Library, googleapisDir, repoRoot, outDir string) error {
	owlbotPath := filepath.Join(outDir, "owlbot.py")
	if _, err := os.Stat(owlbotPath); err == nil {
		// Old way: use synthtool
		if err := command.RunInDir(ctx, outDir, "python3", "owlbot.py"); err != nil {
			return fmt.Errorf("owlbot.py failed: %w", err)
		}
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to check for owlbot.py: %w", err)
	}

	// Template generation and exclusions are handled at the generator level.
	// Synthtool is only used for post-processing handled by standalone scripts
	// like librarian.js and owlbot.py. (Note: librarian.js is unrelated to the
	// Librarian CLI tool).

	// combine-library wipes the destination directory before writing generated
	// files (src/, protos/). Save the keep files it would delete, then restore
	// them afterward.
	backupDir, err := os.MkdirTemp("", "librarian-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create backup dir: %w", err)
	}
	defer os.RemoveAll(backupDir)
	for _, name := range library.Keep {
		src := filepath.Join(outDir, name)
		if _, err := os.Stat(src); err != nil {
			continue // file doesn't exist, nothing to save
		}
		dst := filepath.Join(backupDir, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create backup subdir for %s: %w", name, err)
		}
		if err := os.Rename(src, dst); err != nil {
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

	// Restore keep files.
	for _, name := range library.Keep {
		src := filepath.Join(backupDir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(outDir, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create output subdir for %s: %w", name, err)
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to restore %s: %w", name, err)
		}
	}

	// Copy generated samples from staging into the output directory.
	// combine-library only handles src/ and protos/; samples are generated
	// by gapic-generator-typescript but left in staging.
	if err := copySamplesFromStaging(stagingDir, outDir); err != nil {
		return fmt.Errorf("failed to copy samples from staging: %w", err)
	}

	if err := updateSnippetMetadataVersion(outDir, library.Version); err != nil {
		return fmt.Errorf("failed to update snippet metadata version: %w", err)
	}

	// Remove .OwlBot.yaml produced by the generator. Librarian replaces
	// OwlBot so this file is no longer needed.
	if err := os.Remove(filepath.Join(outDir, ".OwlBot.yaml")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove .OwlBot.yaml: %w", err)
	}

	if err := restoreCopyrightYear(outDir, library.CopyrightYear); err != nil {
		return fmt.Errorf("failed to restore copyright year: %w", err)
	}
	if !slices.Contains(library.Keep, ".repo-metadata.json") {
		if err := writeRepoMetadata(cfg, library, googleapisDir, outDir); err != nil {
			return fmt.Errorf("failed to write repo metadata: %w", err)
		}
	}
	if err := copyMissingProtos(googleapisDir, outDir); err != nil {
		return fmt.Errorf("failed to copy missing protos: %w", err)
	}
	if err := command.RunInDir(ctx, outDir, "compileProtos", "src"); err != nil {
		return fmt.Errorf("failed to compile protos: %w", err)
	}

	// librarian.js is a custom script some libraries use for post-processing.
	// It has nothing to do with the Librarian CLI tool.
	// We execute it from the repository root because many scripts (e.g. secretmanager)
	// use root-relative paths: https://github.com/googleapis/google-cloud-node/blob/1b44bd187289552199b4566f1201974730623a3a/packages/google-cloud-secretmanager/librarian.js#L35
	// TODO(https://github.com/googleapis/librarian/issues/5040) remove librarian.js once it's part
	// of the gapic-generator-typescript
	librarianScript := filepath.Join(outDir, "librarian.js")
	if _, err := os.Stat(librarianScript); err == nil {
		if err := command.RunInDir(ctx, repoRoot, "node", librarianScript); err != nil {
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
			if err := command.RunInDir(ctx, outDir, "npx", "gapic-node-processing", "generate-readme",
				fmt.Sprintf("--source-path=%s", outDir),
				fmt.Sprintf("--string-to-replace=[//]: # \"partials.%s\"", name),
				fmt.Sprintf("--replacement-string=%s", replacement),
			); err != nil {
				return fmt.Errorf("generate-readme (%s) failed: %w", name, err)
			}
		}
	}

	if err := os.RemoveAll(filepath.Join(repoRoot, "owl-bot-staging")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove owl-bot-staging: %w", err)
	}
	return nil
}

// restoreCopyrightYear replaces the copyright year in generated source files
// with the original year from the library configuration.
func restoreCopyrightYear(outDir, year string) error {
	if year == "" {
		return nil
	}
	re := regexp.MustCompile(`Copyright \d{4} Google`)
	replacement := []byte(fmt.Sprintf("Copyright %s Google", year))
	for _, dir := range []string{"src", "test"} {
		d := filepath.Join(outDir, dir)
		if _, err := os.Stat(d); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
		if err := replaceCopyrightInDir(d, re, replacement); err != nil {
			return err
		}
	}
	return nil
}

// replaceCopyrightInDir walks dir and replaces copyright years in .ts and .js
// files using the provided regex and replacement.
func replaceCopyrightInDir(dir string, re *regexp.Regexp, replacement []byte) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".js" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		updated := re.ReplaceAll(content, replacement)
		if bytes.Equal(updated, content) {
			return nil
		}
		return os.WriteFile(path, updated, 0644)
	})
}

// writeRepoMetadata generates .repo-metadata.json for the library.
func writeRepoMetadata(cfg *config.Config, library *config.Library, googleapisDir, outDir string) error {
	if len(library.APIs) == 0 {
		return nil
	}
	metadata, err := repometadata.FromLibrary(cfg, library, googleapisDir)
	if err != nil {
		return err
	}
	metadata.DistributionName = DerivePackageName(library)
	metadata.DefaultVersion = filepath.Base(library.APIs[0].Path)
	metadata.LibraryType = repometadata.GAPICAutoLibraryType
	return metadata.Write(outDir)
}

// copyMissingProtos reads *_proto_list.json files under outDir/src/ and copies
// any referenced protos that are missing from outDir/protos/ using the source
// files in googleapisDir. The generator copies the API's own protos but not
// transitive dependencies (e.g. google/logging/type/log_severity.proto).
func copyMissingProtos(googleapisDir, outDir string) error {
	googleapisDir, err := filepath.Abs(googleapisDir)
	if err != nil {
		return fmt.Errorf("failed to resolve googleapis directory: %w", err)
	}

	lists, err := filepath.Glob(filepath.Join(outDir, "src", "*", "*_proto_list.json"))
	if err != nil {
		return fmt.Errorf("failed to glob proto list files: %w", err)
	}

	for _, listPath := range lists {
		data, err := os.ReadFile(listPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", listPath, err)
		}
		var entries []string
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("failed to parse %s: %w", listPath, err)
		}

		listDir := filepath.Dir(listPath)
		for _, entry := range entries {
			absPath := filepath.Join(listDir, entry)
			absPath = filepath.Clean(absPath)
			if _, err := os.Stat(absPath); err == nil {
				continue
			}

			// Extract the proto-relative path after "protos/".
			const protosPrefix = "protos/"
			idx := strings.Index(entry, protosPrefix)
			if idx < 0 {
				continue
			}
			relPath := entry[idx+len(protosPrefix):]

			srcPath := filepath.Join(googleapisDir, relPath)
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read source proto %s: %w", srcPath, err)
			}
			if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", absPath, err)
			}
			if err := os.WriteFile(absPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write proto %s: %w", absPath, err)
			}
		}
	}
	return nil
}

// copySamplesFromStaging copies generated sample files from the staging
// directory into the output directory. The generator writes samples to
// owl-bot-staging/<lib>/<version>/samples/generated/<version>/ but
// combine-library does not move them.
func copySamplesFromStaging(stagingDir, outDir string) error {
	versions, err := os.ReadDir(stagingDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // staging dir may not exist
		}
		return err
	}
	for _, v := range versions {
		if !v.IsDir() {
			continue
		}
		samplesDir := filepath.Join(stagingDir, v.Name(), "samples")
		if _, err := os.Stat(samplesDir); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
		if err := filepath.WalkDir(samplesDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(samplesDir, path)
			if err != nil {
				return err
			}
			dst := filepath.Join(outDir, "samples", rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(dst, content, 0644)
		}); err != nil {
			return err
		}
	}
	return nil
}

// updateSnippetMetadataVersion updates the clientLibrary.version field in
// snippet metadata JSON files under outDir/samples/generated/. The
// gapic-generator-typescript hardcodes version "0.1.0"; this function replaces
// it with the library's actual version.
func updateSnippetMetadataVersion(outDir, version string) error {
	if version == "" {
		return nil
	}
	pattern := filepath.Join(outDir, "samples", "generated", "*", "snippet_metadata*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob snippet metadata files: %w", err)
	}
	for _, path := range matches {
		if err := updateVersionInFile(path, version); err != nil {
			return fmt.Errorf("failed to update %s: %w", path, err)
		}
	}
	return nil
}

// snippetMetadata is a minimal representation of a snippet metadata JSON file.
// The Snippets field uses [json.RawMessage] to avoid reformatting the array.
type snippetMetadata struct {
	ClientLibrary snippetClientLibrary `json:"clientLibrary"`
	Snippets      json.RawMessage      `json:"snippets"`
}

type snippetClientLibrary struct {
	Name     string          `json:"name"`
	Version  string          `json:"version"`
	Language string          `json:"language"`
	APIs     json.RawMessage `json:"apis"`
}

// updateVersionInFile reads a snippet metadata JSON file, sets the
// clientLibrary.version field to version, and writes it back.
func updateVersionInFile(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var metadata snippetMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}
	metadata.ClientLibrary.Version = version

	// Re-marshal with 2-space indent to match generator output.
	out, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	// Append trailing newline to match convention.
	out = append(out, '\n')
	return os.WriteFile(path, out, 0644)
}

// DerivePackageName returns the npm package name for a library. It uses
// nodejs.package_name if set, otherwise derives it by splitting the library
// name on the second dash (e.g. "google-cloud-batch" → "@google-cloud/batch").
func DerivePackageName(library *config.Library) string {
	if library.Nodejs != nil && library.Nodejs.PackageName != "" {
		return library.Nodejs.PackageName
	}
	return derivePackageNameFromLibraryName(library.Name)
}

func derivePackageNameFromLibraryName(name string) string {
	firstDash := strings.Index(name, "-")
	if firstDash < 0 {
		return name
	}
	secondDash := strings.Index(name[firstDash+1:], "-")
	if secondDash < 0 {
		return name
	}
	secondDash += firstDash + 1
	scope := name[:secondDash]
	pkg := name[secondDash+1:]
	return fmt.Sprintf("@%s/%s", scope, pkg)
}

// DefaultOutput returns the output path for a library.
func DefaultOutput(name, defaultOutput string) string {
	return filepath.Join(defaultOutput, name)
}
