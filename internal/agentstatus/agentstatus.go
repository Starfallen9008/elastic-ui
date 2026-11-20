package agentstatus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Status struct {
	State              string `json:"state"`
	Message            string `json:"message"`
	Version            string `json:"version"`
	AgentID            string `json:"agentId"`
	ComponentsTotal    int    `json:"componentsTotal"`
	ComponentsHealthy  int    `json:"componentsHealthy"`
	ComponentsDegraded int    `json:"componentsDegraded"`
	ComponentsFailed   int    `json:"componentsFailed"`
	CheckedAt          string `json:"checkedAt"`
	Error              string `json:"error,omitempty"`
	Raw                string `json:"raw,omitempty"`
}

type jsonStatus struct {
	Info struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"info"`
	State      int    `json:"state"`
	Message    string `json:"message"`
	Components []struct {
		State   int    `json:"state"`
		Message string `json:"message"`
	} `json:"components"`
}

func Check() Status {
	checkedAt := time.Now().Format(time.RFC3339)
	result := Status{State: "UNAVAILABLE", CheckedAt: checkedAt}

	exe, err := findBinary()
	if err != nil {
		result.Error = err.Error()
		result.Message = err.Error()
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe, "status", "--output", "json")
	cmd.Dir = filepath.Dir(exe)
	hideWindow(cmd)
	output, runErr := cmd.CombinedOutput()
	raw := strings.TrimSpace(string(output))
	if len(raw) > 16384 {
		raw = raw[:16384]
	}
	result.Raw = raw

	if parsed, ok := parseJSON(raw); ok {
		parsed.CheckedAt = checkedAt
		parsed.Raw = raw
		if runErr != nil {
			parsed.Error = runErr.Error()
		}
		return parsed
	}
	if runErr == nil {
		if parsed, ok := parseHuman(raw); ok {
			parsed.CheckedAt = checkedAt
			parsed.Raw = raw
			return parsed
		}
	}

	result.Error = summarizeError(raw, runErr)
	result.Message = result.Error
	return result
}

func findBinary() (string, error) {
	root := `C:\Program Files\Elastic\Agent`
	candidates := []string{}

	if info, err := os.Stat(filepath.Join(root, "elastic-agent.exe")); err == nil && info.Size() > 0 {
		candidates = append(candidates, filepath.Join(root, "elastic-agent.exe"))
	}

	matches, _ := filepath.Glob(filepath.Join(root, "data", "elastic-agent-*", "elastic-agent.exe"))
	sort.Slice(matches, func(i, j int) bool {
		left, leftErr := os.Stat(matches[i])
		right, rightErr := os.Stat(matches[j])
		if leftErr == nil && rightErr == nil {
			return left.ModTime().After(right.ModTime())
		}
		return matches[i] > matches[j]
	})
	candidates = append(candidates, matches...)

	if path, err := exec.LookPath("elastic-agent.exe"); err == nil && path != "" {
		candidates = append(candidates, path)
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("未找到 elastic-agent.exe，请确认 Elastic Agent 已安装")
}

func parseJSON(raw string) (Status, bool) {
	start := strings.Index(raw, "{")
	if start < 0 {
		return Status{}, false
	}
	var payload jsonStatus
	if err := json.Unmarshal([]byte(raw[start:]), &payload); err != nil {
		return Status{}, false
	}
	status := Status{
		State:           stateName(payload.State),
		Message:         payload.Message,
		Version:         payload.Info.Version,
		AgentID:         payload.Info.ID,
		ComponentsTotal: len(payload.Components),
	}
	for _, component := range payload.Components {
		switch component.State {
		case 2:
			status.ComponentsHealthy++
		case 3:
			status.ComponentsDegraded++
		case 4:
			status.ComponentsFailed++
		}
	}
	if status.Message == "" {
		status.Message = status.State
	}
	return status, true
}

func parseHuman(raw string) (Status, bool) {
	status := Status{State: "UNAVAILABLE"}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Status:"):
			status.State = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(line, "Status:")))
		case strings.HasPrefix(line, "Message:"):
			status.Message = strings.TrimSpace(strings.TrimPrefix(line, "Message:"))
		case strings.HasPrefix(line, "Version:"):
			status.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
	}
	if status.State == "UNAVAILABLE" {
		return status, false
	}
	return status, true
}

func stateName(state int) string {
	switch state {
	case 0:
		return "STARTING"
	case 1:
		return "CONFIGURING"
	case 2:
		return "HEALTHY"
	case 3:
		return "DEGRADED"
	case 4:
		return "FAILED"
	case 5:
		return "STOPPING"
	case 6:
		return "UPGRADING"
	case 7:
		return "ROLLBACK"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", state)
	}
}

func summarizeError(raw string, runErr error) string {
	message := strings.TrimSpace(raw)
	if message == "" && runErr != nil {
		message = runErr.Error()
	}
	if strings.Contains(message, "Access is denied") || strings.Contains(message, "Access denied") {
		return "需要管理员权限读取 Elastic Agent 状态，请以管理员身份启动本程序"
	}
	if len(message) > 300 {
		message = message[:300]
	}
	return message
}
