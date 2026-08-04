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
	newLib, err := addWrapper(cfg, lib)
	if err != nil {
		return nil, err
	}
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
	configureWrapper(lib, versionedAPI)
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
func configureWrapper(lib *config.Library, versionedAPI string) {
	lib.APIs = []*config.API{{Path: versionedAPI}}
	lib.Ruby = &config.RubyPackage{
		WrapperOf: []string{fmt.Sprintf("%s:0.0", filepath.Base(versionedAPI))},
	}
}
