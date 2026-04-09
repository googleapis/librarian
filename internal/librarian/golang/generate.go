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
	"github.com/googleapis/librarian/internal/snippetmetadata"
	"github.com/googleapis/librarian/internal/sources"
)

var (
	//go:embed template/_README.md.txt
	readmeTmpl       string
	readmeTmplParsed = template.Must(template.New("readme").Parse(readmeTmpl))

	// Overridable for testing.
	mkdirTemp  = os.MkdirTemp
	commandRun = command.Run
)

// Generate generates a Go client library.
func Generate(ctx context.Context, library *config.Library, srcs *sources.Sources) error {
	outDir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of output directory: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
<<<<<<< HEAD

	tempDir, err := mkdirTemp("", "librarian-gen-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
=======
	tempDir, err := os.MkdirTemp("", "librarian-gen-")
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			err = errors.Join(err, removeErr)
		}
	}()
>>>>>>> main
	// For preview libraries, the API protos are rooted in the
	// googleapis/preview subdirectory, so change the googleapisDir to target
	// that root.
	googleapisDir := srcs.Googleapis
	if isPreview(outDir) {
		googleapisDir = filepath.Join(googleapisDir, "preview")
	}

	for i, api := range library.APIs {
		goAPI := findGoAPI(library, api.Path)
		if goAPI == nil {
			return fmt.Errorf("error finding goAPI associated with API %s: %w", api.Path, errGoAPINotFound)
		}
		if err := generateAPI(ctx, goAPI, googleapisDir, tempDir); err != nil {
			return fmt.Errorf("api %q: %w", api.Path, err)
		}
<<<<<<< HEAD

		if err := moveGeneratedFiles(library, goAPI, tempDir, outDir); err != nil {
			return fmt.Errorf("failed to move generated files: %w", err)
=======
		if err := moveGeneratedFiles(library, goAPI, tempDir, outDir); err != nil {
			return err
>>>>>>> main
		}
		if err := generateClientVersionFile(library, goAPI); err != nil {
			return fmt.Errorf("failed to generate client version file: %w", err)
		}
		api, err := serviceconfig.Find(googleapisDir, api.Path, config.LanguageGo)
		if err != nil {
			return fmt.Errorf("failed to find service configuration: %w", err)
		}
		if err := generateRepoMetadata(api, library, goAPI); err != nil {
			return fmt.Errorf("failed to generate repo metadata: %w", err)
		}
		if i != 0 {
			continue
		}
		if err := generateREADME(library, api, outDir); err != nil {
			return fmt.Errorf("failed to generate README: %w", err)
		}
	}
	if library.Go != nil {
		for _, p := range library.Go.DeleteGenerationOutputPaths {
			if err := os.RemoveAll(filepath.Join(outDir, p)); err != nil {
				return fmt.Errorf("failed to delete generation output path %q: %w", p, err)
			}
		}
	}
<<<<<<< HEAD
	if err := generateInternalVersionFile(outDir, library.CopyrightYear, library.Version); err != nil {
		return fmt.Errorf("failed to generate internal version file: %w", err)
	}

=======
>>>>>>> main
	if _, err := os.Stat(filepath.Join(outDir, "go.mod")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// New client, init the module.
			return initModule(ctx, outDir, modulePath(library))
		}
		return fmt.Errorf("failed to stat go.mod: %w", err)
	}
	return nil
}

func generateAPI(ctx context.Context, goAPI *config.GoAPI, googleapisDir, outDir string) error {
	nestedProtos := goAPI.NestedProtos
	args := []string{
		"protoc",
		"--experimental_allow_proto3_optional",
		"--go_out=" + outDir,
		"-I=" + googleapisDir,
		"--go-grpc_out=" + outDir,
		"--go-grpc_opt=require_unimplemented_servers=false",
	}
	if !goAPI.ProtoOnly {
		gapicOpts, err := buildGAPICOpts(goAPI.Path, goAPI, googleapisDir)
		if err != nil {
			return err
		}
		args = append(args, "--go_gapic_out="+outDir)
		for _, opt := range gapicOpts {
			args = append(args, "--go_gapic_opt="+opt)
		}
	}

	protoFiles, err := collectProtoFiles(googleapisDir, goAPI.Path, nestedProtos)
	if err != nil {
		return err
	}
	args = append(args, protoFiles...)
	return commandRun(ctx, args[0], args[1:]...)
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
	if !goAPI.NoMetadata {
		opts = append(opts, "metadata")
	}
	if goAPI.NoSnippets {
		opts = append(opts, "omit-snippets")
	}
	if sc != nil && sc.HasRESTNumericEnums(config.LanguageGo) {
		opts = append(opts, "rest-numeric-enums")
	}
	if goAPI.DIREGAPIC {
		opts = append(opts, "diregapic")
	}
	if goAPI.EnabledGeneratorFeatures != nil {
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
	opts = append(opts, "release-level="+sc.ReleaseLevel(config.LanguageGo))
	return opts, nil
}

func buildGAPICImportPath(goAPI *config.GoAPI) string {
	return fmt.Sprintf("cloud.google.com/go/%s;%s",
		goAPI.ImportPath, goAPI.ClientPackage)
}

// moveGeneratedFiles moves generated API and snippet files from the protoc output
// directory to their destination in the repository.
func moveGeneratedFiles(library *config.Library, goAPI *config.GoAPI, srcDir, outDir string) error {
	if err := moveAPIDirectory(library, goAPI, srcDir, outDir); err != nil {
		return err
	}
	return moveAndUpdateSnippets(library, goAPI, srcDir, outDir)
}

// moveAPIDirectory moves the generated API directory from the temporary location to its
// final destination in the repository.
func moveAPIDirectory(library *config.Library, goAPI *config.GoAPI, srcDir, outDir string) error {
	libraryDirPrefix := filepath.Join(srcDir, "cloud.google.com", "go")
	librarySrc := filepath.Join(libraryDirPrefix, goAPI.ImportPath)
	libraryDest := filepath.Join(repoRootPath(outDir, library.Name), clientPathFromRepoRoot(library, goAPI))
	if err := os.MkdirAll(libraryDest, 0755); err != nil {
		return err
	}
	return filesystem.MoveAndMerge(librarySrc, libraryDest)
}

// moveAndUpdateSnippets moves the generated snippets from the temporary location to their final
// destination and updates their library versions.
func moveAndUpdateSnippets(library *config.Library, goAPI *config.GoAPI, srcDir, outDir string) error {
	snippetDest := findSnippetDirectory(library, goAPI, outDir)
	if snippetDest == "" {
		return nil
	}
	if err := os.MkdirAll(snippetDest, 0755); err != nil {
		return err
	}
	snippetDirPrefix := filepath.Join(srcDir, "cloud.google.com", "go", "internal", "generated", "snippets")
	snippetSrc := filepath.Join(snippetDirPrefix, goAPI.ImportPath)
	if err := filesystem.MoveAndMerge(snippetSrc, snippetDest); err != nil {
		return err
	}
	// UpdateAllLibraryVersions searches recursively, but since Go APIs are not
	// nested, this only updates the snippets for the current API.
	return snippetmetadata.UpdateAllLibraryVersions(snippetDest, library.Version)
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

// transport get transport from serviceconfig.API for language Go.
//
// The default value is serviceconfig.GRPCRest.
func transport(sc *serviceconfig.API) serviceconfig.Transport {
	if sc != nil {
		return sc.Transport(config.LanguageGo)
	}
	return serviceconfig.GRPCRest
}

// isPreview determines if the given output directory contains the canonical
// preview subdirectory segments as a means of identifying the library as a
// preview library.
func isPreview(output string) bool {
	return strings.Contains(output, "preview/internal")
}
