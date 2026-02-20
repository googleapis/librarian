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

package api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsResourceRenameHeuristicEligible(t *testing.T) {
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
			name:      "compute exact match is eligible",
			serviceID: ".google.cloud.compute",
			want:      true,
		},
		{
			name:      "sql exact match is eligible",
			serviceID: ".google.cloud.sql",
			want:      true,
		},
		{
			name:      "too short service id is not eligible",
			serviceID: ".google.cloud",
			want:      false,
		},
		{
			name:      "non-eligible service with enough parts",
			serviceID: ".google.cloud.other.v1",
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
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("IsHeuristicEligible(%q) mismatch (-want +got):\n%s", test.serviceID, diff)
			}
		})
	}
}

func TestIsCollectionIdentifier(t *testing.T) {
	for _, test := range []struct {
		segment      string
		knownPlurals map[string]bool
		want         bool
	}{
		// Base vocabulary
		{"projects", nil, true},
		{"locations", nil, true},
		{"folders", nil, true},
		{"organizations", nil, true},
		{"billingAccounts", nil, true},

		// Standard plural heuristic
		{"instances", nil, true},
		{"disks", nil, true},
		{"clusters", nil, true},
		{"backups", nil, true},
		{"vms", nil, true},
		{"ips", nil, true},

		// Ignored / Invalid
		{"v1", nil, false},       // Version
		{"us", nil, false},       // Region
		{"address", nil, false},  // Singular exception
		{"status", nil, false},   // Singular exception
		{"ingress", nil, false},  // Singular exception
		{"egress", nil, false},   // Singular exception
		{"access", nil, false},   // Singular exception
		{"analysis", nil, false}, // Singular exception
		{"other", nil, false},    // Random noun not ending in s
		{"s", nil, false},        // Too short
		{"", nil, false},         // Empty

		// Known Plurals (Explicit Match)
		{"fish", map[string]bool{"fish": true}, true},     // Doesn't end in s, but known
		{"people", map[string]bool{"people": true}, true}, // Irregular plural
		{"data", map[string]bool{"data": true}, true},     // Mass noun
		{"status", map[string]bool{"status": true}, true}, // Exception override

		// Known Plurals (No Match / False Cases)
		{"fish", nil, false},                                  // Not known, no 's' suffix -> false
		{"fish", map[string]bool{"sharks": true}, false},      // Map populated, but key missing -> false
		{"status", map[string]bool{"instances": true}, false}, // Exception applies if not in map -> false
	} {
		t.Run(test.segment, func(t *testing.T) {
			got := isCollectionIdentifier(test.segment, test.knownPlurals)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("isCollectionIdentifier(%q) mismatch (-want +got):\n%s", test.segment, diff)
			}
		})
	}
}
