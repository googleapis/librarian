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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type sampleInfoAnnotation struct {
	// Parameters is the set of parameters shown on the sample method for any given RPC sample.
	Parameters []string

	// FormatString is the Swift format string to use with the parameters.
	//
	// For example, this may be `"projects/\(projectId)/secrets/\(secretId)`
	FormatString string

	// Name is the name of the resource field for the samples.
	Name string
}

func (c *codec) annotateSampleInfo(method *api.Method) {
	si := method.SampleInfo
	if si == nil {
		return
	}
	var ann *sampleInfoAnnotation
	if field := si.ResourceNameField; field != nil {
		fieldAnn := field.Codec.(*fieldAnnotations)
		if field.ResourceNamePattern != nil && len(field.ResourceNamePattern.Segments) > 0 {
			ann = c.resourceNameToSampleInfo(field.ResourceNamePattern, fieldAnn)
		} else if fieldAnn != nil {
			ann = &sampleInfoAnnotation{
				Parameters:   []string{fieldAnn.Name},
				Name:         fieldAnn.Name,
				FormatString: fmt.Sprintf("\\(%s)", fieldAnn.Name),
			}
		}
	} else if method.IsAIPStandardUpdate {
		ann = &sampleInfoAnnotation{
			Parameters:   []string{"name"},
			Name:         "name",
			FormatString: "\\(name)",
		}
	}
	si.Codec = ann
}

func (c *codec) resourceNameToSampleInfo(pattern *api.ResourceNamePattern, fieldAnn *fieldAnnotations) *sampleInfoAnnotation {
	var formatString []string
	var parameters []string
	for _, s := range pattern.Segments {
		if s.Literal != "" {
			formatString = append(formatString, s.Literal)
		}
		if s.Variable != "" {
			// Convert variable to snake_case and add _id suffix if needed
			arg := s.Variable
			if !strings.HasSuffix(arg, "_id") && !strings.HasSuffix(arg, "_name") {
				arg += "_id"
			}
			arg = camelCase(arg)
			formatString = append(formatString, "\\("+arg+")")
			parameters = append(parameters, arg)
		}
	}
	if len(formatString) == 0 && len(parameters) == 0 {
		return nil
	}
	name := ""
	if fieldAnn != nil {
		name = fieldAnn.Name
	}
	return &sampleInfoAnnotation{
		FormatString: strings.Join(formatString, "/"),
		Parameters:   parameters,
		Name:         name,
	}
}
