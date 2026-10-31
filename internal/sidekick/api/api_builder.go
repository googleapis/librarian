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

package api

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

// APIBuilder constructs an [API].
type APIBuilder struct {
	a                     API
	paginationOverrides   []PaginationOverride
	resourceNameHeuristic bool
	override              ModelOverride
	commentOverrides      []DocumentationOverride
}

// NewAPIBuilder creates a new [APIBuilder].
func NewAPIBuilder() *APIBuilder {
	return &APIBuilder{
		a: API{
			serviceByID:    make(map[string]*Service),
			methodByID:     make(map[string]*Method),
			messageByID:    make(map[string]*Message),
			enumByID:       make(map[string]*Enum),
			resourceByType: make(map[string]*Resource),
			mapByID:        make(map[string]*Map),
		},
	}
}

// NewAPIBuilderFrom creates a new [APIBuilder] initialized from an existing [API].
func NewAPIBuilderFrom(a *API) *APIBuilder {
	b := NewAPIBuilder()
	if a == nil {
		return b
	}
	b.a = API{
		name:                a.name,
		packageName:         a.packageName,
		title:               a.title,
		description:         a.description,
		revision:            a.revision,
		services:            slices.Clone(a.services),
		messages:            slices.Clone(a.messages),
		externalMessages:    slices.Clone(a.externalMessages),
		enums:               slices.Clone(a.enums),
		externalEnums:       slices.Clone(a.externalEnums),
		resourceDefinitions: slices.Clone(a.resourceDefinitions),
		quickstartService:   a.quickstartService,
		codec:               a.codec,
		serviceByID:         maps.Clone(a.serviceByID),
		methodByID:          maps.Clone(a.methodByID),
		messageByID:         maps.Clone(a.messageByID),
		enumByID:            maps.Clone(a.enumByID),
		resourceByType:      maps.Clone(a.resourceByType),
		mapByID:             maps.Clone(a.mapByID),
		rubyPackage:         a.rubyPackage,
		phpNamespace:        a.phpNamespace,
		csharpNamespace:     a.csharpNamespace,
	}
	if b.a.serviceByID == nil {
		b.a.serviceByID = make(map[string]*Service)
	}
	if b.a.methodByID == nil {
		b.a.methodByID = make(map[string]*Method)
	}
	if b.a.messageByID == nil {
		b.a.messageByID = make(map[string]*Message)
	}
	if b.a.enumByID == nil {
		b.a.enumByID = make(map[string]*Enum)
	}
	if b.a.resourceByType == nil {
		b.a.resourceByType = make(map[string]*Resource)
	}
	if b.a.mapByID == nil {
		b.a.mapByID = make(map[string]*Map)
	}
	return b
}

// NewAnnotatedAPI creates a minimal [API] whose Codec field is set to codec.
func NewAnnotatedAPI(codec any) *API {
	return &API{codec: codec}
}

// WithName sets the API name.
func (b *APIBuilder) WithName(name string) *APIBuilder {
	b.a.name = name
	return b
}

// Name returns the API name currently set on the builder.
func (b *APIBuilder) Name() string {
	return b.a.name
}

// WithPackageName sets the API package name.
func (b *APIBuilder) WithPackageName(packageName string) *APIBuilder {
	b.a.packageName = packageName
	return b
}

// PackageName returns the API package name currently set on the builder.
func (b *APIBuilder) PackageName() string {
	return b.a.packageName
}

// WithTitle sets the API title.
func (b *APIBuilder) WithTitle(title string) *APIBuilder {
	b.a.title = title
	return b
}

// Title returns the API title currently set on the builder.
func (b *APIBuilder) Title() string {
	return b.a.title
}

// WithDescription sets the API description.
func (b *APIBuilder) WithDescription(description string) *APIBuilder {
	b.a.description = description
	return b
}

// Description returns the API description currently set on the builder.
func (b *APIBuilder) Description() string {
	return b.a.description
}

// WithRevision sets the API revision.
func (b *APIBuilder) WithRevision(revision string) *APIBuilder {
	b.a.revision = revision
	return b
}

// WithServices sets the services slice.
func (b *APIBuilder) WithServices(services ...*Service) *APIBuilder {
	b.a.services = services
	return b
}

// AppendService appends a service to the services slice.
func (b *APIBuilder) AppendService(services ...*Service) *APIBuilder {
	b.a.services = append(b.a.services, services...)
	return b
}

// Services returns the services slice currently set on the builder.
func (b *APIBuilder) Services() []*Service {
	return slices.Clone(b.a.services)
}

// SortServices sorts the services slice using cmpFunc.
func (b *APIBuilder) SortServices(cmpFunc func(a, b *Service) int) *APIBuilder {
	slices.SortStableFunc(b.a.services, cmpFunc)
	return b
}

// WithMessages sets the messages slice.
func (b *APIBuilder) WithMessages(messages ...*Message) *APIBuilder {
	if len(messages) == 0 {
		b.a.messages = make([]*Message, 0)
	} else {
		b.a.messages = messages
	}
	return b
}

// AppendMessage appends a message to the messages slice.
func (b *APIBuilder) AppendMessage(messages ...*Message) *APIBuilder {
	b.a.messages = append(b.a.messages, messages...)
	return b
}

// Messages returns the messages slice currently set on the builder.
func (b *APIBuilder) Messages() []*Message {
	return slices.Clone(b.a.messages)
}

// SortMessages sorts the messages slice using cmpFunc.
func (b *APIBuilder) SortMessages(cmpFunc func(a, b *Message) int) *APIBuilder {
	slices.SortStableFunc(b.a.messages, cmpFunc)
	return b
}

// WithExternalMessages sets the external messages slice.
func (b *APIBuilder) WithExternalMessages(messages ...*Message) *APIBuilder {
	b.a.externalMessages = messages
	return b
}

// WithEnums sets the enums slice.
func (b *APIBuilder) WithEnums(enums ...*Enum) *APIBuilder {
	b.a.enums = enums
	return b
}

// AppendEnum appends an enum to the enums slice.
func (b *APIBuilder) AppendEnum(enums ...*Enum) *APIBuilder {
	b.a.enums = append(b.a.enums, enums...)
	return b
}

// Enums returns the enums slice currently set on the builder.
func (b *APIBuilder) Enums() []*Enum {
	return slices.Clone(b.a.enums)
}

// WithExternalEnums sets the external enums slice.
func (b *APIBuilder) WithExternalEnums(enums ...*Enum) *APIBuilder {
	b.a.externalEnums = enums
	return b
}

// WithResourceDefinitions sets the resource definitions slice.
func (b *APIBuilder) WithResourceDefinitions(resources ...*Resource) *APIBuilder {
	b.a.resourceDefinitions = resources
	return b
}

// AppendResourceDefinitions appends resources to the resource definitions slice.
func (b *APIBuilder) AppendResourceDefinitions(resources ...*Resource) *APIBuilder {
	b.a.resourceDefinitions = append(b.a.resourceDefinitions, resources...)
	return b
}

// ResourceDefinitions returns the resource definitions currently set on the builder.
func (b *APIBuilder) ResourceDefinitions() []*Resource {
	return slices.Clone(b.a.resourceDefinitions)
}

// WithQuickstartService sets the quickstart service.
func (b *APIBuilder) WithQuickstartService(s *Service) *APIBuilder {
	b.a.quickstartService = s
	return b
}

// WithRubyPackage sets the Ruby package override.
func (b *APIBuilder) WithRubyPackage(pkg string) *APIBuilder {
	b.a.rubyPackage = pkg
	return b
}

// RubyPackage returns the Ruby package override currently set on the builder.
func (b *APIBuilder) RubyPackage() string {
	return b.a.rubyPackage
}

// WithPhpNamespace sets the PHP namespace override.
func (b *APIBuilder) WithPhpNamespace(ns string) *APIBuilder {
	b.a.phpNamespace = ns
	return b
}

// PhpNamespace returns the PHP namespace override currently set on the builder.
func (b *APIBuilder) PhpNamespace() string {
	return b.a.phpNamespace
}

// WithCsharpNamespace sets the C# namespace override.
func (b *APIBuilder) WithCsharpNamespace(ns string) *APIBuilder {
	b.a.csharpNamespace = ns
	return b
}

// CsharpNamespace returns the C# namespace override currently set on the builder.
func (b *APIBuilder) CsharpNamespace() string {
	return b.a.csharpNamespace
}

// LoadWellKnownTypes populates well-known types into the symbol tables.
func (b *APIBuilder) LoadWellKnownTypes() *APIBuilder {
	b.a.loadWellKnownTypes()
	return b
}

// AddService registers a service in the symbol table.
func (b *APIBuilder) AddService(s *Service) *APIBuilder {
	b.a.addService(s)
	return b
}

// Service looks up a service in the symbol table.
func (b *APIBuilder) Service(id string) *Service {
	return b.a.Service(id)
}

// AddMethod registers a method in the symbol table.
func (b *APIBuilder) AddMethod(m *Method) *APIBuilder {
	b.a.addMethod(m)
	return b
}

// Method looks up a method in the symbol table.
func (b *APIBuilder) Method(id string) *Method {
	return b.a.Method(id)
}

// AddMessage registers a message in the symbol table.
func (b *APIBuilder) AddMessage(m *Message) *APIBuilder {
	b.a.addMessage(m)
	return b
}

// Message looks up a message in the symbol table.
func (b *APIBuilder) Message(id string) *Message {
	return b.a.Message(id)
}

// AddEnum registers an enum in the symbol table.
func (b *APIBuilder) AddEnum(e *Enum) *APIBuilder {
	b.a.addEnum(e)
	return b
}

// Enum looks up an enum in the symbol table.
func (b *APIBuilder) Enum(id string) *Enum {
	return b.a.Enum(id)
}

// AddResource registers a resource in the symbol table.
func (b *APIBuilder) AddResource(r *Resource) *APIBuilder {
	b.a.addResource(r)
	return b
}

// AllResources returns an iterator over the resources in the symbol table.
func (b *APIBuilder) AllResources() iter.Seq[*Resource] {
	return b.a.AllResources()
}

// AddMap registers a map in the symbol table.
func (b *APIBuilder) AddMap(m *Map) *APIBuilder {
	b.a.addMap(m)
	return b
}

// WithMaps registers multiple maps in the symbol table.
func (b *APIBuilder) WithMaps(maps ...*Map) *APIBuilder {
	for _, m := range maps {
		b.a.addMap(m)
	}
	return b
}

// Map looks up a map in the symbol table.
func (b *APIBuilder) Map(id string) *Map {
	return b.a.Map(id)
}

// AllMaps returns an iterator over the maps in the symbol table.
func (b *APIBuilder) AllMaps() iter.Seq[*Map] {
	return b.a.AllMaps()
}

// UpdatePackageName sets the package name from the first service, message, or
// enum if it is not already set.
func (b *APIBuilder) UpdatePackageName() *APIBuilder {
	if b.a.packageName != "" {
		return b
	}
	if len(b.a.services) > 0 {
		b.a.packageName = b.a.services[0].packagez
	} else if len(b.a.messages) > 0 {
		b.a.packageName = b.a.messages[0].packagez
	} else if len(b.a.enums) > 0 {
		b.a.packageName = b.a.enums[0].packagez
	}
	return b
}

// AddDiscoveryLroAnnotations attaches Discovery LRO metadata to LRO-returning
// methods and injects synthetic `getOperation` polling methods onto services.
func (b *APIBuilder) AddDiscoveryLroAnnotations(discoveryConfig *Discovery) error {
	return AddDiscoveryLroAnnotations(&b.a, discoveryConfig)
}

// WithPaginationOverrides configures pagination overrides for [APIBuilder.Build].
func (b *APIBuilder) WithPaginationOverrides(overrides []PaginationOverride) *APIBuilder {
	b.paginationOverrides = overrides
	return b
}

// WithResourceNameHeuristic configures heuristic target resource identification for [APIBuilder.Build].
func (b *APIBuilder) WithResourceNameHeuristic(enabled bool) *APIBuilder {
	b.resourceNameHeuristic = enabled
	return b
}

// WithOverride configures model overrides for [APIBuilder.Build].
func (b *APIBuilder) WithOverride(override ModelOverride) *APIBuilder {
	b.override = override
	return b
}

// WithCommentOverrides configures documentation overrides for [APIBuilder.Build].
func (b *APIBuilder) WithCommentOverrides(overrides []DocumentationOverride) *APIBuilder {
	b.commentOverrides = overrides
	return b
}

// BuildSyntaxTree returns the constructed [API] without running cross-referencing
// or validation passes.
func (b *APIBuilder) BuildSyntaxTree() *API {
	return &NewAPIBuilderFrom(&b.a).a
}

// Build runs pagination detection, recursive field labeling, cross-referencing,
// target resource identification, element skipping, documentation patching,
// validation, and model overrides, and returns the finalized [API].
func (b *APIBuilder) Build() (*API, error) {
	model := b.BuildSyntaxTree()
	UpdateMethodPagination(b.paginationOverrides, model)
	LabelRecursiveFields(model)
	if err := CrossReference(model); err != nil {
		return nil, err
	}
	if err := IdentifyTargetResources(model, b.resourceNameHeuristic); err != nil {
		return nil, err
	}
	if err := SkipModelElements(model, b.override); err != nil {
		return nil, err
	}
	if err := PatchDocumentation(model, b.commentOverrides); err != nil {
		return nil, err
	}
	if err := Validate(model); err != nil {
		return nil, err
	}
	if b.override.Name != "" {
		model.name = b.override.Name
	}
	if b.override.Title != "" {
		model.title = b.override.Title
	}
	if b.override.Description != "" {
		model.description = b.override.Description
	}
	return model, nil
}

// FilterToTypes constructs an [API] model containing only types reachable
// from rootTypeIDs for prost conversion generation. It also returns a sorted slice
// of all non-WKT unused type IDs to exclude via prost_build extern_path, and a boolean
// indicating whether google.rpc.Status is referenced in the reachability path.
// Errors if Any is encountered in the reachability path unless explicitly allowed via allowGrpcAnyFields.
func FilterToTypes(model *API, rootTypeIDs []string, allowGrpcAnyFields []string) (*API, []string, bool, error) {
	type typeItem struct {
		id   string
		path string
	}

	reachableMsgs := make(map[string]bool)
	reachableEnums := make(map[string]bool)
	var queue []typeItem

	for _, id := range rootTypeIDs {
		if id != "" {
			queue = append(queue, typeItem{
				id:   id,
				path: id,
			})
		}
	}

	visited := make(map[string]bool)
	var enqueueMsg func(m *Message, path string)
	var enqueueEnum func(e *Enum, path string)

	enqueueMsg = func(m *Message, path string) {
		if m == nil {
			return
		}
		if !reachableMsgs[m.id] {
			reachableMsgs[m.id] = true
			queue = append(queue, typeItem{
				id:   m.id,
				path: path,
			})
		}
		if m.parent != nil {
			enqueueMsg(m.parent, path)
		}
	}

	enqueueEnum = func(e *Enum, path string) {
		if e == nil {
			return
		}
		reachableEnums[e.id] = true
		if e.parent != nil {
			enqueueMsg(e.parent, path)
		}
	}

	hasGoogleRpcStatus := false
	var unsupportedAnyFields []string

	var enqueueField func(parentPath string, f *Field)
	enqueueField = func(parentPath string, f *Field) {
		if f == nil {
			return
		}
		fieldPath := parentPath + "." + f.name
		if isAnyType(f.typezID) {
			if MarkSkipProtoConversionIfAllowed(f, fieldPath, allowGrpcAnyFields) {
				return
			}
			unsupportedAnyFields = append(unsupportedAnyFields, fieldPath)
			return
		}
		switch f.typez {
		case TypezMessage:
			if f.typezID != "" {
				queue = append(queue, typeItem{
					id:   f.typezID,
					path: fieldPath,
				})
			}
		case TypezEnum:
			if f.typezID != "" {
				enqueueEnum(model.Enum(f.typezID), fieldPath)
			}
		case TypezMap:
			mapType := f.mapType
			if mapType == nil && f.typezID != "" {
				mapType = model.Map(f.typezID)
			}
			if mapType != nil {
				enqueueField(fieldPath, mapType.key)
				enqueueField(fieldPath, mapType.value)
			}
		}
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.id] {
			continue
		}
		visited[item.id] = true
		if isAnyType(item.id) {
			unsupportedAnyFields = append(unsupportedAnyFields, item.id)
			continue
		}

		msg := model.Message(item.id)
		if msg != nil {
			reachableMsgs[msg.id] = true

			if isGoogleRpcStatus(item.id) {
				hasGoogleRpcStatus = true
				continue
			}
			enqueueMsg(msg, item.path)
			for _, f := range msg.fields {
				enqueueField(item.path, f)
			}
			for _, o := range msg.oneOfs {
				for _, f := range o.fields {
					enqueueField(item.path, f)
				}
			}
		}

		enum := model.Enum(item.id)
		if enum != nil {
			enqueueEnum(enum, item.path)
		}
	}

	if len(unsupportedAnyFields) > 0 {
		slices.Sort(unsupportedAnyFields)
		unsupportedAnyFields = slices.Compact(unsupportedAnyFields)
		var sb strings.Builder
		sb.WriteString("cannot generate prost conversion: type google.protobuf.Any is unsupported:\n")
		for _, f := range unsupportedAnyFields {
			fmt.Fprintf(&sb, "  - %s\n", f)
		}
		sb.WriteString("\nTo resolve this, allow dropping Any fields in prost conversion by adding them to allow_grpc_any_fields in librarian.yaml:\n")
		sb.WriteString("    rust:\n")
		sb.WriteString("      allow_grpc_any_fields:\n")
		for _, f := range unsupportedAnyFields {
			fmt.Fprintf(&sb, "        - %s\n", f)
		}
		return nil, nil, false, errors.New(strings.TrimSuffix(sb.String(), "\n"))
	}

	var externalMessages []*Message
	for m := range model.AllMessages() {
		if m.id != "" && reachableMsgs[m.id] && !isWKT(m.id) && m.id != ".google.rpc.Status" && m.parent == nil {
			if m.packagez != model.packageName && m.packagez != ReservedPackageName {
				externalMessages = append(externalMessages, m)
			}
		}
	}
	slices.SortFunc(externalMessages, func(a, b *Message) int { return cmp.Compare(a.id, b.id) })

	var externalEnums []*Enum
	for e := range model.AllEnums() {
		if e.id != "" && reachableEnums[e.id] && !isWKT(e.id) && e.parent == nil {
			if e.packagez != model.packageName && e.packagez != ReservedPackageName {
				externalEnums = append(externalEnums, e)
			}
		}
	}
	slices.SortFunc(externalEnums, func(a, b *Enum) int { return cmp.Compare(a.id, b.id) })

	var filteredMsgs []*Message
	for _, m := range model.messages {
		if reachableMsgs[m.id] {
			filteredMsgs = append(filteredMsgs, m)
		}
	}
	var filteredEnums []*Enum
	for _, e := range model.enums {
		if reachableEnums[e.id] {
			filteredEnums = append(filteredEnums, e)
		}
	}

	hybridModel := API{
		name:                model.name,
		packageName:         model.packageName,
		title:               model.title,
		description:         model.description,
		revision:            model.revision,
		services:            model.services,
		messages:            filteredMsgs,
		externalMessages:    externalMessages,
		enums:               filteredEnums,
		externalEnums:       externalEnums,
		resourceDefinitions: model.resourceDefinitions,
		quickstartService:   model.quickstartService,
		codec:               model.codec,
	}
	for _, s := range hybridModel.services {
		hybridModel.addService(s)
		for _, m := range s.methods {
			hybridModel.addMethod(m)
		}
	}
	for m := range model.AllMessages() {
		hybridModel.addMessage(m)
	}
	for e := range model.AllEnums() {
		hybridModel.addEnum(e)
	}
	for _, r := range hybridModel.resourceDefinitions {
		hybridModel.addResource(r)
	}
	for m := range model.AllMaps() {
		hybridModel.addMap(m)
	}

	var unusedTypes []string
	for m := range model.AllMessages() {
		if m.id != "" && !isWKT(m.id) && !reachableMsgs[m.id] {
			unusedTypes = append(unusedTypes, m.id)
		}
	}
	for e := range model.AllEnums() {
		if e.id != "" && !isWKT(e.id) && !reachableEnums[e.id] {
			unusedTypes = append(unusedTypes, e.id)
		}
	}
	slices.Sort(unusedTypes)

	return &hybridModel, slices.Compact(unusedTypes), hasGoogleRpcStatus, nil
}

func isAnyType(id string) bool {
	return strings.TrimPrefix(id, ".") == "google.protobuf.Any"
}

func isGoogleRpcStatus(id string) bool {
	return strings.TrimPrefix(id, ".") == "google.rpc.Status"
}

func isWKT(id string) bool {
	return strings.HasPrefix(strings.TrimPrefix(id, "."), "google.protobuf.")
}
