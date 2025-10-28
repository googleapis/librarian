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

package gcloud

import (
	"fmt"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/config/gcloudyaml"
	"gopkg.in/yaml.v3"
)

// Generate generates gcloud commands from the model.
func Generate(model *api.API, outdir string, cfg *config.Config) error {
	for _, service := range model.Services {
		for _, method := range service.Methods {
			cmd := newCommand(method, cfg)

			b, err := yaml.Marshal(cmd)
			if err != nil {
				return err
			}

			fmt.Println(string(b))
		}
	}
	return nil
}

func newCommand(method *api.Method, cfg *config.Config) *Command {
	rule := findHelpTextRule(method, cfg)
	cmd := &Command{}
	if rule != nil {
		cmd.HelpText = &CommandHelpText{
			Brief:       rule.HelpText.Brief,
			Description: rule.HelpText.Description,
		}
	}
	return cmd
}

func findHelpTextRule(method *api.Method, cfg *config.Config) *gcloudyaml.HelpTextRule {
	if cfg.Gcloud == nil || cfg.Gcloud.APIs == nil {
		return nil
	}
	for _, api := range cfg.Gcloud.APIs {
		if api.HelpText == nil {
			continue
		}
		for _, rule := range api.HelpText.MethodRules {
			fmt.Printf("DEBUG: Checking selector: '%s' == '%s'\n", rule.Selector, method.MethodFullName())
			if rule.Selector == method.MethodFullName() {
				fmt.Println("DEBUG: Found matching help text rule.")
				return rule
			}
		}
	}
	return nil
}
