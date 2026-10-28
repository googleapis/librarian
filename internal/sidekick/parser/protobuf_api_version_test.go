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
	service := model.Service(id)
	if service == nil {
		t.Fatalf("Cannot find service %s in API State", id)
	}
	createMethod := api.NewTestMethod("Create").
		WithID(".test.Service.Create").
		WithDocumentation("A create method.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v7").
				WithLiteral("thing"),
		).
		WithQueryParameters(map[string]bool{"parent": true}).
		WithAPIVersion("v7_20260206")
	createMethod.SourceServiceID = ".test.Service"
	createMethod.InputTypeID = ".test.Request"
	createMethod.OutputTypeID = ".test.Response"

	makeMethod := api.NewTestMethod("Make").
		WithID(".test.Service.Make").
		WithDocumentation("Another sort of method.").
		WithVerb("POST").
		WithPathTemplate(
			(&api.PathTemplate{}).
				WithLiteral("v7").
				WithLiteral("thing").
				WithVerb("make"),
		).
		WithQueryParameters(map[string]bool{"parent": true}).
		WithAPIVersion("v7_20260206")
	makeMethod.SourceServiceID = ".test.Service"
	makeMethod.InputTypeID = ".test.Request"
	makeMethod.OutputTypeID = ".test.Response"

	want := api.NewTestService("Service").
		WithPackage("test").
		WithDocumentation("A service with an API version.").
		WithDefaultHost("test.googleapis.com").
		WithMethods(createMethod, makeMethod)
	apitest.CheckService(t, service, want)
}
