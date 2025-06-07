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
	test1 := "foo"
	commands := []*Command{
		{Name: test1},
	}

	cmd, err := Lookup(test1, commands)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if cmd.Name != test1 {
		t.Errorf("expected command 'second', got %q", cmd.Name)
	}
}

func TestLookup_Error(t *testing.T) {
	test1 := "foo"
	commands := []*Command{
		{Name: test1},
	}
	test2 := "bar"
	if _, err := Lookup(test2, commands); err == nil {
		t.Fatalf("expected error for command %q", test2)
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
