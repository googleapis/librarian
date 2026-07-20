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

package api

import "fmt"

// Typez represent different field types that may be found in messages.
type Typez int

// These are the different field types as defined in
// descriptorpb.FieldDescriptorProto_Type.
const (
	TypezUndefined Typez = 0
	TypezDouble    Typez = 1
	TypezFloat     Typez = 2
	TypezInt64     Typez = 3
	TypezUint64    Typez = 4
	TypezInt32     Typez = 5
	TypezFixed64   Typez = 6
	TypezFixed32   Typez = 7
	TypezBool      Typez = 8
	TypezString    Typez = 9
	TypezGroup     Typez = 10
	TypezMessage   Typez = 11
	TypezBytes     Typez = 12
	TypezUint32    Typez = 13
	TypezEnum      Typez = 14
	TypezSfixed32  Typez = 15
	TypezSfixed64  Typez = 16
	TypezSint32    Typez = 17
	TypezSint64    Typez = 18
)

var typezName = [...]string{
	"UNDEFINED",
	"DOUBLE",
	"FLOAT",
	"INT64",
	"UINT64",
	"INT32",
	"FIXED64",
	"FIXED32",
	"BOOL",
	"STRING",
	"GROUP",
	"MESSAGE",
	"BYTES",
	"UINT32",
	"ENUM",
	"SFIXED32",
	"SFIXED64",
	"SINT32",
	"SINT64",
}

// String returns the symbolic name for the Typez.
func (t Typez) String() string {
	if t < 0 || int(t) >= len(typezName) {
		return fmt.Sprintf("Typez(%d)", t)
	}
	return typezName[t]
}
