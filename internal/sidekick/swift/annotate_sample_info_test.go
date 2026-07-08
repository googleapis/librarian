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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestAnnotateSampleInfo(t *testing.T) {
	for _, test := range []struct {
		name   string
		method *api.Method
		want   *sampleInfoAnnotation
	}{
		{
			name: "nil SampleInfo",
			method: &api.Method{
				Name: "TestMethod",
			},
			want: nil,
		},
		{
			name: "ResourceNameField with pattern",
			method: &api.Method{
				Name: "TestMethod",
				SampleInfo: &api.SampleInfo{
					ResourceNameField: &api.Field{
						Name: "secret",
						Codec: &fieldAnnotations{
							Name: "secretField",
						},
						ResourceNamePattern: &api.ResourceNamePattern{
							Segments: []api.ResourceNameSegment{
								{Literal: "projects"},
								{Variable: "project"},
								{Literal: "secrets"},
								{Variable: "secret"},
							},
						},
					},
				},
			},
			want: &sampleInfoAnnotation{
				Parameters:   []string{"projectId", "secretId"},
				FormatString: "projects/\\(projectId)/secrets/\\(secretId)",
				Name:         "secretField",
			},
		},
		{
			name: "ResourceNameField without pattern",
			method: &api.Method{
				Name: "TestMethod",
				SampleInfo: &api.SampleInfo{
					ResourceNameField: &api.Field{
						Name: "secret",
						Codec: &fieldAnnotations{
							Name: "secretField",
						},
					},
				},
			},
			want: &sampleInfoAnnotation{
				Parameters:   []string{"secretField"},
				Name:         "secretField",
				FormatString: "\\(secretField)",
			},
		},
		{
			name: "AIP standard update",
			method: &api.Method{
				Name:                "TestMethod",
				IsAIPStandardUpdate: true,
				SampleInfo:          &api.SampleInfo{},
			},
			want: &sampleInfoAnnotation{
				Parameters:   []string{"name"},
				Name:         "name",
				FormatString: "\\(name)",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			codec := &codec{}
			codec.annotateSampleInfo(test.method)

			var got *sampleInfoAnnotation
			if test.method.SampleInfo != nil {
				if test.method.SampleInfo.Codec != nil {
					got = test.method.SampleInfo.Codec.(*sampleInfoAnnotation)
				}
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
