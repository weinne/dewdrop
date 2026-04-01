package app

import (
	"context"

	"github.com/weinne/rclone-auto/gui/internal/core"
)

type Bindings struct {
	api *API
}

func NewBindings(api *API) *Bindings {
	return &Bindings{api: api}
}

func (b *Bindings) GetSnapshot() (core.AppSnapshot, error) {
	return b.api.Snapshot(context.Background())
}

func (b *Bindings) ExecuteAction(actionType string, remote string) (core.ActionResult, error) {
	req := core.ActionRequest{
		Type:   core.ActionType(actionType),
		Remote: remote,
	}

	return b.api.Action(context.Background(), req)
}
