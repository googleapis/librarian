// Copyright 2025 Google LLC
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
	"fmt"
	"slices"
	"strings"
)

// Discovery defines the configuration for discovery docs.
//
// It is too complex to just use key/value pairs in the `Config.Source` field.
type Discovery struct {
	// The ID of the LRO operation type.
	//
	// For example: ".google.cloud.compute.v1.Operation".
	OperationID string

	// Possible prefixes to match the LRO polling RPCs.
	//
	// In discovery-based services there may be multiple resources and RPCs that
	// service as LRO pollers. The order is important, sidekick picks the first
	// match, so the configuration should list preferred matches first.
	Pollers []*Poller
}

// Poller defines how to find a suitable poller RPC.
//
// For operations that may be LROs sidekick will match the URL path of the
// RPC against the prefixes.
type Poller struct {
	// An acceptable prefix for the URL path, for example:
	//     `compute/v1/projects/{project}/zones/{zone}`
	Prefix string

	// The corresponding method ID.
	MethodID string
}

// DiscoveryLro contains old-style long-running operation descriptors.
//
// This is one of the `Method` attributes.
type DiscoveryLro struct {
	// The path parameters required by the polling operation.
	pollingPathParameters []string
	// Language specific annotations.
	codec any
}

// PollingPathParameters returns the path parameters required by the polling operation.
func (d *DiscoveryLro) PollingPathParameters() []string {
	return slices.Clone(d.pollingPathParameters)
}

// Codec returns the language-specific annotations.
func (d *DiscoveryLro) Codec() any {
	return d.codec
}

// SetAnnotations sets the language-specific annotations.
func (d *DiscoveryLro) SetAnnotations(codec any) {
	d.codec = codec
}

// LroServices returns the set of Discovery LRO services.
//
// The discovery doc parser avoids generating LRO annotations for methods in
// this set. These functions return the LRO operation, but are inserted to that
// list, poll, wait for, and cancel LROs. They do not need the annotations and
// generated helpers.
func (d *Discovery) LroServices() map[string]bool {
	found := map[string]bool{}
	for _, poller := range d.Pollers {
		found[poller.serviceID()] = true
	}
	return found
}

// PathParameters returns the list of path parameters associated with a LRO
// poller.
//
// In discovery-based APIs different LRO functions use different polling
// methods. Each one of those methods uses a *subset* of the LRO functions to
// poll the operation. This method returns that subset.
func (p *Poller) PathParameters() []string {
	var parameters []string
	for segment := range strings.SplitSeq(p.Prefix, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			parameters = append(parameters, segment[1:len(segment)-1])
		}
	}
	return parameters
}

func (p *Poller) serviceID() string {
	idx := strings.LastIndex(p.MethodID, ".")
	if idx == -1 {
		return p.MethodID
	}
	return p.MethodID[:idx]
}

// AddDiscoveryLroAnnotations attaches Discovery LRO metadata to LRO-returning
// methods and injects synthetic `getOperation` polling methods onto services.
func AddDiscoveryLroAnnotations(model *API, discoveryConfig *Discovery) error {
	if discoveryConfig == nil {
		return nil
	}
	lroServices := discoveryConfig.LroServices()
	for _, svc := range model.services {
		if _, ok := lroServices[svc.id]; ok {
			continue
		}
		var svcMixin *Method
		for _, method := range svc.methods {
			if method.outputTypeID != discoveryConfig.OperationID {
				continue
			}
			mixin, pathParams := findDiscoveryLroPoller(method, model, discoveryConfig)
			if mixin == nil {
				continue
			}
			method.discoveryLro = NewDiscoveryLroBuilder().
				WithPollingPathParameters(pathParams...).
				Build()
			if svcMixin != nil && mixin != svcMixin {
				return fmt.Errorf("mismatched LRO mixin, want=%v, got=%v", svcMixin, mixin)
			}
			svcMixin = mixin
		}
		if svcMixin == nil {
			continue
		}
		sourceID := ""
		if l := strings.LastIndex(svcMixin.id, "."); l != -1 {
			sourceID = svcMixin.id[0:l]
		}
		method := &Method{
			name:            "getOperation",
			id:              fmt.Sprintf("%s.getOperation", svc.id),
			documentation:   svcMixin.documentation,
			inputTypeID:     svcMixin.inputTypeID,
			outputTypeID:    svcMixin.outputTypeID,
			returnsEmpty:    svcMixin.returnsEmpty,
			pathInfo:        svcMixin.pathInfo,
			pagination:      svcMixin.pagination,
			routing:         svcMixin.routing,
			autoPopulated:   svcMixin.autoPopulated,
			service:         svc,
			sourceServiceID: sourceID,
			isLroPoller:     true,
		}
		svc.methods = append(svc.methods, method)
		model.addMethod(method)
	}
	return nil
}

func findDiscoveryLroPoller(method *Method, model *API, discoveryConfig *Discovery) (*Method, []string) {
	if method.pathInfo == nil {
		return nil, nil
	}
	var flatPath []string
	for _, binding := range method.pathInfo.Bindings() {
		if binding.PathTemplate() == nil {
			continue
		}
		flatPath = append(flatPath, binding.PathTemplate().FlatPath())
	}
	for _, candidate := range discoveryConfig.Pollers {
		for _, path := range flatPath {
			if !strings.HasPrefix(path, candidate.Prefix) {
				continue
			}
			if method := model.Method(candidate.MethodID); method != nil {
				return method, candidate.PathParameters()
			}
		}
	}
	return nil, nil
}
