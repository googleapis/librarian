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
	apiPath := "google/cloud/secretmanager/v1"
	params, err := newInitParams(googleapisDir, apiPath)
	if err != nil {
		t.Fatalf("newInitParams failed: %v", err)
	}
	want := &initParams{
		apiShortName:    "secretmanager",
		productDocs:     "https://cloud.google.com/secret-manager/docs/overview",
		productHomepage: "https://cloud.google.com/secret-manager/",
	}
	if diff := cmp.Diff(want, params, cmp.AllowUnexported(initParams{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
