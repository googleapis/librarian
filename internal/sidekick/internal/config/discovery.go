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

package config

import (
	"strings"
)

// Discovery defines the configuration for discovery docs.
//
// It is too complex to just use key/value pairs in the `Config.Source` field.
type Discovery struct {
	// The ID of the LRO operation type.
	//
	// For example: ".google.cloud.compute.v1.Operation".
	OperationID string `toml:"operation-id"`

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
	MethodID string `toml:"method-id"`
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
	for _, segment := range strings.Split(p.Prefix, "/") {
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
