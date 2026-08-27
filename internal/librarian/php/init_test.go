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

package php

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestNamespace(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "php_namespace option",
			content: `option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			want:    `Google\Cloud\SecretManager`,
		},
		{
			name:    "extra whitespace",
			content: `option php_namespace   =   "Google\\Cloud\\Storage\\V2beta";`,
			want:    `Google\Cloud\Storage`,
		},
		{
			name:    "ignore comments",
			content: `// option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			want:    `Google\Cloud\Test`,
		},
		{
			name:    "no php_namespace option",
			content: `syntax = "proto3";`,
			want:    `Google\Cloud\Test`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			apiPath := "google/cloud/test/v1"
			dir := filepath.Join(tmpDir, apiPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(dir, "service.proto")
			if err := os.WriteFile(file, []byte(test.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := namespace(tmpDir, apiPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNamespace_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		setup   func(t *testing.T, tmpDir string) string
		wantErr error
	}{
		{
			name: "missing api directory",
			setup: func(t *testing.T, tmpDir string) string {
				return "google/cloud/nonexistent/v1"
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "no proto files in directory",
			setup: func(t *testing.T, tmpDir string) string {
				apiPath := "google/cloud/test/v1"
				dir := filepath.Join(tmpDir, apiPath)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a proto"), 0o644); err != nil {
					t.Fatal(err)
				}
				return apiPath
			},
			wantErr: fs.ErrNotExist,
		},
		{
			// this test failed because fake.proto is a directory, not a file.
			name: "ignore directory with proto extension",
			setup: func(t *testing.T, tmpDir string) string {
				apiPath := "google/cloud/test/v1"
				dir := filepath.Join(tmpDir, apiPath, "fake.proto")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				return apiPath
			},
			wantErr: fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			apiPath := test.setup(t, tmpDir)
			_, err := namespace(tmpDir, apiPath)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("namespace() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestComponentName(t *testing.T) {
	for _, test := range []struct {
		name      string
		library   *config.Library
		namespace string
		want      string
	}{
		{
			name:      "google cloud component",
			library:   &config.Library{},
			namespace: `Google\Cloud\SecretManager`,
			want:      "SecretManager",
		},
		{
			name:      "google ads",
			library:   &config.Library{},
			namespace: `Google\Ads\GoogleAds`,
			want:      "AdsGoogleAds",
		},
		{
			name:      "google shopping",
			library:   &config.Library{},
			namespace: `Google\Shopping\Merchant\Conversions`,
			want:      "ShoppingMerchantConversions",
		},
		{
			name: "component name override",
			library: &config.Library{
				PHP: &config.PHPPackage{
					ComponentName: "CustomComponentName",
				},
			},
			namespace: `Google\Cloud\SecretManager`,
			want:      "CustomComponentName",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := componentName(test.library, test.namespace)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewInitParams(t *testing.T) {
	t.Parallel()
	googleapisDir := filepath.Join("..", "..", "testdata", "googleapis")
	for _, test := range []struct {
		name    string
		library *config.Library
		want    *initParams
	}{
		{
			name: "default derived protoPackage",
			library: &config.Library{
				APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			},
			want: &initParams{
				componentName:   "SecretManager",
				phpNamespace:    `Google\Cloud\SecretManager`,
				apiShortName:    "secretmanager",
				productDocs:     "https://cloud.google.com/secret-manager/docs/overview",
				productHomepage: "https://cloud.google.com/secret-manager/",
				protoPackage:    "google.cloud.secretmanager",
				apiVersion:      "v1",
			},
		},
		{
			name: "custom protoPackage override",
			library: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP: &config.PHPAPI{
							ProtoPackage: "google.cloud.secrets",
						},
					},
				},
			},
			want: &initParams{
				componentName:   "SecretManager",
				phpNamespace:    `Google\Cloud\SecretManager`,
				apiShortName:    "secretmanager",
				productDocs:     "https://cloud.google.com/secret-manager/docs/overview",
				productHomepage: "https://cloud.google.com/secret-manager/",
				protoPackage:    "google.cloud.secrets",
				apiVersion:      "v1",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			params, err := newInitParams(googleapisDir, test.library)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, params, cmp.AllowUnexported(initParams{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInitComponentIfMissing(t *testing.T) {
	googleapisDir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "googleapis"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name          string
		library       *config.Library
		setup         func(t *testing.T, repoRoot string)
		wantComponent string
		wantInit      bool
	}{
		{
			name: "component already exists",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			setup: func(t *testing.T, repoRoot string) {
				if err := os.MkdirAll(filepath.Join(repoRoot, "SecretManager"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantComponent: "SecretManager",
			wantInit:      false,
		},
		{
			name: "new component initialized",
			library: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			setup: func(t *testing.T, repoRoot string) {
				devDir := filepath.Join(repoRoot, "dev")
				if err := os.MkdirAll(devDir, 0o755); err != nil {
					t.Fatal(err)
				}
				mockScript := filepath.Join(devDir, "google-cloud")
				scriptContent := "#!/bin/sh\ntouch initialized.txt\n"
				if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantComponent: "SecretManager",
			wantInit:      true,
		},
		{
			name: "new component with component name override",
			library: &config.Library{
				Name: "secretmanager",
				PHP: &config.PHPPackage{
					ComponentName: "CustomSecretManager",
				},
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			setup: func(t *testing.T, repoRoot string) {
				devDir := filepath.Join(repoRoot, "dev")
				if err := os.MkdirAll(devDir, 0o755); err != nil {
					t.Fatal(err)
				}
				mockScript := filepath.Join(devDir, "google-cloud")
				scriptContent := "#!/bin/sh\ntouch initialized.txt\n"
				if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantComponent: "CustomSecretManager",
			wantInit:      true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			t.Chdir(repoRoot)
			if test.setup != nil {
				test.setup(t, repoRoot)
			}
			got, err := initComponentIfMissing(t.Context(), test.library, googleapisDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantComponent, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			_, statErr := os.Stat(filepath.Join(repoRoot, "initialized.txt"))
			wasInitialized := statErr == nil
			if wasInitialized != test.wantInit {
				t.Errorf("wasInitialized = %v, wantInit = %v", wasInitialized, test.wantInit)
			}
		})
	}
}

func TestInitComponentIfMissing_Error(t *testing.T) {
	googleapisDir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "googleapis"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		library *config.Library
		setup   func(t *testing.T, repoRoot string)
		wantErr error
	}{
		{
			name: "no apis configured in library",
			library: &config.Library{
				Name: "empty",
			},
			wantErr: errNoAPIs,
		},
		{
			name: "api service config not found",
			library: &config.Library{
				Name: "nonexistent",
				APIs: []*config.API{
					{Path: "google/cloud/nonexistent/v1"},
				},
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "stat error other than not exist",
			library: &config.Library{
				Name: "secretmanager",
				PHP: &config.PHPPackage{
					ComponentName: "unreadable/SecretManager",
				},
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			setup: func(t *testing.T, repoRoot string) {
				unreadableDir := filepath.Join(repoRoot, "unreadable")
				if err := os.MkdirAll(unreadableDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(unreadableDir, 0o000); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_ = os.Chmod(unreadableDir, 0o755)
				})
			},
			wantErr: fs.ErrPermission,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			t.Chdir(repoRoot)
			if test.setup != nil {
				test.setup(t, repoRoot)
			}
			_, err := initComponentIfMissing(t.Context(), test.library, googleapisDir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("initComponentIfMissing() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestInitComponent(t *testing.T) {
	ctx := t.Context()
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	devDir := filepath.Join(repoRoot, "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mockScript := filepath.Join(devDir, "google-cloud")
	scriptContent := `#!/bin/sh
echo "$@" > dev_output.txt
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatal(err)
	}
	params := &initParams{
		componentName:   "Speech",
		phpNamespace:    `Google\Cloud\Speech\V2`,
		protoPackage:    "google.cloud.speech.v2",
		apiShortName:    "speech",
		apiVersion:      "v2",
		productDocs:     "https://cloud.google.com/speech-to-text/docs",
		productHomepage: "https://cloud.google.com/speech-to-text",
	}
	if err := initComponent(ctx, params); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := os.ReadFile(filepath.Join(repoRoot, "dev_output.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(gotBytes))
	want := `component:new --no-update --component-name=Speech --php-namespace=Google\Cloud\Speech\V2 --proto-package=google.cloud.speech.v2 --api-short-name=speech --api-version=v2 --product-docs=https://cloud.google.com/speech-to-text/docs --product-homepage=https://cloud.google.com/speech-to-text`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestInitComponent_Error(t *testing.T) {
	ctx := t.Context()
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	devDir := filepath.Join(repoRoot, "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mockScript := filepath.Join(devDir, "google-cloud")
	scriptContent := `#!/bin/sh
exit 1
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatal(err)
	}
	params := &initParams{
		componentName: "Speech",
	}
	if err := initComponent(ctx, params); err == nil {
		t.Fatal("initComponent() expected error, got nil")
	}
}

func TestProtoPackage(t *testing.T) {
	for _, test := range []struct {
		name string
		api  *config.API
		want string
	}{
		{
			name: "speech v2",
			api: &config.API{
				Path: "google/cloud/speech/v2",
			},
			want: "google.cloud.speech",
		},
		{
			name: "privateca v1",
			api: &config.API{
				Path: "google/cloud/security/privateca/v1",
			},
			want: "google.cloud.security.privateca",
		},
		{
			name: "generativelanguage v1alpha",
			api: &config.API{
				Path: "google/ai/generativelanguage/v1alpha",
			},
			want: "google.ai.generativelanguage",
		},
		{
			name: "unversioned path",
			api: &config.API{
				Path: "google/identity/accesscontextmanager/type",
			},
			want: "google.identity.accesscontextmanager.type",
		},
		{
			name: "protoPackage override",
			api: &config.API{
				Path: "google/cloud/speech/v2",
				PHP: &config.PHPAPI{
					ProtoPackage: "google.cloud.speech.custom",
				},
			},
			want: "google.cloud.speech.custom",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := protoPackage(test.api)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestComponentNameForLibrary(t *testing.T) {
	googleapisDir := filepath.Join("..", "..", "testdata", "googleapis")
	for _, test := range []struct {
		name    string
		library *config.Library
		want    string
	}{
		{
			name: "derived from proto namespace",
			library: &config.Library{
				Name: "SecretManager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: "SecretManager",
		},
		{
			name: "explicit config override",
			library: &config.Library{
				Name: "AccessContextManager",
				PHP: &config.PHPPackage{
					ComponentName: "AccessContextManager",
				},
				APIs: []*config.API{
					{Path: "google/identity/accesscontextmanager/v1"},
				},
			},
			want: "AccessContextManager",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ComponentNameForLibrary(googleapisDir, test.library)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
