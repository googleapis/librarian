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

package python

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestNewCodec(t *testing.T) {
	for _, test := range []struct {
		name    string
		model   *api.API
		library *config.Library
		want    *codec
	}{
		{
			name: "derived from library configuration",
			model: func() *api.API {
				m := api.NewTestAPI(nil, nil, nil)
				m.Name = "google-cloud-secretmanager"
				return m
			}(),
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				Version:       "2.1.0",
				CopyrightYear: "2026",
			},
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "2.1.0",
			},
		},
		{
			name: "fallback to model name when library name is empty",
			model: func() *api.API {
				m := api.NewTestAPI(nil, nil, nil)
				m.Name = "google-cloud-secretmanager"
				return m
			}(),
			library: &config.Library{
				CopyrightYear: "2026",
			},
			want: &codec{
				GenerationYear: "2026",
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "0.0.0",
			},
		},
		{
			name: "fallback to model package name when library is nil",
			model: func() *api.API {
				m := api.NewTestAPI(nil, nil, nil)
				m.Name = ""
				m.PackageName = "google.cloud.speech.v1"
				return m
			}(),
			library: nil,
			want: &codec{
				GenerationYear: fmt.Sprintf("%04d", time.Now().Year()),
				PackageName:    "google.cloud.speech.v1",
				PackageVersion: "0.0.0",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := newCodec(test.model, test.library, "")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(codec{}, "Model", "Library", "OutDir")); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewCodec_Error(t *testing.T) {
	_, err := newCodec(nil, nil, "")
	if !errors.Is(err, ErrNilModel) {
		t.Errorf("got %v, want %v", err, ErrNilModel)
	}
}

func newTestCodec(t *testing.T, model *api.API, library *config.Library) *codec {
	t.Helper()
	c, err := newCodec(model, library, "")
	if err != nil {
		t.Fatal(err)
	}
	return c
}
