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
		name     string
		path     string
		language string
		want     bool
	}{
		{"matching path and language", "google/api", config.LanguageRust, true},
		{"matching path but not language", "google/ads/admanager/v1", config.LanguageRust, false},
		{"unknown path", "google/does/not/exist/v1", config.LanguageRust, false},
		{"empty path", "", config.LanguageRust, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := HasAPIPath(test.path, test.language)
			if got != test.want {
				t.Errorf("HasAPIPath(%q, %q) = %v, want %v", test.path, test.language, got, test.want)
			}
		})
	}
}

func TestReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name     string
		sc       *API
		language string
		version  string
		want     string
	}{
		{
			name:     "empty release levels defaults to stable",
			sc:       &API{},
			language: config.LanguageRust,
			want:     "stable",
		},
		{
			name:     "go defaults to ga",
			sc:       &API{},
			language: config.LanguageGo,
			want:     "ga",
		},
		{
			name: "language-specific level",
			sc: &API{
				ReleaseLevels: map[string]string{config.LanguageGo: "alpha"},
			},
			language: config.LanguageGo,
			want:     "alpha",
		},
		{
			name: "falls back to all",
			sc: &API{
				ReleaseLevels: map[string]string{config.LanguageAll: "beta"},
			},
			language: config.LanguageRust,
			want:     "beta",
		},
		{
			name: "language-specific overrides all",
			sc: &API{
				ReleaseLevels: map[string]string{
					config.LanguageAll: "beta",
					config.LanguageGo:  "alpha",
				},
			},
			language: config.LanguageGo,
			want:     "alpha",
		},
		{
			name: "other language not affected by go default",
			sc: &API{
				ReleaseLevels: map[string]string{config.LanguagePython: "beta"},
			},
			language: config.LanguageRust,
			want:     "stable",
		},
		{
			name:     "preview due to version 0.x",
			sc:       &API{Path: "google/cloud/secretmanager/v1"},
			language: config.LanguageRust,
			version:  "0.1.0",
			want:     "preview",
		},
		{
			name:     "preview due to alpha path",
			sc:       &API{Path: "google/cloud/secretmanager/v1alpha"},
			language: config.LanguageRust,
			want:     "preview",
		},
		{
			name:     "preview due to beta path",
			sc:       &API{Path: "google/cloud/secretmanager/v1beta"},
			language: config.LanguageRust,
			want:     "preview",
		},
		{
			name:     "go alpha due to alpha path",
			sc:       &API{Path: "google/cloud/secretmanager/v1alpha"},
			language: config.LanguageGo,
			want:     "alpha",
		},
		{
			name:     "go beta due to beta path",
			sc:       &API{Path: "google/cloud/secretmanager/v1beta"},
			language: config.LanguageGo,
			want:     "beta",
		},
		{
			name:     "go ga for stable path",
			sc:       &API{Path: "google/cloud/secretmanager/v1"},
			language: config.LanguageGo,
			version:  "1.0.0",
			want:     "ga",
		},
		{
			name:     "go beta for stable path with 0.x version",
			sc:       &API{Path: "google/cloud/secretmanager/v1"},
			language: config.LanguageGo,
			version:  "0.1.0",
			want:     "beta",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.sc.ReleaseLevel(test.language, test.version)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepoMetadataReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name     string
		sc       *API
		language string
		version  string
		want     string
	}{
		{
			name: "go, stable",
			sc: &API{
				Path: "google/cloud/secretmanager/v1",
			},
			language: config.LanguageGo,
			version:  "1.0.0",
			want:     "stable",
		},
		{
			name:     "go, stable empty path",
			sc:       &API{},
			language: config.LanguageGo,
			version:  "1.0.0",
			want:     "stable",
		},
		{
			name: "go, preview alpha api path",
			sc: &API{
				Path: "google/cloud/secretmanager/v1alpha",
			},
			language: config.LanguageGo,
			want:     "preview",
		},
		{
			name: "go, preview alpha release level",
			sc: &API{
				ReleaseLevels: map[string]string{config.LanguageGo: "alpha"},
			},
			language: config.LanguageGo,
			want:     "preview",
		},
		{
			name: "go, preview beta api path",
			sc: &API{
				Path: "google/cloud/secretmanager/v1beta",
			},
			language: config.LanguageGo,
			want:     "preview",
		},
		{
			name: "go, preview beta release level",
			sc: &API{
				ReleaseLevels: map[string]string{config.LanguageGo: "beta"},
			},
			language: config.LanguageGo,
			want:     "preview",
		},
		{
			name:     "go, preview due to version 0.x",
			sc:       &API{Path: "google/cloud/secretmanager/v1"},
			language: config.LanguageGo,
			version:  "0.1.0",
			want:     "preview",
		},
		{
			name:     "non-go returns raw release level",
			sc:       &API{},
			language: config.LanguageRust,
			want:     "stable",
		},
		{
			name: "non-go returns language-specific level",
			sc: &API{
				ReleaseLevels: map[string]string{config.LanguageRust: "beta"},
			},
			language: config.LanguageRust,
			want:     "beta",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.sc.RepoMetadataReleaseLevel(test.language, test.version)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHasRESTNumericEnums(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *API
		lang string
		want bool
	}{
		{
			name: "empty map enables numeric enums",
			sc:   &API{},
			lang: config.LanguageGo,
			want: true,
		},
		{
			name: "nil list enables numeric enums",
			sc:   &API{SkipRESTNumericEnums: nil},
			lang: config.LanguageNodejs,
			want: true,
		},
		{
			name: "skipped for all languages",
			sc:   &API{SkipRESTNumericEnums: []string{config.LanguageAll}},
			lang: config.LanguageGo,
			want: false,
		},
		{
			name: "skipped for specific language",
			sc:   &API{SkipRESTNumericEnums: []string{config.LanguageNodejs}},
			lang: config.LanguageNodejs,
			want: false,
		},
		{
			name: "skipped for other language only",
			sc:   &API{SkipRESTNumericEnums: []string{"python"}},
			lang: config.LanguageGo,
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.sc.HasRESTNumericEnums(test.lang)
			if got != test.want {
				t.Errorf("HasRESTNumericEnums(%q) = %v, want %v", test.lang, got, test.want)
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

func TestRepoMetadataTransport(t *testing.T) {
	for _, test := range []struct {
		name     string
		sc       *API
		language string
		library  *config.Library
		want     string
	}{
		{
			name: "java, grpc",
			sc: &API{
				Transports: map[string]Transport{config.LanguageJava: GRPC},
			},
			language: config.LanguageJava,
			want:     "grpc",
		},
		{
			name: "java, rest",
			sc: &API{
				Transports: map[string]Transport{config.LanguageJava: Rest},
			},
			language: config.LanguageJava,
			want:     "http",
		},
		{
			name:     "non-java, default",
			sc:       &API{},
			language: config.LanguageGo,
			want:     "grpc+rest",
		},
		{
			name: "non-java, grpc",
			sc: &API{
				Transports: map[string]Transport{config.LanguageGo: GRPC},
			},
			language: config.LanguageGo,
			want:     "grpc",
		},
		{
			name: "non-java, rest",
			sc: &API{
				Transports: map[string]Transport{config.LanguageGo: Rest},
			},
			language: config.LanguageGo,
			want:     "rest",
		},
		{
			name: "java, transport override",
			sc: &API{
				Transports: map[string]Transport{config.LanguageJava: GRPC},
			},
			language: config.LanguageJava,
			library: &config.Library{
				Java: &config.JavaModule{
					TransportOverride: "rest",
				},
			},
			want: "http",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.sc.RepoMetadataTransport(test.language, test.library)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
