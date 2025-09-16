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

package rustrelease

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

func publishedCrate(manifest string) ([]string, error) {
	contents, err := os.ReadFile(manifest)
	if err != nil {
		return nil, err
	}
	info := cargo{
		Package: &crateInfo{
			Publish: true,
		},
	}
	if err := toml.Unmarshal(contents, &info); err != nil {
		return nil, err
	}
	if !info.Package.Publish {
		return nil, nil
	}
	return []string{info.Package.Name}, nil
}
