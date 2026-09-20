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
						WithOperationInfo(&api.OperationInfo{ResponseTypeID: "Resp", MetadataTypeID: "Meta"}),
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
