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

package ruby

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestAdd(t *testing.T) {
	for _, test := range []struct {
		name string
		in   *config.Library
		want *config.Library
	}{
		{
			name: "default library name",
			in: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name:    "google-cloud-secretmanager-v1",
				Version: "0.0.1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
		{
			name: "custom library name",
			in: &config.Library{
				Name: "google-cloud-secret_manager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name:    "google-cloud-secret_manager-v1",
				Version: "0.0.1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Add(&config.Config{}, test.in)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddWrapper(t *testing.T) {
	for _, test := range []struct {
		name string
		in   *config.Library
		want *config.Library
	}{
		{
			name: "versioned client untouched",
			in: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager-v1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
		{
			name: "wrapper client configured for v1",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager"},
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v1:0.0"},
				},
			},
		},
		{
			name: "wrapper client configured for beta",
			in: &config.Library{
				Name: "google-cloud-dialogflow-cx",
				APIs: []*config.API{
					{Path: "google/cloud/dialogflow/cx"},
				},
			},
			want: &config.Library{
				Name: "google-cloud-dialogflow-cx",
				APIs: []*config.API{
					{Path: "google/cloud/dialogflow/cx/v3beta1"},
				},
				Ruby: &config.RubyPackage{
					WrapperOf: []string{"v3beta1:0.0"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Libraries: []*config.Library{
					{
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
						},
					},
					{
						APIs: []*config.API{
							{Path: "google/cloud/dialogflow/cx/v3beta1"},
						},
					},
				},
			}
			got, err := addWrapper(cfg, test.in)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddWrapper_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		in      *config.Library
		wantErr error
	}{
		{
			name: "no APIs",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
			},
			wantErr: errRequiresOneAPI,
		},
		{
			name: "multiple APIs",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v2"},
				},
			},
			wantErr: errRequiresOneAPI,
		},
		{
			name: "missing versioned API for wrapper",
			in: &config.Library{
				Name: "google-cloud-nonexistent",
				APIs: []*config.API{
					{Path: "google/cloud/nonexistent"},
				},
			},
			wantErr: errNoVersionedAPI,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Libraries: []*config.Library{
					{
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
						},
					},
				},
			}
			_, err := addWrapper(cfg, test.in)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("addWrapper() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestSearchVersionedAPI(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{
			name:    "v1 api found",
			apiPath: "google/cloud/secretmanager",
			want:    "google/cloud/secretmanager/v1",
		},
		{
			name:    "beta api found",
			apiPath: "google/cloud/dialogflow/cx",
			want:    "google/cloud/dialogflow/cx/v3beta1",
		},
		{
			name:    "second api in library matched",
			apiPath: "google/cloud/asset",
			want:    "google/cloud/asset/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{
				Libraries: []*config.Library{
					{
						APIs: []*config.API{
							{Path: "google/cloud/secretmanager/v1"},
						},
					},
					{
						APIs: []*config.API{
							{Path: "google/cloud/dialogflow/cx/v3beta1"},
						},
					},
					{
						APIs: []*config.API{
							{Path: "google/cloud/unrelated/v1"},
							{Path: "google/cloud/asset/v1"},
						},
					},
				},
			}
			got, err := searchVersionedAPI(cfg, test.apiPath)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
