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
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	//go:embed template/_README.md.txt
	readmeTmpl string

	readmeTmplParsed = template.Must(template.New("readme").Parse(readmeTmpl))
)

// Generate generates a Go client library.
func Generate(ctx context.Context, library *config.Library, googleapisDir string) error {
	if len(library.APIs) == 0 {
		return fmt.Errorf("no apis configured for library %q", library.Name)
	}

	outdir, err := filepath.Abs(library.Output)
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
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", src)
	}
	if err := move(src, outdir); err != nil {
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

	moduleRoot := filepath.Join(outdir, library.Name)
	absModuleRoot, err := filepath.Abs(moduleRoot)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absModuleRoot, outdir+string(filepath.Separator)) && absModuleRoot != outdir {
		return fmt.Errorf("invalid library name: path traversal detected")
	}
	if err := generateInternalVersionFile(moduleRoot, library.Version); err != nil {
		return err
	}
	for _, api := range library.APIs {
		if err := generateClientVersionFile(library, api.Path); err != nil {
			return err
		}
	}
	api, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, serviceconfig.LangGo)
	if err != nil {
		return err
	}
	if err := generateREADME(library, api, moduleRoot); err != nil {
		return err
	}
	if err := updateSnippetMetadata(library, outdir); err != nil {
		return err
	}
	return nil
}

// Format formats a generated Go library.
func Format(ctx context.Context, library *config.Library) error {
	outDir, err := filepath.Abs(library.Output)
	if err != nil {
		return err
	}
	args := []string{"-w", filepath.Join(outDir, library.Name)}
	snippetDir := filepath.Join(outDir, "internal", "generated", "snippets", library.Name)
	if _, err := os.Stat(snippetDir); err == nil {
		args = append(args, snippetDir)
	}
	return command.Run(ctx, "goimports", args...)
}

func generateAPI(ctx context.Context, api *config.API, library *config.Library, googleapisDir, outdir string) error {
	goAPI := findGoAPI(library, api.Path)
	var nestedProtos []string
	if goAPI != nil {
		nestedProtos = goAPI.NestedProtos
	}

	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"--go_out=" + outdir,
		"-I=" + googleapisDir,
		"--go-grpc_out=" + outdir,
		"--go-grpc_opt=require_unimplemented_servers=false",
	}
	if goAPI == nil || !goAPI.DisableGAPIC {
		gapicOpts, err := buildGAPICOpts(api.Path, library, goAPI, googleapisDir)
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

func buildGAPICOpts(apiPath string, library *config.Library, goAPI *config.GoAPI, googleapisDir string) ([]string, error) {
	sc, err := serviceconfig.Find(googleapisDir, apiPath, serviceconfig.LangGo)
	if err != nil {
		return nil, err
	}
	gc, err := serviceconfig.FindGRPCServiceConfig(googleapisDir, apiPath)
	if err != nil {
		return nil, err
	}

	opts := []string{
		"go-gapic-package=" + buildGAPICImportPath(apiPath, library, goAPI),
		"metadata",
		"rest-numeric-enums",
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
	opts = append(opts, "transport="+getTransport(sc))
	if library.ReleaseLevel != "" {
		opts = append(opts, "release-level="+library.ReleaseLevel)
	}
	return opts, nil
}

func buildGAPICImportPath(apiPath string, library *config.Library, goAPI *config.GoAPI) string {
	version := filepath.Base(apiPath)
	clientDir := library.Name
	if goAPI != nil && goAPI.ClientDirectory != "" {
		clientDir = goAPI.ClientDirectory
	}

	importPath := clientDir
	if goAPI != nil && goAPI.ImportPath != "" {
		importPath = goAPI.ImportPath
	}

	var modulePathVersion string
	if library.Go != nil && library.Go.ModulePathVersion != "" {
		modulePathVersion = "/" + library.Go.ModulePathVersion
	}
	return fmt.Sprintf("cloud.google.com/go/%s%s/api%s;%s",
		importPath, modulePathVersion, version, clientDir)
}

func findGoAPI(library *config.Library, apiPath string) *config.GoAPI {
	if library.Go == nil {
		return &config.GoAPI{}
	}
	for _, ga := range library.Go.GoAPIs {
		if ga.Path == apiPath {
			return ga
		}
	}
	return &config.GoAPI{}
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
	if err := move(filepath.Join(srcDir, version), srcDir); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(srcDir, version)); err != nil {
		return err
	}

	snippetDir := filepath.Join(outputDir, "internal", "generated", "snippets", name)
	snippetVersionDir := filepath.Join(snippetDir, version)
	if _, err := os.Stat(snippetVersionDir); err == nil {
		if err := move(snippetVersionDir, snippetDir); err != nil {
			return err
		}
		if err := os.RemoveAll(snippetVersionDir); err != nil {
			return err
		}
	}
	return nil
}

func move(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if _, err := os.Stat(dstPath); err == nil {
				// Destination exists, merge contents.
				if err := move(srcPath, dstPath); err != nil {
					return err
				}
				continue
			}
		}
		if err := os.Rename(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

// modulePath returns the Go module path for the library. ModulePathVersion is
// set for modules at v2+, e.g. "cloud.google.com/go/pubsub/v2".
func modulePath(library *config.Library) string {
	path := "cloud.google.com/go/" + library.Name
	if library.Go != nil && library.Go.ModulePathVersion != "" {
		path += "/" + library.Go.ModulePathVersion
	}
	return path
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
	f, err := os.Create(filepath.Join(moduleRoot, "README.md"))
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
func updateSnippetMetadata(library *config.Library, output string) error {
	baseDir := filepath.Join(output, "internal", "generated", "snippets", library.Name)
	return filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip the update if the baseDir is not existed.
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasPrefix(d.Name(), "snippet_metadata") {
			return nil
		}
		read, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		newContent := strings.Replace(string(read), "$VERSION", library.Version, 1)
		err = os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return err
		}
		return nil
	})
}

// getTransport get transport from serviceconfig.API for language Go.
//
// The default value is serviceconfig.GRPCRest.
func getTransport(sc *serviceconfig.API) string {
	if sc != nil {
		return sc.Transport("go")
	}
	return string(serviceconfig.GRPCRest)
}
