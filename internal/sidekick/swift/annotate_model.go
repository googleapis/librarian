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
	"github.com/googleapis/librarian/internal/license"
)

type modelAnnotations struct {
	CopyrightYear string
	BoilerPlate   []string
	PackageName   string
	MonorepoRoot  string
}

func (codec *codec) annotateModel() error {
	annotations := &modelAnnotations{
		CopyrightYear: codec.GenerationYear,
		BoilerPlate:   license.HeaderBulk(),
		PackageName:   codec.PackageName,
		MonorepoRoot:  codec.MonorepoRoot,
	}
	codec.Model.Codec = annotations
	for _, message := range codec.Model.Messages {
		if err := codec.annotateMessage(message, annotations); err != nil {
			return err
		}
	}
	for _, service := range codec.Model.Services {
		codec.annotateService(service, annotations)
	}
	return nil
}
