// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package language

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
)

// Generate generates a single library for the specified language.
func Generate(ctx context.Context, language string, library *config.Library) error {
	var err error
	switch language {
	case "testhelper":
		err = testGenerate(library)
	default:
		err = fmt.Errorf("generate not implemented for %q", language)
	}

	if err != nil {
		fmt.Printf("✗ Error generating %s: %v\n", library.Name, err)
		return err
	}

	fmt.Printf("✓ Successfully generated %s\n", library.Name)
	return nil
}
