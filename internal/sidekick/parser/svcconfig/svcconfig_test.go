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

package svcconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/sample"
)

func TestExtractPackageName(t *testing.T) {
	for _, test := range []struct {
		Input *serviceconfig.Service
		Want  *ServiceNames
	}{
		{nil, nil},
		{&serviceconfig.Service{}, nil},
		{sample.ServiceConfig(), &ServiceNames{"google.cloud.secretmanager.v1", "SecretManagerService"}},
	} {
		got := ExtractPackageName(test.Input)
		if diff := cmp.Diff(test.Want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestSplit(t *testing.T) {
	for _, test := range []struct {
		Input       string
		WantPackage string
		WantName    string
	}{
		{"google.cloud.location.Location", "google.cloud.location", "Location"},
		{"Service", "", "Service"},
	} {
		got := splitQualifiedServiceName(test.Input)
		if got.PackageName != test.WantPackage {
			t.Errorf("mismatched package, want=%q, got=%q", test.WantPackage, got.PackageName)
		}
		if got.ServiceName != test.WantName {
			t.Errorf("mismatched name, want=%q, got=%q", test.WantName, got.ServiceName)
		}
	}
}

func TestMixin(t *testing.T) {
	for _, test := range []struct {
		Input string
		Want  bool
	}{
		{"google.cloud.location.Location", true},
		{"google.longrunning.Operations", true},
		{"google.iam.v1.IAMPolicy", true},
		{"google.storage.v2.Storage", false},
	} {
		got := wellKnownMixin(test.Input)
		if got != test.Want {
			t.Errorf("mismatched WellKnownMixin, want=%v, got=%v", test.Want, got)
		}
	}
}
