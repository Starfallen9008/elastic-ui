package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"elastic-ui/internal/config"
	"elastic-ui/internal/service"
	"elastic-ui/internal/shell"
	"elastic-ui/internal/tray"
	"elastic-ui/internal/update"
)

type App struct {
	ctx context.Context

	svc *service.Service

	mu          sync.Mutex
	pendingScan string
	trayReady   bool
	quitting    bool
	quitOnce    sync.Once
	lastRuntime string
	startHidden bool
	showPending bool
	updateMu    sync.Mutex
}

func NewApp() *App {
	return &App{svc: service.New()}
}

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	pending := a.pendingScan
	a.pendingScan = ""
	a.mu.Unlock()

	logf("startup connectError=%v pendingScan=%q", a.svc.ConnectError(), pending)
	runtime.WindowSetTitle(ctx, "Elastic Defend 本机控制台")

	a.trayReady = tray.Start(tray.Actions{
		Show: func() {
			runtime.WindowShow(ctx)
			runtime.WindowUnminimise(ctx)
		},
		Scan: func() {
			runtime.WindowShow(ctx)
			runtime.EventsEmit(ctx, "navigate", "scan")
		},
		Threats: func() {
			runtime.WindowShow(ctx)
			runtime.EventsEmit(ctx, "navigate", "threats")
		},
		NotificationDetails: func(id string) {
			runtime.WindowShow(ctx)
			runtime.WindowUnminimise(ctx)
			runtime.EventsEmit(ctx, "notification-details", id)
		},
		Quit: func() {
			a.mu.Lock()
			a.quitting = true
			a.mu.Unlock()
			runtime.Quit(ctx)
		},
	})
	logf("tray ready=%v", a.trayReady)
	a.mu.Lock()
	showPending := a.showPending
	a.mu.Unlock()
	if showPending || (a.startHidden && !a.trayReady) {
		runtime.WindowShow(ctx)
	}
	go a.watchAlerts()

	if pending != "" {
		go a.HandleScan(pending)
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	quitting := a.quitting
	a.mu.Unlock()
	if quitting || !a.trayReady || !a.svc.CloseToTrayEnabled() {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

func (a *App) ShowWindow() {
	a.mu.Lock()
	ctx := a.ctx
	a.showPending = true
	a.mu.Unlock()
	if ctx != nil {
		runtime.WindowShow(ctx)
		runtime.WindowUnminimise(ctx)
	}
}

func (a *App) shutdown() {
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	if a.trayReady {
		a.quitOnce.Do(func() {
			tray.Quit()
		})
	}
}

func (a *App) watchAlerts() {
	started := time.Now().Add(-5 * time.Second)
	seen := map[string]time.Time{}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		a.mu.Lock()
		quitting := a.quitting
		a.mu.Unlock()
		if quitting {
			return
		}
		if !a.svc.NotificationsEnabled() {
			started = time.Now().Add(-5 * time.Second)
			continue
		}
		state := a.svc.NotificationState()
		if !state.Ready {
			continue
		}
		fresh := []service.Threat{}
		cutoff := started
		if recent := time.Now().Add(-10 * time.Minute); recent.After(cutoff) {
			cutoff = recent
		}
		for offset := 0; offset < 10000; offset += 200 {
			page, err := a.svc.Threats(offset, 200, "", "24h")
			if err != nil {
				logf("alert polling: %v", err)
				break
			}
			older := false
			for _, item := range page.Items {
				at, err := time.Parse(time.RFC3339Nano, item.Time)
				if err != nil || at.Before(cutoff) {
					older = true
					continue
				}
				if _, exists := seen[item.ID]; exists {
					continue
				}
				seen[item.ID] = at
				fresh = append(fresh, item)
			}
			if older || offset+len(page.Items) >= int(page.Total) || len(page.Items) == 0 {
				break
			}
		}
		for i := len(fresh) - 1; i >= 0; i-- {
			if !a.svc.NotificationsEnabled() {
				break
			}
			content := service.NotificationContent(fresh[i])
			tray.ShowNotification(tray.Notification{
				ID: content.ID, Kind: content.Kind, Outcome: content.Outcome,
				Object: content.Object, Rule: content.Rule, MalwareName: content.MalwareName,
			})
		}
		for id, at := range seen {
			if time.Since(at) > 24*time.Hour {
				delete(seen, id)
			}
		}
	}
}

func (a *App) SetNotifications(enabled bool) error {
	mode := "native"
	if enabled {
		mode = "ui"
	}
	return a.SetNotificationMode(mode)
}

func (a *App) SetNotificationMode(mode string) error {
	tray.ClearNotifications()
	if err := a.svc.SetNotificationMode(mode); err != nil {
		return err
	}
	tray.ClearNotifications()
	return nil
}

func (a *App) GetNotificationState() service.NotificationState { return a.svc.NotificationState() }

func (a *App) GetIndependentPolicyState() (*service.IndependentPolicyState, error) {
	return a.svc.IndependentPolicyState()
}
func (a *App) CreateIndependentPolicy(name string) (*service.IndependentPolicyState, error) {
	tray.ClearNotifications()
	return a.svc.CreateIndependentPolicy(name)
}
func (a *App) RenameIndependentPolicy(name string) (*service.IndependentPolicyState, error) {
	return a.svc.RenameIndependentPolicy(name)
}
func (a *App) CompletePolicySetup() error { return a.svc.CompletePolicySetup() }

func (a *App) SetCloseToTray(enabled bool) error {
	return a.svc.SetCloseToTray(enabled)
}

func (a *App) HandleScan(path string) {
	a.mu.Lock()
	ctx := a.ctx
	if ctx == nil {
		a.pendingScan = path
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	logf("handle scan path=%q", path)
	if _, err := a.svc.StartScan(path); err != nil {
		logf("scan error: %v", err)
		runtime.EventsEmit(ctx, "scan-error", err.Error())
	} else {
		logf("scan submitted")
		runtime.EventsEmit(ctx, "navigate", "scan")
	}
	runtime.WindowShow(ctx)
	runtime.WindowUnminimise(ctx)
}

func (a *App) GetSettings() *config.Settings {
	return a.svc.Settings()
}

func (a *App) Configure(in service.ConnectionInput) (*service.ConnectionResult, error) {
	result, err := a.svc.Configure(in)
	if err == nil && result.OK {
		runtime.EventsEmit(a.ctx, "configured")
	}
	return result, err
}

func (a *App) GetStatus() (*service.StatusView, error) {
	status, err := a.svc.Status()
	if err == nil && status != nil {
		a.mu.Lock()
		changed := status.AgentRuntime.State != a.lastRuntime
		a.lastRuntime = status.AgentRuntime.State
		a.mu.Unlock()
		if changed {
			logf("agent runtime state=%s message=%s error=%s", status.AgentRuntime.State, status.AgentRuntime.Message, status.AgentRuntime.Error)
		}
	}
	return status, err
}

func (a *App) RebindLocal() (*service.StatusView, error) {
	if err := a.svc.Rebind(); err != nil {
		return nil, err
	}
	return a.svc.Status()
}

func (a *App) GetThreats(offset int, limit int, query string, rangeKey string) (*service.ThreatPage, error) {
	return a.svc.Threats(offset, limit, query, rangeKey)
}

func (a *App) GetEvents(offset int, limit int, rangeKey string, category string, query string) (*service.EventPage, error) {
	return a.svc.Events(offset, limit, rangeKey, category, query)
}

func (a *App) GetThreatDetail(id string) (*service.ThreatDetail, error) {
	return a.svc.ThreatDetail(id)
}

func (a *App) GetProcessChain(alertID string) (*service.ProcessChain, error) {
	return a.svc.ProcessChain(alertID)
}

func (a *App) MarkRead(ids []string) error {
	return a.svc.MarkRead(ids)
}

func (a *App) StartScan(path string) (*service.ScanTask, error) {
	task, err := a.svc.StartScan(path)
	if err == nil {
		runtime.EventsEmit(a.ctx, "navigate", "scan")
	}
	return task, err
}

func (a *App) GetScans() []*service.ScanTask {
	return a.svc.Scans()
}

func (a *App) GetScanResults(taskID string, offset int) (*service.ScanResults, error) {
	return a.svc.ScanResults(taskID, offset)
}

func (a *App) GetQuarantine(offset int) (*service.ScanResults, error) {
	return a.svc.Quarantine(offset)
}

func (a *App) GetQuarantineFile(alertID string) (*service.ScanFile, error) {
	return a.svc.QuarantineFile(alertID)
}

func (a *App) RecoverQuarantinedFile(alertID string) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "保存从隔离区找回的文件", DefaultFilename: "elastic-recovered-file.zip",
		Filters: []runtime.FileFilter{{DisplayName: "ZIP", Pattern: "*.zip"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := a.svc.SaveQuarantinedFile(alertID, path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) TrustQuarantinedFile(alertID string) error {
	return a.svc.TrustQuarantinedFile(alertID)
}

func (a *App) RecoverQuarantinedFiles(ids []string) ([]service.QuarantineOperation, error) {
	if len(ids) == 0 || len(ids) > 10000 {
		return nil, fmt.Errorf("请选择 1 至 10000 个文件")
	}
	directory, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择隔离文件保存目录"})
	if err != nil || directory == "" {
		return nil, err
	}
	return a.svc.SaveQuarantinedFiles(ids, directory), nil
}

func (a *App) Isolate(confirmHost string) (*service.StatusView, error) {
	return a.svc.Isolate(confirmHost)
}

func (a *App) GetTrustItems() ([]service.TrustItem, error) {
	return a.svc.TrustItems()
}

func (a *App) GetProcesses(force bool) ([]service.ProcessInfo, error) {
	return a.svc.Processes(force)
}

func (a *App) GetFile(path string) (*service.RetrievedFile, error) {
	return a.svc.GetFile(path)
}

func (a *App) KillProcess(pid int, entityID string) error {
	return a.svc.KillProcess(pid, entityID)
}

func (a *App) SuspendProcess(pid int, entityID string) error {
	return a.svc.SuspendProcess(pid, entityID)
}

func (a *App) GetAudit(limit int) ([]service.AuditEntry, error) {
	return a.svc.GetAudit(limit)
}

func (a *App) ExportDiagnostics() (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存诊断包",
		DefaultFilename: "elastic-defend-diagnostics-" + time.Now().Format("20060102-150405") + ".zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP", Pattern: "*.zip"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if err := a.svc.WriteDiagnostics(path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) ClearCache() error {
	return a.svc.ClearCache()
}

func (a *App) CloseWindow() {
	if a.svc.CloseToTrayEnabled() && a.trayReady {
		runtime.WindowHide(a.ctx)
		return
	}
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	runtime.Quit(a.ctx)
}

func (a *App) SetTrayTheme(dark bool) {
	tray.SetTheme(dark)
}

func (a *App) CreateTrust(in service.TrustInput) error {
	return a.svc.CreateTrust(in)
}

func (a *App) DeleteTrust(id string) error {
	return a.svc.DeleteTrust(id)
}

func (a *App) SetShellIntegration(enabled bool) error {
	if err := shell.SetEnabled(enabled); err != nil {
		return err
	}
	settings := a.svc.Settings()
	settings.ShellIntegration = enabled
	return settings.Save()
}

func (a *App) ShellIntegrationEnabled() bool {
	return shell.IsEnabled()
}

func (a *App) StartupEnabled() (bool, error) { return shell.StartupEnabled() }

func (a *App) SetStartupEnabled(enabled bool) error { return shell.SetStartupEnabled(enabled) }

func (a *App) CheckUpdate() (*update.Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return update.Check(ctx)
}

func (a *App) InstallUpdate() error {
	if !a.updateMu.TryLock() {
		return fmt.Errorf("更新正在进行")
	}
	defer a.updateMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	release, err := update.Check(ctx)
	if err != nil {
		return err
	}
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	path, hash, err := update.Download(ctx, *release, filepath.Join(dir, "updates"))
	if err != nil {
		return err
	}
	if err := update.StartInstall(path, hash); err != nil {
		return err
	}
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	runtime.Quit(a.ctx)
	return nil
}

func (a *App) OpenKibana(path string) {
	base := strings.TrimRight(a.svc.KibanaURL(), "/")
	if base == "" {
		return
	}
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	runtime.BrowserOpenURL(a.ctx, base+path)
}

func (a *App) Disconnect() error {
	return a.svc.Disconnect()
}

func (a *App) AppInfo() map[string]string {
	dir, _ := config.Dir()
	failure, _ := os.ReadFile(filepath.Join(dir, "updates", "last-error.txt"))
	return map[string]string{
		"updateError": strings.TrimSpace(string(failure)),
		"version":     update.Version,
		"repository":  "https://github.com/" + update.Repository,
		"configDir":   dir,
		"kibana":      a.svc.KibanaURL(),
		"buildNote":   fmt.Sprintf("Windows / Wails %s", "v2.17.1"),
	}
}
