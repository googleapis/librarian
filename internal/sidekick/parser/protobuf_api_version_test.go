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

package parser

import (
	"testing"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/api/apitest"
)

func TestProtobuf_ApiVersion(t *testing.T) {
	requireProtoc(t)
	serviceConfig := &serviceconfig.Service{
		Name:  "unused.googleapis.com",
		Title: "A test-only API",
	}

	model, err := makeAPIForProtobuf(serviceConfig, newTestCodeGeneratorRequest(t, "api_version.proto"))
	if err != nil {
		t.Fatalf("Failed to make API for Protobuf %v", err)
	}
	id := ".test.Service"
	service, ok := model.State.ServiceByID[id]
	if !ok {
		t.Fatalf("Cannot find service %s in API State", id)
	}
	want := &api.Service{
		Name:          "Service",
		ID:            ".test.Service",
		Package:       "test",
		Documentation: "A service with an API version.",
		DefaultHost:   "test.googleapis.com",
		Methods: []*api.Method{
			{
				Name:            "Create",
				ID:              ".test.Service.Create",
				SourceServiceID: ".test.Service",
				Documentation:   "A create method.",
				InputTypeID:     ".test.Request",
				OutputTypeID:    ".test.Response",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "POST",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("v7").
								WithLiteral("thing"),
							QueryParameters: map[string]bool{"parent": true}},
					},
				},
				APIVersion: "v7_20260206",
			},
			{
				Name:            "Make",
				ID:              ".test.Service.Make",
				SourceServiceID: ".test.Service",
				Documentation:   "Another sort of method.",
				InputTypeID:     ".test.Request",
				OutputTypeID:    ".test.Response",
				PathInfo: &api.PathInfo{
					Bindings: []*api.PathBinding{
						{
							Verb: "POST",
							PathTemplate: api.NewPathTemplate().
								WithLiteral("v7").
								WithLiteral("thing").
								WithVerb("make"),
							QueryParameters: map[string]bool{"parent": true}},
					},
				},
				APIVersion: "v7_20260206",
			},
		},
	}
	apitest.CheckService(t, service, want)
}
