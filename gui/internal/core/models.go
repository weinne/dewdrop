package core

import "time"

type TrayState string

const (
	TrayStateIdle    TrayState = "idle"
	TrayStateSyncing TrayState = "syncing"
	TrayStateError   TrayState = "error"
)

type RemoteState struct {
	Name             string `json:"name"`
	LocalPath        string `json:"localPath"`
	MountUnit        string `json:"mountUnit"`
	SyncServiceUnit  string `json:"syncServiceUnit"`
	SyncTimerUnit    string `json:"syncTimerUnit"`
	MountActive      bool   `json:"mountActive"`
	MountEnabled     bool   `json:"mountEnabled"`
	SyncActive       bool   `json:"syncActive"`
	SyncTimerEnabled bool   `json:"syncTimerEnabled"`
	LastError        string `json:"lastError,omitempty"`
}

type AppSnapshot struct {
	TrayState TrayState     `json:"trayState"`
	Remotes   []RemoteState `json:"remotes"`
	CheckedAt time.Time     `json:"checkedAt"`
}

type ActionType string

const (
	ActionStartMount       ActionType = "start-mount"
	ActionStopMount        ActionType = "stop-mount"
	ActionStartSync        ActionType = "start-sync"
	ActionStopSync         ActionType = "stop-sync"
	ActionSetupMount       ActionType = "setup-mount"
	ActionSetupSync        ActionType = "setup-sync"
	ActionEnableMountAuto  ActionType = "enable-mount-autostart"
	ActionDisableMountAuto ActionType = "disable-mount-autostart"
	ActionEnableSyncAuto   ActionType = "enable-sync-autostart"
	ActionDisableSyncAuto  ActionType = "disable-sync-autostart"
	ActionReloadUnits      ActionType = "reload-units"
	ActionDiagnoseRemote   ActionType = "diagnose-remote"
	ActionRepairRemote     ActionType = "repair-remote"
	ActionRemoveService    ActionType = "remove-service"
)

type ActionRequest struct {
	Type      ActionType `json:"type"`
	Remote    string     `json:"remote,omitempty"`
	LocalPath string     `json:"localPath,omitempty"`
	AutoStart bool       `json:"autoStart,omitempty"`
}

type ActionResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}
