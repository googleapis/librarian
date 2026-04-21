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
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type enumValueAnnotations struct {
	CaseName    string
	Number      int32
	StringValue string
	DocLines    []string
}

func (codec *codec) annotateUniqueEnumValue(ev *api.EnumValue) *api.EnumValue {
	docLines := codec.formatDocumentation(ev.Documentation)
	ann := &enumValueAnnotations{
		CaseName:    enumValueCaseName(ev),
		Number:      ev.Number,
		StringValue: ev.Name,
		DocLines:    docLines,
	}
	ev.Codec = ann
	return ev
}

func (codec *codec) annotateEnumValue(ev *api.EnumValue, unique map[int32]*enumValueAnnotations) (*api.EnumValue, error) {
	if ev.Codec != nil {
		return ev, nil
	}
	existing, ok := unique[ev.Number]
	if !ok {
		return nil, fmt.Errorf("expected an existing annotation for %s as it duplicates the integer value %d", ev.Name, ev.Number)
	}
	ann := &enumValueAnnotations{
		CaseName:    existing.CaseName,
		Number:      ev.Number,
		StringValue: ev.Name,
	}
	ev.Codec = ann
	return ev, nil
}
