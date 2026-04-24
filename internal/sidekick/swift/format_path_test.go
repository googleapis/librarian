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

package swift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestPathExpression(t *testing.T) {
	for _, test := range []struct {
		name     string
		template *api.PathTemplate
		want     string
	}{
		{
			name: "literals only",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithLiteral("operations"),
			want: "/v1/operations",
		},
		{
			name: "with variable",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("name"),
			want: "/v1/\\(pathVariable0)",
		},
		{
			name: "with multiple variables",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("name").WithLiteral("separator").WithVariableNamed("second"),
			want: "/v1/\\(pathVariable0)/separator/\\(pathVariable1)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := pathExpression(test.template)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPathVariables(t *testing.T) {
	requestMessage := &api.Message{
		Name:    "CreateSecretRequest",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.CreateSecretRequest",
		Fields: []*api.Field{
			{
				Name:  "name",
				Typez: api.TypezString,
			},
			{
				Name:  "second",
				Typez: api.TypezString,
			},
		},
	}
	model := api.NewTestAPI([]*api.Message{requestMessage}, nil, []*api.Service{})
	codec := newTestCodec(t, model, nil)

	for _, f := range requestMessage.Fields {
		if err := codec.annotateField(f); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range []struct {
		name     string
		template *api.PathTemplate
		want     []*pathVariable
		wantErr  bool
	}{
		{
			name:     "no variables",
			template: api.NewPathTemplate().WithLiteral("v1"),
			want:     nil,
		},
		{
			name: "one variable",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("name"),
			want: []*pathVariable{
				{
					Name:       "pathVariable0",
					Expression: ".name as String?",
					Test:       "!pathVariable0.isEmpty",
					FieldPath:  "name",
				},
			},
		},
		{
			name: "two variables",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("name").
				WithLiteral("sep").
				WithVariableNamed("second"),
			want: []*pathVariable{
				{
					Name:       "pathVariable0",
					Expression: ".name as String?",
					Test:       "!pathVariable0.isEmpty",
					FieldPath:  "name",
				},
				{
					Name:       "pathVariable1",
					Expression: ".second as String?",
					Test:       "!pathVariable1.isEmpty",
					FieldPath:  "second",
				},
			},
		},
		{
			name: "error - lookup field missing",
			template: api.NewPathTemplate().
				WithLiteral("v1").
				WithVariableNamed("missing"),
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.pathVariables(requestMessage, test.template)
			if (err != nil) != test.wantErr {
				t.Fatalf("pathVariables() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("pathVariables() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewPathVariable(t *testing.T) {
	secretMessage := &api.Message{
		Name:    "Secret",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.Secret",
		Fields: []*api.Field{
			{
				Name:  "name",
				Typez: api.TypezString,
			},
			{
				Name:     "description",
				Typez:    api.TypezString,
				Optional: true,
			},
		},
	}

	requestMessage := &api.Message{
		Name:    "CreateSecretRequest",
		Package: "google.cloud.secretmanager.v1",
		ID:      ".google.cloud.secretmanager.v1.CreateSecretRequest",
		Fields: []*api.Field{
			{
				Name:  "parent",
				Typez: api.TypezString,
			},
			{
				Name:     "display_name",
				Typez:    api.TypezString,
				Optional: true,
			},
			{
				Name:     "secret",
				Typez:    api.TypezMessage,
				TypezID:  ".google.cloud.secretmanager.v1.Secret",
				Optional: true,
			},
			{
				Name:  "data",
				Typez: api.TypezBytes,
			},
			{
				Name:    "oneof_field",
				Typez:   api.TypezString,
				IsOneOf: true,
			},
		},
	}

	model := api.NewTestAPI([]*api.Message{secretMessage, requestMessage}, nil, []*api.Service{})
	model.State = &api.APIState{
		MessageByID: map[string]*api.Message{
			".google.cloud.secretmanager.v1.Secret":              secretMessage,
			".google.cloud.secretmanager.v1.CreateSecretRequest": requestMessage,
		},
	}

	codec := newTestCodec(t, model, nil)
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		message  *api.Message
		variable *api.PathVariable
		count    int
		want     *pathVariable
		wantErr  bool
	}{
		{
			name:    "non-optional string",
			message: requestMessage,
			variable: &api.PathVariable{
				FieldPath: []string{"parent"},
			},
			count: 0,
			want: &pathVariable{
				Name:       "pathVariable0",
				Expression: ".parent as String?",
				Test:       "!pathVariable0.isEmpty",
				FieldPath:  "parent",
			},
		},
		{
			name:    "optional string",
			message: requestMessage,
			variable: &api.PathVariable{
				FieldPath: []string{"display_name"},
			},
			count: 1,
			want: &pathVariable{
				Name:       "pathVariable1",
				Expression: ".displayName",
				Test:       "!pathVariable1.isEmpty",
				FieldPath:  "display_name",
			},
		},
		{
			name:    "error - oneof field",
			message: requestMessage,
			variable: &api.PathVariable{
				FieldPath: []string{"oneof_field"},
			},
			count:   6,
			wantErr: true,
		},
		{
			name:    "nested non-optional string",
			message: requestMessage,
			variable: &api.PathVariable{
				FieldPath: []string{"secret", "name"},
			},
			count: 2,
			want: &pathVariable{
				Name:       "pathVariable2",
				Expression: ".secret.map({ $0.name })",
				Test:       "!pathVariable2.isEmpty",
				FieldPath:  "secret.name",
			},
		},
		{
			name:    "nested optional string",
			message: requestMessage,
			variable: &api.PathVariable{
				FieldPath: []string{"secret", "description"},
			},
			count: 3,
			want: &pathVariable{
				Name:       "pathVariable3",
				Expression: ".secret.flatMap({ $0.description })",
				Test:       "!pathVariable3.isEmpty",
				FieldPath:  "secret.description",
			},
		},
		{
			name:    "error - unsupported type bytes",
			message: requestMessage,
			variable: &api.PathVariable{
				FieldPath: []string{"data"},
			},
			count:   4,
			wantErr: true,
		},
		{
			name:    "error - lookup field missing",
			message: requestMessage,
			variable: &api.PathVariable{
				FieldPath: []string{"missing"},
			},
			count:   5,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codec.newPathVariable(test.message, test.variable, test.count)
			if (err != nil) != test.wantErr {
				t.Fatalf("newPathVariable() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("newPathVariable() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
