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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const testdataGoogleapis = "../../testdata/googleapis"

func TestBuildGAPICOpts(t *testing.T) {
	googleapisDir, err := filepath.Abs(testdataGoogleapis)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		apiPath string
		gemName string
		want    []string
	}{
		{
			name:    "secretmanager v1",
			apiPath: "google/cloud/secretmanager/v1",
			gemName: "google-cloud-secret_manager-v1",
			want: []string{
				"ruby-cloud-gem-name=google-cloud-secret_manager-v1",
				"service-yaml=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_v1.yaml"),
				"grpc-service-config=" + filepath.Join(googleapisDir, "google/cloud/secretmanager/v1/secretmanager_grpc_service_config.json"),
				"transport=grpc+rest",
				"ruby-cloud-rest-numeric-enums=true",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildGAPICOpts(test.apiPath, test.gemName, googleapisDir)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTransport(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   *serviceconfig.API
		want serviceconfig.Transport
	}{
		{
			name: "nil api",
			sc:   nil,
			want: serviceconfig.GRPCRest,
		},
		{
			name: "rest only",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.Rest,
				},
			},
			want: serviceconfig.Rest,
		},
		{
			name: "rest and grpc",
			sc: &serviceconfig.API{
				Transports: map[string]serviceconfig.Transport{
					config.LanguageRuby: serviceconfig.GRPCRest,
				},
			},
			want: serviceconfig.GRPCRest,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := transport(test.sc)
			if got != test.want {
				t.Errorf("transport() = %v, want %v", got, test.want)
			}
		})
	}
}
