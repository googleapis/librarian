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

package php

import (
	"context"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
)

func TestGenerate(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	lib := &config.Library{}
	src := &sources.Sources{}

	if err := Generate(ctx, cfg, lib, src); err != nil {
		t.Errorf("Generate() returned error: %v", err)
	}
}

func TestClean(t *testing.T) {
	lib := &config.Library{}

	if err := Clean(lib); err != nil {
		t.Errorf("Clean() returned error: %v", err)
	}
}

func TestFormat(t *testing.T) {
	ctx := context.Background()
	lib := &config.Library{}

	if err := Format(ctx, lib); err != nil {
		t.Errorf("Format() returned error: %v", err)
	}
}
