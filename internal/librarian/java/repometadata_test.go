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

package java

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sample"
)

func TestRepoMetadata_write(t *testing.T) {
	s := sample.RepoMetadata()
	want := &repoMetadata{
		APIShortname:         s.APIShortname,
		NamePretty:           s.NamePretty,
		ProductDocumentation: s.ProductDocumentation,
		APIDescription:       s.APIDescription,
		ClientDocumentation:  "https://cloud.google.com/java/docs/reference/google-cloud-secretmanager/latest/overview",
		ReleaseLevel:         s.ReleaseLevel,
		Transport:            "grpc",
		Language:             "java",
		Repo:                 "googleapis/google-cloud-java",
		RepoShort:            "java-secretmanager",
		DistributionName:     "com.google.cloud:google-cloud-secretmanager",
		LibraryType:          s.LibraryType,
		CodeownerTeam:        "cloud-java-team",
		IssueTracker:         s.IssueTracker,
		RestDocumentation:    "https://example.com/rest",
		RpcDocumentation:     "https://example.com/rpc",
		RecommendedPackage:   "com.google.cloud.secretmanager.v1",
		MinJavaVersion:       8,
	}
	tmpDir := t.TempDir()
	err := want.write(tmpDir)
	if err != nil {
		t.Fatalf("write() = %v, want nil", err)
	}

	gotPath := filepath.Join(tmpDir, ".repo-metadata.json")
	if _, err := os.Stat(gotPath); err != nil {
		t.Fatalf("os.Stat(%q) = %v, want nil", gotPath, err)
	}

	gotBytes, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) = %v, want nil", gotPath, err)
	}

	const wantJSON = `{
  "api_shortname": "secretmanager",
  "name_pretty": "Secret Manager",
  "product_documentation": "https://cloud.google.com/secret-manager/",
  "api_description": "Stores sensitive data such as API keys, passwords, and certificates.\nProvides convenience while improving security.",
  "client_documentation": "https://cloud.google.com/java/docs/reference/google-cloud-secretmanager/latest/overview",
  "release_level": "stable",
  "transport": "grpc",
  "language": "java",
  "repo": "googleapis/google-cloud-java",
  "repo_short": "java-secretmanager",
  "distribution_name": "com.google.cloud:google-cloud-secretmanager",
  "library_type": "GAPIC_AUTO",
  "requires_billing": false,
  "codeowner_team": "cloud-java-team",
  "issue_tracker": "https://issuetracker.google.com/issues/new?component=784854\u0026template=1380926",
  "rest_documentation": "https://example.com/rest",
  "rpc_documentation": "https://example.com/rpc",
  "recommended_package": "com.google.cloud.secretmanager.v1",
  "min_java_version": 8
}`
	if diff := cmp.Diff(wantJSON, string(gotBytes)); diff != "" {
		t.Errorf("write() mismatch (-want +got):\n%s", diff)
	}
}

func TestDeriveRepoMetadata_Overrides(t *testing.T) {
	t.Parallel()
	apiPath := "google/cloud/secretmanager/v1"
	googleapis := "internal/testdata/googleapis"

	cfg := sample.Config()
	cfg.Language = config.LanguageJava
	cfg.Repo = "googleapis/google-cloud-java"
	library := &config.Library{
		Name: "secretmanager",
		APIs: []*config.API{{Path: apiPath}},
		Java: &config.JavaModule{
			GroupID:                      "com.custom",
			DistributionNameOverride:     "com.custom:custom-artifact",
			APIIDOverride:                "custom.googleapis.com",
			APIDescriptionOverride:       "Custom description",
			NamePrettyOverride:           "Custom Pretty Name",
			ProductDocumentationOverride: "https://custom.docs",
			ClientDocumentationOverride:  "https://custom.client.docs",
			BillingNotRequired:           true,
			LibraryTypeOverride:          "OTHER",
		},
	}
	got, err := deriveRepoMetadata(cfg, library, googleapis)
	if err != nil {
		t.Fatalf("deriveRepoMetadata failed: %v", err)
	}
	s := sample.RepoMetadata()
	want := &repoMetadata{
		NamePretty:           "Custom Pretty Name",
		ProductDocumentation: "https://custom.docs",
		APIDescription:       "Custom description",
		ClientDocumentation:  "https://custom.client.docs",
		ReleaseLevel:         s.ReleaseLevel,
		Transport:            "both",
		Language:             cfg.Language,
		Repo:                 cfg.Repo,
		RepoShort:            "java-secretmanager",
		DistributionName:     "com.custom:custom-artifact",
		APIID:                "custom.googleapis.com",
		LibraryType:          "OTHER",
		RequiresBilling:      false,
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(repoMetadata{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
