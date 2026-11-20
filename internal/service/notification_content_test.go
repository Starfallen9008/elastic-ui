package service

import "testing"

func TestNotificationContentShowsChineseTypeAndAffectedObject(t *testing.T) {
	tests := []struct {
		item                  Threat
		kind, outcome, object string
	}{
		{item: Threat{RuleName: "Malware Detection", Blocked: true, Action: "quarantine", FilePath: `C:\Downloads\bad.exe`, ProcessPath: `C:\Windows\explorer.exe`}, kind: "恶意软件防护", outcome: "已隔离威胁文件", object: `C:\Downloads\bad.exe`},
		{item: Threat{RuleName: "Ransomware Prevention", Blocked: true, Action: "kill_process", FilePath: `C:\Documents\report.docx`, ProcessPath: `C:\bad.exe`}, kind: "勒索软件防护", outcome: "已结束威胁进程", object: `C:\bad.exe`},
		{item: Threat{Category: "memory", Blocked: false, ProcessName: "app.exe"}, kind: "内存攻击防护", outcome: "检测到威胁", object: "app.exe"},
		{item: Threat{Category: "behavior", Blocked: true, Action: "kill_process", ProcessPath: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, RuleName: "PowerShell Script with Screen Capture Capability"}, kind: "恶意行为防护", outcome: "已结束威胁进程", object: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`},
	}
	for _, test := range tests {
		got := NotificationContent(test.item)
		if got.Kind != test.kind || got.Outcome != test.outcome || got.Object != test.object {
			t.Fatalf("unexpected content: %+v", got)
		}
	}
}
