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

package python

import (
	"github.com/googleapis/librarian/internal/license"
)

// modelAnnotations decorates api.API with Python-specific metadata.
type modelAnnotations struct {
	CopyrightYear  string
	BoilerPlate    []string
	PackageName    string
	PackageVersion string
	Messages       []*messageAnnotations
	Enums          []*enumAnnotations
	Services       []*serviceAnnotations
}

func (c *codec) annotateModel() error {
	ann := &modelAnnotations{
		CopyrightYear:  c.GenerationYear,
		BoilerPlate:    license.HeaderBulk(),
		PackageName:    c.PackageName,
		PackageVersion: c.PackageVersion,
	}
	c.Model.Codec = ann

	// Annotate messages first so services can reference message types.
	for _, message := range c.Model.Messages {
		if err := c.annotateMessage(message, ann); err != nil {
			return err
		}
		if mAnn, ok := message.Codec.(*messageAnnotations); ok {
			ann.Messages = append(ann.Messages, mAnn)
		}
	}

	// Annotate enums.
	for _, enum := range c.Model.Enums {
		if err := c.annotateEnum(enum, ann); err != nil {
			return err
		}
		if eAnn, ok := enum.Codec.(*enumAnnotations); ok {
			ann.Enums = append(ann.Enums, eAnn)
		}
	}

	// Annotate services after messages and enums.
	for _, service := range c.Model.Services {
		if err := c.annotateService(service, ann); err != nil {
			return err
		}
		if sAnn, ok := service.Codec.(*serviceAnnotations); ok {
			ann.Services = append(ann.Services, sAnn)
		}
	}

	return nil
}
