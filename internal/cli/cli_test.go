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
	"bytes"
	"context"
	"flag"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseAndSetFlags(t *testing.T) {
	var (
		strFlag string
		intFlag int
	)

	cmd := &Command{
		Short: "test is used for testing",
		Long:  "This is the long documentation for command test.",
		Usage: "foobar test [arguments]",
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
		{Short: "foo runs the foo command"},
		{Short: "bar runs the bar command"},
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
			if cmd.Name() != test.name {
				t.Errorf("got = %q, want = %q", cmd.Name(), test.name)
			}
		})
	}
}

func TestRun(t *testing.T) {
	executed := false
	cmd := &Command{
		Short: "run runs the command",
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

func TestUsage(t *testing.T) {
	preamble := `Test prints test information.

Usage:
  test [flags]

`

	for _, test := range []struct {
		name  string
		flags []func(fs *flag.FlagSet)
		want  string
	}{
		{
			name:  "no flags",
			flags: nil,
			want:  preamble,
		},
		{
			name: "with string flag",
			flags: []func(fs *flag.FlagSet){
				func(fs *flag.FlagSet) {
					fs.String("name", "default", "name flag")
				},
			},
			want: fmt.Sprintf(`%sFlags:
  -name string
    	name flag (default "default")


`, preamble),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &Command{
				Short: "test prints test information",
				Usage: "test [flags]",
				Long:  "Test prints test information.",
			}
			initFlags(c)
			c.SetFlags(test.flags)

			var buf bytes.Buffer
			c.usage(&buf)
			got := buf.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch(-want + got):\n%s", diff)
			}
		})
	}
}
