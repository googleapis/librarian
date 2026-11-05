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

import (
	"fmt"
	"strings"
)

// DocumentationOverride describes overrides for the documentation of a single element.
//
// This should be used sparingly. Generally we should prefer updating the
// comments upstream, and then getting a new version of the services'
// specification. The exception may be when the fixes take a long time, or are
// specific to one language.
type DocumentationOverride struct {
	ID      string
	Match   string
	Replace string
}

// PatchDocumentation overrides the documentation of the API model with the provided overrides.
func PatchDocumentation(model *API, overrides []DocumentationOverride) error {
	for _, override := range overrides {
		id := override.ID
		if msg := model.Message(id); msg != nil {
			if err := patchElementDocs(&msg.documentation, &override); err != nil {
				return err
			}
			continue
		}
		if enu := model.Enum(id); enu != nil {
			if err := patchElementDocs(&enu.documentation, &override); err != nil {
				return err
			}
			continue
		}
		if svc := model.Service(id); svc != nil {
			if err := patchElementDocs(&svc.documentation, &override); err != nil {
				return err
			}
			continue
		}
		idx := strings.LastIndex(id, ".")
		if idx == -1 {
			return fmt.Errorf("cannot find element %s to apply comment overrides", id)
		}
		parentId := id[0:idx]
		childId := id[idx+1:]
		if msg := model.Message(parentId); msg != nil {
			if err := patchFieldDocs(msg, childId, &override); err != nil {
				return err
			}
			continue
		}
		if enu := model.Enum(parentId); enu != nil {
			if err := patchEnumValueDocs(enu, childId, &override); err != nil {
				return err
			}
			continue
		}
		if svc := model.Service(parentId); svc != nil {
			if err := patchMethodDocs(svc, childId, &override); err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("cannot find element %s to apply comment overrides, only searched for messages, enums and services", id)
	}
	return nil
}

func patchFieldDocs(msg *Message, fieldName string, override *DocumentationOverride) error {
	for _, field := range msg.fields {
		if field.name != fieldName {
			continue
		}
		if err := patchElementDocs(&field.documentation, override); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("cannot find field %s in message %s to apply comment override", fieldName, msg.id)
}

func patchEnumValueDocs(enu *Enum, name string, override *DocumentationOverride) error {
	for _, v := range enu.values {
		if v.name != name {
			continue
		}
		if err := patchElementDocs(&v.documentation, override); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("cannot find field %s in message %s to apply comment override", name, enu.id)
}

func patchMethodDocs(svc *Service, name string, override *DocumentationOverride) error {
	for _, m := range svc.methods {
		if m.name != name {
			continue
		}
		if err := patchElementDocs(&m.documentation, override); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("cannot find field %s in message %s to apply comment override", name, svc.id)
}

func patchElementDocs(documentation *string, override *DocumentationOverride) error {
	new := strings.ReplaceAll(*documentation, strings.ReplaceAll(override.Match, "\r\n", "\n"), override.Replace)
	if *documentation == new {
		return fmt.Errorf("comment override for %s did not match", override.ID)
	}
	*documentation = new
	return nil
}
