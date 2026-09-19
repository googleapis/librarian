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

package cpp

import (
	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

type modelAnnotations struct {
	CopyrightYear string
	BoilerPlate   []string
}

func (c *codec) annotateModel(model *api.API) error {
	year := "2026"
	if c.config != nil && c.config.InitialCopyrightYear != "" {
		year = c.config.InitialCopyrightYear
	}
	modelAnn := &modelAnnotations{
		CopyrightYear: year,
		BoilerPlate:   license.HeaderBulk(),
	}
	model.Codec = modelAnn
	for _, m := range model.Messages {
		if err := c.annotateMessage(m); err != nil {
			return err
		}
	}
	for _, e := range model.Enums {
		if err := c.annotateEnum(e); err != nil {
			return err
		}
	}
	for _, s := range model.Services {
		if err := c.annotateService(s, modelAnn, model); err != nil {
			return err
		}
	}
	return nil
}
