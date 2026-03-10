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
)

func TestRepoMetadata_write(t *testing.T) {
	want := &repoMetadata{
		APIShortname:         "secretmanager",
		NamePretty:           "Secret Manager",
		ProductDocumentation: "https://cloud.google.com/secret-manager/",
		APIDescription:       "Stores sensitive data such as API keys, passwords, and certificates. Provides convenience while improving security.",
		ClientDocumentation:  "https://cloud.google.com/java/docs/reference/google-cloud-secretmanager/latest/overview",
		ReleaseLevel:         "stable",
		Transport:            "grpc",
		Language:             "java",
		Repo:                 "googleapis/google-cloud-java",
		RepoShort:            "java-secretmanager",
		DistributionName:     "com.google.cloud:google-cloud-secretmanager",
		LibraryType:          "GAPIC_AUTO",
		CodeownerTeam:        "cloud-java-team",
		IssueTracker:         "https://issuetracker.google.com/issues/new?component=187210&template=0",
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
  "api_description": "Stores sensitive data such as API keys, passwords, and certificates. Provides convenience while improving security.",
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
  "issue_tracker": "https://issuetracker.google.com/issues/new?component=187210\u0026template=0",
  "rest_documentation": "https://example.com/rest",
  "rpc_documentation": "https://example.com/rpc",
  "recommended_package": "com.google.cloud.secretmanager.v1",
  "min_java_version": 8
}`
	if diff := cmp.Diff(wantJSON, string(gotBytes)); diff != "" {
		t.Errorf("write() mismatch (-want +got):\n%s", diff)
	}
}
