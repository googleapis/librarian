// Copyright 2025 Google LLC
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

package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDisco_Parse(t *testing.T) {
	// Mixing Compute and Secret Manager like this is fine in tests.
	got, err := ParseDisco(discoSourceFile, secretManagerYamlFullPath, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	wantName := "secretmanager"
	wantTitle := "Secret Manager API"
	wantDescription := "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security."
	wantPackageName := "google.cloud.secretmanager.v1"
	if got.Name != wantName {
		t.Errorf("want = %q; got = %q", wantName, got.Name)
	}
	if got.Title != wantTitle {
		t.Errorf("want = %q; got = %q", wantTitle, got.Title)
	}
	if diff := cmp.Diff(got.Description, wantDescription); diff != "" {
		t.Errorf("description mismatch (-want, +got):\n%s", diff)
	}
	if got.PackageName != wantPackageName {
		t.Errorf("want = %q; got = %q", wantPackageName, got.PackageName)
	}
}

func TestDisco_ParseNoServiceConfig(t *testing.T) {
	got, err := ParseDisco(discoSourceFile, "", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	wantName := "compute"
	wantTitle := "Compute Engine API"
	wantDescription := "Creates and runs virtual machines on Google Cloud Platform. "
	if got.Name != wantName {
		t.Errorf("want = %q; got = %q", wantName, got.Name)
	}
	if got.Title != wantTitle {
		t.Errorf("want = %q; got = %q", wantTitle, got.Title)
	}
	if diff := cmp.Diff(got.Description, wantDescription); diff != "" {
		t.Errorf("description mismatch (-want, +got):\n%s", diff)
	}
}

func TestDisco_ParseBadFiles(t *testing.T) {
	if _, err := ParseDisco("-invalid-file-name-", secretManagerYamlFullPath, map[string]string{}); err == nil {
		t.Fatalf("expected error with invalid source file name")
	}

	if _, err := ParseDisco(discoSourceFile, "-invalid-file-name-", map[string]string{}); err == nil {
		t.Fatalf("expected error with invalid service config yaml file name")
	}
}
