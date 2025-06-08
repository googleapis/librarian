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

package cli

import (
	"context"
	"flag"
	"testing"
)

func TestParseAndSetFlags(t *testing.T) {
	var (
		strFlag string
		intFlag int
	)

	cmd := &Command{
		Name:  "test",
		Short: "test command is used for testing",
	}
	cmd.SetFlags([]func(fs *flag.FlagSet){
		func(fs *flag.FlagSet) {
			fs.StringVar(&strFlag, "name", "default", "name flag")
			fs.IntVar(&intFlag, "count", 0, "count flag")
		},
	})

	args := []string{"-name=foo", "-count=5"}
	if err := cmd.Parse(args); err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if strFlag != "foo" {
		t.Errorf("expected name=foo, got %q", strFlag)
	}
	if intFlag != 5 {
		t.Errorf("expected count=5, got %d", intFlag)
	}
}

func TestLookup(t *testing.T) {
	commands := []*Command{
		{Name: "foo"},
		{Name: "bar"},
	}

	for _, test := range []struct {
		name    string
		wantErr bool
	}{
		{"foo", false},
		{"bar", false},
		{"baz", true}, // not found case
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd, err := Lookup(test.name, commands)
			if test.wantErr {
				if err == nil {
					t.Fatal(err)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}
			if cmd.Name != test.name {
				t.Errorf("got = %q, want = %q", cmd.Name, test.name)
			}
		})
	}
}

func TestRun(t *testing.T) {
	executed := false
	cmd := &Command{
		Name: "run",
		Run: func(ctx context.Context) error {
			executed = true
			return nil
		},
	}

	if err := cmd.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !executed {
		t.Errorf("cmd.Run was not executed")
	}
}
