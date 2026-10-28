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

package codec_sample

import "github.com/googleapis/librarian/internal/sidekick/api"

type messageAnnotations struct {
	Name string
}

func (c *codec) annotateMessage(m *api.Message) error {
	mAnn := &messageAnnotations{
		Name: m.Name + "Sample",
	}
	m.Codec = mAnn
	for _, f := range m.Fields {
		if err := c.annotateField(f, mAnn); err != nil {
			return err
		}
	}
	for _, o := range m.OneOfs {
		if err := c.annotateOneOf(o, mAnn); err != nil {
			return err
		}
	}
	for _, child := range m.Messages {
		if err := c.annotateMessage(child); err != nil {
			return err
		}
	}
	for _, e := range m.Enums {
		if err := c.annotateEnum(e); err != nil {
			return err
		}
	}
	return nil
}
