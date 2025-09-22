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

package api

// LoadWellKnownTypes adds well-known types to `state`.
//
// Some source specification formats (Discovery, OpenAPI) must manually add the
// well-known types. In Protobuf these types are automatically defined in the
// protoc output.
func (model *API) LoadWellKnownTypes() {
	for _, message := range wellKnownMessages {
		model.State.MessageByID[message.ID] = message
	}
	model.State.EnumByID[".google.protobuf.NullValue"] = &Enum{
		Name:    "NullValue",
		Package: "google.protobuf",
		ID:      ".google.protobuf.NullValue",
	}
}

var wellKnownMessages = []*Message{
	{
		ID:      ".google.protobuf.Any",
		Name:    "Any",
		Package: "google.protobuf",
	},
	{
		ID:      ".google.protobuf.Struct",
		Name:    "Struct",
		Package: "google.protobuf",
	},
	{
		ID:      ".google.protobuf.Value",
		Name:    "Value",
		Package: "google.protobuf",
	},
	{
		ID:      ".google.protobuf.ListValue",
		Name:    "ListValue",
		Package: "google.protobuf",
	},
	{
		ID:      ".google.protobuf.Empty",
		Name:    "Empty",
		Package: "google.protobuf",
	},
	{
		ID:      ".google.protobuf.FieldMask",
		Name:    "FieldMask",
		Package: "google.protobuf",
		Fields: []*Field{
			{
				Name:     "paths",
				JSONName: "paths",
				Typez:    STRING_TYPE,
				Repeated: true,
			},
		},
	},
	{
		ID:      ".google.protobuf.Duration",
		Name:    "Duration",
		Package: "google.protobuf",
	},
	{
		ID:      ".google.protobuf.Timestamp",
		Name:    "Timestamp",
		Package: "google.protobuf",
	},
	{ID: ".google.protobuf.BytesValue", Name: "BytesValue", Package: "google.protobuf"},
	{ID: ".google.protobuf.UInt64Value", Name: "UInt64Value", Package: "google.protobuf"},
	{ID: ".google.protobuf.Int64Value", Name: "Int64Value", Package: "google.protobuf"},
	{ID: ".google.protobuf.UInt32Value", Name: "UInt32Value", Package: "google.protobuf"},
	{ID: ".google.protobuf.Int32Value", Name: "Int32Value", Package: "google.protobuf"},
	{ID: ".google.protobuf.FloatValue", Name: "FloatValue", Package: "google.protobuf"},
	{ID: ".google.protobuf.DoubleValue", Name: "DoubleValue", Package: "google.protobuf"},
	{ID: ".google.protobuf.BoolValue", Name: "BoolValue", Package: "google.protobuf"},
}
