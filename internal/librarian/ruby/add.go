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

package ruby

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

const defaultVersion = "0.0.1"

var (
	errRequiresOneAPI = errors.New("ruby libraries must have a single API")
	errNoVersionedAPI = errors.New("no versioned API found")
)

// Add initializes a new Ruby library configuration.
func Add(cfg *config.Config, lib *config.Library) (*config.Library, error) {
	lib.Version = defaultVersion
	// Ruby generation does not require copyright_year in librarian.yaml,
	// so we reset it here to avoid redundancy in librarian.yaml.
	lib.CopyrightYear = ""
	newLib, err := addWrapper(cfg, lib)
	if err != nil {
		return nil, err
	}
	copyFromSiblingAPI(cfg, newLib)
	return newLib, nil
}

// addWrapper initializes a new Ruby main client library configuration.
func addWrapper(cfg *config.Config, lib *config.Library) (*config.Library, error) {
	if len(lib.APIs) != 1 {
		return nil, fmt.Errorf("%w: got %d", errRequiresOneAPI, len(lib.APIs))
	}
	apiPath := lib.APIs[0].Path
	// No need to add wrapperOf for a versioned client.
	if serviceconfig.ExtractVersion(apiPath) != "" {
		return lib, nil
	}
	versionedAPI, err := searchVersionedAPI(cfg, apiPath)
	if err != nil {
		return nil, err
	}
	lib = configureWrapper(lib, versionedAPI)
	return lib, nil
}

// searchVersionedAPI finds a versioned API in the config that is a prefix of the given API path,
// otherwise it returns an error.
func searchVersionedAPI(cfg *config.Config, apiPath string) (string, error) {
	for _, lib := range cfg.Libraries {
		for _, api := range lib.APIs {
			if strings.HasPrefix(api.Path, apiPath+"/v") {
				return api.Path, nil
			}
		}
	}
	return "", fmt.Errorf("%w: %q", errNoVersionedAPI, apiPath)
}

// configureWrapper configures the library to be a main client.
func configureWrapper(lib *config.Library, versionedAPI string) *config.Library {
	lib.APIs = []*config.API{{Path: versionedAPI}}
	if lib.Ruby == nil {
		lib.Ruby = &config.RubyPackage{}
	}
	lib.Ruby.WrapperOf = []string{fmt.Sprintf("%s:0.0", filepath.Base(versionedAPI))}
	return lib
}

func copyFromSiblingAPI(cfg *config.Config, lib *config.Library) {
	if lib.Ruby != nil && len(lib.Ruby.WrapperOf) > 0 {
		return // wrappers don't inherit ruby_cloud_opts
	}
	for _, existingLib := range cfg.Libraries {
		existingAPI := existingLib.APIs[0]
		if !hasSamePathPrefix(existingAPI.Path, lib.APIs[0].Path) {
			continue
		}
		copyCloudOpts(existingAPI, lib.APIs[0])
	}
}

// hasSamePathPrefix reports whether the two API paths have the same prefix when the
// version suffix is trimmed. The caller is responsible for ensuring that the two API
// paths are not the same.
func hasSamePathPrefix(oneAPI, otherAPI string) bool {
	return trimVersion(oneAPI) == trimVersion(otherAPI)
}

func trimVersion(apiPath string) string {
	version := serviceconfig.ExtractVersion(apiPath)
	if version == "" {
		return apiPath
	}
	return strings.TrimSuffix(apiPath, "/"+version)
}

func copyCloudOpts(existingAPI, newAPI *config.API) {
	if existingAPI.Ruby == nil || existingAPI.Ruby.RubyCloudOpts == nil {
		return
	}
	opts := *existingAPI.Ruby.RubyCloudOpts
	if newAPI.Ruby == nil {
		newAPI.Ruby = &config.RubyAPI{}
	}
	newAPI.Ruby.RubyCloudOpts = &opts
	log.Printf(
		"WARNING: customized ruby_cloud_opts is copied from %q to %q, but this may not be correct for %q. "+
			"Please review and edit ruby_cloud_opts in librarian.yaml if needed.",
		existingAPI.Path, newAPI.Path, newAPI.Path,
	)
}
