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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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

func TestParseProto(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		content string
		wantPkg string
		wantNS  string
	}{
		{
			name: "with php_namespace option",
			content: `package google.cloud.secretmanager.v1;
option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			wantPkg: `google.cloud.secretmanager`,
			wantNS:  `Google\Cloud\SecretManager`,
		},
		{
			name: "extra whitespace",
			content: `package   google.cloud.storage.v2beta  ;
option php_namespace   =   "Google\\Cloud\\Storage\\V2beta";`,
			wantPkg: `google.cloud.storage`,
			wantNS:  `Google\Cloud\Storage`,
		},
		{
			name: "ignore comments",
			content: `// package google.ignored.v1;
package google.cloud.test.v1;
// option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			wantPkg: `google.cloud.test`,
			wantNS:  `Google\Cloud\Test`,
		},
		{
			name: "no php_namespace option",
			content: `syntax = "proto3";
package google.cloud.test.v1;`,
			wantPkg: `google.cloud.test`,
			wantNS:  `Google\Cloud\Test`,
		},
		{
			name: "without version suffix",
			content: `package google.backstory;
option php_namespace = "Google\\Backstory";`,
			wantPkg: `google.backstory`,
			wantNS:  `Google\Backstory`,
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
			gotPkg, gotNS, err := parseProto(tmpDir, apiPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantPkg, gotPkg); diff != "" {
				t.Errorf("pkg mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantNS, gotNS); diff != "" {
				t.Errorf("namespace mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
