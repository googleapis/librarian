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
	"github.com/googleapis/librarian/internal/config"
)

func TestAPIsNoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, api := range APIs {
		if seen[api.Path] {
			t.Errorf("duplicate API path: %s", api.Path)
		}
		seen[api.Path] = true
	}
}

func TestAPIsAlphabeticalOrder(t *testing.T) {
	for i := 1; i < len(APIs); i++ {
		prev := APIs[i-1].Path
		curr := APIs[i].Path
		if prev > curr {
			t.Errorf("APIs not in alphabetical order: %q comes after %q", prev, curr)
		}
	}
}

func TestHasAPIPath(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want bool
	}{
		{"matching path and language", "google/api", true},
		{"unknown path", "google/does/not/exist/v1", false},
		{"empty path", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := HasAPIPath(test.path)
			if got != test.want {
				t.Errorf("HasAPIPath(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestGetTransport(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *API
		lang string
		want Transport
	}{
		{
			name: "empty serviceconfig",
			sc:   &API{},
			lang: config.LanguageGo,
			want: GRPCRest,
		},
		{
			name: "go specific transport",
			sc: &API{
				Transports: map[string]Transport{
					config.LanguageGo: GRPC,
				},
			},
			lang: config.LanguageGo,
			want: GRPC,
		},
		{
			name: "other language transport",
			sc: &API{
				Transports: map[string]Transport{
					config.LanguageGo: GRPC,
				},
			},
			lang: config.LanguagePython,
			want: GRPCRest,
		},
		{
			name: "all language transport",
			sc: &API{
				Transports: map[string]Transport{
					config.LanguageAll: GRPC,
				},
			},
			lang: config.LanguageGo,
			want: GRPC,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.sc.Transport(test.lang)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
