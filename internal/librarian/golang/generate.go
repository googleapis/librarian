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

// Package golang provides functionality for generating Go client libraries.
package golang

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const (
	releaseLevelGA = "ga"
)

var (
	//go:embed template/_README.md.txt
	readmeTmpl       string
	readmeTmplParsed = template.Must(template.New("readme").Parse(readmeTmpl))
)

// Generate generates all the given libraries in sequence.
func Generate(ctx context.Context, libraries []*config.Library, googleapisDir string) error {
	for _, library := range libraries {
		if err := generate(ctx, library, googleapisDir); err != nil {
			return err
		}
	}
	return nil
}

// generate generates a Go client library.
func generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return nil
	}

	outdir, err := filepath.Abs(repoRootPath(library))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return err
	}

	for _, api := range library.APIs {
		if err := generateAPI(ctx, api, library, googleapisDir, outdir); err != nil {
			return fmt.Errorf("api %q: %w", api.Path, err)
		}
	}

	src := filepath.Join(outdir, "cloud.google.com", "go")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("cannot access directory %q: %w", src, err)
	}
	if err := filesystem.MoveAndMerge(src, outdir); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(outdir, "cloud.google.com")); err != nil {
		return err
	}

	if err := fixVersioning(outdir, library.Name, modulePath(library)); err != nil {
		return err
	}
	if library.Go != nil {
		for _, p := range library.Go.DeleteGenerationOutputPaths {
			if err := os.RemoveAll(filepath.Join(outdir, p)); err != nil {
				return err
			}
		}
	}

	absModuleRoot, err := filepath.Abs(outdir)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absModuleRoot, outdir+string(filepath.Separator)) && absModuleRoot != outdir {
		return fmt.Errorf("invalid library name: path traversal detected")
	}
	if err := generateInternalVersionFile(outdir, library.Version); err != nil {
		return err
	}
	for i, api := range library.APIs {
		if err := generateClientVersionFile(library, api.Path); err != nil {
			return err
		}
		api, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageGo)
		if err != nil {
			return err
		}
		if err := generateRepoMetadata(api, library); err != nil {
			return err
		}
		if i != 0 {
			continue
		}
		if err := generateREADME(library, api, outdir); err != nil {
			return err
		}
	}
	if err := updateSnippetMetadata(library); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(absModuleRoot, "go.mod")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// New client, init the module.
			return initModule(ctx, absModuleRoot, modulePath(library))
		}
		return err
	}
	return nil
}

func generateAPI(ctx context.Context, api *config.API, library *config.Library, googleapisDir, outdir string) error {
	goAPI := findGoAPI(library, api.Path)
	if goAPI == nil {
		return fmt.Errorf("could not find Go API %q in library %q: %w", api.Path, library.Name, errGoAPINotFound)
	}
	nestedProtos := goAPI.NestedProtos
	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"--go_out=" + outdir,
		"-I=" + googleapisDir,
		"--go-grpc_out=" + outdir,
		"--go-grpc_opt=require_unimplemented_servers=false",
	}
	if !goAPI.ProtoOnly {
		gapicOpts, err := buildGAPICOpts(api.Path, goAPI, googleapisDir)
		if err != nil {
			return err
		}
		args = append(args, "--go_gapic_out="+outdir)
		for _, opt := range gapicOpts {
			args = append(args, "--go_gapic_opt="+opt)
		}
	}

	protoFiles, err := collectProtoFiles(googleapisDir, api.Path, nestedProtos)
	if err != nil {
		return err
	}
	args = append(args, protoFiles...)
	return command.Run(ctx, args[0], args[1:]...)
}

func buildGAPICOpts(apiPath string, goAPI *config.GoAPI, googleapisDir string) ([]string, error) {
	sc, err := serviceconfig.Find(googleapisDir, apiPath, config.LanguageGo)
	if err != nil {
		return nil, err
	}
	gc, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}

	opts := []string{"go-gapic-package=" + buildGAPICImportPath(goAPI)}
	if goAPI == nil || !goAPI.NoMetadata {
		opts = append(opts, "metadata")
	}
	if hasRESTNumericEnums(sc) {
		opts = append(opts, "rest-numeric-enums")
	}
	if goAPI != nil && goAPI.DIREGAPIC {
		opts = append(opts, "diregapic")
	}
	if goAPI != nil && goAPI.EnabledGeneratorFeatures != nil {
		opts = append(opts, goAPI.EnabledGeneratorFeatures...)
	}
	if sc != nil {
		opts = append(opts, "api-service-config="+filepath.Join(googleapisDir, sc.ServiceConfig))
	}
	if gc != "" {
		opts = append(opts, "grpc-service-config="+filepath.Join(googleapisDir, gc))
	}
	// TODO(https://github.com/googleapis/librarian/issues/3775): assuming
	// transport is library-wide for now, until we have figured out the config
	// for transports.
	if trans := transport(sc); trans != "" {
		opts = append(opts, fmt.Sprintf("transport=%s", trans))
	}
	level := releaseLevel(sc)
	opts = append(opts, "release-level="+level)
	return opts, nil
}

func buildGAPICImportPath(goAPI *config.GoAPI) string {
	return fmt.Sprintf("cloud.google.com/go/%s;%s",
		goAPI.ImportPath, goAPI.ClientPackage)
}

// fixVersioning moves {name}/{version}/* up to {name}/ for versioned modules.
func fixVersioning(outputDir, library, modPath string) error {
	// parts is the module path split by "/".
	// For example, "cloud.google.com/go/bigquery/v2" becomes:
	// parts[0]: "cloud.google.com"
	// parts[1]: "go"
	// parts[2]: library ID (e.g., "bigquery")
	// parts[3]: version (e.g., "v2")
	parts := strings.Split(modPath, "/")
	if len(parts) == 3 {
		return nil
	}
	if len(parts) != 4 {
		return fmt.Errorf("unexpected module path: %s", modPath)
	}

	name, version := parts[2], parts[3]
	if library == name+"/"+version {
		return nil
	}

	srcDir := filepath.Join(outputDir, name)
	versionDir := filepath.Join(srcDir, version)
	if err := filesystem.MoveAndMerge(versionDir, srcDir); err != nil {
		return err
	}
	if err := os.RemoveAll(versionDir); err != nil {
		return err
	}

	snippetDir := filepath.Join(outputDir, "internal", "generated", "snippets", name)
	snippetVersionDir := filepath.Join(snippetDir, version)
	if _, err := os.Stat(snippetVersionDir); err == nil {
		if err := filesystem.MoveAndMerge(snippetVersionDir, snippetDir); err != nil {
			return err
		}
		if err := os.RemoveAll(snippetVersionDir); err != nil {
			return err
		}
	}
	return nil
}

func collectProtoFiles(googleapisDir, apiPath string, nestedProtos []string) ([]string, error) {
	apiDir := filepath.Join(googleapisDir, apiPath)
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read API directory %s: %w", apiDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".proto" {
			files = append(files, filepath.Join(apiDir, entry.Name()))
		}
	}

	for _, nested := range nestedProtos {
		files = append(files, filepath.Join(apiDir, nested))
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .proto files found in %s", apiDir)
	}
	return files, nil
}

func generateREADME(library *config.Library, api *serviceconfig.API, moduleRoot string) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("no APIs configured")
	}
	readmePath := filepath.Join(moduleRoot, "README.md")
	// Skip generating README if it's in the keep list.
	// Handwritten/veneer libraries should have the top-level README in the keep list.
	// TODO(https://github.com/googleapis/librarian/issues/4113): investigate the difference between
	// GAPIC and handwritten libraries.
	for _, k := range library.Keep {
		path := filepath.Join(moduleRoot, k)
		if path == readmePath {
			return nil
		}
	}
	f, err := os.Create(readmePath)
	if err != nil {
		return err
	}
	err = readmeTmplParsed.Execute(f, map[string]string{
		"Name":       api.Title,
		"ModulePath": modulePath(library),
	})
	cerr := f.Close()
	if err != nil {
		return err
	}
	return cerr
}

// updateSnippetMetadata updates the snippet metadata files with the correct library version.
func updateSnippetMetadata(library *config.Library) error {
	for _, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			return fmt.Errorf("error finding Go API %s: %w", api.Path, errGoAPINotFound)
		}
		// Proto-only client doesn't have generated snippets, skip updating.
		if goAPI.ProtoOnly {
			continue
		}
		baseDir := snippetDirectory(repoRootPath(library), clientPathFromLibraryRoot(library, goAPI))
		if err := updateSnippetDirectory(baseDir, library.Version); err != nil {
			return err
		}
	}
	return nil
}

func updateSnippetDirectory(baseDir, version string) error {
	return filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(d.Name(), "snippet_metadata") {
			return nil
		}
		read, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		newContent := strings.Replace(string(read), "$VERSION", version, 1)
		err = os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return err
		}
		return nil
	})
}

func hasRESTNumericEnums(sc *serviceconfig.API) bool {
	if len(sc.NoRESTNumericEnums) == 0 {
		return true
	}
	if _, ok := sc.NoRESTNumericEnums[config.LanguageAll]; ok {
		return false
	}
	if _, ok := sc.NoRESTNumericEnums[config.LanguageGo]; ok {
		return false
	}
	return true
}

// releaseLevel determines the release level for an API.
func releaseLevel(sc *serviceconfig.API) string {
	if rl, ok := sc.ReleaseLevels[config.LanguageGo]; ok {
		return rl
	}
	return releaseLevelGA
}

// transport get transport from serviceconfig.API for language Go.
//
// The default value is serviceconfig.GRPCRest.
func transport(sc *serviceconfig.API) serviceconfig.Transport {
	if sc != nil {
		return sc.Transport(config.LanguageGo)
	}
	return serviceconfig.GRPCRest
}
