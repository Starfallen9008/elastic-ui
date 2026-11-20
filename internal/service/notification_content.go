package service

import "strings"

type AlertNotification struct {
	ID          string
	Kind        string
	Outcome     string
	Object      string
	Rule        string
	MalwareName string
}

func NotificationContent(item Threat) AlertNotification {
	kind := "威胁检测"
	category := strings.ToLower(item.DetectionType + " " + item.Category + " " + item.RuleName)
	switch {
	case strings.Contains(category, "ransom"):
		kind = "勒索软件防护"
	case strings.Contains(category, "memory"):
		kind = "内存攻击防护"
	case strings.Contains(category, "behavior"):
		kind = "恶意行为防护"
	case strings.Contains(category, "malware"):
		kind = "恶意软件防护"
	case strings.Contains(category, "device"):
		kind = "设备访问控制"
	case strings.Contains(category, "credential"):
		kind = "凭据访问防护"
	case strings.Contains(category, "network"):
		kind = "网络威胁防护"
	}
	outcome := "检测到威胁"
	action := strings.ToLower(item.Action)
	if item.Blocked {
		outcome = "已拦截威胁"
		switch {
		case strings.Contains(action, "kill"), strings.Contains(action, "terminat"):
			outcome = "已结束威胁进程"
		case strings.Contains(action, "quarantin"):
			outcome = "已隔离威胁文件"
		case strings.Contains(action, "suspend"):
			outcome = "已暂停威胁进程"
		case strings.Contains(action, "rollback"):
			outcome = "已回滚威胁操作"
		}
	}
	object := item.FilePath
	if strings.Contains(action, "kill") || strings.Contains(action, "terminat") || object == "" {
		object = item.ProcessPath
	}
	if object == "" {
		object = item.ProcessName
	}
	if object == "" {
		object = "告警未提供对象路径，请查看详情"
	}
	return AlertNotification{
		ID: item.ID, Kind: kind, Outcome: outcome, Object: object,
		Rule: item.RuleName, MalwareName: item.MalwareName,
	}
}
