// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"context"
	"log"
	"testing"
)

// TODO(https://github.com/googleapis/librarian/issues/202): add better tests
// for librarian.Run.
func TestRun(t *testing.T) {
	if err := Run(context.Background(), []string{"version"}...); err != nil {
		log.Fatal(err)
	}
}
