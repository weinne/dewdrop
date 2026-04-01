package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/weinne/rclone-auto/gui/internal/adapters/config"
	"github.com/weinne/rclone-auto/gui/internal/adapters/rclone"
	"github.com/weinne/rclone-auto/gui/internal/adapters/systemd"
	"github.com/weinne/rclone-auto/gui/internal/app"
	"github.com/weinne/rclone-auto/gui/internal/bootstrap"
	"github.com/weinne/rclone-auto/gui/internal/core"
	"github.com/weinne/rclone-auto/gui/internal/desktop"
	"github.com/weinne/rclone-auto/gui/internal/runner"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := bootstrap.CheckRuntimeDependencies(); err != nil {
		fail(fmt.Errorf("nao foi possivel iniciar: %w", err))
	}

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
	bindings := app.NewBindings(api)
	desktopApp := desktop.NewApplication(api, bindings)

	err = wails.Run(&options.App{
		Title:  "Dewdrop",
		Width:  1180,
		Height: 760,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: desktopApp.Startup,
		OnShutdown: func(ctx context.Context) {
			desktopApp.Shutdown()
		},
		Bind: []interface{}{
			desktopApp,
		},
	})
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
