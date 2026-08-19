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
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type methodAnnotations struct {
	Name           string
	DocLines       []string
	PathVariables  []*pathVariable
	RoutingParams  []*routingParam
	PathExpression string
	HTTPMethod     string
	HasBody        bool
	IsBodyWildcard bool
	BodyField      string
	QueryParams    []*api.Field
	Pagination     *paginationAnnotations
	LRO            *lroAnnotations
	DiscoveryLRO   *discoveryLroAnnotations
	ReturnType     string

	// ResponseEncoding sets the `$alt` query parameter value.
	//
	// To understand the motivation, see the field by the same in the the
	// `codec` data structure.
	ResponseEncoding string
}

type paginationAnnotations struct {
	ItemType string
}

type lroAnnotations struct {
	ReturnType      string
	MetadataType    string
	ResponseIsEmpty bool
}

// discoveryLroAnnotations defines the annotations for a discovery-based LRO.
type discoveryLroAnnotations struct {
	// The return type for the LRO.
	//
	// This is always the operation type (e.g.
	// `.google.cloud.compute.v1.Operation`) converted to the corresponding
	// Swift type (e.g. `GoogleCloudComputeV1.Operation`)
	ReturnType string

	// The names of the path parameters.
	//
	// The LRO body must initialize these parameters.
	PollingPathParameters []string
}

// pathVariable describes a variable used to build a request URL path.
//
// Most services have a single path variable, something like `request.parent` or `request.name`,
// where the field is a (required) string.
//
// In general they can take more complex forms, including:
//   - `request.secret.name` where `secret` is optional and name is a string, typically a full
//     resource name.
//   - `request.name` where `name` is an optional string (common in OpenAPI and discovery docs).
//   - `request.value` where the value is some enum, or integer field.
//   - `request.project` and `request.resource` where each is a string and are combined to construct
//     the path (again, common in OpenAPI and discovery docs).
//
// And of course all of these can be combined, such as nested fields that point to enums or nested
// fields that point to nested fields.
type pathVariable struct {
	Name       string
	Expression string
	Test       string
	FieldPath  string
}

// HasQueryParams returns true if the method's default binding has query parameters
//
// The mustache templates use this to (1) use a `var query` vs. `let query` for the collection of
// query parameters, and (2) generate the query parameter encoder only once, and only if needed.
func (ann *methodAnnotations) HasQueryParams() bool {
	return len(ann.QueryParams) != 0
}

// HasPathVariables returns true if the method has path variables for routing headers.
func (ann *methodAnnotations) HasPathVariables() bool {
	return len(ann.PathVariables) != 0
}

// HasRoutingParams returns true if the method has routing parameters for gRPC x-goog-request-params header.
func (ann *methodAnnotations) HasRoutingParams() bool {
	return len(ann.RoutingParams) != 0
}

// PlainRPC returns true if the method is not a pagination or LRO.
func (ann *methodAnnotations) PlainRPC() bool {
	return ann.LRO == nil && ann.Pagination == nil && ann.DiscoveryLRO == nil
}

func (c *codec) annotateMethod(method *api.Method, modelAnn *modelAnnotations) error {
	if method.InputType != nil {
		if err := c.annotateMessage(method.InputType, modelAnn); err != nil {
			return err
		}
	}
	var returnType string
	if method.OutputType != nil {
		if err := c.annotateMessage(method.OutputType, modelAnn); err != nil {
			return err
		}
		var err error
		returnType, err = c.fullyQualifiedMessageTypeName(method.OutputType)
		if err != nil {
			return err
		}
	}
	docLines, err := c.formatDocumentation(method.Documentation, method.Scopes())
	if err != nil {
		return err
	}
	var pathExpressionStr string
	var pathVariables []*pathVariable
	var routingParams []*routingParam
	var httpMethod string
	var hasBody bool
	var isBodyWildcard bool
	var bodyField string
	var queryParams []*api.Field

	// Extract routing parameters for gRPC per AIP-4222:
	// - Prefer explicit google.api.routing annotations if present.
	// - Fall back to extracting path parameters from the google.api.http path template.
	if method.HasRouting() {
		routingParams = c.routingParamsFromRouting(method.Routing)
	} else if method.PathInfo != nil && len(method.PathInfo.Bindings) > 0 {
		routingParams = c.routingParamsFromPathTemplate(method.PathInfo.Bindings[0].PathTemplate)
	}

	// In Protobuf APIs (such as Storage Control or pure gRPC services), some RPC
	// methods do not define google.api.http annotations.
	// - HTTP/REST methods: Extract httpMethod, pathExpressionStr, bodyField,
	//   queryParams, and pathVariables as before.
	// - Pure gRPC methods (without HTTP annotations): Safely bypass the HTTP
	//   extraction without crashing, leaving those fields as their default zero values.
	if method.PathInfo != nil && len(method.PathInfo.Bindings) > 0 {
		binding := method.PathInfo.Bindings[0]
		hasBody = method.PathInfo.BodyFieldPath != ""
		isBodyWildcard = method.PathInfo.BodyFieldPath == "*"
		if hasBody && !isBodyWildcard {
			bodyField = camelCase(method.PathInfo.BodyFieldPath)
		}
		var err error
		pathVariables, err = c.pathVariables(method.InputType, binding.PathTemplate)
		if err != nil {
			return err
		}
		pathExpressionStr = pathExpression(binding.PathTemplate)
		httpMethod = binding.Verb
		queryParams = language.QueryParams(method, binding)
	}

	var pagination *paginationAnnotations
	if method.Pagination != nil && method.OutputType != nil && method.OutputType.Pagination != nil {
		itemField := method.OutputType.Pagination.PageableItem
		itemFieldCodec, ok := itemField.Codec.(*fieldAnnotations)
		if !ok {
			return fmt.Errorf("internal error: pageable item field %q is not annotated", itemField.Name)
		}
		pagination = &paginationAnnotations{
			ItemType: itemFieldCodec.BaseFieldType,
		}
	}
	var lro *lroAnnotations
	if method.IsLRO && method.OperationInfo != nil {
		respMsg, err := lookupMessage(c.Model, method.OperationInfo.ResponseTypeID)
		if err != nil {
			return err
		}
		if err := c.annotateMessage(respMsg, modelAnn); err != nil {
			return err
		}
		respTypeName, err := c.messageTypeName(respMsg)
		if err != nil {
			return err
		}
		metaMsg, err := lookupMessage(c.Model, method.OperationInfo.MetadataTypeID)
		if err != nil {
			return err
		}
		if err := c.annotateMessage(metaMsg, modelAnn); err != nil {
			return err
		}
		metaTypeName, err := c.messageTypeName(metaMsg)
		if err != nil {
			return err
		}
		responseIsEmpty := respMsg.ID == ".google.protobuf.Empty"
		if responseIsEmpty {
			respTypeName = "Swift.Void"
		}
		lro = &lroAnnotations{
			ReturnType:      respTypeName,
			MetadataType:    metaTypeName,
			ResponseIsEmpty: responseIsEmpty,
		}
	}

	var discoveryLRO *discoveryLroAnnotations
	if method.DiscoveryLro != nil {
		discoveryLRO = &discoveryLroAnnotations{
			ReturnType: returnType,
		}
		for _, p := range method.DiscoveryLro.PollingPathParameters {
			discoveryLRO.PollingPathParameters = append(discoveryLRO.PollingPathParameters, camelCase(p))
		}
	}
	method.Codec = &methodAnnotations{
		Name:             camelCase(method.Name),
		DocLines:         docLines,
		PathExpression:   pathExpressionStr,
		PathVariables:    pathVariables,
		RoutingParams:    routingParams,
		HTTPMethod:       httpMethod,
		HasBody:          hasBody,
		IsBodyWildcard:   isBodyWildcard,
		BodyField:        bodyField,
		QueryParams:      queryParams,
		Pagination:       pagination,
		LRO:              lro,
		ReturnType:       returnType,
		DiscoveryLRO:     discoveryLRO,
		ResponseEncoding: c.ResponseEncoding,
	}
	if method.SampleInfo != nil {
		c.annotateSampleInfo(method)
	}
	return nil
}

func (a *methodAnnotations) Idempotent() bool {
	return a.HTTPMethod == "GET" || a.HTTPMethod == "PUT"
}
