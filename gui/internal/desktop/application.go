package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/weinne/rclone-auto/gui/internal/app"
	"github.com/weinne/rclone-auto/gui/internal/core"
)

type Application struct {
	ctx    context.Context
	api    *app.API
	bridge *app.Bindings

	cancel context.CancelFunc
	mu     sync.Mutex
}

func NewApplication(api *app.API, bridge *app.Bindings) *Application {
	return &Application{api: api, bridge: bridge}
}

func (a *Application) Startup(ctx context.Context) {
	a.ctx = ctx
	a.startPolling(3 * time.Second)
}

func (a *Application) Shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}

func (a *Application) GetSnapshot() (core.AppSnapshot, error) {
	return a.bridge.GetSnapshot()
}

func (a *Application) ExecuteAction(actionType string, remote string) (core.ActionResult, error) {
	return a.bridge.ExecuteAction(actionType, remote)
}

func (a *Application) ExecuteActionWithOptions(actionType string, remote string, localPath string, autoStart bool) (core.ActionResult, error) {
	return a.api.Action(context.Background(), core.ActionRequest{
		Type:      core.ActionType(actionType),
		Remote:    remote,
		LocalPath: localPath,
		AutoStart: autoStart,
	})
}

func (a *Application) OpenLocalFolder(remote string) error {
	snapshot, err := a.api.Snapshot(context.Background())
	if err != nil {
		return err
	}

	for _, item := range snapshot.Remotes {
		if item.Name == remote {
			if item.LocalPath == "" {
				return errors.New("local path not configured")
			}
			cmd := exec.Command("xdg-open", item.LocalPath)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("open local folder: %w", err)
			}
			return nil
		}
	}

	return errors.New("remote not found")
}

func (a *Application) OpenCloudLoginWizard() error {
	if _, err := exec.LookPath("rclone"); err != nil {
		return errors.New("rclone is not installed or not available in PATH")
	}

	loginCmd := `rclone config; code=$?; echo ""; if test $code -ne 0; then echo "Assistente finalizou com erro ($code)."; else echo "Assistente finalizado."; fi; echo "Pressione ENTER para fechar..."; read _`

	for _, candidate := range terminalCandidates() {
		args := append([]string{}, candidate.args...)
		args = append(args, "sh", "-lc", loginCmd)

		cmd := exec.Command(candidate.name, args...)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	if _, err := exec.LookPath("sh"); err == nil {
		cmd := exec.Command("sh", "-lc", loginCmd)
		if runErr := cmd.Run(); runErr == nil {
			return nil
		}
	}

	return errors.New("could not launch terminal for rclone login")
}

func (a *Application) CreateCloudRemote(remoteName string, provider string) (core.ActionResult, error) {
	remoteName = strings.TrimSpace(remoteName)
	provider = strings.TrimSpace(provider)

	if remoteName == "" {
		return core.ActionResult{}, errors.New("remote name is required")
	}
	if provider == "" {
		return core.ActionResult{}, errors.New("provider is required")
	}

	rcloneBin, err := resolveRcloneBinary()
	if err != nil {
		return core.ActionResult{}, err
	}

	if provider == "onedrive" {
		if err := createOneDriveRemoteAuto(rcloneBin, remoteName); err != nil {
			return core.ActionResult{}, err
		}
		return core.ActionResult{OK: true, Message: "account connected"}, nil
	}

	// Non-interactive create. OAuth providers will open browser-based login when needed.
	cmd := exec.Command(rcloneBin, "config", "create", remoteName, provider)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return core.ActionResult{}, fmt.Errorf("create remote failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return core.ActionResult{OK: true, Message: "account connected"}, nil
}

type rcloneConfigPrompt struct {
	State  string             `json:"State"`
	Option rcloneConfigOption `json:"Option"`
	Error  string             `json:"Error"`
	Result string             `json:"Result"`
}

type rcloneConfigOption struct {
	Name       string               `json:"Name"`
	Default    any                  `json:"Default"`
	DefaultStr string               `json:"DefaultStr"`
	Required   bool                 `json:"Required"`
	Examples   []rcloneConfigChoice `json:"Examples"`
}

type rcloneConfigChoice struct {
	Value string `json:"Value"`
	Help  string `json:"Help"`
}

func createOneDriveRemoteAuto(rcloneBin string, remoteName string) error {
	cleanup := func() {
		_ = exec.Command(rcloneBin, "config", "delete", remoteName).Run()
	}

	prompt, err := runConfigCreatePrompt(rcloneBin, remoteName, "onedrive")
	if err != nil {
		cleanup()
		return err
	}

	for i := 0; i < 80 && prompt.State != ""; i++ {
		answer, answerErr := choosePromptAnswer("onedrive", prompt)
		if answerErr != nil {
			cleanup()
			return answerErr
		}

		next, stepErr := runConfigContinuePrompt(rcloneBin, remoteName, prompt.State, answer)
		if stepErr != nil {
			cleanup()
			return stepErr
		}

		prompt = next
	}

	if prompt.State != "" {
		cleanup()
		return errors.New("create remote failed: fluxo de configuracao do OneDrive excedeu o limite de etapas")
	}

	return nil
}

func runConfigCreatePrompt(rcloneBin string, remoteName string, provider string) (rcloneConfigPrompt, error) {
	cmd := exec.Command(rcloneBin, "config", "create", remoteName, provider, "--non-interactive", "--all")
	out, err := cmd.CombinedOutput()
	prompt, isPrompt := parseRclonePrompt(out)
	if isPrompt {
		if prompt.Error != "" {
			return prompt, fmt.Errorf("create remote failed: %s", prompt.Error)
		}
		return prompt, nil
	}

	if err != nil {
		return rcloneConfigPrompt{}, fmt.Errorf("create remote failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return rcloneConfigPrompt{}, nil
}

func runConfigContinuePrompt(rcloneBin string, remoteName string, state string, result string) (rcloneConfigPrompt, error) {
	cmd := exec.Command(rcloneBin, "config", "update", remoteName, "--continue", "--state", state, "--result", result)
	out, err := cmd.CombinedOutput()
	prompt, isPrompt := parseRclonePrompt(out)
	if isPrompt {
		if prompt.Error != "" {
			return prompt, fmt.Errorf("create remote failed: %s", prompt.Error)
		}
		return prompt, nil
	}

	if err != nil {
		return rcloneConfigPrompt{}, fmt.Errorf("create remote failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return rcloneConfigPrompt{}, nil
}

func parseRclonePrompt(out []byte) (rcloneConfigPrompt, bool) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return rcloneConfigPrompt{}, false
	}

	var prompt rcloneConfigPrompt
	if err := json.Unmarshal(trimmed, &prompt); err != nil {
		return rcloneConfigPrompt{}, false
	}

	if prompt.State == "" && prompt.Option.Name == "" && prompt.Error == "" && prompt.Result == "" {
		return rcloneConfigPrompt{}, false
	}

	return prompt, true
}

func choosePromptAnswer(provider string, prompt rcloneConfigPrompt) (string, error) {
	name := strings.TrimSpace(prompt.Option.Name)

	if provider == "onedrive" {
		switch name {
		case "region":
			return preferChoice(prompt.Option.Examples, "global")
		case "drive_type":
			if v, err := preferChoice(prompt.Option.Examples, "personal"); err == nil {
				return v, nil
			}
			if v, err := preferChoice(prompt.Option.Examples, "business"); err == nil {
				return v, nil
			}
		case "config_type":
			if v, err := preferChoice(prompt.Option.Examples, "onedrive"); err == nil {
				return v, nil
			}
		}
	}

	if prompt.Result != "" {
		return prompt.Result, nil
	}

	if prompt.Option.DefaultStr != "" {
		return prompt.Option.DefaultStr, nil
	}

	if prompt.Option.Default != nil {
		switch value := prompt.Option.Default.(type) {
		case string:
			return value, nil
		case bool:
			if value {
				return "true", nil
			}
			return "false", nil
		case float64:
			if float64(int64(value)) == value {
				return strconv.FormatInt(int64(value), 10), nil
			}
			return strconv.FormatFloat(value, 'f', -1, 64), nil
		case []any:
			parts := make([]string, 0, len(value))
			for _, item := range value {
				parts = append(parts, fmt.Sprint(item))
			}
			if len(parts) > 0 {
				return strings.Join(parts, " "), nil
			}
		default:
			return fmt.Sprint(value), nil
		}
	}

	if len(prompt.Option.Examples) > 0 {
		return prompt.Option.Examples[0].Value, nil
	}

	if prompt.Option.Required {
		return "", fmt.Errorf("create remote failed: o provedor exige valor manual para '%s'", prompt.Option.Name)
	}

	return "", nil
}

func preferChoice(choices []rcloneConfigChoice, value string) (string, error) {
	for _, choice := range choices {
		if strings.EqualFold(choice.Value, value) {
			return choice.Value, nil
		}
	}

	if len(choices) > 0 {
		return choices[0].Value, nil
	}

	return "", fmt.Errorf("nenhuma opcao disponivel")
}

func (a *Application) SetPollingInterval(intervalMs int) {
	if intervalMs < 1000 {
		intervalMs = 1000
	}
	a.startPolling(time.Duration(intervalMs) * time.Millisecond)
}

type terminalLaunch struct {
	name string
	args []string
}

func terminalCandidates() []terminalLaunch {
	return []terminalLaunch{
		{name: "x-terminal-emulator", args: []string{"-e"}},
		{name: "konsole", args: []string{"-e"}},
		{name: "gnome-terminal", args: []string{"--"}},
		{name: "xfce4-terminal", args: []string{"-e"}},
		{name: "xterm", args: []string{"-e"}},
	}
}

func resolveRcloneBinary() (string, error) {
	if bin, err := exec.LookPath("rclone"); err == nil {
		return bin, nil
	}

	home, err := os.UserHomeDir()
	if err == nil {
		candidate := filepath.Join(home, ".local", "bin", "rclone")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}

	return "", errors.New("rclone não encontrado. Instale o rclone e tente novamente")
}

func (a *Application) startPolling(interval time.Duration) {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	pollCtx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.mu.Unlock()

	poller := app.NewPoller(a.api, interval)
	updates := make(chan core.AppSnapshot, 1)

	go func() {
		_ = poller.Run(pollCtx, updates)
	}()

	go func() {
		for snapshot := range updates {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "snapshot", snapshot)
			}
		}
	}()
}
