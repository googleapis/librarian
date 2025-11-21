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

package legacycli

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseAndSetFlags(t *testing.T) {
	var (
		strFlag string
		intFlag int
	)

	cmd := &Command{
		Short:     "test is used for testing",
		Long:      "This is the long documentation for command test.",
		UsageLine: "foobar test [arguments]",
	}
	cmd.Init()
	cmd.Flags.StringVar(&strFlag, "name", "default", "name flag")
	cmd.Flags.IntVar(&intFlag, "count", 0, "count flag")

	args := []string{"-name=foo", "-count=5"}
	if err := cmd.Flags.Parse(args); err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if strFlag != "foo" {
		t.Errorf("expected name=foo, got %q", strFlag)
	}
	if intFlag != 5 {
		t.Errorf("expected count=5, got %d", intFlag)
	}
}

func TestAction(t *testing.T) {
	executed := false
	cmd := &Command{
		Short: "run runs the command",
		Action: func(ctx context.Context, cmd *Command) error {
			executed = true
			return nil
		},
	}

	if err := cmd.Action(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	if !executed {
		t.Errorf("cmd.Action was not executed")
	}
}

func TestNamePanicsOnEmptyShort(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("The code did not panic")
		}
	}()
	c := &Command{Short: ""}
	c.Name()
}

func TestUsagePanicsOnMissingDoc(t *testing.T) {
	for _, test := range []struct {
		name string
		cmd  *Command
	}{
		{"missing short", &Command{UsageLine: "l", Long: "l"}},
		{"missing usage", &Command{Short: "s", Long: "l"}},
		{"missing long", &Command{Short: "s", UsageLine: "l"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("The code did not panic")
				}
			}()
			test.cmd.Init()
			var buf bytes.Buffer
			test.cmd.usage(&buf)
		})
	}
}

func TestUsage(t *testing.T) {
	preamble := `Test prints test information.

Usage:

  test [flags]

`

	for _, test := range []struct {
		name        string
		flags       []func(fs *flag.FlagSet)
		subcommands []*Command
		want        string
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
		{
			name: "with subcommand",
			subcommands: []*Command{
				{Short: "sub runs a subcommand"},
			},
			want: fmt.Sprintf(`%sCommands:

  sub                        runs a subcommand

`, preamble),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &Command{
				Short:     "test prints test information",
				UsageLine: "test [flags]",
				Long:      "Test prints test information.",
				Commands:  test.subcommands,
			}
			c.Init()
			for _, fn := range test.flags {
				fn(c.Flags)
			}

			var buf bytes.Buffer
			c.usage(&buf)
			got := buf.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch(-want + got):\n%s", diff)
			}
		})
	}
}

func TestLookupCommand(t *testing.T) {
	sub1sub1 := &Command{
		Short:     "sub1sub1 does something",
		UsageLine: "sub1sub1",
		Long:      "sub1sub1 does something",
	}
	sub1 := &Command{
		Short:     "sub1 does something",
		UsageLine: "sub1",
		Long:      "sub1 does something",
		Commands:  []*Command{sub1sub1},
	}
	sub2 := &Command{
		Short:     "sub2 does something",
		UsageLine: "sub2",
		Long:      "sub2 does something",
	}
	root := &Command{
		Short:     "root does something",
		UsageLine: "root",
		Long:      "root does something",
		Commands: []*Command{
			sub1,
			sub2,
		},
	}
	root.Init()
	sub1.Init()
	sub2.Init()
	sub1sub1.Init()

	for _, test := range []struct {
		name     string
		cmd      *Command
		args     []string
		wantCmd  *Command
		wantArgs []string
		wantErr  bool
	}{
		{
			name:    "no args",
			cmd:     root,
			wantCmd: root,
		},
		{
			name:    "find sub1",
			cmd:     root,
			args:    []string{"sub1"},
			wantCmd: sub1,
		},
		{
			name:    "find sub2",
			cmd:     root,
			args:    []string{"sub2"},
			wantCmd: sub2,
		},
		{
			name:    "find sub1sub1",
			cmd:     root,
			args:    []string{"sub1", "sub1sub1"},
			wantCmd: sub1sub1,
		},
		{
			name:     "find sub1sub1 with args",
			cmd:      root,
			args:     []string{"sub1", "sub1sub1", "arg1"},
			wantCmd:  sub1sub1,
			wantArgs: []string{"arg1"},
		},
		{
			name:    "unknown command",
			cmd:     root,
			args:    []string{"unknown"},
			wantErr: true,
		},
		{
			name:    "unknown subcommand",
			cmd:     root,
			args:    []string{"sub1", "unknown"},
			wantErr: true,
		},
		{
			name:     "find sub1 with flag arguments",
			cmd:      root,
			args:     []string{"sub1", "-h"},
			wantCmd:  sub1,
			wantArgs: []string{"-h"},
		},
		{
			name:     "find sub1sub1 with flag arguments",
			cmd:      root,
			args:     []string{"sub1", "sub1sub1", "-h"},
			wantCmd:  sub1sub1,
			wantArgs: []string{"-h"},
		},
		{
			name:     "find sub1 with a flag argument in between subcommands",
			cmd:      root,
			args:     []string{"sub1", "-h", "sub1sub1"},
			wantCmd:  sub1,
			wantArgs: []string{"-h", "sub1sub1"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotCmd, gotArgs, err := lookupCommand(test.cmd, test.args)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if gotCmd != test.wantCmd {
				var gotName, wantName string
				if gotCmd != nil {
					gotName = gotCmd.Name()
				}
				if test.wantCmd != nil {
					wantName = test.wantCmd.Name()
				}
				t.Errorf("gotCmd.Name() = %q, want %q", gotName, wantName)
			}
			if diff := cmp.Diff(test.wantArgs, gotArgs, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRun(t *testing.T) {
	actionExecuted := false
	subcmd := &Command{
		Short:     "bar is a subcommand",
		Long:      "bar is a subcommand.",
		UsageLine: "bar",
		Action: func(ctx context.Context, cmd *Command) error {
			actionExecuted = true
			return nil
		},
	}
	subcmd.Init()

	root := &Command{
		Short:     "foo is the root command",
		Long:      "foo is the root command.",
		UsageLine: "foo",
		Commands:  []*Command{subcmd},
	}
	root.Init()

	noaction := &Command{
		Short:     "noaction has no action",
		Long:      "noaction has no action.",
		UsageLine: "noaction",
	}
	noaction.Init()

	for _, test := range []struct {
		name           string
		cmd            *Command
		args           []string
		wantErr        bool
		actionExecuted bool
	}{
		{
			name:           "execute foo with subcommand bar",
			cmd:            root,
			args:           []string{"bar"},
			actionExecuted: true,
		},
		{
			name:    "unknown subcommand",
			cmd:     root,
			args:    []string{"unknown"},
			wantErr: true,
		},
		{
			name:    "flag parse error",
			cmd:     subcmd,
			args:    []string{"-unknown"},
			wantErr: true,
		},
		{
			name:    "no action defined on command with no subcommands",
			cmd:     noaction,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actionExecuted = false
			err := test.cmd.Run(t.Context(), test.args)
			if err != nil {
				if !test.wantErr {
					t.Errorf("error = %v, wantErr %v", err, test.wantErr)
				}
			}
			if actionExecuted != test.actionExecuted {
				t.Errorf("actionExecuted = %v, want %v", actionExecuted, test.actionExecuted)
			}
		})
	}
}

func TestNewCommandSet(t *testing.T) {
	cmd := NewCommandSet(nil, "short", "usage", "long usageLine")
	if len(cmd.Commands) != 1 || cmd.Commands[0].Name() != "version" {
		t.Errorf("NewCommandSet(nil) did not produce a single 'version' command")
	}
}
