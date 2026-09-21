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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type methodAnnotations struct {
	Name                string
	DocLines            []string
	PathVariables       []*pathVariable
	PathBindings        []*pathBindingAnnotations
	HasMultipleBindings bool
	RoutingParams       []*routingParam
	PathExpression      string
	HTTPMethod          string
	HasBody             bool
	IsBodyWildcard      bool
	BodyField           string
	QueryParams         []*api.Field
	Pagination          *paginationAnnotations
	LRO                 *lroAnnotations
	DiscoveryLRO        *discoveryLroAnnotations
	ReturnType          string

	// ResponseEncoding sets the `$alt` query parameter value.
	//
	// To understand the motivation, see the field by the same in the the
	// `codec` data structure.
	ResponseEncoding string

	// --- Diagnose flags ---
	//
	// These mark the generated declarations that need `@diagnose` to suppress
	// a deprecation warning. They vary along two axes.
	//
	// The first is what the declaration names. Every declaration names the
	// request and response types, and the pagination and LRO signatures also
	// name the item type and the operation response type; all of those form
	// the *types* axis. Some declarations additionally name individual request
	// fields, either as parameters or by reading and writing them; those form
	// the *fields* axis.
	//
	// The second is whether the declaration is itself deprecated when the
	// method or its service is. The public client and protocol declarations
	// carry `@available(*, deprecated)`, and Swift does not report deprecated
	// references inside a deprecated declaration. The internal stub, retry,
	// logging and transport declarations never carry it, so they always need
	// the attribute.

	// DiagnoseTypes guards the public declarations that name only types.
	DiagnoseTypes bool

	// DiagnoseFields guards the public declarations that also name individual
	// request fields, such as the generated overloads.
	//
	// This is deliberately an over-approximation for the per-signature
	// overloads. It is computed over every request field, while each overload
	// names only the subset its `api.MethodSignature` selects, and the
	// pagination overload names only the page token. An overload whose own
	// fields are all live can therefore carry the attribute. Suppressing a
	// warning that is not raised is inert, and the alternative is a separate
	// annotation computed per signature.
	DiagnoseFields bool

	// DiagnoseStubTypes guards the internal declarations that name only types.
	DiagnoseStubTypes bool

	// DiagnoseStubFields guards the internal declarations that also name
	// individual request fields, such as the transports.
	DiagnoseStubFields bool

	// DiagnoseSnippet guards the sample function in the generated snippet.
	//
	// A snippet is a standalone executable. Nothing in it is deprecated, so
	// unlike the public client it needs the attribute even when the method or
	// its service is.
	DiagnoseSnippet bool
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
	// JSONFieldPath is FieldPath using the ProtoJSON field names, e.g. `secret.name`.
	//
	// The generated code uses this to skip the path parameters when serializing the request body.
	JSONFieldPath    string
	MatchingSegments []string
	TemplateString   string
}

func (p *pathVariable) HasMatchingSegments() bool {
	return len(p.MatchingSegments) > 0
}

func (p *pathVariable) MatchingSegmentsExpression() string {
	return "[" + strings.Join(p.MatchingSegments, ", ") + "]"
}

type pathBindingAnnotations struct {
	HTTPMethod       string
	PathExpression   string
	PathVariables    []*pathVariable
	QueryParams      []*api.Field
	HasQueryParams   bool
	HasPathVariables bool
	ResponseEncoding string
	// OmittedBodyFields are the ProtoJSON paths bound by this binding's path template.
	//
	// Only set for methods using `body: "*"`. Per the `google.api.http` rules the request body
	// contains the fields *not* bound by the path template.
	OmittedBodyFields []string
}

// OmittedBodyFieldsExpression returns OmittedBodyFields as a Swift array literal.
func (b *pathBindingAnnotations) OmittedBodyFieldsExpression() string {
	quoted := make([]string, 0, len(b.OmittedBodyFields))
	for _, field := range b.OmittedBodyFields {
		quoted = append(quoted, fmt.Sprintf("%q", field))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// HasOmittedBodyFields returns true if the request body must skip some path parameters.
func (ann *methodAnnotations) HasOmittedBodyFields() bool {
	idx := slices.IndexFunc(ann.PathBindings, func(b *pathBindingAnnotations) bool {
		return len(b.OmittedBodyFields) > 0
	})
	return idx != -1
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
	var pathBindings []*pathBindingAnnotations
	if method.PathInfo != nil && len(method.PathInfo.Bindings) > 0 {
		hasBody = method.PathInfo.BodyFieldPath != ""
		isBodyWildcard = method.PathInfo.BodyFieldPath == "*"
		if hasBody && !isBodyWildcard {
			bodyField = camelCase(method.PathInfo.BodyFieldPath)
		}
		for _, binding := range method.PathInfo.Bindings {
			pVars, err := c.pathVariables(method.InputType, binding.PathTemplate)
			if err != nil {
				return err
			}
			pExpr := pathExpression(binding.PathTemplate)
			qParams := language.QueryParams(method, binding)
			// With `body: "*"` the request body only contains the fields not bound by the path
			// template.
			var omittedBodyFields []string
			if isBodyWildcard {
				for _, pVar := range pVars {
					omittedBodyFields = append(omittedBodyFields, pVar.JSONFieldPath)
				}
			}
			pathBindings = append(pathBindings, &pathBindingAnnotations{
				HTTPMethod:        binding.Verb,
				PathExpression:    pExpr,
				PathVariables:     pVars,
				QueryParams:       qParams,
				HasQueryParams:    len(qParams) > 0,
				HasPathVariables:  len(pVars) > 0,
				ResponseEncoding:  c.ResponseEncoding,
				OmittedBodyFields: omittedBodyFields,
			})
		}
		primary := pathBindings[0]
		pathVariables = primary.PathVariables
		pathExpressionStr = primary.PathExpression
		httpMethod = primary.HTTPMethod
		queryParams = primary.QueryParams
	}

	// The pagination and LRO signatures name a type that is neither the input
	// nor the output type, so `diagnoseMethodTypes` cannot see it.
	diagnoseSignatureTypes := false

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
		// The pagination signatures return `any AsyncSequence<ItemType, ...>`.
		itemTypeDeprecated, err := c.fieldTypeDeprecated(itemField)
		if err != nil {
			return err
		}
		diagnoseSignatureTypes = diagnoseSignatureTypes || itemTypeDeprecated
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
		// The LRO signatures return `any GoogleGax.PollableOperation<ReturnType>`.
		// `MetadataType` needs nothing: no template names it.
		diagnoseSignatureTypes = diagnoseSignatureTypes || respMsg.Deprecated
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
	diagnoseTypes := diagnoseMethodTypes(method) || diagnoseSignatureTypes
	diagnoseFields, err := c.diagnoseRequestFields(method)
	if err != nil {
		return err
	}
	diagnoseFields = diagnoseFields || diagnoseTypes
	deprecatedScope := inDeprecatedScope(method)
	method.Codec = &methodAnnotations{
		Name:                camelCase(method.Name),
		DocLines:            docLines,
		PathExpression:      pathExpressionStr,
		PathVariables:       pathVariables,
		PathBindings:        pathBindings,
		HasMultipleBindings: len(pathBindings) > 1,
		RoutingParams:       routingParams,
		HTTPMethod:          httpMethod,
		HasBody:             hasBody,
		IsBodyWildcard:      isBodyWildcard,
		BodyField:           bodyField,
		QueryParams:         queryParams,
		Pagination:          pagination,
		LRO:                 lro,
		ReturnType:          returnType,
		DiscoveryLRO:        discoveryLRO,
		ResponseEncoding:    c.ResponseEncoding,
		DiagnoseTypes:       diagnoseTypes && !deprecatedScope,
		DiagnoseFields:      diagnoseFields && !deprecatedScope,
		DiagnoseStubTypes:   diagnoseTypes,
		DiagnoseStubFields:  diagnoseFields,
		DiagnoseSnippet:     diagnoseFields || deprecatedScope,
	}
	if method.SampleInfo != nil {
		c.annotateSampleInfo(method)
	}
	return nil
}

// diagnoseMethodTypes reports whether the request or the response type is
// deprecated.
//
// Every generated signature names both. The pagination item type and the LRO
// response type also appear in generated signatures, but they are neither the
// input nor the output type; `annotateMethod` folds them in separately, where
// they have already been resolved.
func diagnoseMethodTypes(method *api.Method) bool {
	return (method.InputType != nil && method.InputType.Deprecated) ||
		(method.OutputType != nil && method.OutputType.Deprecated)
}

// diagnoseRequestFields reports whether any field of the request message is
// deprecated, or names a deprecated type.
//
// The generated overloads assign the selected request fields by name, and the
// transports read them to build the path and the query string.
//
// This is the same predicate as `messageAnnotations.DiagnoseCodable` for the
// request message. It is recomputed rather than read from the annotation to
// keep `annotateMethod` independent of whether the request message has been
// annotated yet, which is not guaranteed for mixin methods whose request types
// live outside this model.
//
// Returning early for a deprecated request message is an optimisation, not a
// correctness requirement: `diagnoseMethodTypes` already covers that case, and
// the caller ORs the two together.
func (c *codec) diagnoseRequestFields(method *api.Method) (bool, error) {
	if method.InputType == nil || method.InputType.Deprecated {
		return false, nil
	}
	for _, field := range method.InputType.Fields {
		if field.Deprecated {
			return true, nil
		}
		deprecatedType, err := c.fieldTypeDeprecated(field)
		if err != nil {
			return false, err
		}
		if deprecatedType {
			return true, nil
		}
	}
	return false, nil
}

// inDeprecatedScope reports whether the public declarations generated for this
// method are themselves deprecated.
//
// Swift does not report deprecated references inside a deprecated declaration,
// so those declarations need no attribute. Every public declaration that this
// governs is preceded by `{{#Deprecated}} @available(*, deprecated) {{/Deprecated}}`
// in the templates.
func inDeprecatedScope(method *api.Method) bool {
	return method.Deprecated || (method.Service != nil && method.Service.Deprecated)
}

func (a *methodAnnotations) Idempotent() bool {
	return a.HTTPMethod == "GET" || a.HTTPMethod == "PUT"
}
