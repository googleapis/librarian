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

package discovery

import (
	"testing"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/api/apitest"
)

func TestService(t *testing.T) {
	model, err := PublicCaDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := model.State.ServiceByID["..projects"]; ok {
		t.Errorf("expected no service for `projects` resource as it has no methods")
	}

	id := "..externalAccountKeys"
	got, ok := model.State.ServiceByID[id]
	if !ok {
		t.Fatalf("expected service %s in the API model", id)
	}
	want := &api.Service{
		Name:          "externalAccountKeys",
		ID:            id,
		Package:       "",
		Documentation: "Service for the `externalAccountKeys` resource.",
		Methods: []*api.Method{
			{
				ID:            "..externalAccountKeys.create",
				Name:          "create",
				Documentation: "Creates a new ExternalAccountKey bound to the project.",
				InputTypeID:   "..ExternalAccountKey",
				OutputTypeID:  "..ExternalAccountKey",
			},
		},
	}
	apitest.CheckService(t, got, want)
}

func TestServiceTopLevelMethodErrors(t *testing.T) {
	model, err := PublicCaDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	input := resource{
		Methods: []*method{
			{MediaUpload: &mediaUpload{}},
		},
	}
	if err := addServiceRecursive(model, &input); err == nil {
		t.Errorf("expected error in addServiceRecursive invalid top-level method, got=%v", model.Services)
	}
}

func TestServiceChildMethodErrors(t *testing.T) {
	model, err := PublicCaDisco(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	input := resource{
		Resources: []*resource{
			{
				Methods: []*method{
					{MediaUpload: &mediaUpload{}},
				},
			},
		},
	}
	if err := addServiceRecursive(model, &input); err == nil {
		t.Errorf("expected error in addServiceRecursive invalid child method, got=%v", model.Services)
	}
}
