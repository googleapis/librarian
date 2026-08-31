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
			message: api.NewTestMessage("SimpleMessage").WithPackage("test.v1").WithFields(
				api.NewTestField("field1").WithType(api.TypezString),
				api.NewTestField("field2").WithType(api.TypezString),
			),
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
			message: api.NewTestMessage("MessageWithSkippedString").WithPackage("test.v1").WithFields(
				api.NewTestField("field1").WithType(api.TypezString),
				api.NewTestField("field2").WithType(api.TypezString),
			),
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
			message: api.NewTestMessage("MessageWithSkippedAny").WithPackage("test.v1").WithFields(
				api.NewTestField("field1").WithType(api.TypezString),
				api.NewTestField("field2").WithMessageType(&api.Message{ID: api.WktAnyID}),
			),
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

func TestGenerateConvertOneOf(t *testing.T) {
	typeSchemaMsg := api.NewTestMessage("TypeSchema")
	inlineSchemaMsg := api.NewTestMessage("InlineSchema").WithFields(
		api.NewTestField("items").WithMessageType(typeSchemaMsg).WithOptional(),
	)
	refSchemaMsg := api.NewTestMessage("ReferenceSchema").WithFields(
		api.NewTestField("tool").WithType(api.TypezString),
	)
	typeSchemaMsg.WithOneOfs(
		api.NewTestOneOf("schema").WithFields(
			api.NewTestField("inline_schema").WithMessageType(inlineSchemaMsg),
			api.NewTestField("reference_schema").WithMessageType(refSchemaMsg),
			api.NewTestField("name").WithType(api.TypezString),
		),
	)

	outDir := t.TempDir()
	model := api.NewTestAPI([]*api.Message{typeSchemaMsg, inlineSchemaMsg, refSchemaMsg}, nil, nil)
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
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

	wantToProto := `impl gaxi::prost::ToProto<type_schema::Schema> for crate::model::type_schema::Schema {
    type Output = type_schema::Schema;
    fn to_proto(self) -> std::result::Result<Self::Output, gaxi::prost::ConvertError> {
        match self {
            Self::InlineSchema(v) => Ok(Self::Output::InlineSchema(std::boxed::Box::new((*v).to_proto()?))),
            Self::ReferenceSchema(v) => Ok(Self::Output::ReferenceSchema((*v).to_proto()?)),
            Self::Name(v) => Ok(Self::Output::Name(v.to_proto()?)),
        }
    }
}`
	gotToProto := extractBlock(t, string(contents), "impl gaxi::prost::ToProto<type_schema::Schema>", "\n        }\n    }\n}")
	if diff := cmp.Diff(wantToProto, gotToProto); diff != "" {
		t.Errorf("mismatch ToProto (-want +got):\n%s", diff)
	}
}

func TestGenerateConvertAcronyms(t *testing.T) {
	dataStoreSpecMsg := api.NewTestMessage("DataStoreSpec").
		WithPackage("test.v1").
		WithID(".test.v1.VertexAISearch.DataStoreSpec").
		WithFields(
			api.NewTestField("data_store").WithType(api.TypezString),
		)

	vertexAiSearchMsg := api.NewTestMessage("VertexAISearch").
		WithPackage("test.v1").
		WithFields(
			api.NewTestField("serving_config").WithType(api.TypezString),
		)

	ipVersionEnum := &api.Enum{
		Name:    "IPVersion",
		ID:      ".test.v1.IPVersion",
		Package: "test.v1",
		Values: []*api.EnumValue{
			{Name: "IP_VERSION_UNSPECIFIED", ID: ".test.v1.IPVersion.IP_VERSION_UNSPECIFIED", Number: 0},
			{Name: "IPV4", ID: ".test.v1.IPVersion.IPV4", Number: 1},
			{Name: "IPV6", ID: ".test.v1.IPVersion.IPV6", Number: 2},
		},
	}

	nestedEnum := &api.Enum{
		Name:    "IPMode",
		ID:      ".test.v1.VertexAISearch.IPMode",
		Package: "test.v1",
		Values: []*api.EnumValue{
			{Name: "IP_MODE_UNSPECIFIED", ID: ".test.v1.VertexAISearch.IPMode.IP_MODE_UNSPECIFIED", Number: 0},
			{Name: "DYNAMIC_IP", ID: ".test.v1.VertexAISearch.IPMode.DYNAMIC_IP", Number: 1},
		},
	}

	retrievalMsg := api.NewTestMessage("Retrieval").
		WithPackage("test.v1").
		WithOneOfs(
			api.NewTestOneOf("source").WithFields(
				api.NewTestField("vertex_ai_search").WithMessageType(vertexAiSearchMsg),
				api.NewTestField("disable_attribution").WithType(api.TypezBool),
			),
		)

	extDnsConfigMsg := api.NewTestMessage("DNSConfig").WithPackage("google.type")

	outDir := t.TempDir()
	model := api.NewTestAPI([]*api.Message{vertexAiSearchMsg, dataStoreSpecMsg, retrievalMsg}, []*api.Enum{ipVersionEnum, nestedEnum}, nil)
	model.ExternalMessages = []*api.Message{extDnsConfigMsg}
	if err := api.CrossReference(model); err != nil {
		t.Fatal(err)
	}

	cfg := &parser.ModelConfig{
		SpecificationFormat: libconfig.SpecProtobuf,
		Codec: map[string]string{
			"package:wkt":               "source=google.protobuf,package=google-cloud-wkt",
			"package:google-cloud-type": "source=google.type,package=google-cloud-type",
			"template-override":         "templates/convert-prost",
		},
	}
	if err := Generate(t.Context(), model, outDir, cfg); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(filepath.Join(outDir, "convert.rs"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(contents)

	for _, test := range []struct {
		name      string
		start     string
		end       string
		wantBlock string
	}{
		{
			name:  "top-level message with acronym ToProto",
			start: "impl gaxi::prost::ToProto<VertexAiSearch>",
			end:   "\n        })\n    }\n}",
			wantBlock: `impl gaxi::prost::ToProto<VertexAiSearch> for crate::model::VertexAISearch {
    type Output = VertexAiSearch;
    fn to_proto(self) -> std::result::Result<VertexAiSearch, gaxi::prost::ConvertError> {
        Ok(Self::Output {
            serving_config: self.serving_config.to_proto()?,
        })
    }
}`,
		},
		{
			name:  "top-level message with acronym FromProto",
			start: "impl gaxi::prost::FromProto<crate::model::VertexAISearch>",
			end:   "\n        )\n    }\n}",
			wantBlock: `impl gaxi::prost::FromProto<crate::model::VertexAISearch> for VertexAiSearch {
    fn cnv(self) -> std::result::Result<crate::model::VertexAISearch, gaxi::prost::ConvertError> {
        Ok(
            crate::model::VertexAISearch::new()
                .set_serving_config(self.serving_config)
        )
    }
}`,
		},
		{
			name:  "nested message inside message with acronym ToProto",
			start: "impl gaxi::prost::ToProto<vertex_ai_search::DataStoreSpec>",
			end:   "\n        })\n    }\n}",
			wantBlock: `impl gaxi::prost::ToProto<vertex_ai_search::DataStoreSpec> for crate::model::vertex_ai_search::DataStoreSpec {
    type Output = vertex_ai_search::DataStoreSpec;
    fn to_proto(self) -> std::result::Result<vertex_ai_search::DataStoreSpec, gaxi::prost::ConvertError> {
        Ok(Self::Output {
            data_store: self.data_store.to_proto()?,
        })
    }
}`,
		},
		{
			name:  "enum with acronym ToProto",
			start: "impl gaxi::prost::ToProto<IpVersion>",
			end:   "    }\n}",
			wantBlock: `impl gaxi::prost::ToProto<IpVersion> for crate::model::IPVersion {
    type Output = i32;
    fn to_proto(self) -> std::result::Result<Self::Output, gaxi::prost::ConvertError> {
        self.value().ok_or(gaxi::prost::ConvertError::EnumNoIntegerValue("crate::model::IPVersion"))
    }
}`,
		},
		{
			name:  "nested enum with acronym inside message with acronym ToProto",
			start: "impl gaxi::prost::ToProto<vertex_ai_search::IpMode>",
			end:   "    }\n}",
			wantBlock: `impl gaxi::prost::ToProto<vertex_ai_search::IpMode> for crate::model::vertex_ai_search::IPMode {
    type Output = i32;
    fn to_proto(self) -> std::result::Result<Self::Output, gaxi::prost::ConvertError> {
        self.value().ok_or(gaxi::prost::ConvertError::EnumNoIntegerValue("crate::model::vertex_ai_search::IPMode"))
    }
}`,
		},
		{
			name:  "oneof on message with branch referencing message with acronym ToProto",
			start: "impl gaxi::prost::ToProto<retrieval::Source>",
			end:   "\n        }\n    }\n}",
			wantBlock: `impl gaxi::prost::ToProto<retrieval::Source> for crate::model::retrieval::Source {
    type Output = retrieval::Source;
    fn to_proto(self) -> std::result::Result<Self::Output, gaxi::prost::ConvertError> {
        match self {
            Self::VertexAiSearch(v) => Ok(Self::Output::VertexAiSearch((*v).to_proto()?)),
            Self::DisableAttribution(v) => Ok(Self::Output::DisableAttribution(v.to_proto()?)),
        }
    }
}`,
		},
		{
			name:  "external message with acronym ToProto",
			start: "impl gaxi::prost::ToProto<crate::prost::google::r#type::DnsConfig>",
			end:   "\n        })\n    }\n}",
			wantBlock: `impl gaxi::prost::ToProto<crate::prost::google::r#type::DnsConfig> for google_cloud_type::model::DNSConfig {
    type Output = crate::prost::google::r#type::DnsConfig;
    fn to_proto(self) -> std::result::Result<crate::prost::google::r#type::DnsConfig, gaxi::prost::ConvertError> {
        Ok(Self::Output {
        })
    }
}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extractBlock(t, contentStr, test.start, test.end)
			if diff := cmp.Diff(test.wantBlock, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
