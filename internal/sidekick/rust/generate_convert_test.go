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

package rust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	libconfig "github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

func TestGenerateConvert(t *testing.T) {
	for _, test := range []struct {
		name       string
		message    *api.Message
		skippedIDs []string
		want       string
	}{
		{
			name: "simple message with two string fields",
			message: &api.Message{
				Name:    "SimpleMessage",
				Package: "test.v1",
				ID:      ".test.v1.SimpleMessage",
				Fields: []*api.Field{
					{
						Name:     "field1",
						JSONName: "field1",
						ID:       ".test.v1.SimpleMessage.field1",
						Typez:    api.TypezString,
					},
					{
						Name:     "field2",
						JSONName: "field2",
						ID:       ".test.v1.SimpleMessage.field2",
						Typez:    api.TypezString,
					},
				},
			},
			want: `impl gaxi::prost::ToProto<SimpleMessage> for crate::model::SimpleMessage {
    type Output = SimpleMessage;
    fn to_proto(self) -> std::result::Result<SimpleMessage, gaxi::prost::ConvertError> {
        Ok(Self::Output {
            field1: self.field1.to_proto()?,
            field2: self.field2.to_proto()?,
        })
    }
}

impl gaxi::prost::FromProto<crate::model::SimpleMessage> for SimpleMessage {
    fn cnv(self) -> std::result::Result<crate::model::SimpleMessage, gaxi::prost::ConvertError> {
        Ok(
            crate::model::SimpleMessage::new()
                .set_field1(self.field1)
                .set_field2(self.field2)
        )
    }
}`,
		},
		{
			name: "message with two singular string fields and one skipped",
			message: &api.Message{
				Name:    "MessageWithSkippedString",
				Package: "test.v1",
				ID:      ".test.v1.MessageWithSkippedString",
				Fields: []*api.Field{
					{
						Name:     "field1",
						JSONName: "field1",
						ID:       ".test.v1.MessageWithSkippedString.field1",
						Typez:    api.TypezString,
					},
					{
						Name:     "field2",
						JSONName: "field2",
						ID:       ".test.v1.MessageWithSkippedString.field2",
						Typez:    api.TypezString,
					},
				},
			},
			skippedIDs: []string{".test.v1.MessageWithSkippedString.field2"},
			want: `impl gaxi::prost::ToProto<MessageWithSkippedString> for crate::model::MessageWithSkippedString {
    type Output = MessageWithSkippedString;
    fn to_proto(self) -> std::result::Result<MessageWithSkippedString, gaxi::prost::ConvertError> {
        Ok(Self::Output {
            field1: self.field1.to_proto()?,
            ..Self::Output::default()
        })
    }
}

impl gaxi::prost::FromProto<crate::model::MessageWithSkippedString> for MessageWithSkippedString {
    fn cnv(self) -> std::result::Result<crate::model::MessageWithSkippedString, gaxi::prost::ConvertError> {
        Ok(
            crate::model::MessageWithSkippedString::new()
                .set_field1(self.field1)
        )
    }
}`,
		},
		{
			name: "message with skipped any field",
			message: &api.Message{
				Name:    "MessageWithSkippedAny",
				Package: "test.v1",
				ID:      ".test.v1.MessageWithSkippedAny",
				Fields: []*api.Field{
					{
						Name:     "field1",
						JSONName: "field1",
						ID:       ".test.v1.MessageWithSkippedAny.field1",
						Typez:    api.TypezString,
					},
					{
						Name:     "field2",
						JSONName: "field2",
						ID:       ".test.v1.MessageWithSkippedAny.field2",
						Typez:    api.TypezMessage,
						TypezID:  api.WktAnyID,
					},
				},
			},
			skippedIDs: []string{".test.v1.MessageWithSkippedAny.field2"},
			want: `impl gaxi::prost::ToProto<MessageWithSkippedAny> for crate::model::MessageWithSkippedAny {
    type Output = MessageWithSkippedAny;
    fn to_proto(self) -> std::result::Result<MessageWithSkippedAny, gaxi::prost::ConvertError> {
        Ok(Self::Output {
            field1: self.field1.to_proto()?,
            ..Self::Output::default()
        })
    }
}

impl gaxi::prost::FromProto<crate::model::MessageWithSkippedAny> for MessageWithSkippedAny {
    fn cnv(self) -> std::result::Result<crate::model::MessageWithSkippedAny, gaxi::prost::ConvertError> {
        Ok(
            crate::model::MessageWithSkippedAny::new()
                .set_field1(self.field1)
        )
    }
}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			model := api.NewTestAPI([]*api.Message{test.message}, []*api.Enum{}, []*api.Service{})
			model.PackageName = "test.v1"
			if err := api.CrossReference(model); err != nil {
				t.Fatal(err)
			}
			if len(test.skippedIDs) > 0 {
				if err := api.SkipModelElements(model, api.ModelOverride{SkippedIDs: test.skippedIDs}); err != nil {
					t.Fatal(err)
				}
			}

			cfg := &parser.ModelConfig{
				SpecificationFormat: libconfig.SpecProtobuf,
				Codec: map[string]string{
					"package:wkt":       "source=google.protobuf,package=google-cloud-wkt",
					"template-override": "templates/convert-prost",
				},
			}
			if err := Generate(t.Context(), model, outDir, cfg); err != nil {
				t.Fatal(err)
			}

			contents, err := os.ReadFile(filepath.Join(outDir, "convert.rs"))
			if err != nil {
				t.Fatal(err)
			}
			got := extractBlock(t, string(contents), "impl gaxi::prost::ToProto<"+test.message.Name+">", "\n        )\n    }\n}")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
