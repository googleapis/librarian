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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/internal/api"
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
	if diff := cmp.Diff(wantDescription, got.Description); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if got.PackageName != wantPackageName {
		t.Errorf("want = %q; got = %q", wantPackageName, got.PackageName)
	}
}

func TestDisco_FindSources(t *testing.T) {
	got, err := ParseDisco(discoSourceFileRelative, "", map[string]string{
		"test-root": testdataDir,
		"roots":     "undefined,test",
	})
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
	if diff := cmp.Diff(wantDescription, got.Description); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
	if diff := cmp.Diff(wantDescription, got.Description); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDisco_ParsePagination(t *testing.T) {
	model, err := ParseDisco(discoSourceFile, "", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	updateMethodPagination(nil, model)
	wantID := "..zones.list"
	got, ok := model.State.MethodByID[wantID]
	if !ok {
		t.Fatalf("expected method %s in the API model", wantID)
	}
	wantPagination := &api.Field{
		Name:     "pageToken",
		JSONName: "pageToken",
		ID:       "..zones.listRequest.pageToken",
		Typez:    api.STRING_TYPE,
		TypezID:  "string",
		Optional: true,
	}
	if diff := cmp.Diff(wantPagination, got.Pagination, cmpopts.IgnoreFields(api.Field{}, "Documentation")); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestDisco_ParseDeprecatedEnum(t *testing.T) {
	model, err := ParseDisco(discoSourceFile, "", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	wantEnum := &api.Enum{
		ID: "..AcceleratorTypeAggregatedList.warning.code",
	}
	got, ok := model.State.EnumByID[wantEnum.ID]
	if !ok {
		t.Fatalf("expected method %s in the API model", wantEnum.ID)
	}
	if len(got.Values) < 7 {
		t.Fatalf("expected at least 7 values in the enum value list, got=%v", got)
	}
	if !got.Values[6].Deprecated {
		t.Errorf("expected a deprecated enum value, got=%v", got.Values[6])
	}
}

func TestDisco_ParseBadFiles(t *testing.T) {
	if _, err := ParseDisco("-invalid-file-name-", secretManagerYamlFullPath, map[string]string{}); err == nil {
		t.Fatalf("expected error with missing source file")
	}

	if _, err := ParseDisco(discoSourceFile, "-invalid-file-name-", map[string]string{}); err == nil {
		t.Fatalf("expected error with missing service config yaml file")
	}

	if _, err := ParseDisco(secretManagerYamlFullPath, secretManagerYamlFullPath, map[string]string{}); err == nil {
		t.Fatalf("expected error with invalid source file contents")
	}
}
