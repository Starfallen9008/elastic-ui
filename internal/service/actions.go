package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeka/zip"

	"elastic-ui/internal/config"
	"elastic-ui/internal/kibana"
)

type ProcessInfo struct {
	PID      int    `json:"pid"`
	EntityID string `json:"entityId"`
	User     string `json:"user"`
	Command  string `json:"command"`
}

type AuditEntry struct {
	Time   string `json:"time"`
	Action string `json:"action"`
	Target string `json:"target"`
	Result string `json:"result"`
	Detail string `json:"detail,omitempty"`
}

type RetrievedFile struct {
	Path        string `json:"path"`
	FileName    string `json:"fileName"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	SavedPath   string `json:"savedPath"`
	ZipPath     string `json:"zipPath"`
	SavedDir    string `json:"savedDir"`
	DownloadURI string `json:"downloadUri"`
	IsDirectory bool   `json:"isDirectory"`
}

type getFileOutput struct {
	Type    string `json:"type"`
	Content struct {
		Code     string `json:"code"`
		ZipSize  int64  `json:"zip_size"`
		Contents []struct {
			Path      string `json:"path"`
			SHA256    string `json:"sha256"`
			Size      int64  `json:"size"`
			FileName  string `json:"file_name"`
			Type      string `json:"type"`
			Directory string `json:"directory"`
		} `json:"contents"`
		DownloadURI string `json:"downloadUri"`
	} `json:"content"`
}

type procsOutput struct {
	Type    string `json:"type"`
	Content struct {
		Entries []struct {
			PID      int    `json:"pid"`
			EntityID string `json:"entity_id"`
			User     string `json:"user"`
			Command  string `json:"command"`
		} `json:"entries"`
	} `json:"content"`
}

func (s *Service) Processes(force bool) ([]ProcessInfo, error) {
	if !force {
		s.mu.RLock()
		cache := s.procsCache
		cacheAt := s.procsCacheAt
		s.mu.RUnlock()
		if len(cache) > 0 && time.Since(cacheAt) < 60*time.Second {
			return append([]ProcessInfo{}, cache...), nil
		}
	}
	s.mu.RLock()
	agent, kb := s.agent, s.kb
	s.mu.RUnlock()
	if agent == nil || kb == nil {
		return nil, errors.New("尚未绑定本机")
	}
	ctx := context.Background()
	action, err := kb.RunningProcesses(ctx, agent.ID)
	if err != nil {
		return nil, err
	}
	final, err := waitAction(ctx, kb, action.ID, 300*time.Second)
	if err != nil {
		return nil, err
	}
	if final.Status != "successful" {
		return nil, fmt.Errorf("进程列表读取失败: %s", final.Status)
	}
	raw, ok := final.Outputs[agent.ID]
	if !ok {
		return nil, errors.New("动作没有返回进程列表")
	}
	var out procsOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	procs := make([]ProcessInfo, 0, len(out.Content.Entries))
	for _, entry := range out.Content.Entries {
		procs = append(procs, ProcessInfo{
			PID:      entry.PID,
			EntityID: entry.EntityID,
			User:     entry.User,
			Command:  entry.Command,
		})
	}
	s.mu.Lock()
	s.procsCache = procs
	s.procsCacheAt = time.Now()
	s.mu.Unlock()
	return append([]ProcessInfo{}, procs...), nil
}

func (s *Service) KillProcess(pid int, entityID string) error {
	return s.processAction("kill_process", pid, entityID)
}

func (s *Service) SuspendProcess(pid int, entityID string) error {
	return s.processAction("suspend_process", pid, entityID)
}

func (s *Service) processAction(command string, pid int, entityID string) error {
	s.mu.RLock()
	agent, kb := s.agent, s.kb
	s.mu.RUnlock()
	if agent == nil || kb == nil {
		return errors.New("尚未绑定本机")
	}
	if pid <= 0 && entityID == "" {
		return errors.New("缺少 PID 或 entity_id")
	}
	target := fmt.Sprintf("PID %d", pid)
	if entityID != "" {
		target = entityID
	}
	ctx := context.Background()
	var action *kibana.Action
	var err error
	switch command {
	case "kill_process":
		action, err = kb.KillProcess(ctx, agent.ID, pid, entityID)
	case "suspend_process":
		action, err = kb.SuspendProcess(ctx, agent.ID, pid, entityID)
	default:
		return fmt.Errorf("不支持的动作 %s", command)
	}
	if err != nil {
		s.audit(command, target, "failed", err.Error())
		return err
	}
	final, err := waitAction(ctx, kb, action.ID, 300*time.Second)
	if err != nil {
		s.audit(command, target, "failed", err.Error())
		return err
	}
	s.audit(command, target, final.Status, action.ID)
	if final.Status != "successful" {
		return fmt.Errorf("%s 未成功: %s", command, final.Status)
	}
	return nil
}

func (s *Service) GetFile(path string) (*RetrievedFile, error) {
	s.mu.RLock()
	agent, kb := s.agent, s.kb
	s.mu.RUnlock()
	if agent == nil || kb == nil {
		return nil, errors.New("尚未绑定本机")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("文件路径不能为空")
	}
	if strings.HasPrefix(path, `\\`) {
		return nil, errors.New("不支持网络路径")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, fmt.Errorf("文件不可访问: %w", err)
	}
	if info.IsDir() {
		return nil, errors.New("Elastic Defend 不支持取回目录，请指定具体文件")
	}
	ctx := context.Background()
	action, err := kb.GetFile(ctx, agent.ID, absolute)
	if err != nil {
		s.audit("get_file", absolute, "failed", err.Error())
		return nil, err
	}
	final, err := waitAction(ctx, kb, action.ID, 300*time.Second)
	if err != nil {
		s.audit("get_file", absolute, "failed", err.Error())
		return nil, err
	}
	if final.Status != "successful" {
		s.audit("get_file", absolute, final.Status, action.ID)
		return nil, fmt.Errorf("取回文件失败: %s", final.Status)
	}
	raw, ok := final.Outputs[agent.ID]
	if !ok {
		return nil, errors.New("动作没有返回文件信息")
	}
	var out getFileOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out.Content.Contents) == 0 || out.Content.DownloadURI == "" {
		return nil, errors.New("动作没有返回可下载文件")
	}
	data, err := kb.Download(ctx, out.Content.DownloadURI)
	if err != nil {
		s.audit("get_file", absolute, "failed", err.Error())
		return nil, err
	}
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}
	saveDir := filepath.Join(dir, "files", time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(saveDir, 0o700); err != nil {
		return nil, err
	}
	zipPath := filepath.Join(saveDir, "download.zip")
	if err := os.WriteFile(zipPath, data, 0o600); err != nil {
		return nil, err
	}
	extracted, err := extractZip(data, saveDir)
	if err != nil {
		return nil, err
	}
	meta := out.Content.Contents[0]
	isDir := meta.Type == "directory"
	savedPath := extracted
	if isDir {
		savedPath = saveDir
	}
	result := &RetrievedFile{
		Path:        meta.Path,
		FileName:    meta.FileName,
		Size:        meta.Size,
		SHA256:      meta.SHA256,
		SavedPath:   savedPath,
		ZipPath:     zipPath,
		SavedDir:    saveDir,
		DownloadURI: out.Content.DownloadURI,
		IsDirectory: isDir,
	}
	s.audit("get_file", absolute, "success", savedPath)
	return result, nil
}

func extractZip(data []byte, dir string) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	first := ""
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name == "upload.info" {
			continue
		}
		file.SetPassword("elastic")
		name := filepath.Clean(filepath.FromSlash(file.Name))
		if name == "." || filepath.IsAbs(name) || strings.HasPrefix(name, "..") {
			continue
		}
		dest := filepath.Join(dir, name)
		if !strings.HasPrefix(dest, dir+string(os.PathSeparator)) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return "", err
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		dst, err := os.Create(dest)
		if err != nil {
			_ = src.Close()
			return "", err
		}
		_, copyErr := io.Copy(dst, src)
		_ = dst.Close()
		_ = src.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if first == "" {
			first = dest
		}
	}
	if first == "" {
		return "", errors.New("下载包中没有文件")
	}
	return first, nil
}

func (s *Service) countAlertsBetween(ctx context.Context, start, end string) int64 {
	s.mu.RLock()
	esc, agent := s.esc, s.agent
	s.mu.RUnlock()
	if esc == nil || agent == nil {
		return 0
	}
	body := map[string]any{
		"size": 0,
		"query": map[string]any{"bool": map[string]any{"filter": []any{
			map[string]any{"term": map[string]any{"agent.id": agent.ID}},
			map[string]any{"range": map[string]any{"@timestamp": map[string]any{"gte": start, "lte": end}}},
		}}},
	}
	var resp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
		} `json:"hits"`
	}
	if err := esc.Search(ctx, "logs-endpoint.alerts-default", body, &resp); err != nil {
		return 0
	}
	return resp.Hits.Total.Value
}

func (s *Service) ClearCache() error {
	s.mu.Lock()
	s.scans = nil
	s.read = map[string]bool{}
	s.procsCache = nil
	s.procsCacheAt = time.Time{}
	s.mu.Unlock()
	return s.saveLocalState()
}

func (s *Service) WriteDiagnostics(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	write := func(name string, data []byte) error {
		if len(data) == 0 {
			data = []byte("none\n")
		}
		entry, err := writer.Create(name)
		if err != nil {
			return err
		}
		_, err = entry.Write(data)
		return err
	}

	settings := s.Settings()
	dir, _ := config.Dir()
	info := fmt.Sprintf(
		"Elastic Defend 本机控制台 诊断包\n生成时间: %s\nKibana: %s\nElasticsearch: %s\n主机: %s\nAgent: %s\n策略: %s\n",
		time.Now().Format(time.RFC3339),
		settings.Profile.KibanaURL,
		settings.Profile.ESURL,
		settings.Profile.HostName,
		settings.Profile.AgentID,
		settings.Profile.PolicyName,
	)
	_ = write("info.txt", []byte(info))
	if data, err := json.MarshalIndent(settings, "", "  "); err == nil {
		_ = write("config.json", data)
	}
	_ = write("app.log", tailFile(filepath.Join(dir, "app.log"), 1<<20))
	_ = write("audit.log", tailFile(filepath.Join(dir, "audit.log"), 1<<20))
	if status, err := s.Status(); err == nil {
		if data, err := json.MarshalIndent(status, "", "  "); err == nil {
			_ = write("status.json", data)
		}
	} else {
		_ = write("status-error.txt", []byte(err.Error()))
	}
	return nil
}

func tailFile(path string, max int64) []byte {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil
	}
	offset := info.Size() - max
	if offset < 0 {
		offset = 0
	}
	_, _ = file.Seek(offset, io.SeekStart)
	data, _ := io.ReadAll(file)
	return data
}

func (s *Service) GetAudit(limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "audit.log"))
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEntry{}, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	out := make([]AuditEntry, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		var entry AuditEntry
		if json.Unmarshal([]byte(lines[i]), &entry) == nil {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (s *Service) audit(action, target, result, detail string) {
	dir, err := config.Dir()
	if err != nil {
		return
	}
	entry := AuditEntry{
		Time:   time.Now().Format(time.RFC3339),
		Action: action,
		Target: target,
		Result: result,
		Detail: detail,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(dir, "audit.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(data, '\n'))
}

func waitAction(ctx context.Context, kb *kibana.Client, id string, timeout time.Duration) (*kibana.Action, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		action, err := kb.GetAction(ctx, id)
		if err != nil {
			return nil, err
		}
		if action.Status == "successful" || action.Status == "failed" || action.IsExpired {
			return action, nil
		}
	}
	return nil, errors.New("动作等待超时")
}
