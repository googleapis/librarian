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

package importconfig

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	// Just test that it can be initialized without error.
	// We don't want to run the actual commands as they require complex setup.
	ctx := context.Background()
	err := Run(ctx, "import-configs", "--help")
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}
