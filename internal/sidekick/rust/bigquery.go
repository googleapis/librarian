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

package rust

import (
	"fmt"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type bigQueryRunQueryFieldAnnotations struct {
	PrimType                 string
	KeyType                  string
	ValueType                string
	IsOrClear                bool
	IsCopy                   bool
	HasQueryRequest          bool
	HasJobConfigurationQuery bool
	HasJobConfiguration      bool
}

// generateBigQueryRunQueryFields generates the forwarding fields for the unified RunQuery builder.
func generateBigQueryRunQueryFields(model *api.API) ([]*api.Field, error) {
	var qrMsg, jcqMsg, jcMsg *api.Message

	// Find the target low-level messages in the model
	for _, msg := range model.Messages {
		switch msg.Name {
		case "QueryRequest":
			qrMsg = msg
		case "JobConfigurationQuery":
			jcqMsg = msg
		case "JobConfiguration":
			jcMsg = msg
		}
	}

	if qrMsg == nil || jcqMsg == nil || jcMsg == nil {
		return nil, fmt.Errorf("failed to locate QueryRequest, JobConfigurationQuery, or JobConfiguration messages")
	}

	// Index fields by their name and collect unique names
	var allFieldNames []string
	qrFields := make(map[string]*api.Field)
	for _, f := range qrMsg.Fields {
		qrFields[f.Name] = f
		allFieldNames = append(allFieldNames, f.Name)
	}

	jcqFields := make(map[string]*api.Field)
	for _, f := range jcqMsg.Fields {
		jcqFields[f.Name] = f
		allFieldNames = append(allFieldNames, f.Name)
	}

	jcFields := make(map[string]*api.Field)
	for _, f := range jcMsg.Fields {
		jcFields[f.Name] = f
		allFieldNames = append(allFieldNames, f.Name)
	}

	slices.Sort(allFieldNames)
	allFieldNames = slices.Compact(allFieldNames)

	skippedFields := []string{"query", "copy", "load", "extract", "format_options", "kind", "job_type"}
	var setters []*api.Field

	for _, fieldName := range allFieldNames {
		if slices.Contains(skippedFields, fieldName) {
			continue
		}

		qrF := qrFields[fieldName]
		jcqF := jcqFields[fieldName]
		jcF := jcFields[fieldName]

		isOutputOnly := func(f *api.Field) bool {
			if f == nil {
				return true
			}
			return slices.Contains(f.Behavior, api.FieldBehaviorOutputOnly)
		}

		if isOutputOnly(qrF) && isOutputOnly(jcqF) && isOutputOnly(jcF) {
			continue
		}

		// Generate normal and or_clear setters where applicable
		variations := []struct {
			suffix    string
			isOrClear bool
		}{
			{suffix: "", isOrClear: false},
			{suffix: "_or_clear", isOrClear: true},
		}

		for _, v := range variations {
			methodName := fmt.Sprintf("set%s_%s", v.suffix, fieldName)

			// Determine which model targets have this setter variation
			hasQueryRequest := hasSetter(qrF, v.isOrClear)
			hasJobConfigurationQuery := hasSetter(jcqF, v.isOrClear)
			hasJobConfiguration := hasSetter(jcF, v.isOrClear)

			if !hasQueryRequest && !hasJobConfigurationQuery && !hasJobConfiguration {
				continue
			}

			// Get the primary field and its annotations for types
			primaryField := firstNonNull(qrF, jcqF, jcF)

			fAnn := primaryField.Codec.(*fieldAnnotations)

			primType := strings.ReplaceAll(fAnn.PrimitiveFieldType, "crate::model", "google_cloud_bigquery_v2::model")
			keyType := strings.ReplaceAll(fAnn.KeyType, "crate::model", "google_cloud_bigquery_v2::model")
			valType := strings.ReplaceAll(fAnn.ValueType, "crate::model", "google_cloud_bigquery_v2::model")

			// Build the list of reference links to all targets that support this setter
			var links []string
			if hasQueryRequest {
				links = append(links, fmt.Sprintf("[%s][google_cloud_bigquery_v2::model::QueryRequest::%s]", fieldName, fieldName))
			}
			if hasJobConfigurationQuery {
				links = append(links, fmt.Sprintf("[%s][google_cloud_bigquery_v2::model::JobConfigurationQuery::%s]", fieldName, fieldName))
			}
			if hasJobConfiguration {
				links = append(links, fmt.Sprintf("[%s][google_cloud_bigquery_v2::model::JobConfiguration::%s]", fieldName, fieldName))
			}

			var docLine string
			if v.isOrClear {
				docLine = fmt.Sprintf("Sets or clears the value of %s.", strings.Join(links, " and "))
			} else {
				docLine = fmt.Sprintf("Sets the value of %s.", strings.Join(links, " and "))
			}

			// primitive and wkt wrapper types in Rust implementing the Copy trait
			isCopy := primType == "bool" || primType == "i32" || primType == "i64" || primType == "f64" ||
				primType == "wkt::BoolValue" || primType == "wkt::Int32Value" || primType == "wkt::Int64Value" ||
				primType == "wkt::UInt32Value" || primType == "wkt::UInt64Value" || primType == "wkt::FloatValue" ||
				primType == "wkt::DoubleValue"

			setters = append(setters, &api.Field{
				Name:          methodName,
				Map:           primaryField.Map,
				Repeated:      primaryField.Repeated,
				Documentation: docLine,
				Codec: &bigQueryRunQueryFieldAnnotations{
					PrimType:                 primType,
					KeyType:                  keyType,
					ValueType:                valType,
					IsOrClear:                v.isOrClear,
					IsCopy:                   isCopy,
					HasQueryRequest:          hasQueryRequest,
					HasJobConfigurationQuery: hasJobConfigurationQuery,
					HasJobConfiguration:      hasJobConfiguration,
				},
			})
		}
	}

	return setters, nil
}

func hasSetter(f *api.Field, isOrClear bool) bool {
	if f == nil {
		return false
	}
	return !isOrClear || f.Optional
}

func firstNonNull(fields ...*api.Field) *api.Field {
	for _, f := range fields {
		if f != nil {
			return f
		}
	}
	return nil
}
