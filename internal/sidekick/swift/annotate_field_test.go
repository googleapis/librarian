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

func TestAnnotateField(t *testing.T) {
	field := &api.Field{
		Name:          "secret_payload",
		Documentation: "The secret version payload.",
		ID:            ".test.SecretVersion.secret_payload",
		Typez:         api.STRING_TYPE,
	}
	msg := &api.Message{
		Name:    "Secret",
		ID:      ".test.SecretVersion",
		Package: "test",
		Fields:  []*api.Field{field},
	}
	model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
	codec := newTestCodec(t, model, map[string]string{})
	if err := codec.annotateModel(); err != nil {
		t.Fatal(err)
	}
	want := &fieldAnnotations{
		Name:      "secretPayload",
		DocLines:  []string{"The secret version payload."},
		FieldType: "String",
	}

	if diff := cmp.Diff(want, field.Codec); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAnnotateField_TypeNames(t *testing.T) {
	for _, test := range []struct {
		name     string
		typez    api.Typez
		wantType string
	}{
		{"string", api.STRING_TYPE, "String"},
		{"int32", api.INT32_TYPE, "Int32"},
		{"bytes", api.BYTES_TYPE, "Data"},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := &api.Field{
				Name:          "test_field",
				ID:            ".test.TestMessage.test_field",
				Typez:         test.typez,
				Documentation: "Test documentation.",
			}
			msg := &api.Message{
				Name:    "TestMessage",
				ID:      ".test.TestMessage",
				Package: "test",
				Fields:  []*api.Field{field},
			}
			model := api.NewTestAPI([]*api.Message{msg}, []*api.Enum{}, []*api.Service{})
			codec := newTestCodec(t, model, map[string]string{})
			if err := codec.annotateModel(); err != nil {
				t.Fatal(err)
			}
			want := &fieldAnnotations{
				Name:      "testField",
				FieldType: test.wantType,
				DocLines:  []string{"Test documentation."},
			}
			if diff := cmp.Diff(want, field.Codec); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
