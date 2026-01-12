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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestNewParam(t *testing.T) {
	// Helper to create a basic field
	makeField := func(name string, typez api.Typez) *api.Field {
		return &api.Field{
			Name:     name,
			JSONName: name, // simplify default
			Typez:    typez,
			Behavior: []api.FieldBehavior{api.FIELD_BEHAVIOR_OPTIONAL},
		}
	}

	for _, test := range []struct {
		name     string
		field    *api.Field
		apiField string
		method   *api.Method
		want     Param
		wantErr  bool
	}{
		{
			name:     "String Field",
			field:    makeField("description", api.STRING_TYPE),
			apiField: "description",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "description",
				APIField: "description",
				Type:     "str", // String is default/empty
				HelpText: "Value for the `description` field.",
				Required: false,
				Repeated: false,
			},
		},
		{
			name:     "Long Field",
			field:    makeField("capacity_gib", api.INT64_TYPE),
			apiField: "capacityGib",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "capacity-gib",
				APIField: "capacityGib",
				Type:     "long",
				HelpText: "Value for the `capacity-gib` field.",
				Required: false,
				Repeated: false,
			},
		},
		{
			name: "Repeated Field",
			field: &api.Field{
				Name:     "labels",
				JSONName: "labels",
				Typez:    api.STRING_TYPE,
				Repeated: true,
			},
			apiField: "labels",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "labels",
				APIField: "labels",
				Type:     "str",
				HelpText: "Value for the `labels` field.",
				Required: false,
				Repeated: true,
			},
		},
		{
			name: "Required Field",
			field: &api.Field{
				Name:     "name",
				JSONName: "name",
				Typez:    api.STRING_TYPE,
				Behavior: []api.FieldBehavior{api.FIELD_BEHAVIOR_REQUIRED},
			},
			apiField: "name",
			method:   &api.Method{Name: "CreateInstance"},
			want: Param{
				ArgName:  "name",
				APIField: "name",
				Type:     "str",
				HelpText: "Value for the `name` field.",
				Required: true,
				Repeated: false,
			},
		},
		{
			name: "Clearable Map (Update)",
			field: &api.Field{
				Name:     "labels",
				JSONName: "labels",
				Typez:    api.STRING_TYPE,
				Map:      true,
			},
			apiField: "labels",
			method:   &api.Method{Name: "UpdateInstance"},
			want: Param{
				ArgName:   "labels",
				APIField:  "labels",
				HelpText:  "Value for the `labels` field.",
				Repeated:  true,
				Clearable: true,
				Spec: []ArgSpec{
					{APIField: "key"},
					{APIField: "value"},
				},
			},
		},
		{
			name: "Clearable Repeated Field (Update)",
			field: &api.Field{
				Name:     "access_points",
				JSONName: "accessPoints",
				Typez:    api.STRING_TYPE,
				Repeated: true,
			},
			apiField: "accessPoints",
			method:   &api.Method{Name: "UpdateInstance"},
			want: Param{
				ArgName:   "access-points",
				APIField:  "accessPoints",
				Type:      "str",
				HelpText:  "Value for the `access-points` field.",
				Repeated:  true,
				Clearable: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := newParam(test.field, test.apiField, &Config{}, &api.API{}, &api.Service{}, test.method)
			if (err != nil) != test.wantErr {
				t.Errorf("newParam() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			// Ignore fields that are hard to mock or irrelevant for basic mapping test
			if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(Param{}, "ResourceSpec")); diff != "" {
				t.Errorf("newParam() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
