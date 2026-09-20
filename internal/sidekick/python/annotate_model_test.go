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
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateModel(t *testing.T) {
	for _, test := range []struct {
		name    string
		model   *api.API
		library *config.Library
		want    *modelAnnotations
	}{
		{
			name: "basic model with message, enum, and service",
			model: func() *api.API {
				msg := api.NewTestMessage("Secret").WithFields(
					api.NewTestField("name").WithType(api.TypezString),
				)
				enum := api.NewTestEnum("SecretStatus").WithValues(
					api.NewTestEnumValue("ENABLED", 1),
				)
				svc := api.NewTestService("SecretManagerService")
				m := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{enum}, []*api.Service{svc})
				m.Name = "google-cloud-secretmanager"
				return m
			}(),
			library: &config.Library{
				Name:          "google-cloud-secretmanager",
				Version:       "1.0.0",
				CopyrightYear: "2026",
			},
			want: &modelAnnotations{
				CopyrightYear:  "2026",
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "1.0.0",
				Messages: []*messageAnnotations{
					{Name: "Secret"},
				},
				Enums: []*enumAnnotations{
					{Name: "SecretStatus"},
				},
				Services: []*serviceAnnotations{
					{Name: "SecretManagerService"},
				},
			},
		},
		{
			name: "model with nil library defaults",
			model: func() *api.API {
				msg := api.NewTestMessage("Secret")
				m := api.NewTestAPI([]*api.Message{msg}, nil, nil)
				m.Name = "google-cloud-secretmanager"
				return m
			}(),
			library: nil,
			want: &modelAnnotations{
				CopyrightYear:  strconv.Itoa(time.Now().Year()),
				PackageName:    "google-cloud-secretmanager",
				PackageVersion: "0.0.0",
				Messages: []*messageAnnotations{
					{Name: "Secret"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			codec := newTestCodec(t, test.model, test.library)
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}

			ann, ok := test.model.Codec.(*modelAnnotations)
			if !ok {
				t.Fatalf("got %T, want *modelAnnotations", test.model.Codec)
			}

			if diff := cmp.Diff(test.want, ann,
				cmpopts.IgnoreFields(modelAnnotations{}, "BoilerPlate"),
				cmpopts.IgnoreFields(messageAnnotations{}, "Model", "Message", "Fields", "OneOfs", "DocLines"),
				cmpopts.IgnoreFields(enumAnnotations{}, "Model", "Enum", "Values", "DocLines"),
				cmpopts.IgnoreFields(serviceAnnotations{}, "Model", "Service", "Methods", "ProtoName", "ClientName", "AsyncClientName", "DocLines"),
			); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
