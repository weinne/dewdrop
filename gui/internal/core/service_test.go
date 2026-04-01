package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeRemoteLister struct {
	remotes []string
	err     error
}

func (f fakeRemoteLister) ListRemotes(context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.remotes, nil
}

type fakePathReader struct {
	paths map[string]string
}

func (f fakePathReader) Get(remoteName string) (string, error) {
	if p, ok := f.paths[remoteName]; ok {
		return p, nil
	}
	return "", errors.New("missing")
}

func (f fakePathReader) Set(remoteName string, localPath string) error {
	f.paths[remoteName] = localPath
	return nil
}

type fakeSystemd struct {
	active  map[string]bool
	enabled map[string]bool
	started []string
	stopped []string
	enabledOps []string
	disabledOps []string
	reloaded bool
}

func (f *fakeSystemd) IsActive(_ context.Context, unit string) (bool, error) {
	return f.active[unit], nil
}

func (f *fakeSystemd) IsEnabled(_ context.Context, unit string) (bool, error) {
	return f.enabled[unit], nil
}

func (f *fakeSystemd) Start(_ context.Context, unit string) error {
	f.started = append(f.started, unit)
	return nil
}

func (f *fakeSystemd) Stop(_ context.Context, unit string) error {
	f.stopped = append(f.stopped, unit)
	return nil
}

func (f *fakeSystemd) Enable(_ context.Context, unit string) error {
	f.enabledOps = append(f.enabledOps, unit)
	return nil
}

func (f *fakeSystemd) Disable(_ context.Context, unit string) error {
	f.disabledOps = append(f.disabledOps, unit)
	return nil
}

func (f *fakeSystemd) DaemonReload(context.Context) error {
	f.reloaded = true
	return nil
}

func TestBuildSnapshotSyncingWins(t *testing.T) {
	ctrl := &fakeSystemd{
		active: map[string]bool{
			"rclone-sync-drive.service": true,
		},
		enabled: map[string]bool{},
	}

	svc := NewService(
		fakeRemoteLister{remotes: []string{"drive"}},
		fakePathReader{paths: map[string]string{"drive": "/home/test/Nuvem/drive"}},
		ctrl,
		"/home/test/Nuvem",
	)

	snapshot, err := svc.BuildSnapshot(context.Background())
	if err != nil {
		t.Fatalf("BuildSnapshot returned error: %v", err)
	}

	if snapshot.TrayState != TrayStateSyncing {
		t.Fatalf("TrayState mismatch: got %q", snapshot.TrayState)
	}
}

func TestBuildSnapshotErrorWhenPathFails(t *testing.T) {
	ctrl := &fakeSystemd{active: map[string]bool{}, enabled: map[string]bool{}}

	svc := NewService(
		fakeRemoteLister{remotes: []string{"drive"}},
		fakePathReader{paths: map[string]string{}},
		ctrl,
		"/home/test/Nuvem",
	)

	snapshot, err := svc.BuildSnapshot(context.Background())
	if err != nil {
		t.Fatalf("BuildSnapshot returned error: %v", err)
	}

	if len(snapshot.Remotes) != 1 {
		t.Fatalf("Remotes length mismatch: got %d", len(snapshot.Remotes))
	}

	if snapshot.Remotes[0].LastError == "" {
		t.Fatalf("expected LastError to be populated")
	}

	if snapshot.TrayState != TrayStateError {
		t.Fatalf("TrayState mismatch: got %q", snapshot.TrayState)
	}
}

func TestExecuteActionStartMount(t *testing.T) {
	ctrl := &fakeSystemd{active: map[string]bool{}, enabled: map[string]bool{}}
	svc := NewService(
		fakeRemoteLister{},
		fakePathReader{paths: map[string]string{}},
		ctrl,
		"/home/test/Nuvem",
	)

	result, err := svc.ExecuteAction(context.Background(), ActionRequest{Type: ActionStartMount, Remote: "drive"})
	if err != nil {
		t.Fatalf("ExecuteAction returned error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected result.OK to be true")
	}
	if len(ctrl.started) != 1 || ctrl.started[0] != "rclone-mount-drive.service" {
		t.Fatalf("unexpected started units: %#v", ctrl.started)
	}
}

func TestExecuteActionReloadUnits(t *testing.T) {
	ctrl := &fakeSystemd{active: map[string]bool{}, enabled: map[string]bool{}}
	svc := NewService(
		fakeRemoteLister{},
		fakePathReader{paths: map[string]string{}},
		ctrl,
		"/home/test/Nuvem",
	)

	result, err := svc.ExecuteAction(context.Background(), ActionRequest{Type: ActionReloadUnits})
	if err != nil {
		t.Fatalf("ExecuteAction returned error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected result.OK to be true")
	}
	if !ctrl.reloaded {
		t.Fatalf("expected daemon reload to be called")
	}
}

func TestExecuteActionRequiresRemote(t *testing.T) {
	ctrl := &fakeSystemd{active: map[string]bool{}, enabled: map[string]bool{}}
	svc := NewService(
		fakeRemoteLister{},
		fakePathReader{paths: map[string]string{}},
		ctrl,
		"/home/test/Nuvem",
	)

	_, err := svc.ExecuteAction(context.Background(), ActionRequest{Type: ActionStopMount})
	if err == nil {
		t.Fatalf("expected error when remote is missing")
	}
}

func TestExecuteActionSetupMountCreatesUnit(t *testing.T) {
	ctrl := &fakeSystemd{active: map[string]bool{}, enabled: map[string]bool{}}
	paths := fakePathReader{paths: map[string]string{}}

	tmp := t.TempDir()
	svc := NewService(
		fakeRemoteLister{},
		paths,
		ctrl,
		filepath.Join(tmp, "Nuvem"),
	)
	svc.systemdDir = filepath.Join(tmp, "systemd")

	_, err := svc.ExecuteAction(context.Background(), ActionRequest{
		Type:      ActionSetupMount,
		Remote:    "drive",
		AutoStart: true,
	})
	if err != nil {
		t.Fatalf("setup mount returned error: %v", err)
	}

	unitPath := filepath.Join(svc.systemdDir, "rclone-mount-drive.service")
	if _, err := os.Stat(unitPath); err != nil {
		t.Fatalf("expected mount unit to exist: %v", err)
	}
	if len(ctrl.started) == 0 {
		t.Fatalf("expected unit start to be triggered")
	}
}
