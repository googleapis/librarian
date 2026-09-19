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

type serviceAnnotations struct {
	Name          string
	CopyrightYear string
	BoilerPlate   []string
	Model         *modelAnnotations
	SourceFile    string
}

func (c *codec) annotateService(s *api.Service) error {
	var modelAnn *modelAnnotations
	year := "2026"
	if c.config != nil && c.config.InitialCopyrightYear != "" {
		year = c.config.InitialCopyrightYear
	}
	boilerPlate := license.HeaderBulk()
	if s.Model != nil {
		if ann, ok := s.Model.Codec.(*modelAnnotations); ok {
			modelAnn = ann
			year = ann.CopyrightYear
			boilerPlate = ann.BoilerPlate
		}
	}
	var sourceFile string
	if s.Model != nil {
		if loc, ok := s.Model.DefinitionLocation(s.ID); ok {
			sourceFile = loc.Filename
		}
	}
	sAnn := &serviceAnnotations{
		Name:          s.Name,
		CopyrightYear: year,
		BoilerPlate:   boilerPlate,
		Model:         modelAnn,
		SourceFile:    sourceFile,
	}
	s.Codec = sAnn
	for _, m := range s.Methods {
		if err := c.annotateMethod(m, sAnn); err != nil {
			return err
		}
	}
	return nil
}
