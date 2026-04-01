package app

import (
	"context"
	"testing"
	"time"

	"github.com/weinne/rclone-auto/gui/internal/core"
)

type stubSnapshotProvider struct {
	snap core.AppSnapshot
	err  error
}

func TestPollerEmitsSnapshots(t *testing.T) {
	api := &API{service: nil}

	// Build a local wrapper so we can test loop semantics without touching system state.
	apiSnapshot := func(context.Context) (core.AppSnapshot, error) {
		return core.AppSnapshot{TrayState: core.TrayStateIdle}, nil
	}

	poller := NewPoller(api, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan core.AppSnapshot, 2)

	go func() {
		_ = runPollLoop(ctx, poller.interval, out, apiSnapshot)
	}()

	select {
	case snap := <-out:
		if snap.TrayState != core.TrayStateIdle {
			t.Fatalf("unexpected tray state: %s", snap.TrayState)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("timed out waiting for snapshot")
	}

	cancel()
}
