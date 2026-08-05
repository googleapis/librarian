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
		namespace string
		want      string
	}{
		{
			name:      "google cloud component",
			namespace: `Google\Cloud\SecretManager`,
			want:      "SecretManager",
		},
		{
			name:      "google ads",
			namespace: `Google\Ads\GoogleAds`,
			want:      "AdsGoogleAds",
		},
		{
			name:      "google shopping",
			namespace: `Google\Shopping\Merchant\Conversions`,
			want:      "ShoppingMerchantConversions",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := componentName(test.namespace)
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
		name string
		api  *config.API
		want *initParams
	}{
		{
			name: "default derived protoPackage",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
			},
			want: &initParams{
				apiShortName:    "secretmanager",
				productDocs:     "https://cloud.google.com/secret-manager/docs/overview",
				productHomepage: "https://cloud.google.com/secret-manager/",
				protoPackage:    "google.cloud.secretmanager",
				apiVersion:      "v1",
			},
		},
		{
			name: "custom protoPackage override",
			api: &config.API{
				Path: "google/cloud/secretmanager/v1",
				PHP: &config.PHPAPI{
					ProtoPackage: "google.cloud.secrets",
				},
			},
			want: &initParams{
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
			params, err := newInitParams(googleapisDir, test.api)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, params, cmp.AllowUnexported(initParams{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
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
