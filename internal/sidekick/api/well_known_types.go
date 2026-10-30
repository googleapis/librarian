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
	model.AddEnum(NewEnumBuilder().
		WithID(WktNullValueID).
		WithName("NullValue").
		WithPackage("google.protobuf").
		Build())
}

var wellKnownMessages = []*Message{
	{
		id:       WktAnyID,
		name:     "Any",
		packagez: "google.protobuf",
	},
	{
		id:       WktStructID,
		name:     "Struct",
		packagez: "google.protobuf",
	},
	{
		id:       WktValueID,
		name:     "Value",
		packagez: "google.protobuf",
	},
	{
		id:       WktListValueID,
		name:     "ListValue",
		packagez: "google.protobuf",
	},
	{
		id:       WktEmptyID,
		name:     "Empty",
		packagez: "google.protobuf",
	},
	{
		id:       WktFieldMaskID,
		name:     "FieldMask",
		packagez: "google.protobuf",
		fields: []*Field{
			{
				name:     "paths",
				jsonName: "paths",
				typez:    TypezString,
				repeated: true,
			},
		},
	},
	{
		id:       WktDurationID,
		name:     "Duration",
		packagez: "google.protobuf",
	},
	{
		id:       WktTimestampID,
		name:     "Timestamp",
		packagez: "google.protobuf",
	},
	{id: WktBytesValueID, name: "BytesValue", packagez: "google.protobuf"},
	{id: WktUInt64ValueID, name: "UInt64Value", packagez: "google.protobuf"},
	{id: WktInt64ValueID, name: "Int64Value", packagez: "google.protobuf"},
	{id: WktUInt32ValueID, name: "UInt32Value", packagez: "google.protobuf"},
	{id: WktInt32ValueID, name: "Int32Value", packagez: "google.protobuf"},
	{id: WktFloatValueID, name: "FloatValue", packagez: "google.protobuf"},
	{id: WktDoubleValueID, name: "DoubleValue", packagez: "google.protobuf"},
	{id: WktBoolValueID, name: "BoolValue", packagez: "google.protobuf"},
}
