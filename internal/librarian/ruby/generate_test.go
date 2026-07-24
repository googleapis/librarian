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

package ruby

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

const testdataGoogleapis = "../../testdata/googleapis"

func TestBuildGAPICOpts(t *testing.T) {
	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		api     *config.API
		gemName string
		want    []string
	}{
		{
			name: "basic case with service and grpc configs",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			gemName: "google-cloud-secret_manager-v1",
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager-v1",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
				"ruby-cloud-rest-numeric-enums=true",
			},
		},
		{
			name: "rest transport from sdk.yaml",
			api: &config.API{
				Path: "google/cloud/compute/v1",
			},
			gemName: "google-cloud-compute-v1",
			want: []string{
				"ruby-cloud-gem-name=google-cloud-compute-v1",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/compute/v1/compute_v1.yaml"),
				"transport=rest",
				"ruby-cloud-rest-numeric-enums=true",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildGAPICOpts(test.api, test.gemName, googleapisDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTransport(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *serviceconfig.API
		want serviceconfig.Transport
	}{
		{
			name: "nil api",
			sc:   nil,
			want: serviceconfig.GRPCRest,
		},
		{
			name: "rest only",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.Rest,
				},
			},
			want: serviceconfig.Rest,
		},
		{
			name: "rest and grpc",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.GRPCRest,
				},
			},
			want: serviceconfig.GRPCRest,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := transport(test.sc)
			if got != test.want {
				t.Errorf("transport() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCollectProtoFiles(t *testing.T) {
	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		apiPath string
		want    []string
	}{
		{
			name:    "standard api path",
			apiPath: "google/cloud/secretmanager/v1",
			want: []string{
				filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
				filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
			},
		},
		{
			name:    "nested api path",
			apiPath: "google/cloud/gkehub/v1/configmanagement",
			want: []string{
				filepath.Join(googleapisDir, "google/cloud/gkehub/v1/configmanagement/configmanagement.proto"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := collectProtoFiles(googleapisDir, test.apiPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectProtoFiles_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}

	_, err = collectProtoFiles(googleapisDir, "non/existent/path")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("collectProtoFiles() error = %v, wantErr %v", err, fs.ErrNotExist)
	}
}

func TestGenerate_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}
	fileAsDir := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(fileAsDir, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		library *config.Library
		srcs    *sources.Sources
		wantErr error
	}{
		{
			name: "no apis",
			library: &config.Library{
				Name: "test-lib",
				APIs: []*config.API{},
			},
			srcs:    &sources.Sources{},
			wantErr: errNoAPIs,
		},
		{
			name: "non existent api path",
			library: &config.Library{
				Name:   "test-lib",
				Output: t.TempDir(),
				APIs: []*config.API{
					{
						Path: "non/existent/path",
					},
				},
			},
			srcs:    &sources.Sources{Googleapis: googleapisDir},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "invalid output dir",
			library: &config.Library{
				Name:   "test-lib",
				Output: filepath.Join(fileAsDir, "sub"),
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
			srcs:    &sources.Sources{},
			wantErr: syscall.ENOTDIR,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := Generate(t.Context(), nil, test.library, test.srcs)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Generate() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestToolsEnv(t *testing.T) {
	for _, test := range []struct {
		name        string
		gemPath     string
		wantGemPath string
	}{
		{
			name:        "default gem path",
			gemPath:     "",
			wantGemPath: "",
		},
		{
			name:        "custom gem path set",
			gemPath:     "/custom/gem/path",
			wantGemPath: "/custom/gem/path",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GEM_PATH", test.gemPath)
			env, err := toolsEnv()
			if err != nil {
				t.Fatal(err)
			}
			if env["PATH"] == "" {
				t.Error("toolsEnv() PATH is empty")
			}
			if env["GEM_HOME"] == "" {
				t.Error("toolsEnv() GEM_HOME is empty")
			}
			if test.wantGemPath != "" && !strings.Contains(env["GEM_PATH"], test.wantGemPath) {
				t.Errorf("toolsEnv() GEM_PATH = %q, want to contain %q", env["GEM_PATH"], test.wantGemPath)
			}
		})
	}
}

func setupDummyProtoc(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	t.Setenv("LIBRARIAN_BIN", binDir)

	installDir := filepath.Join(binDir, "ruby_tools", "bin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}

	protocPath := filepath.Join(binDir, "protoc")
	script := `#!/bin/sh
rubyOut=""
rubyCloudOut=""
for arg in "$@"; do
  case "$arg" in
    --ruby_cloud_out=*) rubyCloudOut="${arg#--ruby_cloud_out=}" ;;
    --ruby_out=*) rubyOut="${arg#--ruby_out=}" ;;
  esac
done
if [ -n "$rubyCloudOut" ]; then
  mkdir -p "$rubyCloudOut/lib/google/cloud/secret_manager"
  touch "$rubyCloudOut/lib/google/cloud/secret_manager/v1.rb"
  touch "$rubyCloudOut/CHANGELOG.md"
fi
if [ -n "$rubyOut" ]; then
  mkdir -p "$rubyOut/google/cloud/secret_manager"
  touch "$rubyOut/google/cloud/secret_manager/v1_pb.rb"
fi
exit 0
`
	if err := os.WriteFile(protocPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, plugin := range []string{"grpc_tools_ruby_protoc_plugin", "protoc-gen-ruby_cloud"} {
		pPathInBin := filepath.Join(binDir, plugin)
		pPathInInstallDir := filepath.Join(installDir, plugin)
		pScript := "#!/bin/sh\nexit 0\n"
		if err := os.WriteFile(pPathInBin, []byte(pScript), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pPathInInstallDir, []byte(pScript), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("PATH", binDir+string(filepath.ListSeparator)+os.Getenv("PATH"))
}

func TestGenerate(t *testing.T) {
	setupDummyProtoc(t)

	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	changelogPath := filepath.Join(outDir, "CHANGELOG.md")
	const existingContent = "# Initial Changelog Content\n"
	if err := os.WriteFile(changelogPath, []byte(existingContent), 0o644); err != nil {
		t.Fatal(err)
	}
	library := &config.Library{
		Name:   "google-cloud-secret_manager-v1",
		Output: outDir,
		APIs: []*config.API{
			{
				Path: "google/cloud/secretmanager/v1",
			},
		},
	}
	err = Generate(t.Context(), nil, library, &sources.Sources{Googleapis: googleapisDir})
	if err != nil {
		t.Fatal(err)
	}
	wantFile := filepath.Join(outDir, "lib", "google", "cloud", "secret_manager", "v1.rb")
	if _, err := os.Stat(wantFile); err != nil {
		t.Errorf("expected generated file %s to exist: %v", wantFile, err)
	}
	wantPbFile := filepath.Join(outDir, "lib", "google", "cloud", "secret_manager", "v1_pb.rb")
	if _, err := os.Stat(wantPbFile); err != nil {
		t.Errorf("expected generated pb file %s to exist: %v", wantPbFile, err)
	}
	gotChangelog, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(existingContent, string(gotChangelog)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateAPI(t *testing.T) {
	setupDummyProtoc(t)

	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}
	stagingDir := t.TempDir()
	api := &config.API{Path: "google/cloud/secretmanager/v1"}
	gemName := "google-cloud-secret_manager-v1"

	err = generateAPI(t.Context(), api, gemName, nil, googleapisDir, stagingDir)
	if err != nil {
		t.Fatalf("generateAPI() error = %v", err)
	}
	wantFile := filepath.Join(stagingDir, "lib", "google", "cloud", "secret_manager", "v1.rb")
	if _, err := os.Stat(wantFile); err != nil {
		t.Errorf("expected generated file %s to exist: %v", wantFile, err)
	}
	wantPbFile := filepath.Join(stagingDir, "lib", "google", "cloud", "secret_manager", "v1_pb.rb")
	if _, err := os.Stat(wantPbFile); err != nil {
		t.Errorf("expected generated pb file %s to exist: %v", wantPbFile, err)
	}
}

func TestGenerateAPI_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}
	api := &config.API{Path: "non/existent/path"}
	err = generateAPI(t.Context(), api, "gem-name", nil, googleapisDir, t.TempDir())
	if err == nil {
		t.Error("generateAPI() error = nil, want error")
	}
}

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name          string
		libName       string
		defaultOutput string
		want          string
	}{
		{
			name:          "empty default output",
			libName:       "google-cloud-secret_manager-v1",
			defaultOutput: "",
			want:          "google-cloud-secret_manager-v1",
		},
		{
			name:          "with default output directory",
			libName:       "google-cloud-secret_manager-v1",
			defaultOutput: "gems",
			want:          filepath.Join("gems", "google-cloud-secret_manager-v1"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.libName, test.defaultOutput)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
