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
	"fmt"

	"github.com/googleapis/librarian/internal/cli"
)

var CmdVersion = &cli.Command{
	Name:  "version",
	Short: "Prints version information.",
	Usage: "librarian version",
	Long:  "Prints version information for the librarian binary.",
	Run: func(ctx context.Context) error {
		fmt.Println(cli.Version())
		return nil
	},
}
