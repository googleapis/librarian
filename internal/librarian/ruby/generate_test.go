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
	"slices"
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
		library *config.Library
		cfg     *config.Config
		want    []string
	}{
		{
			name: "basic case with service and grpc configs",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager-v1",
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager-v1",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
			},
		},
		{
			name: "rest transport from sdk.yaml",
			api: &config.API{
				Path: "google/cloud/compute/v1",
			},
			library: &config.Library{
				Name: "google-cloud-compute-v1",
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-compute-v1",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/compute/v1/compute_v1.yaml"),
				"ruby-cloud-description=Compute Engine is an infrastructure as a service (IaaS) product that offers self-managed virtual machine (VM) instances and bare metal instances.",
				"ruby-cloud-summary=Compute Engine is an infrastructure as a service (IaaS) product that offers self-managed virtual machine (VM) instances and bare metal instances.",
				"ruby-cloud-generate-transports=rest",
			},
		},
		{
			name: "ruby cloud opts with migration version",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						MigrationVersion: "1.0",
					},
				},
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager",
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
				"ruby-cloud-migration-version=1.0",
			},
		},
		{
			name: "ruby cloud opts with service override",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						ServiceOverride: "SecretManager=secretmanager",
					},
				},
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager",
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
				"ruby-cloud-service-override=SecretManager=secretmanager",
			},
		},
		{
			name: "ruby cloud opts with gem namespace",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						GemNamespace: "Google::Cloud::SecretManager::V1",
					},
				},
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager",
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
				"ruby-cloud-gem-namespace=Google::Cloud::SecretManager::V1",
			},
		},
		{
			// The generic endpoint option is used for APIs like Grafeas (grafeas/v1), but tested here using Secret Manager testdata.
			name: "ruby cloud opts with generic endpoint",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						GenericEndpoint: true,
					},
				},
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager",
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
				"ruby-cloud-generic-endpoint=true",
			},
		},
		{
			name: "ruby cloud opts with all options",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						EnvPrefix:           "SECRET_MANAGER",
						ExtraDependencies:   "google-cloud-core=~>1.6",
						FactoryMethodSuffix: "service",
						GemNamespace:        "Google::Cloud::SecretManager::V1",
						GenericEndpoint:     true,
						MigrationVersion:    "1.0",
						NamespaceOverride:   "SecretManager=Secretmanager",
						PathOverride:        "secret_manager=secretmanager",
						RenamedFrom:         "google-cloud-secret_manager-old",
						ServiceOverride:     "SecretManager=secretmanager",
						Title:               "Secret Manager, V1",
						WrapperGemOverride:  "google-cloud-secret_manager",
						YardStrict:          "false",
					},
				},
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager",
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
				"ruby-cloud-env-prefix=SECRET_MANAGER",
				"ruby-cloud-extra-dependencies=google-cloud-core=~>1.6",
				"ruby-cloud-factory-method-suffix=service",
				"ruby-cloud-gem-namespace=Google::Cloud::SecretManager::V1",
				"ruby-cloud-generic-endpoint=true",
				"ruby-cloud-migration-version=1.0",
				"ruby-cloud-namespace-override=SecretManager=Secretmanager",
				"ruby-cloud-path-override=secret_manager=secretmanager",
				"ruby-cloud-renamed-from=google-cloud-secret_manager-old",
				"ruby-cloud-service-override=SecretManager=secretmanager",
				"ruby-cloud-title=Secret Manager\\, V1",
				"ruby-cloud-wrapper-gem-override=google-cloud-secret_manager",
				"ruby-cloud-yard-strict=false",
			},
		},
		{
			name: "wrapper library with wrapper_of option",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager",
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v1:0.29"},
				},
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
				"ruby-cloud-wrapper-of=v1:0.29",
			},
		},
		{
			name: "api with ruby-cloud-gem-name and ruby-cloud-wrapper-of overrides",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				Ruby: &config.RubyAPI{
					RubyCloudOpts: &config.RubyCloudOpts{
						GemName:   "google-cloud-secret_manager-custom",
						WrapperOf: "v1:1.0",
					},
				},
			},
			library: &config.Library{
				Name: "google-cloud-secret_manager",
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v1:0.29"},
				},
			},
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager-custom",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"ruby-cloud-description=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"ruby-cloud-summary=Stores sensitive data such as API keys\\, passwords\\, and certificates. Provides convenience while improving security.",
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"ruby-cloud-generate-transports=grpc;rest",
				"ruby-cloud-rest-numeric-enums=true",
				"ruby-cloud-wrapper-of=v1:1.0",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			serviceConfig, err := serviceconfig.Find(googleapisDir, test.api.Path, config.LanguageRuby)
			if err != nil {
				t.Fatal(err)
			}
			got, err := buildGAPICOpts(test.api, test.library, googleapisDir, serviceConfig)
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
		name          string
		serviceConfig *serviceconfig.API
		want          serviceconfig.Transport
	}{
		{
			name:          "nil api",
			serviceConfig: nil,
			want:          serviceconfig.GRPCRest,
		},
		{
			name: "rest only",
			serviceConfig: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.Rest,
				},
			},
			want: serviceconfig.Rest,
		},
		{
			name: "rest and grpc",
			serviceConfig: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.GRPCRest,
				},
			},
			want: serviceconfig.GRPCRest,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := transport(test.serviceConfig)
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
		name             string
		apiPath          string
		additionalProtos []string
		want             []string
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
		{
			name:    "recursive search protos",
			apiPath: "google/cloud/aiplatform/v1",
			want: []string{
				filepath.Join(googleapisDir, "google/cloud/aiplatform/v1/schema/schema.proto"),
			},
		},
		{
			name:             "with additional protos",
			apiPath:          "google/cloud/secretmanager/v1",
			additionalProtos: []string{"google/cloud/location/locations.proto"},
			want: []string{
				filepath.Join(googleapisDir, "google/cloud/location/locations.proto"),
				filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/resources.proto"),
				filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/service.proto"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := collectProtoFiles(googleapisDir, test.apiPath, test.additionalProtos)
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

	_, err = collectProtoFiles(googleapisDir, "non/existent/path", nil)
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
grpcOut=""
for arg in "$@"; do
  case "$arg" in
    --ruby_cloud_out=*) rubyCloudOut="${arg#--ruby_cloud_out=}" ;;
    --ruby_out=*) rubyOut="${arg#--ruby_out=}" ;;
    --grpc_out=*) grpcOut="${arg#--grpc_out=}" ;;
  esac
done
if [ -n "$rubyCloudOut" ]; then
  mkdir -p "$rubyCloudOut/lib/google/cloud/secret_manager/v1"
  touch "$rubyCloudOut/lib/google/cloud/secret_manager/v1/version.rb"
  touch "$rubyCloudOut/lib/google/cloud/secret_manager/v1.rb"
  touch "$rubyCloudOut/CHANGELOG.md"
  mkdir -p "$rubyCloudOut/snippets"
  cat << 'EOF' > "$rubyCloudOut/snippets/snippet_metadata_google.cloud.secretmanager.v1.json"
{
  "clientLibrary": {
    "name": "google-cloud-secret_manager-v1",
    "version": "",
    "language": "RUBY"
  }
}
EOF
fi
if [ -n "$rubyOut" ]; then
  mkdir -p "$rubyOut/google/cloud/secret_manager"
  touch "$rubyOut/google/cloud/secret_manager/v1_pb.rb"
  for arg in "$@"; do
    case "$arg" in
      *common_resources.proto)
        mkdir -p "$rubyOut/google/cloud"
        touch "$rubyOut/google/cloud/common_resources_pb.rb"
        ;;
    esac
  done
fi
if [ -n "$grpcOut" ]; then
  mkdir -p "$grpcOut/google/cloud/secret_manager"
  touch "$grpcOut/google/cloud/secret_manager/v1_services_pb.rb"
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
	repoMetadataPath := filepath.Join(outDir, ".repo-metadata.json")
	const existingRepoMetadataContent = "{\n  \"release_level\": \"ga\"\n}\n"
	if err := os.WriteFile(repoMetadataPath, []byte(existingRepoMetadataContent), 0o644); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(outDir, "lib", "google", "cloud", "secret_manager", "v1", "version.rb")
	if err := os.MkdirAll(filepath.Dir(versionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const existingVersionContent = `module Google
  module Cloud
    module SecretManager
      module V1
        VERSION = "1.2.3"
      end
    end
  end
end
`
	if err := os.WriteFile(versionPath, []byte(existingVersionContent), 0o644); err != nil {
		t.Fatal(err)
	}
	snippetMetadataPath := filepath.Join(outDir, "snippets", "snippet_metadata_google.cloud.secretmanager.v1.json")
	if err := os.MkdirAll(filepath.Dir(snippetMetadataPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const existingSnippetMetadataContent = "{\n  \"client_library\": {\n    \"version\": \"1.2.0\"\n  }\n}\n"
	if err := os.WriteFile(snippetMetadataPath, []byte(existingSnippetMetadataContent), 0o644); err != nil {
		t.Fatal(err)
	}
	library := &config.Library{
		Name:    "google-cloud-secret_manager-v1",
		Version: "1.2.3",
		Output:  outDir,
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
	wantServicesPbFile := filepath.Join(outDir, "lib", "google", "cloud", "secret_manager", "v1_services_pb.rb")
	if _, err := os.Stat(wantServicesPbFile); err != nil {
		t.Errorf("expected generated services pb file %s to exist: %v", wantServicesPbFile, err)
	}
	gotChangelog, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(existingContent, string(gotChangelog)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	gotRepoMetadata, err := os.ReadFile(repoMetadataPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(existingRepoMetadataContent, string(gotRepoMetadata)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	gotVersion, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(existingVersionContent, string(gotVersion)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	gotSnippetMetadata, err := os.ReadFile(snippetMetadataPath)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(existingSnippetMetadataContent, string(gotSnippetMetadata)); diff != "" {
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
	library := &config.Library{Name: "google-cloud-secret_manager-v1"}

	err = generateAPI(t.Context(), api, library, nil, googleapisDir, stagingDir)
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
	wantServicesPbFile := filepath.Join(stagingDir, "lib", "google", "cloud", "secret_manager", "v1_services_pb.rb")
	if _, err := os.Stat(wantServicesPbFile); err != nil {
		t.Errorf("expected generated services pb file %s to exist: %v", wantServicesPbFile, err)
	}
	unexpectedCommonPbFile := filepath.Join(stagingDir, "lib", "google", "cloud", "common_resources_pb.rb")
	if _, err := os.Stat(unexpectedCommonPbFile); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected common_resources_pb.rb %s to be removed, but err = %v", unexpectedCommonPbFile, err)
	}
}

func TestBuildProtocArgs(t *testing.T) {
	for _, test := range []struct {
		name          string
		googleapisDir string
		stagingDir    string
		libStagingDir string
		installDir    string
		isWrapper     bool
		serviceConfig *serviceconfig.API
		gapicOpts     []string
		protoFiles    []string
		want          []string
	}{
		{
			name:          "standard api defaults to grpc and rest",
			googleapisDir: "/googleapis",
			stagingDir:    "/staging",
			libStagingDir: "/staging/lib",
			installDir:    "/install",
			isWrapper:     false,
			serviceConfig: nil,
			protoFiles:    []string{"/googleapis/google/cloud/secretmanager/v1/service.proto"},
			want: []string{
				"--experimental_allow_proto3_optional",
				"-I=/googleapis",
				"--ruby_cloud_out=/staging",
				"--ruby_out=/staging/lib",
				"--grpc_out=/staging/lib",
				"--plugin=protoc-gen-grpc=/install/bin/grpc_tools_ruby_protoc_plugin",
				"/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name:          "rest-only transport with all",
			googleapisDir: "/googleapis",
			stagingDir:    "/staging",
			libStagingDir: "/staging/lib",
			installDir:    "/install",
			isWrapper:     false,
			serviceConfig: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageAll: serviceconfig.Rest,
				},
			},
			gapicOpts:  []string{"ruby-cloud-gem-name=google-cloud-compute-v1"},
			protoFiles: []string{"/googleapis/google/cloud/compute/v1/compute.proto"},
			want: []string{
				"--experimental_allow_proto3_optional",
				"-I=/googleapis",
				"--ruby_cloud_out=/staging",
				"--ruby_out=/staging/lib",
				"--ruby_cloud_opt=ruby-cloud-gem-name=google-cloud-compute-v1",
				"/googleapis/google/cloud/compute/v1/compute.proto",
			},
		},
		{
			name:          "rest-only transport with ruby",
			googleapisDir: "/googleapis",
			stagingDir:    "/staging",
			libStagingDir: "/staging/lib",
			installDir:    "/install",
			isWrapper:     false,
			serviceConfig: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.Rest,
				},
			},
			protoFiles: []string{"/googleapis/google/cloud/compute/v1/compute.proto"},
			want: []string{
				"--experimental_allow_proto3_optional",
				"-I=/googleapis",
				"--ruby_cloud_out=/staging",
				"--ruby_out=/staging/lib",
				"/googleapis/google/cloud/compute/v1/compute.proto",
			},
		},
		{
			name:          "grpc-only transport with ruby",
			googleapisDir: "/googleapis",
			stagingDir:    "/staging",
			libStagingDir: "/staging/lib",
			installDir:    "/install",
			isWrapper:     false,
			serviceConfig: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.GRPC,
				},
			},
			protoFiles: []string{"/googleapis/google/cloud/secretmanager/v1/service.proto"},
			want: []string{
				"--experimental_allow_proto3_optional",
				"-I=/googleapis",
				"--ruby_cloud_out=/staging",
				"--ruby_out=/staging/lib",
				"--grpc_out=/staging/lib",
				"--plugin=protoc-gen-grpc=/install/bin/grpc_tools_ruby_protoc_plugin",
				"/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
		{
			name:          "wrapper gem omits ruby_out and grpc_out",
			googleapisDir: "/googleapis",
			stagingDir:    "/staging",
			libStagingDir: "/staging/lib",
			installDir:    "/install",
			isWrapper:     true,
			serviceConfig: nil,
			gapicOpts:     []string{"ruby-cloud-gem-name=google-cloud-secret_manager"},
			protoFiles:    []string{"/googleapis/google/cloud/secretmanager/v1/service.proto"},
			want: []string{
				"--experimental_allow_proto3_optional",
				"-I=/googleapis",
				"--ruby_cloud_out=/staging",
				"--ruby_cloud_opt=ruby-cloud-gem-name=google-cloud-secret_manager",
				"/googleapis/google/cloud/secretmanager/v1/service.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := buildProtocArgs(test.googleapisDir, test.stagingDir, test.libStagingDir, test.installDir, test.isWrapper, test.serviceConfig, test.gapicOpts, test.protoFiles)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateAPI_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}
	api := &config.API{Path: "non/existent/path"}
	library := &config.Library{Name: "gem-name"}
	err = generateAPI(t.Context(), api, library, nil, googleapisDir, t.TempDir())
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

func TestEscapeRubyCloudOptValue(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "Cloud Asset API",
			want:  "Cloud Asset API",
		},
		{
			name:  "contains commas",
			input: "API keys, passwords, and certificates.",
			want:  `API keys\, passwords\, and certificates.`,
		},
		{
			name:  "contains backslashes and commas",
			input: `path\to\file, and more`,
			want:  `path\\to\\file\, and more`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := escapeRubyCloudOptValue(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteAfterGeneration(t *testing.T) {
	for _, test := range []struct {
		name      string
		api       *config.API
		files     []string
		wantFiles []string
	}{
		{
			name: "nil ruby configuration",
			api:  &config.API{},
			files: []string{
				"google/cloud/secret_manager/v1/version.rb",
			},
			wantFiles: []string{
				"google/cloud/secret_manager/v1/version.rb",
			},
		},
		{
			name: "nil delete generation output paths",
			api: &config.API{
				Ruby: &config.RubyAPI{},
			},
			files: []string{
				"google/cloud/secret_manager/v1/version.rb",
			},
			wantFiles: []string{
				"google/cloud/secret_manager/v1/version.rb",
			},
		},
		{
			name: "empty delete generation output paths",
			api: &config.API{
				Ruby: &config.RubyAPI{
					DeleteGenerationOutputPaths: []string{},
				},
			},
			files: []string{
				"google/cloud/secret_manager/v1/version.rb",
			},
			wantFiles: []string{
				"google/cloud/secret_manager/v1/version.rb",
			},
		},
		{
			name: "delete files and directories",
			api: &config.API{
				Ruby: &config.RubyAPI{
					DeleteGenerationOutputPaths: []string{
						"google/cloud/secret_manager/v1/to_delete.rb",
						"google/cloud/secret_manager/v1/delete_dir",
					},
				},
			},
			files: []string{
				"google/cloud/secret_manager/v1/to_delete.rb",
				"google/cloud/secret_manager/v1/delete_dir/nested.rb",
				"google/cloud/secret_manager/v1/version.rb",
			},
			wantFiles: []string{
				"google/cloud/secret_manager/v1/version.rb",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			stagingDir := t.TempDir()
			libDir := filepath.Join(stagingDir, "lib")
			for _, file := range test.files {
				p := filepath.Join(libDir, file)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte("test"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := deleteAfterGeneration(test.api, stagingDir); err != nil {
				t.Fatal(err)
			}
			var gotFiles []string
			if _, err := os.Stat(libDir); err == nil {
				err := filepath.WalkDir(libDir, func(path string, entry fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if entry.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(libDir, path)
					if err != nil {
						return err
					}
					gotFiles = append(gotFiles, filepath.ToSlash(rel))
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			slices.Sort(gotFiles)
			slices.Sort(test.wantFiles)
			if diff := cmp.Diff(test.wantFiles, gotFiles); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteAfterGeneration_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		api     *config.API
		setup   func(t *testing.T, stagingDir string)
		wantErr error
	}{
		{
			name: "path does not exist",
			api: &config.API{
				Ruby: &config.RubyAPI{
					DeleteGenerationOutputPaths: []string{
						"google/cloud/secret_manager/v1/nonexistent.rb",
					},
				},
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "cannot delete file from read-only directory",
			api: &config.API{
				Ruby: &config.RubyAPI{
					DeleteGenerationOutputPaths: []string{"readonly/file.rb"},
				},
			},
			setup: func(t *testing.T, stagingDir string) {
				readOnlyDir := filepath.Join(stagingDir, "lib", "readonly")
				if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(readOnlyDir, "file.rb"), []byte("test"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(readOnlyDir, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_ = os.Chmod(readOnlyDir, 0o755)
				})
			},
			wantErr: fs.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			stagingDir := t.TempDir()
			if test.setup != nil {
				test.setup(t, stagingDir)
			}
			gotErr := deleteAfterGeneration(test.api, stagingDir)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("deleteAfterGeneration() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestIsWrapperLibrary(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    bool
	}{
		{
			name: "versioned library without wrapper_of",
			library: &config.Library{
				Name: "google-cloud-secret_manager-v1",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: false,
		},
		{
			name: "wrapper library with library wrapper_of",
			library: &config.Library{
				Name: "google-cloud-secret_manager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v1:0.29"},
				},
			},
			want: true,
		},
		{
			name: "wrapper library with api wrapper_of",
			library: &config.Library{
				Name: "google-cloud-secret_manager",
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{
								WrapperOf: "v1:0.29",
							},
						},
					},
				},
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isWrapperLibrary(test.library)
			if got != test.want {
				t.Errorf("isWrapperLibrary() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsMultiWrapper(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		want    bool
	}{
		{
			name: "single wrapper with matching name",
			library: &config.Library{
				Name: "google-cloud-secret_manager",
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v1:0.29"},
				},
			},
			want: false,
		},
		{
			name: "multi wrapper with multiple apis",
			library: &config.Library{
				Name: "google-cloud-workflows",
				APIs: []*config.API{
					{Path: "google/cloud/workflows/v1"},
					{
						Path: "google/cloud/workflows/executions/v1",
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{
								GemName:   "google-cloud-workflows-executions",
								WrapperOf: "v1:1.2",
							},
						},
					},
				},
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v1:2.0"},
				},
			},
			want: true,
		},
		{
			name: "multi wrapper with different main gem",
			library: &config.Library{
				Name: "google-cloud-beyond_corp",
				APIs: []*config.API{
					{
						Path: "google/cloud/beyondcorp/appconnections/v1",
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{
								GemName:   "google-cloud-beyond_corp-app_connections",
								WrapperOf: "v1:0.4",
							},
						},
					},
					{
						Path: "google/cloud/beyondcorp/appconnectors/v1",
						Ruby: &config.RubyAPI{
							RubyCloudOpts: &config.RubyCloudOpts{
								GemName:   "google-cloud-beyond_corp-app_connectors",
								WrapperOf: "v1:0.4",
							},
						},
					},
				},
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v1:0.4"},
				},
			},
			want: true,
		},
		{
			name: "non-wrapper with multiple apis",
			library: &config.Library{
				Name: "google-cloud-secret_manager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v1beta"},
				},
			},
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isMultiWrapper(test.library)
			if got != test.want {
				t.Errorf("isMultiWrapper() = %v, want %v", got, test.want)
			}
		})
	}
}
