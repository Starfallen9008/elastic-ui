package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"elastic-ui/internal/model"
	"github.com/yeka/zip"
)

type ScanFile struct {
	ID             string `json:"id"`
	Time           string `json:"time"`
	Path           string `json:"path"`
	MalwareName    string `json:"malwareName"`
	SHA256         string `json:"sha256"`
	Outcome        string `json:"outcome"`
	Detail         string `json:"detail"`
	QuarantinePath string `json:"quarantinePath"`
	Recoverable    bool   `json:"recoverable"`
	Availability   string `json:"availability"`
	Recovery       string `json:"recovery"`
	SavedPath      string `json:"savedPath"`
	RecoveredAt    string `json:"recoveredAt"`
	RecoveryError  string `json:"recoveryError"`
	TrustStatus    string `json:"trustStatus"`
}

type ScanResults struct {
	Items  []ScanFile `json:"items"`
	Total  int64      `json:"total"`
	Notice string     `json:"notice"`
}

type QuarantineOperation struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

func (s *Service) TrustQuarantinedFile(alertID string) error {
	s.mu.RLock()
	esc, agent := s.esc, s.agent
	s.mu.RUnlock()
	if esc == nil || agent == nil {
		return errors.New("尚未绑定本机")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a, err := s.alertSource(ctx, esc, alertID)
	if err != nil {
		return err
	}
	if a.Agent.ID != agent.ID {
		return errors.New("只能加白本机隔离文件")
	}
	if !scanFile(model.AlertHit{Source: *a}).Recoverable {
		return errors.New("这条告警没有成功隔离的文件记录")
	}
	if a.File.Hash.SHA256 == "" {
		return errors.New("文件未记录 SHA256，无法加白")
	}
	items, err := s.TrustItems()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.MatchKey == trustRuleKey("hash", a.File.Hash.SHA256) {
			return nil
		}
	}
	return s.CreateTrust(TrustInput{Kind: "hash", Value: a.File.Hash.SHA256, Description: "隔离区：" + a.File.Path})
}

func (s *Service) SaveQuarantinedFiles(ids []string, directory string) []QuarantineOperation {
	results := []QuarantineOperation{}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		result := QuarantineOperation{ID: id}
		// Reserve a unique name so existing archives are never overwritten in a batch.
		file, err := os.CreateTemp(directory, "elastic-recovered-*.zip")
		if err == nil {
			result.Path = file.Name()
			err = file.Close()
			if err == nil {
				err = s.SaveQuarantinedFile(id, result.Path)
			}
			if err != nil {
				_ = os.Remove(result.Path)
			}
		}
		if err != nil {
			result.Error = err.Error()
			result.Path = ""
		}
		results = append(results, result)
	}
	return results
}

func scanFile(hit model.AlertHit) ScanFile {
	a := hit.Source
	f := ScanFile{ID: hit.ID, Time: a.Timestamp, Path: a.File.Path, MalwareName: a.MalwareName(), SHA256: a.File.Hash.SHA256, Outcome: "仅检测", Detail: a.File.Ext.QuarantineMessage, QuarantinePath: a.File.Ext.QuarantinePath}
	if q := a.File.Ext.QuarantineResult; q != nil {
		if *q {
			f.Outcome = "已隔离"
			f.Recoverable = f.QuarantinePath != ""
		} else {
			f.Outcome = "隔离失败"
		}
	} else {
		for _, r := range a.Responses {
			if r.Result != 0 {
				continue
			}
			switch r.Action.Action {
			case "quarantine", "quarantine_file":
				f.Outcome = "已隔离"
				f.Recoverable = f.QuarantinePath != ""
			case "delete_file":
				f.Outcome = "已删除"
			case "kill_process":
				if f.Outcome == "仅检测" {
					f.Outcome = "已终止进程"
				}
			}
		}
		if f.Outcome == "仅检测" {
			for _, t := range a.Event.Type {
				if t == "denied" {
					f.Outcome = "已阻止"
				}
			}
		}
	}
	return f
}

// Scan alerts do not carry a response-action ID. Restrict by host, demand,
// execution window and path; tell the caller that concurrent scans can overlap.
func (s *Service) ScanResults(taskID string, offset int) (*ScanResults, error) {
	s.mu.RLock()
	var task ScanTask
	for _, t := range s.scans {
		if t.ID == taskID {
			task = *t
			break
		}
	}
	s.mu.RUnlock()
	if task.ID == "" {
		return nil, errors.New("扫描记录不存在")
	}
	end := task.CompletedAt
	if end == "" {
		end = time.Now().UTC().Format(time.RFC3339Nano)
	}
	filters := []any{
		map[string]any{"term": map[string]any{"event.action": "demand"}},
		map[string]any{"range": map[string]any{"@timestamp": map[string]any{"gte": task.StartedAt, "lte": end}}},
		scanPathFilter(task.Path),
	}
	result, err := s.queryScanFiles(filters, offset)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	for _, t := range s.scans {
		if t.ID == taskID {
			t.Found = result.Total
			break
		}
	}
	s.mu.Unlock()
	_ = s.saveLocalState()
	return result, nil
}

func scanPathFilter(path string) map[string]any {
	path = strings.TrimRight(path, `\/`)
	escaped := strings.NewReplacer(`\`, `\\`, "*", `\*`, "?", `\?`).Replace(path)
	return map[string]any{"bool": map[string]any{"minimum_should_match": 1, "should": []any{
		map[string]any{"wildcard": map[string]any{"file.path": map[string]any{"value": escaped, "case_insensitive": true}}},
		map[string]any{"wildcard": map[string]any{"file.path": map[string]any{"value": escaped + `\\*`, "case_insensitive": true}}},
	}}}
}

func (s *Service) Quarantine(offset int) (*ScanResults, error) {
	r, err := s.queryScanFiles([]any{map[string]any{"term": map[string]any{"file.Ext.quarantine_result": true}}, map[string]any{"exists": map[string]any{"field": "file.Ext.quarantine_path"}}}, offset)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	agent := s.agent
	s.mu.RUnlock()
	if agent != nil {
		s.enrichQuarantineFiles(agent.ID, r.Items)
	}
	return r, nil
}

func (s *Service) queryScanFiles(filters []any, offset int) (*ScanResults, error) {
	s.mu.RLock()
	esc, agent := s.esc, s.agent
	s.mu.RUnlock()
	if esc == nil || agent == nil {
		return nil, errors.New("尚未绑定本机")
	}
	if offset < 0 || offset > 9900 {
		return nil, errors.New("分页超出范围")
	}
	filters = append(filters, map[string]any{"term": map[string]any{"agent.id": agent.ID}})
	body := map[string]any{"from": offset, "size": 50, "track_total_hits": true, "sort": []any{map[string]any{"@timestamp": "desc"}}, "query": map[string]any{"bool": map[string]any{"filter": filters}}, "_source": []string{"@timestamp", "event", "file", "Responses"}}
	quarantineOnly := false
	for _, filter := range filters {
		item, ok := filter.(map[string]any)
		if !ok {
			continue
		}
		term, ok := item["term"].(map[string]any)
		if ok && term["file.Ext.quarantine_result"] == true {
			quarantineOnly = true
		}
	}
	if quarantineOnly {
		body["collapse"] = map[string]any{"field": "file.Ext.quarantine_path"}
		body["aggs"] = map[string]any{"unique_files": map[string]any{"cardinality": map[string]any{
			"field": "file.Ext.quarantine_path", "precision_threshold": 40000,
		}}}
	}
	var resultEnvelope struct {
		model.SearchResponse
		Aggregations struct {
			UniqueFiles struct {
				Value int64 `json:"value"`
			} `json:"unique_files"`
		} `json:"aggregations"`
	}
	if err := esc.Search(context.Background(), "logs-endpoint.alerts-*", body, &resultEnvelope); err != nil {
		return nil, err
	}
	resp := resultEnvelope.SearchResponse
	result := &ScanResults{Items: []ScanFile{}, Total: resp.Hits.Total.Value}
	if quarantineOnly {
		result.Total = resultEnvelope.Aggregations.UniqueFiles.Value
	}
	for _, hit := range resp.Hits.Hits {
		result.Items = append(result.Items, scanFile(hit))
	}
	return result, nil
}

// The path and host are resolved from the original alert, never from UI input.
// Elastic get-file removes quarantine XOR; keep the recovered bytes inside ZIP.
func (s *Service) SaveQuarantinedFile(alertID, destination string) (err error) {
	s.mu.RLock()
	esc, kb, agent := s.esc, s.kb, s.agent
	s.mu.RUnlock()
	if esc == nil || kb == nil || agent == nil {
		return errors.New("尚未绑定本机")
	}
	defer func() {
		if err != nil {
			s.audit("recover_quarantine", alertID, "failed", err.Error())
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	a, err := s.alertSource(ctx, esc, alertID)
	if err != nil {
		return err
	}
	if a.Agent.ID != agent.ID {
		return errors.New("只能找回本机隔离文件")
	}
	if !scanFile(model.AlertHit{Source: *a}).Recoverable {
		return errors.New("这条告警没有成功隔离的文件记录")
	}
	defer func() {
		if stateErr := s.recordQuarantineRecovery(agent.ID, a.File.Ext.QuarantinePath, destination, err); stateErr != nil {
			s.audit("record_quarantine_recovery", alertID, "failed", stateErr.Error())
		}
	}()
	action, err := kb.GetFile(ctx, agent.ID, a.File.Ext.QuarantinePath)
	if err != nil {
		return err
	}
	final, err := waitAction(ctx, kb, action.ID, 300*time.Second)
	if err != nil {
		return err
	}
	if !final.WasSuccessful && final.Status != "successful" {
		return fmt.Errorf("找回失败（%s），文件可能已被清理或恢复", final.Status)
	}
	var output getFileOutput
	if err := json.Unmarshal(final.Outputs[agent.ID], &output); err != nil {
		return fmt.Errorf("无法读取找回结果: %w", err)
	}
	if output.Content.DownloadURI == "" {
		return errors.New("找回结果没有下载地址")
	}
	data, err := kb.Download(ctx, output.Content.DownloadURI)
	if err != nil {
		return err
	}
	if err := validateRecoveredArchive(data, a.File.Hash.SHA256); err != nil {
		return err
	}
	// A save dialog already confirms overwrite. Stage beside the destination so
	// failures never leave a partial archive or truncate an existing file.
	file, err := os.CreateTemp(filepath.Dir(destination), ".elastic-recovery-*.zip")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(tmp, destination); err != nil {
		return err
	}
	s.audit("recover_quarantine", a.File.Path, "success", destination)
	return nil
}

func validateRecoveredArchive(data []byte, expected string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("找回文件不是有效 ZIP: %w", err)
	}
	count := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() || f.Name == "upload.info" {
			continue
		}
		count++
		if !f.IsEncrypted() {
			return errors.New("找回包未加密，未保存")
		}
		f.SetPassword("elastic")
		stream, err := f.Open()
		if err != nil {
			return err
		}
		h := sha256.New()
		n, copyErr := io.Copy(h, io.LimitReader(stream, 101<<20))
		stream.Close()
		if copyErr != nil {
			return copyErr
		}
		if n > 100<<20 {
			return errors.New("找回文件超过 100 MB")
		}
		if expected != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expected) {
			return errors.New("找回内容的 SHA256 与告警不一致，未保存")
		}
	}
	if count != 1 {
		return errors.New("找回包中的文件数量异常，未保存")
	}
	return nil
}
