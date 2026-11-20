//go:build windows

package main

import (
	"embed"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"elastic-ui/internal/shell"
	"elastic-ui/internal/tray"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logf("start args=%v", os.Args)
	defer func() {
		if r := recover(); r != nil {
			logf("panic: %v", r)
			panic(r)
		}
	}()

	scanPath := parseScanArg(os.Args[1:])

	if generatingBindings {
		_ = wails.Run(&options.App{Bind: []interface{}{&App{}}})
		return
	}
	for _, arg := range os.Args[1:] {
		if arg == "--uninstall-integration" {
			_ = shell.SetStartupEnabled(false)
			_ = shell.SetEnabled(false)
			return
		}
		if arg == "--refresh-integration" {
			if enabled, _ := shell.StartupEnabled(); enabled {
				_ = shell.SetStartupEnabled(true)
			}
			if shell.IsEnabled() {
				_ = shell.SetEnabled(true)
			}
			return
		}
		if arg == "--preview-notification" {
			previewNotification()
			return
		}
	}
	release, alreadyRunning, err := shell.AcquireSingleInstance()
	logf("single instance already=%v err=%v", alreadyRunning, err)
	if err == nil {
		defer release()
		if alreadyRunning {
			if scanPath != "" {
				_ = shell.ForwardScan(scanPath)
			} else if !hasArgument("--autostart") {
				_ = shell.ForwardScan("\x00show")
			}
			logf("forwarded scan and exit")
			return
		}
	}

	app := NewApp()
	for _, arg := range os.Args[1:] {
		if arg == "--autostart" {
			app.startHidden = true
		}
	}
	app.pendingScan = scanPath
	_, _ = shell.StartPipeServer(func(message string) {
		if message == "\x00show" {
			app.ShowWindow()
			return
		}
		app.HandleScan(message)
	})

	err = wails.Run(&options.App{
		Title:            "Elastic Defend 本机控制台",
		Width:            1280,
		Height:           840,
		MinWidth:         1024,
		MinHeight:        680,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 17, G: 19, B: 24, A: 1},
		Frameless:        true,
		StartHidden:      app.startHidden,
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind:             []interface{}{app},
	})
	if err != nil {
		logf("wails.Run error: %v", err)
		println("Error:", err.Error())
	}
	app.shutdown()
	logf("exit")
}

// Local preview uses the production popup renderer without accessing Elastic.
func previewNotification() {
	done := make(chan struct{}, 1)
	if !tray.Start(tray.Actions{NotificationDetails: func(string) { done <- struct{}{} }}) {
		return
	}
	defer tray.Quit()
	tray.ShowNotification(tray.Notification{
		ID: "preview-alert", Kind: "恶意软件防护", Outcome: "已拦截威胁",
		Object:      `C:\Users\19393\AppData\Local\Temp\opencode\notification-preview\example-threat-with-complete-filename.exe`,
		MalwareName: "测试报毒名：Example.Test.Malware",
	})
	select {
	case <-done:
		time.Sleep(350 * time.Millisecond)
	case <-time.After(16 * time.Second):
	}
}

func parseScanArg(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--scan" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], "--scan=") {
			return strings.TrimPrefix(args[i], "--scan=")
		}
	}
	return ""
}

func hasArgument(argument string) bool {
	for _, value := range os.Args[1:] {
		if value == argument {
			return true
		}
	}
	return false
}
