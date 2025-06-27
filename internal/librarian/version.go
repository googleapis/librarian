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
	"github.com/googleapis/librarian/internal/config"
)

var cmdVersion = &cli.Command{
	Short:     "version prints the version information",
	UsageLine: "librarian version",
	Long:      "Version prints version information for the librarian binary.",
	Run: func(ctx context.Context, cfg *config.Config) error {
		fmt.Println(cli.Version())
		return nil
	},
}

func init() {
	cmdVersion.Init()
}
