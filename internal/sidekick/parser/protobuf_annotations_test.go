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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestProcessRule_HttpBodyWildcard(t *testing.T) {
	httpBodyMsg := api.NewTestMessage("HttpBody").
		WithPackage("google.api").
		WithFields(
			api.NewTestField("content_type").WithType(api.TypezString),
			api.NewTestField("data").WithType(api.TypezBytes),
		)
	rawPredictReq := api.NewTestMessage("RawPredictRequest").
		WithFields(
			api.NewTestField("endpoint").WithType(api.TypezString),
			api.NewTestField("http_body").WithMessageType(httpBodyMsg),
			api.NewTestField("extra_param").WithType(api.TypezString),
		)
	regularReq := api.NewTestMessage("RegularRequest").
		WithFields(
			api.NewTestField("endpoint").WithType(api.TypezString),
			api.NewTestField("extra_param").WithType(api.TypezString),
		)
	model := api.NewTestAPI([]*api.Message{httpBodyMsg, rawPredictReq, regularReq}, nil, nil)

	for _, test := range []struct {
		name            string
		msgID           string
		body            string
		wantBodyField   string
		wantQueryParams map[string]bool
	}{
		{
			name:            "wildcard with top-level HttpBody field resolves to field name",
			msgID:           ".test.RawPredictRequest",
			body:            "*",
			wantBodyField:   "http_body",
			wantQueryParams: map[string]bool{"extra_param": true},
		},
		{
			name:            "wildcard when request is HttpBody itself stays wildcard",
			msgID:           ".google.api.HttpBody",
			body:            "*",
			wantBodyField:   "*",
			wantQueryParams: map[string]bool{},
		},
		{
			name:            "field body when request is HttpBody itself resolves to wildcard",
			msgID:           ".google.api.HttpBody",
			body:            "data",
			wantBodyField:   "*",
			wantQueryParams: map[string]bool{},
		},
		{
			name:            "wildcard without HttpBody field stays wildcard",
			msgID:           ".test.RegularRequest",
			body:            "*",
			wantBodyField:   "*",
			wantQueryParams: map[string]bool{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			rule := &httpRule{
				Pattern: &httpRulePost{
					Post: "/v1/{endpoint=projects/*/locations/*/endpoints/*}:rawPredict",
				},
				Body: test.body,
			}
			got, err := processRule(rule, model, test.msgID)
			if err != nil {
				t.Fatal(err)
			}
			want := &api.PathInfo{
				BodyFieldPath: test.wantBodyField,
				Bindings: []*api.PathBinding{
					{
						Verb: "POST",
						PathTemplate: (&api.PathTemplate{}).
							WithLiteral("v1").
							WithVariable(api.NewPathVariable("endpoint").
								WithLiteral("projects").
								WithMatch().
								WithLiteral("locations").
								WithMatch().
								WithLiteral("endpoints").
								WithMatch()).
							WithVerb("rawPredict"),
						QueryParameters: test.wantQueryParams,
					},
				},
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
