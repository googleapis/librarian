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

package gcloud

import (
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

func TestGenerateService(t *testing.T) {
	for _, test := range []struct {
		name    string
		service *api.Service
		model   *api.API
		wantErr bool
	}{
		{
			name: "Valid Service",
			service: &api.Service{
				Name:        "parallelstore.googleapis.com",
				DefaultHost: "parallelstore.googleapis.com",
				Methods: []*api.Method{
					{
						Name: "CreateInstance",
						InputType: &api.Message{
							Fields: []*api.Field{},
						},
						// Annotations needed for resource resolution would be complex to mock here completely
						// without a full parser run or extensive manual setup.
						// So we test the basic flow: it should create the service directory.
					},
				},
			},
			model: &api.API{
				Title: "Parallelstore API",
			},
			wantErr: false,
		},
		{
			name: "Empty DefaultHost",
			service: &api.Service{
				Name:        "parallelstore.googleapis.com",
				DefaultHost: "",
				Package:     "google.cloud.parallelstore.v1",
			},
			model: &api.API{
				Title: "Parallelstore API",
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.model.Services = []*api.Service{test.service}
			_, err := newCommandTreeBuilder(test.model, &provider.Config{}).build()
			if (err != nil) != test.wantErr {
				t.Errorf("newCommandTreeBuilder().build() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
