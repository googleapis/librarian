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

// Package svcconfig contains helper functions to parse service config files.
package svcconfig

import (
	"strings"

	"cloud.google.com/go/librarian/internal/serviceconfig"
)

// ServiceNames contains the package and (unqualified) name of a service.
type ServiceNames struct {
	PackageName string
	ServiceName string
}

// ExtractPackageName determines the package name and service implied by a
// service config file.
func ExtractPackageName(serviceConfig *serviceconfig.Service) *ServiceNames {
	if serviceConfig == nil {
		return nil
	}
	for _, api := range serviceConfig.Apis {
		if wellKnownMixin(api.Name) {
			continue
		}
		names := splitQualifiedServiceName(api.Name)
		return &names
	}
	return nil
}

// splitQualifiedServiceName splits a service name into the package name and the
// unqualified service name.
func splitQualifiedServiceName(name string) ServiceNames {
	li := strings.LastIndex(name, ".")
	if li == -1 {
		return ServiceNames{PackageName: "", ServiceName: name}
	}
	return ServiceNames{PackageName: name[:li], ServiceName: name[li+1:]}
}

// wellKnownMixin returns true if the qualified service name is one of the
// well-known mixins.
func wellKnownMixin(qualifiedServiceName string) bool {
	return strings.HasPrefix(qualifiedServiceName, "google.cloud.location.Location") ||
		strings.HasPrefix(qualifiedServiceName, "google.longrunning.Operations") ||
		strings.HasPrefix(qualifiedServiceName, "google.iam.v1.IAMPolicy")
}
