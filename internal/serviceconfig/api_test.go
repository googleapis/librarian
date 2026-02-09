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

func TestGetReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *API
		lang string
		want string
	}{
		{
			name: "empty serviceconfig",
			sc:   &API{},
			lang: "go",
			want: "",
		},
		{
			name: "go specific release level",
			sc: &API{
				ReleaseLevels: map[string]string{
					"go": "beta",
				},
			},
			lang: "go",
			want: "beta",
		},
		{
			name: "other language transport",
			sc: &API{
				ReleaseLevels: map[string]string{
					"go": "beta",
				},
			},
			lang: "python",
			want: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.sc.ReleaseLevel(test.lang)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetTransport(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *API
		lang string
		want string
	}{
		{
			name: "empty serviceconfig",
			sc:   &API{},
			lang: "go",
			want: "grpc+rest",
		},
		{
			name: "go specific transport",
			sc: &API{
				Transports: map[string]Transport{
					"go": GRPC,
				},
			},
			lang: "go",
			want: "grpc",
		},
		{
			name: "other language transport",
			sc: &API{
				Transports: map[string]Transport{
					"go": GRPC,
				},
			},
			lang: "python",
			want: "grpc+rest",
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
