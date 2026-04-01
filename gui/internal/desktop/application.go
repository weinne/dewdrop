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

func (a *Application) DeleteCloudRemote(remoteName string) (core.ActionResult, error) {
	remoteName = strings.TrimSpace(remoteName)
	if remoteName == "" {
		return core.ActionResult{}, errors.New("remote name is required")
	}

	errList := make([]string, 0, 2)

	if _, err := a.api.Action(context.Background(), core.ActionRequest{Type: core.ActionRemoveService, Remote: remoteName}); err != nil {
		errList = append(errList, fmt.Sprintf("remove service: %v", err))
	}

	rcloneBin, err := resolveRcloneBinary()
	if err != nil {
		errList = append(errList, err.Error())
	} else {
		cmd := exec.Command(rcloneBin, "config", "delete", remoteName)
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			output := strings.TrimSpace(string(out))
			if !isMissingRemoteError(output) {
				if output == "" {
					errList = append(errList, fmt.Sprintf("delete remote: %v", runErr))
				} else {
					errList = append(errList, fmt.Sprintf("delete remote: %v: %s", runErr, output))
				}
			}
		}
	}

	if len(errList) > 0 {
		return core.ActionResult{}, errors.New(strings.Join(errList, "; "))
	}

	return core.ActionResult{OK: true, Message: "account deleted"}, nil
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

		var (
			next    rcloneConfigPrompt
			stepErr error
		)
		for attempt := 0; attempt < 4; attempt++ {
			next, stepErr = runConfigContinuePrompt(rcloneBin, remoteName, prompt.State, answer)
			if stepErr == nil {
				break
			}
			if !isTransientOneDriveError(stepErr.Error()) || attempt == 3 {
				cleanup()
				return stepErr
			}
			time.Sleep(oneDriveRetryDelay(attempt))
		}

		prompt = next
	}

	if prompt.State != "" {
		cleanup()
		return errors.New("create remote failed: fluxo de configuracao do OneDrive excedeu o limite de etapas")
	}

	if err := validateOneDriveRemote(rcloneBin, remoteName); err != nil {
		cleanup()
		return err
	}

	return nil
}

func validateOneDriveRemote(rcloneBin string, remoteName string) error {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		cmd := exec.Command(
			rcloneBin,
			"--retries", "1",
			"--low-level-retries", "1",
			"--timeout", "20s",
			"lsd",
			remoteName+":",
		)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}

		message := strings.TrimSpace(string(out))
		if message == "" {
			lastErr = fmt.Errorf("create remote failed: validacao final do OneDrive falhou: %w", err)
		} else {
			lastErr = fmt.Errorf("create remote failed: validacao final do OneDrive falhou: %s", message)
		}

		if !isTransientOneDriveError(lastErr.Error()) || attempt == 3 {
			return lastErr
		}

		time.Sleep(oneDriveRetryDelay(attempt))
	}

	if lastErr != nil {
		return lastErr
	}

	return errors.New("create remote failed: validacao final do OneDrive falhou")
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
	if len(trimmed) == 0 {
		return rcloneConfigPrompt{}, false
	}

	candidates := make([][]byte, 0, 8)

	if trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		candidates = append(candidates, trimmed)
	}

	if object := extractFirstJSONObject(trimmed); len(object) > 0 {
		candidates = append(candidates, object)
	}

	for _, line := range bytes.Split(trimmed, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) < 2 {
			continue
		}
		if line[0] == '{' && line[len(line)-1] == '}' {
			candidates = append(candidates, line)
		}
	}

	for i := len(candidates) - 1; i >= 0; i-- {
		var prompt rcloneConfigPrompt
		if err := json.Unmarshal(candidates[i], &prompt); err != nil {
			continue
		}

		if prompt.State == "" && prompt.Option.Name == "" && prompt.Error == "" && prompt.Result == "" {
			continue
		}

		return prompt, true
	}

	return rcloneConfigPrompt{}, false
}

func extractFirstJSONObject(data []byte) []byte {
	start := -1
	depth := 0
	inString := false
	escaped := false

	for i, b := range data {
		if escaped {
			escaped = false
			continue
		}

		if b == '\\' {
			escaped = true
			continue
		}

		if b == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch b {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return bytes.TrimSpace(data[start : i+1])
			}
		}
	}

	return nil
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
		case "drive_id":
			if v := strings.TrimSpace(prompt.Option.DefaultStr); v != "" {
				return v, nil
			}
			if v, ok := preferOneDriveDriveID(prompt.Option.Examples); ok {
				return v, nil
			}
			if !prompt.Option.Required {
				return "", nil
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
	target := strings.ToLower(strings.TrimSpace(value))
	if target == "" {
		return "", fmt.Errorf("valor alvo vazio")
	}

	for _, choice := range choices {
		if strings.EqualFold(strings.TrimSpace(choice.Value), target) {
			return choice.Value, nil
		}
	}

	for _, choice := range choices {
		help := strings.ToLower(strings.TrimSpace(choice.Help))
		if help != "" && strings.Contains(help, target) {
			return choice.Value, nil
		}
	}

	if len(choices) > 0 {
		return choices[0].Value, nil
	}

	return "", fmt.Errorf("nenhuma opcao disponivel")
}

func isMissingRemoteError(output string) bool {
	normalized := strings.ToLower(output)
	return strings.Contains(normalized, "didn't find section") ||
		strings.Contains(normalized, "not found in config file") ||
		strings.Contains(normalized, "could not find section")
}

func preferOneDriveDriveID(choices []rcloneConfigChoice) (string, bool) {
	if len(choices) == 0 {
		return "", false
	}

	for _, choice := range choices {
		help := strings.ToLower(strings.TrimSpace(choice.Help))
		if help == "" {
			continue
		}

		// Prefer personal/root-ish drives and avoid SharePoint/DocumentLibrary-like entries.
		if strings.Contains(help, "personal") ||
			(strings.Contains(help, "onedrive") && !strings.Contains(help, "sharepoint") && !strings.Contains(help, "library") && !strings.Contains(help, "site")) {
			value := strings.TrimSpace(choice.Value)
			if value != "" {
				return value, true
			}
		}
	}

	for _, choice := range choices {
		help := strings.ToLower(strings.TrimSpace(choice.Help))
		if strings.Contains(help, "sharepoint") || strings.Contains(help, "library") || strings.Contains(help, "site") {
			continue
		}
		value := strings.TrimSpace(choice.Value)
		if value != "" {
			return value, true
		}
	}

	for _, choice := range choices {
		value := strings.TrimSpace(choice.Value)
		if value != "" {
			return value, true
		}
	}

	return "", false
}

func isTransientOneDriveError(message string) bool {
	normalized := strings.ToLower(message)
	return strings.Contains(normalized, "service unavailable") ||
		strings.Contains(normalized, "servicenotavailable") ||
		strings.Contains(normalized, "http error 503") ||
		strings.Contains(normalized, "timeout") ||
		strings.Contains(normalized, "temporar") ||
		strings.Contains(normalized, "try again") ||
		strings.Contains(normalized, "rate limit") ||
		strings.Contains(normalized, "too many requests") ||
		strings.Contains(normalized, "http error 429")
}

func oneDriveRetryDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 4 {
		attempt = 4
	}
	return time.Duration(1<<attempt) * time.Second
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
