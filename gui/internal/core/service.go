package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RemoteLister interface {
	ListRemotes(ctx context.Context) ([]string, error)
}

type RemotePathReader interface {
	Get(remoteName string) (string, error)
}

type RemotePathWriter interface {
	Set(remoteName string, localPath string) error
}

type RemotePathDeleter interface {
	Delete(remoteName string) error
}

type SystemdController interface {
	IsActive(ctx context.Context, unit string) (bool, error)
	IsEnabled(ctx context.Context, unit string) (bool, error)
	Start(ctx context.Context, unit string) error
	Stop(ctx context.Context, unit string) error
	Enable(ctx context.Context, unit string) error
	Disable(ctx context.Context, unit string) error
	DaemonReload(ctx context.Context) error
}

type Service struct {
	remoteLister RemoteLister
	pathReader   RemotePathReader
	systemd      SystemdController
	cloudDir     string
	systemdDir   string
	rcloneBin    string
}

func NewService(remoteLister RemoteLister, pathReader RemotePathReader, systemd SystemdController, cloudDir string) *Service {
	home, _ := os.UserHomeDir()
	systemdDir := filepath.Join(home, ".config", "systemd", "user")
	rcloneBin := "rclone"
	if bin, err := exec.LookPath("rclone"); err == nil {
		rcloneBin = bin
	} else if home != "" {
		candidate := filepath.Join(home, ".local", "bin", "rclone")
		if _, statErr := os.Stat(candidate); statErr == nil {
			rcloneBin = candidate
		}
	}

	return &Service{
		remoteLister: remoteLister,
		pathReader:   pathReader,
		systemd:      systemd,
		cloudDir:     cloudDir,
		systemdDir:   systemdDir,
		rcloneBin:    rcloneBin,
	}
}

func (s *Service) BuildSnapshot(ctx context.Context) (AppSnapshot, error) {
	remotes, err := s.ListRemoteStates(ctx)
	if err != nil {
		return AppSnapshot{}, err
	}

	return AppSnapshot{
		TrayState: deriveTrayState(remotes),
		Remotes:   remotes,
		CheckedAt: time.Now(),
	}, nil
}

func (s *Service) ListRemoteStates(ctx context.Context) ([]RemoteState, error) {
	names, err := s.remoteLister.ListRemotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list remotes: %w", err)
	}

	states := make([]RemoteState, 0, len(names))
	for _, name := range names {
		state := RemoteState{
			Name:            name,
			MountUnit:       mountUnit(name),
			SyncServiceUnit: syncServiceUnit(name),
			SyncTimerUnit:   syncTimerUnit(name),
		}

		localPath, localErr := s.pathReader.Get(name)
		if localErr != nil {
			state.LastError = localErr.Error()
			localPath = filepath.Join(s.cloudDir, name)
		}
		state.LocalPath = localPath

		mountActive, mountActiveErr := s.systemd.IsActive(ctx, state.MountUnit)
		state.MountActive = mountActive
		if mountActiveErr != nil {
			appendError(&state, fmt.Errorf("%s active: %w", state.MountUnit, mountActiveErr))
		}

		mountEnabled, mountEnabledErr := s.systemd.IsEnabled(ctx, state.MountUnit)
		state.MountEnabled = mountEnabled
		if mountEnabledErr != nil {
			appendError(&state, fmt.Errorf("%s enabled: %w", state.MountUnit, mountEnabledErr))
		}

		syncActive, syncActiveErr := s.systemd.IsActive(ctx, state.SyncServiceUnit)
		state.SyncActive = syncActive
		if syncActiveErr != nil {
			appendError(&state, fmt.Errorf("%s active: %w", state.SyncServiceUnit, syncActiveErr))
		}

		syncEnabled, syncEnabledErr := s.systemd.IsEnabled(ctx, state.SyncTimerUnit)
		state.SyncTimerEnabled = syncEnabled
		if syncEnabledErr != nil {
			appendError(&state, fmt.Errorf("%s enabled: %w", state.SyncTimerUnit, syncEnabledErr))
		}

		states = append(states, state)
	}

	return states, nil
}

func (s *Service) StartMount(ctx context.Context, remoteName string) error {
	if err := s.systemd.Start(ctx, mountUnit(remoteName)); err != nil {
		return fmt.Errorf("start mount: %w", err)
	}
	return nil
}

func (s *Service) StopMount(ctx context.Context, remoteName string) error {
	if err := s.systemd.Stop(ctx, mountUnit(remoteName)); err != nil {
		return fmt.Errorf("stop mount: %w", err)
	}
	return nil
}

func (s *Service) StartSyncTimer(ctx context.Context, remoteName string) error {
	if err := s.systemd.Start(ctx, syncTimerUnit(remoteName)); err != nil {
		return fmt.Errorf("start sync timer: %w", err)
	}
	return nil
}

func (s *Service) StopSyncTimer(ctx context.Context, remoteName string) error {
	if err := s.systemd.Stop(ctx, syncTimerUnit(remoteName)); err != nil {
		return fmt.Errorf("stop sync timer: %w", err)
	}
	return nil
}

func (s *Service) SetMountAutoStart(ctx context.Context, remoteName string, enabled bool) error {
	unit := mountUnit(remoteName)
	if enabled {
		if err := s.systemd.Enable(ctx, unit); err != nil {
			return fmt.Errorf("enable mount autostart: %w", err)
		}
		return nil
	}

	if err := s.systemd.Disable(ctx, unit); err != nil {
		return fmt.Errorf("disable mount autostart: %w", err)
	}
	return nil
}

func (s *Service) SetSyncAutoStart(ctx context.Context, remoteName string, enabled bool) error {
	unit := syncTimerUnit(remoteName)
	if enabled {
		if err := s.systemd.Enable(ctx, unit); err != nil {
			return fmt.Errorf("enable sync autostart: %w", err)
		}
		return nil
	}

	if err := s.systemd.Disable(ctx, unit); err != nil {
		return fmt.Errorf("disable sync autostart: %w", err)
	}
	return nil
}

func (s *Service) ReloadUnits(ctx context.Context) error {
	if err := s.systemd.DaemonReload(ctx); err != nil {
		return fmt.Errorf("systemd daemon-reload: %w", err)
	}
	return nil
}

func (s *Service) ExecuteAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	req.Remote = strings.TrimSpace(req.Remote)
	req.LocalPath = strings.TrimSpace(req.LocalPath)

	switch req.Type {
	case ActionStartMount:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.StartMount(ctx, req.Remote); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "mount started"}, nil
	case ActionStopMount:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.StopMount(ctx, req.Remote); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "mount stopped"}, nil
	case ActionStartSync:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.StartSyncTimer(ctx, req.Remote); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "sync timer started"}, nil
	case ActionStopSync:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.StopSyncTimer(ctx, req.Remote); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "sync timer stopped"}, nil
	case ActionSetupMount:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.SetupRemote(ctx, req.Remote, "mount", req.LocalPath, req.AutoStart); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "mount prepared"}, nil
	case ActionSetupSync:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.SetupRemote(ctx, req.Remote, "sync", req.LocalPath, req.AutoStart); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "sync prepared"}, nil
	case ActionEnableMountAuto:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.SetMountAutoStart(ctx, req.Remote, true); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "mount autostart enabled"}, nil
	case ActionDisableMountAuto:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.SetMountAutoStart(ctx, req.Remote, false); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "mount autostart disabled"}, nil
	case ActionEnableSyncAuto:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.SetSyncAutoStart(ctx, req.Remote, true); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "sync autostart enabled"}, nil
	case ActionDisableSyncAuto:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.SetSyncAutoStart(ctx, req.Remote, false); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "sync autostart disabled"}, nil
	case ActionReloadUnits:
		if err := s.ReloadUnits(ctx); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "units reloaded"}, nil
	case ActionDiagnoseRemote:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		details, err := s.DiagnoseRemote(ctx, req.Remote)
		if err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: details}, nil
	case ActionRepairRemote:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.RepairRemote(ctx, req.Remote); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "repair completed"}, nil
	case ActionRemoveService:
		if req.Remote == "" {
			return ActionResult{}, errors.New("remote is required")
		}
		if err := s.RemoveRemoteService(ctx, req.Remote); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{OK: true, Message: "service removed"}, nil
	default:
		return ActionResult{}, fmt.Errorf("unknown action type: %s", req.Type)
	}
}

type remoteInspection struct {
	remoteName        string
	localPath         string
	mountActive       bool
	mountEnabled      bool
	syncActive        bool
	syncTimerEnabled  bool
	mountUnitFile     bool
	syncSvcUnitFile   bool
	syncTimerUnitFile bool
	errors            []string
}

func (s *Service) inspectRemote(ctx context.Context, remoteName string) remoteInspection {
	inspection := remoteInspection{remoteName: remoteName}

	localPath, err := s.pathReader.Get(remoteName)
	if err != nil {
		inspection.errors = append(inspection.errors, fmt.Sprintf("local path: %v", err))
		localPath = filepath.Join(s.cloudDir, remoteName)
	}
	inspection.localPath = localPath

	mountActive, err := s.systemd.IsActive(ctx, mountUnit(remoteName))
	if err != nil {
		inspection.errors = append(inspection.errors, fmt.Sprintf("%s active: %v", mountUnit(remoteName), err))
	}
	inspection.mountActive = mountActive

	mountEnabled, err := s.systemd.IsEnabled(ctx, mountUnit(remoteName))
	if err != nil {
		inspection.errors = append(inspection.errors, fmt.Sprintf("%s enabled: %v", mountUnit(remoteName), err))
	}
	inspection.mountEnabled = mountEnabled

	syncActive, err := s.systemd.IsActive(ctx, syncServiceUnit(remoteName))
	if err != nil {
		inspection.errors = append(inspection.errors, fmt.Sprintf("%s active: %v", syncServiceUnit(remoteName), err))
	}
	inspection.syncActive = syncActive

	syncTimerEnabled, err := s.systemd.IsEnabled(ctx, syncTimerUnit(remoteName))
	if err != nil {
		inspection.errors = append(inspection.errors, fmt.Sprintf("%s enabled: %v", syncTimerUnit(remoteName), err))
	}
	inspection.syncTimerEnabled = syncTimerEnabled

	inspection.mountUnitFile = fileExists(filepath.Join(s.systemdDir, mountUnit(remoteName)))
	inspection.syncSvcUnitFile = fileExists(filepath.Join(s.systemdDir, syncServiceUnit(remoteName)))
	inspection.syncTimerUnitFile = fileExists(filepath.Join(s.systemdDir, syncTimerUnit(remoteName)))

	return inspection
}

func (s *Service) DiagnoseRemote(ctx context.Context, remoteName string) (string, error) {
	remoteName = strings.TrimSpace(remoteName)
	if remoteName == "" {
		return "", errors.New("remote is required")
	}

	info := s.inspectRemote(ctx, remoteName)

	lines := []string{
		fmt.Sprintf("Diagnostico da conta: %s", remoteName),
		fmt.Sprintf("Caminho local: %s", info.localPath),
		fmt.Sprintf("Mount: active=%t enabled=%t unit_file=%t", info.mountActive, info.mountEnabled, info.mountUnitFile),
		fmt.Sprintf("Sync: active=%t timer_enabled=%t service_file=%t timer_file=%t", info.syncActive, info.syncTimerEnabled, info.syncSvcUnitFile, info.syncTimerUnitFile),
	}

	probableCause := inferProbableCause(info.errors)
	if probableCause != "" {
		lines = append(lines, "Causa provavel: "+probableCause)
	}

	if len(info.errors) > 0 {
		lines = append(lines, "Erros detectados:")
		for _, item := range info.errors {
			lines = append(lines, "- "+item)
		}
	}

	if unitStatus := s.unitStatusSnippet(ctx, mountUnit(remoteName)); unitStatus != "" {
		lines = append(lines, "", "systemctl status mount:", unitStatus)
	}
	if unitStatus := s.unitStatusSnippet(ctx, syncServiceUnit(remoteName)); unitStatus != "" {
		lines = append(lines, "", "systemctl status sync service:", unitStatus)
	}
	if unitStatus := s.unitStatusSnippet(ctx, syncTimerUnit(remoteName)); unitStatus != "" {
		lines = append(lines, "", "systemctl status sync timer:", unitStatus)
	}

	if len(info.errors) == 0 && probableCause == "" {
		lines = append(lines, "", "Nenhum problema estrutural detectado no momento.")
	}

	return strings.Join(lines, "\n"), nil
}

func (s *Service) RepairRemote(ctx context.Context, remoteName string) error {
	remoteName = strings.TrimSpace(remoteName)
	if remoteName == "" {
		return errors.New("remote is required")
	}

	info := s.inspectRemote(ctx, remoteName)

	mode := "mount"
	autoStart := info.mountEnabled || info.mountActive

	if info.syncSvcUnitFile || info.syncTimerUnitFile || info.syncActive || info.syncTimerEnabled {
		mode = "sync"
		autoStart = info.syncTimerEnabled || info.syncActive
	}

	if err := s.SetupRemote(ctx, remoteName, mode, info.localPath, autoStart); err != nil {
		return fmt.Errorf("repair setup: %w", err)
	}

	return nil
}

func (s *Service) RemoveRemoteService(ctx context.Context, remoteName string) error {
	remoteName = strings.TrimSpace(remoteName)
	if remoteName == "" {
		return errors.New("remote is required")
	}

	errList := make([]string, 0)

	stopUnits := []string{mountUnit(remoteName), syncTimerUnit(remoteName), syncServiceUnit(remoteName)}
	for _, unit := range stopUnits {
		if err := s.systemd.Stop(ctx, unit); err != nil && !isIgnorableSystemdError(err) {
			errList = append(errList, fmt.Sprintf("stop %s: %v", unit, err))
		}
	}

	disableUnits := []string{mountUnit(remoteName), syncTimerUnit(remoteName)}
	for _, unit := range disableUnits {
		if err := s.systemd.Disable(ctx, unit); err != nil && !isIgnorableSystemdError(err) {
			errList = append(errList, fmt.Sprintf("disable %s: %v", unit, err))
		}
	}

	unitFiles := []string{
		filepath.Join(s.systemdDir, mountUnit(remoteName)),
		filepath.Join(s.systemdDir, syncServiceUnit(remoteName)),
		filepath.Join(s.systemdDir, syncTimerUnit(remoteName)),
	}

	for _, unitFile := range unitFiles {
		if err := os.Remove(unitFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			errList = append(errList, fmt.Sprintf("remove %s: %v", unitFile, err))
		}
	}

	if err := s.systemd.DaemonReload(ctx); err != nil {
		errList = append(errList, fmt.Sprintf("daemon-reload: %v", err))
	}

	if deleter, ok := s.pathReader.(RemotePathDeleter); ok {
		if err := deleter.Delete(remoteName); err != nil {
			errList = append(errList, fmt.Sprintf("cleanup remote-paths entry: %v", err))
		}
	}

	if len(errList) > 0 {
		return errors.New(strings.Join(errList, "; "))
	}

	return nil
}

func (s *Service) unitStatusSnippet(ctx context.Context, unit string) string {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "status", unit, "--no-pager", "--lines", "8")
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	if text == "" && err != nil {
		return err.Error()
	}
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	if len(lines) > 14 {
		lines = lines[:14]
		lines = append(lines, "... (saida truncada)")
	}

	return strings.Join(lines, "\n")
}

func inferProbableCause(items []string) string {
	if len(items) == 0 {
		return ""
	}

	combined := strings.ToLower(strings.Join(items, " | "))

	switch {
	case strings.Contains(combined, "exit status 4") || strings.Contains(combined, "not found") || strings.Contains(combined, "not loaded") || strings.Contains(combined, "could not be found"):
		return "unidade systemd ausente ou ainda nao criada"
	case strings.Contains(combined, "permission denied") || strings.Contains(combined, "access denied"):
		return "permissao insuficiente para operar servicos do systemd --user"
	case strings.Contains(combined, "fusermount") || strings.Contains(combined, "fuse"):
		return "dependencia FUSE indisponivel para montagem"
	case strings.Contains(combined, "rclone mount") || strings.Contains(combined, "rclone bisync") || strings.Contains(combined, "execstart"):
		return "falha ao executar rclone ou configuracao invalida da conta"
	default:
		return "falha operacional nos servicos da conta; verifique o status detalhado abaixo"
	}
}

func isIgnorableSystemdError(err error) bool {
	if err == nil {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not loaded") ||
		strings.Contains(msg, "could not be found") ||
		strings.Contains(msg, "not-found") ||
		strings.Contains(msg, "does not exist")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Service) SetupRemote(ctx context.Context, remoteName string, mode string, localPath string, autoStart bool) error {
	if strings.TrimSpace(remoteName) == "" {
		return errors.New("remote is required")
	}
	if localPath == "" {
		localPath = filepath.Join(s.cloudDir, remoteName)
	}

	if err := os.MkdirAll(localPath, 0o755); err != nil {
		return fmt.Errorf("create local folder: %w", err)
	}
	if err := os.MkdirAll(s.systemdDir, 0o755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}

	if writer, ok := s.pathReader.(RemotePathWriter); ok {
		if err := writer.Set(remoteName, localPath); err != nil {
			return fmt.Errorf("persist local path: %w", err)
		}
	}

	switch mode {
	case "mount":
		if err := s.writeMountUnit(remoteName, localPath); err != nil {
			return err
		}
		if err := s.systemd.DaemonReload(ctx); err != nil {
			return err
		}
		if autoStart {
			if err := s.systemd.Enable(ctx, mountUnit(remoteName)); err != nil {
				return err
			}
		}
		if err := s.systemd.Start(ctx, mountUnit(remoteName)); err != nil {
			return err
		}
	case "sync":
		if err := s.writeSyncUnits(remoteName, localPath); err != nil {
			return err
		}
		if err := s.systemd.DaemonReload(ctx); err != nil {
			return err
		}
		if autoStart {
			if err := s.systemd.Enable(ctx, syncTimerUnit(remoteName)); err != nil {
				return err
			}
		}
		if err := s.systemd.Start(ctx, syncTimerUnit(remoteName)); err != nil {
			return err
		}
	default:
		return errors.New("mode must be mount or sync")
	}

	return nil
}

func (s *Service) writeMountUnit(remoteName string, localPath string) error {
	unmount := "fusermount3 -u"
	if _, err := exec.LookPath("fusermount3"); err != nil {
		unmount = "fusermount -u"
	}

	unitPath := filepath.Join(s.systemdDir, mountUnit(remoteName))
	unit := fmt.Sprintf(`[Unit]
Description=Mount %s
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=notify
ExecStart=%s mount %s: "%s" --vfs-cache-mode full --no-modtime
ExecStop=%s "%s"
Restart=on-failure
RestartSec=15

[Install]
WantedBy=graphical-session.target
`, remoteName, s.rcloneBin, remoteName, localPath, unmount, localPath)

	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("write mount unit: %w", err)
	}
	return nil
}

func (s *Service) writeSyncUnits(remoteName string, localPath string) error {
	servicePath := filepath.Join(s.systemdDir, syncServiceUnit(remoteName))
	timerPath := filepath.Join(s.systemdDir, syncTimerUnit(remoteName))

	service := fmt.Sprintf(`[Unit]
Description=Sync %s

[Service]
Type=oneshot
ExecStart=%s bisync "%s:" "%s" --create-empty-src-dirs --compare size,modtime,checksum --slow-hash-sync-only --resync --verbose
`, remoteName, s.rcloneBin, remoteName, localPath)

	timer := fmt.Sprintf(`[Unit]
Description=Timer 15m %s

[Timer]
OnBootSec=5min
OnUnitActiveSec=15min

[Install]
WantedBy=timers.target
`, remoteName)

	if err := os.WriteFile(servicePath, []byte(service), 0o644); err != nil {
		return fmt.Errorf("write sync service: %w", err)
	}
	if err := os.WriteFile(timerPath, []byte(timer), 0o644); err != nil {
		return fmt.Errorf("write sync timer: %w", err)
	}

	return nil
}

func appendError(state *RemoteState, err error) {
	if err == nil {
		return
	}

	if state.LastError == "" {
		state.LastError = err.Error()
		return
	}

	state.LastError = state.LastError + "; " + err.Error()
}

func deriveTrayState(states []RemoteState) TrayState {
	if len(states) == 0 {
		return TrayStateIdle
	}

	hasError := false
	for _, state := range states {
		if state.SyncActive {
			return TrayStateSyncing
		}
		if state.LastError != "" {
			hasError = true
		}
	}

	if hasError {
		return TrayStateError
	}

	return TrayStateIdle
}

func mountUnit(remoteName string) string {
	return "rclone-mount-" + remoteName + ".service"
}

func syncServiceUnit(remoteName string) string {
	return "rclone-sync-" + remoteName + ".service"
}

func syncTimerUnit(remoteName string) string {
	return "rclone-sync-" + remoteName + ".timer"
}

var ErrNotImplemented = errors.New("not implemented")
