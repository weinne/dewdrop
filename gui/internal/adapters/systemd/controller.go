package systemd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/weinne/rclone-auto/gui/internal/runner"
)

type Controller struct {
	runner runner.CommandRunner
}

func NewController(commandRunner runner.CommandRunner) *Controller {
	return &Controller{runner: commandRunner}
}

func (c *Controller) IsActive(ctx context.Context, unit string) (bool, error) {
	_, err := c.runner.Run(ctx, "systemctl", "--user", "is-active", "--quiet", unit)
	if err == nil {
		return true, nil
	}
	if isExitCode(err, 3) || isExitCode(err, 4) {
		return false, nil
	}
	return false, err
}

func (c *Controller) IsEnabled(ctx context.Context, unit string) (bool, error) {
	_, err := c.runner.Run(ctx, "systemctl", "--user", "is-enabled", "--quiet", unit)
	if err == nil {
		return true, nil
	}
	if isExitCode(err, 1) || isExitCode(err, 3) || isExitCode(err, 4) {
		return false, nil
	}
	return false, err
}

func (c *Controller) Start(ctx context.Context, unit string) error {
	_, err := c.runner.Run(ctx, "systemctl", "--user", "start", unit)
	if err != nil {
		return fmt.Errorf("systemctl start %s: %w", unit, err)
	}
	return nil
}

func (c *Controller) Stop(ctx context.Context, unit string) error {
	_, err := c.runner.Run(ctx, "systemctl", "--user", "stop", unit)
	if err != nil {
		return fmt.Errorf("systemctl stop %s: %w", unit, err)
	}
	return nil
}

func (c *Controller) Enable(ctx context.Context, unit string) error {
	_, err := c.runner.Run(ctx, "systemctl", "--user", "enable", unit)
	if err != nil {
		return fmt.Errorf("systemctl enable %s: %w", unit, err)
	}
	return nil
}

func (c *Controller) Disable(ctx context.Context, unit string) error {
	_, err := c.runner.Run(ctx, "systemctl", "--user", "disable", unit)
	if err != nil {
		return fmt.Errorf("systemctl disable %s: %w", unit, err)
	}
	return nil
}

func (c *Controller) DaemonReload(ctx context.Context) error {
	_, err := c.runner.Run(ctx, "systemctl", "--user", "daemon-reload")
	if err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	return nil
}

func isExitCode(err error, expected int) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}

	return exitErr.ExitCode() == expected
}
