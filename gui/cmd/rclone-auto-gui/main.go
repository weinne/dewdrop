package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/weinne/rclone-auto/gui/internal/app"
	"github.com/weinne/rclone-auto/gui/internal/adapters/config"
	"github.com/weinne/rclone-auto/gui/internal/adapters/rclone"
	"github.com/weinne/rclone-auto/gui/internal/adapters/systemd"
	"github.com/weinne/rclone-auto/gui/internal/core"
	"github.com/weinne/rclone-auto/gui/internal/bootstrap"
	"github.com/weinne/rclone-auto/gui/internal/runner"
)

func main() {
	if err := bootstrap.CheckRuntimeDependencies(); err != nil {
		fail(err)
	}

	watch := flag.Bool("watch", false, "Continuously print snapshot updates")
	interval := flag.Duration("interval", 4*time.Second, "Polling interval for --watch")
	action := flag.String("action", "", "Execute a backend action (start-mount, stop-mount, start-sync, stop-sync, enable-mount-autostart, disable-mount-autostart, enable-sync-autostart, disable-sync-autostart, reload-units, diagnose-remote, repair-remote, remove-service)")
	remote := flag.String("remote", "", "Remote name for action that targets a specific remote")
	flag.Parse()

	home, err := os.UserHomeDir()
	if err != nil {
		fail(err)
	}

	cloudDir := filepath.Join(home, "Nuvem")
	remotePathsFile := filepath.Join(home, ".config", "rclone-auto", "remote-paths.conf")

	run := runner.ExecRunner{}
	remoteProvider := rclone.NewProvider(run, "rclone")
	pathStore := config.NewRemotePathStore(remotePathsFile, cloudDir)
	systemdController := systemd.NewController(run)
	svc := core.NewService(remoteProvider, pathStore, systemdController, cloudDir)
	api := app.NewAPI(svc)

	ctx := context.Background()
	if *action != "" {
		result, err := api.Action(ctx, core.ActionRequest{
			Type:   core.ActionType(*action),
			Remote: *remote,
		})
		if err != nil {
			fail(err)
		}
		writeJSON(result)
		return
	}

	if *watch {
		watchSnapshots(ctx, api, *interval)
		return
	}

	snapshot, err := api.Snapshot(ctx)
	if err != nil {
		fail(err)
	}

	writeJSON(snapshot)
}

func watchSnapshots(ctx context.Context, api *app.API, interval time.Duration) {
	err := api.Watch(ctx, interval, func(snapshot core.AppSnapshot) error {
		writeJSON(snapshot)
		return nil
	})
	if err != nil {
		fail(err)
	}
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
