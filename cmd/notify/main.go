//go:build windows

package main

import (
	"fmt"
	"time"

	"elastic-ui/internal/tray"
)

// This fixture never connects to Elastic or changes the selected notification mode.
func main() {
	details := make(chan string, 1)
	if !tray.Start(tray.Actions{NotificationDetails: func(id string) { details <- id }}) {
		fmt.Println("custom notification window unavailable")
		return
	}
	defer tray.Quit()
	tray.ShowNotification(tray.Notification{
		ID: "fixture-alert", Kind: "恶意软件防护", Outcome: "已拦截威胁",
		Object:      `C:\Users\19393\AppData\Local\Temp\opencode\notification-preview\example-threat-with-complete-filename.exe`,
		MalwareName: "测试报毒名：Example.Test.Malware",
	})
	select {
	case id := <-details:
		fmt.Println("details:", id)
		<-time.After(350 * time.Millisecond)
	case <-time.After(16 * time.Second):
		fmt.Println("custom notification fixture completed")
	}
}
