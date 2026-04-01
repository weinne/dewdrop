package app

import (
	"context"
	"time"

	"github.com/weinne/rclone-auto/gui/internal/core"
)

type API struct {
	service *core.Service
}

func NewAPI(service *core.Service) *API {
	return &API{service: service}
}

func (a *API) Snapshot(ctx context.Context) (core.AppSnapshot, error) {
	return a.service.BuildSnapshot(ctx)
}

func (a *API) Action(ctx context.Context, req core.ActionRequest) (core.ActionResult, error) {
	return a.service.ExecuteAction(ctx, req)
}

func (a *API) Watch(ctx context.Context, interval time.Duration, onSnapshot func(core.AppSnapshot) error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		snapshot, err := a.Snapshot(ctx)
		if err != nil {
			return err
		}
		if err := onSnapshot(snapshot); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
