package app

import (
	"context"
	"time"

	"github.com/weinne/rclone-auto/gui/internal/core"
)

type Poller struct {
	api      *API
	interval time.Duration
}

func NewPoller(api *API, interval time.Duration) *Poller {
	if interval <= 0 {
		interval = 3 * time.Second
	}

	return &Poller{api: api, interval: interval}
}

func (p *Poller) Run(ctx context.Context, out chan<- core.AppSnapshot) error {
	return runPollLoop(ctx, p.interval, out, p.api.Snapshot)
}

func runPollLoop(ctx context.Context, interval time.Duration, out chan<- core.AppSnapshot, next func(context.Context) (core.AppSnapshot, error)) error {
	defer close(out)

	for {
		snapshot, err := next(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- snapshot:
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
