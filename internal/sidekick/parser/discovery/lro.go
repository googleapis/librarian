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

package discovery

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

func lroAnnotations(model *api.API, cfg *config.Config) error {
	if cfg == nil || cfg.Discovery == nil {
		return nil
	}
	lroServices := cfg.Discovery.LroServices()
	for _, svc := range model.Services {
		if _, ok := lroServices[svc.ID]; ok {
			continue
		}
		var svcMixin *api.Method
		for _, method := range svc.Methods {
			if method.OutputTypeID != cfg.Discovery.OperationID {
				continue
			}
			mixin, pathParams := lroFindPoller(method, model, cfg.Discovery)
			if mixin == nil {
				continue
			}
			method.DiscoveryLro = &api.DiscoveryLro{
				PollingPathParameters: pathParams,
			}
			if svcMixin != nil && mixin != svcMixin {
				return fmt.Errorf("mismatched LRO mixin, want=%v, got=%v", svcMixin, mixin)
			}
			svcMixin = mixin
		}
		if svcMixin == nil {
			continue
		}
		method := &api.Method{
			Name:            "getOperation",
			ID:              fmt.Sprintf("%s.getOperation", svc.ID),
			Documentation:   svcMixin.Documentation,
			InputTypeID:     svcMixin.InputTypeID,
			OutputTypeID:    svcMixin.OutputTypeID,
			ReturnsEmpty:    svcMixin.ReturnsEmpty,
			PathInfo:        svcMixin.PathInfo,
			Pagination:      svcMixin.Pagination,
			Routing:         svcMixin.Routing,
			AutoPopulated:   svcMixin.AutoPopulated,
			Service:         svc,
			SourceService:   svcMixin.Service,
			SourceServiceID: svcMixin.SourceServiceID,
		}
		svc.Methods = append(svc.Methods, method)
		model.State.MethodByID[method.ID] = method
	}
	return nil
}

func lroFindPoller(method *api.Method, model *api.API, discoveryConfig *config.Discovery) (*api.Method, []string) {
	var flatPath []string
	for _, binding := range method.PathInfo.Bindings {
		flatPath = append(flatPath, binding.PathTemplate.FlatPath())
	}
	for _, candidate := range discoveryConfig.Pollers {
		for _, path := range flatPath {
			if !strings.HasPrefix(path, candidate.Prefix) {
				continue
			}
			if method, ok := model.State.MethodByID[candidate.MethodID]; ok {
				return method, candidate.PathParameters()
			}
		}
	}
	return nil, nil
}
