package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"elastic-ui/internal/config"
	"elastic-ui/internal/model"
)

type QuarantineState struct {
	Recovery    string `json:"recovery"`
	SavedPath   string `json:"savedPath"`
	RecoveredAt string `json:"recoveredAt"`
	LastError   string `json:"lastError"`
}

func quarantineKey(agent, path string) string {
	return agent + ":" + strings.ToLower(path)
}

func quarantineStatePath() string {
	dir, err := config.Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "quarantine-state.json")
}

func readQuarantineState() map[string]QuarantineState {
	states := map[string]QuarantineState{}
	data, err := os.ReadFile(quarantineStatePath())
	if err == nil {
		_ = json.Unmarshal(data, &states)
	}
	if states == nil {
		states = map[string]QuarantineState{}
	}
	return states
}

func (s *Service) recordQuarantineRecovery(agent, path, destination string, recoveryErr error) error {
	s.quarantineMu.Lock()
	defer s.quarantineMu.Unlock()
	states := readQuarantineState()
	key := quarantineKey(agent, path)
	state := states[key]
	if recoveryErr != nil {
		state.Recovery = "failed"
		state.LastError = recoveryErr.Error()
	} else {
		state.Recovery = "recovered"
		state.SavedPath = destination
		state.RecoveredAt = time.Now().UTC().Format(time.RFC3339)
		state.LastError = ""
	}
	states[key] = state
	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return err
	}
	pathName := quarantineStatePath()
	if pathName == "" {
		return errors.New("无法保存恢复记录")
	}
	if err := os.WriteFile(pathName+".tmp", data, 0o600); err != nil {
		return err
	}
	return os.Rename(pathName+".tmp", pathName)
}

// Inspect only metadata of a direct child of the local Endpoint quarantine folder.
func quarantineAvailability(path string) string {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	if len(volume) != 2 || volume[1] != ':' || !filepath.IsAbs(clean) || !strings.EqualFold(filepath.Dir(clean), volume+`\.equarantine`) {
		return "unknown"
	}
	// A missing or inaccessible drive is not evidence that a file was removed.
	if _, err := os.Stat(volume + `\`); err != nil {
		return "unavailable"
	}
	if parent, err := os.Lstat(filepath.Dir(clean)); err == nil && parent.Mode()&os.ModeSymlink != 0 {
		return "unknown"
	}
	info, err := os.Lstat(clean)
	if errors.Is(err, os.ErrNotExist) {
		return "missing"
	}
	if err != nil {
		return "unavailable"
	}
	if !info.Mode().IsRegular() {
		return "unknown"
	}
	return "available"
}

func (s *Service) enrichQuarantineFiles(agentID string, files []ScanFile) {
	s.quarantineMu.Lock()
	states := readQuarantineState()
	s.quarantineMu.Unlock()
	trusted, trustErr := s.TrustItems()
	for i := range files {
		f := &files[i]
		if f.QuarantinePath == "" {
			continue
		}
		f.Availability = quarantineAvailability(f.QuarantinePath)
		state := states[quarantineKey(agentID, f.QuarantinePath)]
		f.Recovery = state.Recovery
		f.SavedPath = state.SavedPath
		f.RecoveredAt = state.RecoveredAt
		f.RecoveryError = state.LastError
		f.Recoverable = f.Recoverable && f.Availability != "missing"
		f.TrustStatus = "none"
		if trustErr != nil {
			f.TrustStatus = "unknown"
			continue
		}
		for _, item := range trusted {
			if f.SHA256 != "" && item.MatchKey == trustRuleKey("hash", f.SHA256) {
				f.TrustStatus = "configured"
				break
			}
		}
	}
}

func (s *Service) QuarantineFile(alertID string) (*ScanFile, error) {
	s.mu.RLock()
	esc, agent := s.esc, s.agent
	s.mu.RUnlock()
	if esc == nil || agent == nil {
		return nil, errors.New("尚未绑定本机")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a, err := s.alertSource(ctx, esc, alertID)
	if err != nil {
		return nil, err
	}
	if a.Agent.ID != agent.ID {
		return nil, errors.New("只能查看本机隔离文件")
	}
	files := []ScanFile{scanFile(model.AlertHit{ID: alertID, Source: *a})}
	s.enrichQuarantineFiles(agent.ID, files)
	return &files[0], nil
}
