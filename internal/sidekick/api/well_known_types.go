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

const (
	// WktAnyID is the well-known type ID for google.protobuf.Any.
	WktAnyID = ".google.protobuf.Any"
	// WktStructID is the well-known type ID for google.protobuf.Struct.
	WktStructID = ".google.protobuf.Struct"
	// WktValueID is the well-known type ID for google.protobuf.Value.
	WktValueID = ".google.protobuf.Value"
	// WktListValueID is the well-known type ID for google.protobuf.ListValue.
	WktListValueID = ".google.protobuf.ListValue"
	// WktEmptyID is the well-known type ID for google.protobuf.Empty.
	WktEmptyID = ".google.protobuf.Empty"
	// WktFieldMaskID is the well-known type ID for google.protobuf.FieldMask.
	WktFieldMaskID = ".google.protobuf.FieldMask"
	// WktDurationID is the well-known type ID for google.protobuf.Duration.
	WktDurationID = ".google.protobuf.Duration"
	// WktTimestampID is the well-known type ID for google.protobuf.Timestamp.
	WktTimestampID = ".google.protobuf.Timestamp"
	// WktNullValueID is the well-known type ID for google.protobuf.NullValue.
	WktNullValueID = ".google.protobuf.NullValue"
	// WktBytesValueID is the well-known type ID for google.protobuf.BytesValue.
	WktBytesValueID = ".google.protobuf.BytesValue"
	// WktUInt64ValueID is the well-known type ID for google.protobuf.UInt64Value.
	WktUInt64ValueID = ".google.protobuf.UInt64Value"
	// WktInt64ValueID is the well-known type ID for google.protobuf.Int64Value.
	WktInt64ValueID = ".google.protobuf.Int64Value"
	// WktUInt32ValueID is the well-known type ID for google.protobuf.UInt32Value.
	WktUInt32ValueID = ".google.protobuf.UInt32Value"
	// WktInt32ValueID is the well-known type ID for google.protobuf.Int32Value.
	WktInt32ValueID = ".google.protobuf.Int32Value"
	// WktFloatValueID is the well-known type ID for google.protobuf.FloatValue.
	WktFloatValueID = ".google.protobuf.FloatValue"
	// WktDoubleValueID is the well-known type ID for google.protobuf.DoubleValue.
	WktDoubleValueID = ".google.protobuf.DoubleValue"
	// WktBoolValueID is the well-known type ID for google.protobuf.BoolValue.
	WktBoolValueID = ".google.protobuf.BoolValue"
)

// LoadWellKnownTypes adds well-known types to `state`.
//
// Some source specification formats (Discovery, OpenAPI) must manually add the
// well-known types. In Protobuf these types are automatically defined in the
// protoc output.
func (model *API) LoadWellKnownTypes() {
	for _, message := range wellKnownMessages {
		model.AddMessage(message)
	}
	model.AddEnum(&Enum{
		ID:      WktNullValueID,
		Name:    "NullValue",
		Package: "google.protobuf",
	})
}

var wellKnownMessages = []*Message{
	{
		ID:      WktAnyID,
		Name:    "Any",
		Package: "google.protobuf",
	},
	{
		ID:      WktStructID,
		Name:    "Struct",
		Package: "google.protobuf",
	},
	{
		ID:      WktValueID,
		Name:    "Value",
		Package: "google.protobuf",
	},
	{
		ID:      WktListValueID,
		Name:    "ListValue",
		Package: "google.protobuf",
	},
	{
		ID:      WktEmptyID,
		Name:    "Empty",
		Package: "google.protobuf",
	},
	{
		ID:      WktFieldMaskID,
		Name:    "FieldMask",
		Package: "google.protobuf",
		Fields: []*Field{
			{
				Name:     "paths",
				JSONName: "paths",
				Typez:    TypezString,
				Repeated: true,
			},
		},
	},
	{
		ID:      WktDurationID,
		Name:    "Duration",
		Package: "google.protobuf",
	},
	{
		ID:      WktTimestampID,
		Name:    "Timestamp",
		Package: "google.protobuf",
	},
	{ID: WktBytesValueID, Name: "BytesValue", Package: "google.protobuf"},
	{ID: WktUInt64ValueID, Name: "UInt64Value", Package: "google.protobuf"},
	{ID: WktInt64ValueID, Name: "Int64Value", Package: "google.protobuf"},
	{ID: WktUInt32ValueID, Name: "UInt32Value", Package: "google.protobuf"},
	{ID: WktInt32ValueID, Name: "Int32Value", Package: "google.protobuf"},
	{ID: WktFloatValueID, Name: "FloatValue", Package: "google.protobuf"},
	{ID: WktDoubleValueID, Name: "DoubleValue", Package: "google.protobuf"},
	{ID: WktBoolValueID, Name: "BoolValue", Package: "google.protobuf"},
}
