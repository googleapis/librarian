package automation

import (
	"context"
	"log/slog"

	"github.com/googleapis/librarian/internal/config"
)

// runCommandFn is a function type that matches RunCommand, for mocking in tests.
var runCommandFn = RunCommand

type automationRunner struct {
	build    bool
	command  string
	forceRun bool
	project  string
	push     bool
}

func newAutomationRunner(cfg *config.Config) *automationRunner {
	return &automationRunner{
		build:    cfg.Build,
		command:  cfg.CommandName,
		forceRun: cfg.ForceRun,
		project:  cfg.Project,
		push:     cfg.Push,
	}
}

func (r *automationRunner) run(ctx context.Context) error {
	err := runCommandFn(ctx, r.command, r.project, r.push, r.build, r.forceRun)
	if err != nil {
		slog.Error("error running command", slog.Any("err", err))
		return err
	}
	return nil
}
