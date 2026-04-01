package rclone

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/weinne/rclone-auto/gui/internal/runner"
)

type Provider struct {
	runner runner.CommandRunner
	binary string
}

func NewProvider(commandRunner runner.CommandRunner, binary string) *Provider {
	if binary == "" {
		binary = discoverRcloneBinary()
	}

	return &Provider{runner: commandRunner, binary: binary}
}

func discoverRcloneBinary() string {
	if bin, err := exec.LookPath("rclone"); err == nil {
		return bin
	}

	home, err := os.UserHomeDir()
	if err == nil {
		candidate := filepath.Join(home, ".local", "bin", "rclone")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
	}

	return "rclone"
}

func (p *Provider) ListRemotes(ctx context.Context) ([]string, error) {
	out, err := p.runner.Run(ctx, p.binary, "listremotes")
	if err != nil {
		return nil, fmt.Errorf("rclone listremotes: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return []string{}, nil
	}

	lines := strings.Split(out, "\n")
	remotes := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))

	for _, line := range lines {
		name := strings.TrimSpace(strings.TrimSuffix(line, ":"))
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		remotes = append(remotes, name)
	}

	sort.Strings(remotes)
	return remotes, nil
}
