package main

import (
	"fmt"
	"os"

	"elastic-ui/internal/service"
)

func main() {
	svc := service.New()
	if err := svc.ConnectError(); err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}
	status, err := svc.Status()
	if err != nil {
		fmt.Println("status error:", err)
		os.Exit(1)
	}
	fmt.Printf("host=%s agent=%s policy=%s\n", status.HostName, status.AgentStatus, status.PolicyName)
	fmt.Printf("protection_update=%s age=%d\n", status.ProtectionUpdateVersion, status.ProtectionUpdateAge)
	for _, check := range status.Checks {
		fmt.Printf("check ok=%v %s: %s\n", check.OK, check.Name, check.Detail)
	}
	for _, window := range []string{"24h", "7d", "30d"} {
		page, err := svc.Threats(0, 3, "", window)
		if err != nil {
			fmt.Printf("threats %s error: %v\n", window, err)
			continue
		}
		fmt.Printf("threats %s total=%d page=%d\n", window, page.Total, len(page.Items))
	}
	events, err := svc.Events(0, 3, "24h", "all", "")
	if err != nil {
		fmt.Println("events error:", err)
	} else {
		fmt.Printf("events total=%d page=%d\n", events.Total, len(events.Items))
		for _, event := range events.Items {
			fmt.Printf("event %s | %s | %s | %s\n", event.Time, event.Category, event.Action, event.ProcessName)
		}
	}

	for _, keyword := range []string{"../", `..\`, "win.ini", "boot.ini", "<script>", "onerror=", "union select", "xp_cmdshell", "/etc/passwd", "etc/shadow"} {
		page, err := svc.Threats(0, 1, keyword, "24h")
		if err != nil {
			fmt.Printf("search %-14q error: %v\n", keyword, err)
			continue
		}
		fmt.Printf("search %-14q total=%d\n", keyword, page.Total)
	}
}
