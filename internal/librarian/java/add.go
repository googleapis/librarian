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

package java

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

var (
	// ErrSourceRequired indicates that the googleapis source is required.
	ErrSourceRequired = errors.New("googleapis source is required to derive library name")
	// ErrServiceNameNotFound indicates that the service name was not found in the service config.
	ErrServiceNameNotFound = errors.New("service name not found in service config")
	// ErrAPIValidation indicates that the API validation failed.
	ErrAPIValidation = errors.New("API validation failed")
)

const (
	defaultVersion   = "0.1.0-SNAPSHOT"
	fakeGroupID      = "please-configure-java-group-id"
	googleapisSuffix = ".googleapis.com"
)

// Add initializes a new Java library with default values.
func Add(lib *config.Library) *config.Library {
	lib.Version = defaultVersion
	// Java generation defaults to the system year for license headers,
	// so we reset it here to avoid redundancy in librarian.yaml.
	lib.CopyrightYear = ""

	// We use the first API to infer the GroupID and distribution name override.
	// It is unrealistic for a single library to mix cloud and non-cloud APIs.
	apiPath := lib.APIs[0].Path
	switch {
	case strings.HasPrefix(apiPath, "google/shopping/"):
		return setJavaConfig(lib, "com.google.shopping")
	case strings.HasPrefix(apiPath, "google/maps/"):
		return setJavaConfig(lib, "com.google.maps")
	case strings.HasPrefix(apiPath, "google/ads/"):
		return setJavaConfig(lib, "com.google.api-ads")
	}
	if !strings.HasPrefix(apiPath, "google/cloud/") {
		log.Printf(
			"WARNING: unrecognized non-cloud API path %q. Setting fake GroupID %q. "+
				"Please manually configure java.group_id and java.distribution_name_override in librarian.yaml.",
			apiPath, fakeGroupID,
		)
		setJavaConfig(lib, fakeGroupID)
	}
	return lib
}

func setJavaConfig(lib *config.Library, groupID string) *config.Library {
	if lib.Java == nil {
		lib.Java = &config.JavaModule{}
	}
	lib.Java.GroupID = groupID
	lib.Java.DistributionNameOverride = groupID + ":google-" + lib.Name
	return lib
}

// DefaultLibraryName derives a default library name from an API path by parsing
// name from service configuration YAML and taking its subdomain.
func DefaultLibraryName(srcs *sources.Sources, api string) (string, error) {
	if srcs == nil || srcs.Googleapis == "" {
		return "", ErrSourceRequired
	}
	apiConfig, err := serviceconfig.Find(srcs.Googleapis, api, config.LanguageJava)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAPIValidation, err)
	}
	if apiConfig.ServiceName == "" {
		return "", fmt.Errorf("%w for %s", ErrServiceNameNotFound, api)
	}
	if !strings.HasSuffix(apiConfig.ServiceName, googleapisSuffix) {
		return "", fmt.Errorf("%w: service name %q does not end with %q", ErrAPIValidation, apiConfig.ServiceName, googleapisSuffix)
	}
	subdomain := strings.TrimSuffix(apiConfig.ServiceName, googleapisSuffix)
	return subdomain, nil
}
