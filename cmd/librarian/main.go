// Copyright 2024 Google LLC
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

// Package main is the CLI for the Librarian project, which provides
// automation for common operations required by the Google Cloud SDK,
// including onboarding libraries for APIs, regenerating those libraries,
// and releasing them to package managers.
//
// This command includes common language-agnostic logic and delegates
// to Docker images for language-specific operations such as the actual
// library generation, building and testing.
package main

import (
	"context"
	"log"
	"os"

	"github.com/googleapis/librarian/internal/librarian"
)

func main() {
	ctx := context.Background()
	if err := librarian.Run(ctx, os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}
