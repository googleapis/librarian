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

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
)

func TestBuildNodejsLibraries(t *testing.T) {
	got, err := buildNodejsLibraries("testdata/google-cloud-node", "testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	want := []*config.Library{
		{
			Name:    "google-cloud-secretmanager",
			Version: "6.1.0",
			APIs: []*config.API{
				{Path: "google/cloud/secretmanager/v1"},
			},
			Nodejs: &config.NodejsPackage{
				ExtraProtocParameters: []string{"metadata"},
				PackageName:           "@google-cloud/secret-manager",
			},
		},
		{
			Name:    "google-cloud-speech",
			Version: "7.2.0",
			APIs: []*config.API{
				{Path: "google/cloud/speech/v1"},
			},
			Nodejs: &config.NodejsPackage{
				Dependencies: map[string]string{
					"@google-cloud/common": "^6.0.0",
					"pumpify":              "^2.0.1",
				},
				ExtraProtocParameters: []string{"metadata"},
			},
		},
		{
			Name:    "google-cloud-translate",
			Version: "9.1.0",
			APIs: []*config.API{
				{Path: "google/cloud/translate/v3"},
			},
			Nodejs: &config.NodejsPackage{
				BundleConfig: "google/cloud/translate/v3/translate_gapic.yaml",
				ExtraProtocParameters: []string{
					"metadata",
					"auto-populate-field-oauth-scope",
				},
				HandwrittenLayer: true,
				MainService:      "translate",
				Mixins:           "none",
			},
		},
		{
			Name:    "google-cloud-workstations",
			Version: "1.3.0",
			APIs: []*config.API{
				{Path: "google/cloud/workstations/v1"},
			},
			Nodejs: &config.NodejsPackage{
				ExtraProtocParameters: []string{"metadata"},
			},
		},
	}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(func(a, b *config.Library) bool { return a.Name < b.Name })); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDeriveNpmPackageName(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard",
			input: "google-cloud-batch",
			want:  "@google-cloud/batch",
		},
		{
			name:  "multi-word suffix",
			input: "google-cloud-access-approval",
			want:  "@google-cloud/access-approval",
		},
		{
			name:  "no second dash",
			input: "google",
			want:  "google",
		},
		{
			name:  "one dash only",
			input: "google-cloud",
			want:  "google-cloud",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveNpmPackageName(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseBazelNodejsInfo(t *testing.T) {
	for _, test := range []struct {
		name string
		api  string
		want *nodejsGapicInfo
	}{
		{
			name: "secretmanager",
			api:  "google/cloud/secretmanager/v1",
			want: &nodejsGapicInfo{
				packageName:           "@google-cloud/secret-manager",
				extraProtocParameters: []string{"metadata"},
			},
		},
		{
			name: "translate with all fields",
			api:  "google/cloud/translate/v3",
			want: &nodejsGapicInfo{
				packageName:  "@google-cloud/translate",
				bundleConfig: "google/cloud/translate/v3/translate_gapic.yaml",
				extraProtocParameters: []string{
					"metadata",
					"auto-populate-field-oauth-scope",
				},
				handwrittenLayer: true,
				mainService:      "translate",
				mixins:           "none",
			},
		},
		{
			name: "no nodejs rule",
			api:  "google/cloud/no-gapic",
			want: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseBazelNodejsInfo("testdata/googleapis", test.api)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(nodejsGapicInfo{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
