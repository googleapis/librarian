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

package cpp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateService_OperationsMixinAndStub(t *testing.T) {
	for _, test := range []struct {
		name                string
		setup               func() []*api.Method
		wantOperationsMixin bool
		wantOperationsStub  bool
	}{
		{
			name: "regular unary method",
			setup: func() []*api.Method {
				return []*api.Method{
					api.NewTestMethod("Unary").
						WithInput(api.NewTestMessage("Req")).
						WithOutput(api.NewTestMessage("Resp")),
				}
			},
			wantOperationsMixin: false,
			wantOperationsStub:  false,
		},
		{
			name: "lro method enables operations stub",
			setup: func() []*api.Method {
				return []*api.Method{
					api.NewTestMethod("LongRunning").
						WithInput(api.NewTestMessage("Req")).
						WithOutput(api.NewTestMessage("Operation")).
						WithOperationInfo(api.NewTestOperationInfo("Resp", "Meta")),
				}
			},
			wantOperationsMixin: false,
			wantOperationsStub:  true,
		},
		{
			name: "operations mixin method enables both",
			setup: func() []*api.Method {
				m := api.NewTestMethod("ListOperations").
					WithInput(api.NewTestMessage("Req")).
					WithOutput(api.NewTestMessage("Operation"))
				m.SourceServiceID = "google.longrunning.Operations"
				return []*api.Method{m}
			},
			wantOperationsMixin: true,
			wantOperationsStub:  true,
		},
		{
			name: "poller operation does not enable mixin",
			setup: func() []*api.Method {
				m := api.NewTestMethod("GetOperation").
					WithInput(api.NewTestMessage("Req")).
					WithOutput(api.NewTestMessage("Operation"))
				m.SourceServiceID = "google.longrunning.Operations"
				return []*api.Method{m}
			},
			wantOperationsMixin: false,
			wantOperationsStub:  false,
		},
		{
			name: "poller operation with default unrouted path does not enable mixin",
			setup: func() []*api.Method {
				m := api.NewTestMethod("GetOperation").
					WithInput(api.NewTestMessage("Req")).
					WithOutput(api.NewTestMessage("Operation")).
					WithVerb("GET").
					WithPathTemplate((&api.PathTemplate{}).
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("name").WithLiteral("operations").WithMatchRecursive()))
				m.SourceServiceID = "google.longrunning.Operations"
				return []*api.Method{m}
			},
			wantOperationsMixin: false,
			wantOperationsStub:  false,
		},
		{
			name: "routed GetOperation enables operations mixin and stub",
			setup: func() []*api.Method {
				m := api.NewTestMethod("GetOperation").
					WithInput(api.NewTestMessage("Req")).
					WithOutput(api.NewTestMessage("Operation")).
					WithVerb("GET").
					WithPathTemplate((&api.PathTemplate{}).
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("name").
							WithLiteral("organizations").WithMatch().
							WithLiteral("locations").WithMatch().
							WithLiteral("operations").WithMatch()))
				m.SourceServiceID = "google.longrunning.Operations"
				return []*api.Method{m}
			},
			wantOperationsMixin: true,
			wantOperationsStub:  true,
		},
		{
			name: "routed CancelOperation enables operations mixin and stub",
			setup: func() []*api.Method {
				m := api.NewTestMethod("CancelOperation").
					WithInput(api.NewTestMessage("Req")).
					WithOutput(api.NewTestMessage("Operation")).
					WithVerb("POST").
					WithPathTemplate((&api.PathTemplate{}).
						WithLiteral("v1").
						WithVariable(api.NewPathVariable("name").
							WithLiteral("projects").WithMatch().
							WithLiteral("locations").WithMatch().
							WithLiteral("operations").WithMatch()))
				m.SourceServiceID = "google.longrunning.Operations"
				return []*api.Method{m}
			},
			wantOperationsMixin: true,
			wantOperationsStub:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc := api.NewTestService("TestService").WithMethods(test.setup()...)
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			modelAnn := &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			}
			c := newCodec(nil)
			if err := c.annotateService(svc, modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got := svc.Codec.(*serviceAnnotations)
			if got.HasOperationsMixin != test.wantOperationsMixin {
				t.Errorf("HasOperationsMixin mismatch: want %v, got %v", test.wantOperationsMixin, got.HasOperationsMixin)
			}
			if got.HasOperationsStub != test.wantOperationsStub {
				t.Errorf("HasOperationsStub mismatch: want %v, got %v", test.wantOperationsStub, got.HasOperationsStub)
			}
		})
	}
}

func TestAnnotateService_HasAsyncRpcs(t *testing.T) {
	for _, test := range []struct {
		name          string
		genAsyncRPCs  []string
		hasLro        bool
		wantAsyncRpcs bool
	}{
		{
			name:          "no async rpcs and no lro",
			genAsyncRPCs:  nil,
			hasLro:        false,
			wantAsyncRpcs: false,
		},
		{
			name:          "lro without genAsyncRPCs does not set HasAsyncRpcs",
			genAsyncRPCs:  nil,
			hasLro:        true,
			wantAsyncRpcs: false,
		},
		{
			name:          "genAsyncRPCs sets HasAsyncRpcs",
			genAsyncRPCs:  []string{"Unary"},
			hasLro:        false,
			wantAsyncRpcs: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			methods := []*api.Method{
				api.NewTestMethod("Unary").
					WithInput(api.NewTestMessage("Req")).
					WithOutput(api.NewTestMessage("Resp")),
			}
			if test.hasLro {
				methods = append(methods,
					api.NewTestMethod("LongRunning").
						WithInput(api.NewTestMessage("LroReq")).
						WithOutput(api.NewTestMessage("Operation")).
						WithOperationInfo(api.NewTestOperationInfo("Resp", "Meta")))
			}
			svc := api.NewTestService("TestService").WithMethods(methods...)
			model := api.NewTestAPI(nil, nil, []*api.Service{svc})
			modelAnn := &modelAnnotations{
				CopyrightYear: "2026",
				BoilerPlate:   license.HeaderBulk(),
			}
			c := newCodec(&config.CppLibrary{GenAsyncRPCs: test.genAsyncRPCs})
			if err := c.annotateService(svc, modelAnn, model); err != nil {
				t.Fatal(err)
			}
			got := svc.Codec.(*serviceAnnotations)
			if diff := cmp.Diff(test.wantAsyncRpcs, got.HasAsyncRpcs); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
