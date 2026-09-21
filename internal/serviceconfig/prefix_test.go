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

package serviceconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMatchPrefix(t *testing.T) {
	type result struct {
		Value     string
		Remainder string
		OK        bool
	}
	prefixes := map[string]string{
		"google/shopping/merchant": "@google-shopping",
		"google/shopping":          "@google-shopping",
		"google/maps/isochrones":   "@google-maps",
		"google/maps":              "@googlemaps",
		"google/chat":              "@google-apps",
		"google/area120":           "@google/area120",
	}
	for _, test := range []struct {
		name     string
		apiPath  string
		prefixes map[string]string
		want     result
	}{
		{
			name:     "empty apiPath",
			apiPath:  "",
			prefixes: prefixes,
			want:     result{},
		},
		{
			name:     "nil prefixes",
			apiPath:  "google/shopping/v1",
			prefixes: nil,
			want:     result{},
		},
		{
			name:     "empty prefixes",
			apiPath:  "google/shopping/v1",
			prefixes: map[string]string{},
			want:     result{},
		},
		{
			name:     "nested prefix matches longest prefix",
			apiPath:  "google/shopping/merchant/accounts/v1",
			prefixes: prefixes,
			want:     result{Value: "@google-shopping", Remainder: "accounts", OK: true},
		},
		{
			name:     "fallback to shorter prefix when nested prefix does not match",
			apiPath:  "google/shopping/css/v1",
			prefixes: prefixes,
			want:     result{Value: "@google-shopping", Remainder: "css", OK: true},
		},
		{
			name:     "exact match without remainder",
			apiPath:  "google/chat/v1",
			prefixes: prefixes,
			want:     result{Value: "@google-apps", Remainder: "", OK: true},
		},
		{
			name:     "specific sub-path override",
			apiPath:  "google/maps/isochrones/v1",
			prefixes: prefixes,
			want:     result{Value: "@google-maps", Remainder: "", OK: true},
		},
		{
			name:     "general prefix with remainder",
			apiPath:  "google/maps/routing/v2",
			prefixes: prefixes,
			want:     result{Value: "@googlemaps", Remainder: "routing", OK: true},
		},
		{
			name:     "multi-level remainder",
			apiPath:  "google/maps/fleetengine/delivery/v1",
			prefixes: prefixes,
			want:     result{Value: "@googlemaps", Remainder: "fleetengine/delivery", OK: true},
		},
		{
			name:     "exact match including version",
			apiPath:  "google/exact/v1",
			prefixes: map[string]string{"google/exact/v1": "com.google.exact"},
			want:     result{Value: "com.google.exact", Remainder: "", OK: true},
		},
		{
			name:     "no matching prefix",
			apiPath:  "google/cloud/secretmanager/v1",
			prefixes: prefixes,
			want:     result{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			val, rem, ok := MatchPrefix(test.apiPath, test.prefixes)
			got := result{Value: val, Remainder: rem, OK: ok}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
