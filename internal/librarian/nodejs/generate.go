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
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	cloudCommonResourcesProto = "google/cloud/common_resources.proto"
)

// IsMixedLibrary reports whether the library has handwritten code wrapping
// generated or librarian-managed code.
func IsMixedLibrary(lib *config.Library) bool {
	return lib.Output != "" && len(lib.APIs) == 0
}

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
		// TODO(https://github.com/googleapis/google-cloud-node/issues/8149): Do not
		// generate v1small. This package is not meant to be used and will be
		// deprecated and removed in a future major release. Remove this workaround once resolved.
		if api.Path == "google/cloud/compute/v1small" {
			continue
		}
		if err := generateAPI(ctx, api, library, googleapisDir, repoRoot); err != nil {
			return fmt.Errorf("failed to generate api %q: %w", api.Path, err)
		}
	}
	if err := runPostProcessor(ctx, cfg, library, googleapisDir, repoRoot, outdir); err != nil {
		return fmt.Errorf("failed to run post processor: %w", err)
	}

	if library.Name == "google-cloud-compute" {
		if err := injectV1SmallExports(outdir); err != nil {
			return fmt.Errorf("failed to inject v1small exports: %w", err)
		}
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
	for index := range protos {
		rel, err := filepath.Rel(googleapisDir, protos[index])
		if err != nil {
			return fmt.Errorf("failed to make path %s relative: %w", protos[index], err)
		}
		protos[index] = rel
	}

	// Add additional protos from configuration.
	protos = append(protos, nodejsAPI.AdditionalProtos...)

	args, err := buildGeneratorArgs(api, library, googleapisDir, stagingDir, nodejsAPI)
	if err != nil {
		return err
	}
	cmdArgs := append(args[1:], protos...)
	return command.RunInDir(ctx, googleapisDir, args[0], cmdArgs...)
}

// resolveNodejsAPI returns the Node.js-specific configuration for the given API,
// applying default values if no explicit configuration is found in the library.
func resolveNodejsAPI(library *config.Library, api *config.API) *config.NodejsAPI {
	res := &config.NodejsAPI{
		Path: api.Path,
	}

	var apiConfig *config.NodejsAPI
	if library.Nodejs != nil {
		for _, nodejsAPI := range library.Nodejs.NodejsAPIs {
			if nodejsAPI.Path == api.Path {
				apiConfig = nodejsAPI
				break
			}
		}
	}

	omitCommon := false
	if apiConfig != nil {
		omitCommon = apiConfig.OmitCommonResources
		res.DIREGAPIC = apiConfig.DIREGAPIC
		res.OmitCommonResources = apiConfig.OmitCommonResources
	}

	var protos []string
	if !omitCommon {
		protos = append(protos, cloudCommonResourcesProto)
	}

	if library.Nodejs == nil {
		res.AdditionalProtos = protos
		return res
	}

	// Add package-level additional protos.
	protos = append(protos, library.Nodejs.AdditionalProtos...)

	// Add API-level additional protos.
	if apiConfig != nil {
		protos = append(protos, apiConfig.AdditionalProtos...)
	}

	res.AdditionalProtos = unique(protos)
	return res
}

func unique(ss []string) []string {
	m := make(map[string]bool)
	var res []string
	for _, s := range ss {
		if _, ok := m[s]; !ok {
			m[s] = true
			res = append(res, s)
		}
	}
	return res
}

// buildGeneratorArgs constructs the gapic-generator-typescript arguments,
// excluding proto files.
func buildGeneratorArgs(api *config.API, library *config.Library, googleapisDir, stagingDir string, nodejsAPI *config.NodejsAPI) ([]string, error) {
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return nil, fmt.Errorf("failed to find protoc: %w", err)
	}

	args := []string{
		"gapic-generator-typescript",
		"--protoc=" + protocPath,
		"--common-proto-path=.",
		"-I", ".",
		"--output-dir", stagingDir,
	}

	grpcConfigPath, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, api.Path)
	if err != nil {
		return nil, err
	}
	if grpcConfigPath != "" {
		args = append(args, "--grpc-service-config", grpcConfigPath)
	}

	apiMetadata, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageNodejs)
	if err != nil {
		return nil, err
	}
	if apiMetadata != nil && apiMetadata.ServiceConfig != "" {
		args = append(args, "--service-yaml", apiMetadata.ServiceConfig)
	}

	args = append(args, "--package-name", derivePackageName(library))
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

	if nodejsAPI.DIREGAPIC {
		args = append(args, "--diregapic")
	}

	if library.Nodejs != nil {
		if library.Nodejs.BundleConfig != "" {
			args = append(args, "--bundle-config", library.Nodejs.BundleConfig)
		}
		if library.Nodejs.ESM {
			args = append(args, "--format=esm")
		}
		for _, param := range library.Nodejs.ExtraProtocParameters {
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
	// combine-library wipes the destination directory before writing generated
	// files (src/, protos/). Save the keep files it would delete, then restore
	// them afterward.
	backupDir, err := os.MkdirTemp(filepath.Dir(outDir), "librarian-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create backup dir: %w", err)
	}
	defer os.RemoveAll(backupDir)
	// Backup keep files.
	if err := moveKeep(library.Keep, outDir, backupDir); err != nil {
		return err
	}

	stagingDir := filepath.Join(repoRoot, "owl-bot-staging", library.Name)
	combineArgs := []string{
		"combine-library",
		"--source-path", stagingDir,
		"--destination-path", outDir,
		"--default-version", resolveDefaultVersion(library),
	}
	if library.Nodejs != nil && library.Nodejs.ESM {
		combineArgs = append(combineArgs, "--is-esm")
	}
	if err := command.Run(ctx, "gapic-node-processing", combineArgs...); err != nil {
		return fmt.Errorf("combine-library: %w", err)
	}
	// Restore keep files.
	if err := moveKeep(library.Keep, backupDir, outDir); err != nil {
		return err
	}
	// Copy generated samples from staging into the output directory.
	// combine-library only handles src/ and protos/; samples are generated
	// by gapic-generator-typescript but left in staging.
	if err := copySamplesFromStaging(stagingDir, outDir); err != nil {
		return fmt.Errorf("failed to copy samples from staging: %w", err)
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
	protoDir := "src"
	var compileArgs []string
	if library.Nodejs != nil && library.Nodejs.ESM {
		protoDir = "esm/src"
		compileArgs = []string{"--esm"}
	}
	runArgs := append([]string{protoDir}, compileArgs...)

	if err := command.RunInDir(ctx, outDir, "compileProtos", runArgs...); err != nil {
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

	if err := os.RemoveAll(stagingDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove package staging: %w", err)
	}

	if err := removeRedundantLinterFiles(library, outDir); err != nil {
		return fmt.Errorf("failed to remove redundant linter files: %w", err)
	}

	// Remove google/cloud/common_resources.proto from the protos directory.
	// We don't need it in the repo (it isn't in googleapis-gen) and we don't
	// want it to be in the diff.
	if err := os.Remove(filepath.Join(outDir, "protos", cloudCommonResourcesProto)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove %s: %w", cloudCommonResourcesProto, err)
	}
	return nil
}

// TODO(https://github.com/googleapis/google-cloud-node/issues/8286): gapic-generator-typescript
// unconditionally generates redundant linter configuration files (.eslintignore, .eslintrc.json, etc.).
// This post-processing cleanup function removes them unless explicitly kept in librarian.yaml.
// Once gapic-generator-typescript is updated to stop generating them, this function must be removed.
func removeRedundantLinterFiles(library *config.Library, outDir string) error {
	keepSet := make(map[string]bool)
	for _, k := range library.Keep {
		keepSet[filepath.Clean(k)] = true
	}

	linterFiles := []string{
		".eslintignore",
		".eslintrc.json",
		".prettierignore",
		".prettierrc.js",
		".prettierrc.cjs",
	}

	for _, lf := range linterFiles {
		if keepSet[lf] {
			continue
		}
		path := filepath.Join(outDir, lf)
		if err := os.Remove(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("failed to remove redundant linter file %s: %w", path, err)
		}
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

// TODO(https://github.com/googleapis/librarian/issues/6340): Remove this function
// and all .repo-metadata.json generation once the documentation pipeline is
// migrated to read from librarian.yaml directly.
// writeRepoMetadata generates .repo-metadata.json for the library.
func writeRepoMetadata(cfg *config.Config, library *config.Library, googleapisDir, outDir string) error {
	if len(library.APIs) == 0 {
		return nil
	}
	metadata, err := repometadata.FromLibrary(cfg, library, googleapisDir)
	if err != nil {
		return err
	}
	metadata.DistributionName = derivePackageName(library)
	metadata.LibraryType = repometadata.GAPICAutoLibraryType
	metadata.DefaultVersion = resolveDefaultVersion(library)

	if pkgSuffix, ok := strings.CutPrefix(metadata.DistributionName, "@google-cloud/"); ok {
		metadata.ClientDocumentation = fmt.Sprintf("https://cloud.google.com/nodejs/docs/reference/%s/latest", pkgSuffix)
	}

	if library.Nodejs != nil && library.Nodejs.ClientDocumentationOverride != "" {
		metadata.ClientDocumentation = library.Nodejs.ClientDocumentationOverride
	}

	if strings.HasPrefix(metadata.ProductDocumentation, "https://cloud.google.com/") {
		if !strings.HasSuffix(metadata.ProductDocumentation, "/docs") && !strings.HasSuffix(metadata.ProductDocumentation, "/docs/") {
			metadata.ProductDocumentation = strings.TrimSuffix(metadata.ProductDocumentation, "/") + "/docs"
		}
	}

	if err := metadata.Write(outDir); err != nil {
		return err
	}

	// Go's json.MarshalIndent escapes HTML characters by default, but we want a
	// literal ampersand in the .repo-metadata.json.
	path := filepath.Join(outDir, ".repo-metadata.json")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content = bytes.ReplaceAll(content, []byte(`\u0026`), []byte(`&`))
	return os.WriteFile(path, content, 0644)
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
			absPath := filepath.Clean(filepath.Join(listDir, entry))
			if _, err := os.Stat(absPath); err == nil {
				continue
			}
			// Extract the proto-relative path after "protos/".
			_, relPath, ok := strings.Cut(entry, "protos/")
			if !ok {
				continue
			}
			srcPath := filepath.Join(googleapisDir, relPath)
			if err := filesystem.CopyFile(srcPath, absPath); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", srcPath, absPath, err)
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

// derivePackageName returns the npm package name for a library.
// It uses nodejs.package_name if set, otherwise derives it by splitting the
// library name on the second dash (e.g. "google-cloud-batch" → "@google-cloud/batch").
func derivePackageName(library *config.Library) string {
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

// resolveDefaultVersion returns the default API version (v1, v1beta etc) for
// a library, using the Node-specific override if present, or the path of the
// first API otherwise. If the library has no override and no APIs, an empty
// string is returned.
// TODO(https://github.com/googleapis/librarian/issues/6357): remove default version.
func resolveDefaultVersion(library *config.Library) string {
	if library.Nodejs != nil && library.Nodejs.DefaultVersion != "" {
		return library.Nodejs.DefaultVersion
	}
	if len(library.APIs) == 0 {
		return ""
	}
	return filepath.Base(library.APIs[0].Path)
}

// TODO(https://github.com/googleapis/google-cloud-node/issues/8149):
// This function is a temporary workaround to preserve v1small exports in the compute library.
// It must be deleted once v1small is formally deprecated and removed.
func injectV1SmallExports(outDir string) error {
	indexPath := filepath.Join(outDir, "src", "index.ts")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}
	content := string(data)

	// Avoid double injection if the function is called multiple times.
	if strings.Contains(content, "v1small") {
		return nil
	}

	// 1. Inject the import
	importLine := "import * as v1small from './v1small';\nimport * as v1 from './v1';"
	updated := strings.Replace(content, "import * as v1 from './v1';", importLine, 1)
	if updated == content {
		return fmt.Errorf("could not find v1 import in %s", indexPath)
	}
	content = updated

	// 2. Inject into export blocks (both named and default)
	// We search for \"{v1,\" and replace with \"{v1small, v1,\"
	updated = strings.ReplaceAll(content, "{v1,", "{v1small, v1,")
	if updated == content {
		return fmt.Errorf("could not find v1 export in %s", indexPath)
	}
	content = updated

	return os.WriteFile(indexPath, []byte(content), 0644)
}

func moveKeep(files []string, srcDir, dstDir string) error {
	for _, name := range files {
		src := filepath.Join(srcDir, name)
		if _, err := os.Stat(src); err != nil {
			continue // file doesn't exist, nothing to save
		}
		dst := filepath.Join(dstDir, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create destination subdirectory for %s: %w", name, err)
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s: %w", name, err)
		}
	}
	return nil
}
