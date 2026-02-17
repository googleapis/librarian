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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsHeuristicEligible(t *testing.T) {
	for _, test := range []struct {
		name      string
		serviceID string
		want      bool
	}{
		{
			name:      "compute v1 is eligible",
			serviceID: ".google.cloud.compute.v1.Instances",
			want:      true,
		},
		{
			name:      "compute v1beta1 is eligible",
			serviceID: ".google.cloud.compute.v1beta1.Instances",
			want:      true,
		},
		{
			name:      "sql v1 is eligible",
			serviceID: ".google.cloud.sql.v1.Instances",
			want:      true,
		},
		{
			name:      "bigquery v2 is eligible",
			serviceID: ".google.cloud.bigquery.v2.TableService",
			want:      true,
		},
		{
			name:      "kms is not eligible",
			serviceID: ".google.cloud.kms.v1.KeyManagementService",
			want:      false,
		},
		{
			name:      "pubsub is not eligible",
			serviceID: ".google.cloud.pubsub.v1.Publisher",
			want:      false,
		},
		{
			name:      "empty service id is not eligible",
			serviceID: "",
			want:      false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := IsHeuristicEligible(test.serviceID)
			if got != test.want {
				t.Errorf("IsHeuristicEligible(%q) = %v, want %v", test.serviceID, got, test.want)
			}
		})
	}
}

func TestBaseVocabulary(t *testing.T) {
	got := BaseVocabulary()
	want := map[string]bool{
		"projects":        true,
		"locations":       true,
		"folders":         true,
		"organizations":   true,
		"billingAccounts": true,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
