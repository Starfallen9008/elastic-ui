package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"elastic-ui/internal/model"
)

type EventItem struct {
	ID          string `json:"id"`
	Time        string `json:"time"`
	Category    string `json:"category"`
	Action      string `json:"action"`
	ProcessName string `json:"processName"`
	ProcessPID  int    `json:"processPid"`
	ProcessPath string `json:"processPath"`
	FilePath    string `json:"filePath"`
	DestIP      string `json:"destIp"`
	DestPort    int    `json:"destPort"`
	Protocol    string `json:"protocol"`
	User        string `json:"user"`
	Message     string `json:"message"`
	Raw         string `json:"raw"`
}

type EventPage struct {
	Items []EventItem `json:"items"`
	Total int64       `json:"total"`
}

func (s *Service) Events(offset, limit int, rangeKey, category, query string) (*EventPage, error) {
	s.mu.RLock()
	agent, esc := s.agent, s.esc
	s.mu.RUnlock()
	if agent == nil || esc == nil {
		return nil, errors.New("尚未绑定本机")
	}
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	switch rangeKey {
	case "1h", "24h", "7d", "30d":
	default:
		rangeKey = "24h"
	}
	filter := []any{
		map[string]any{"term": map[string]any{"agent.id": agent.ID}},
		map[string]any{"range": map[string]any{"@timestamp": map[string]any{"gte": "now-" + rangeKey}}},
	}
	if category != "" && category != "all" {
		filter = append(filter, map[string]any{"term": map[string]any{"event.category": category}})
	}
	boolQuery := map[string]any{"filter": filter}
	if strings.TrimSpace(query) != "" {
		boolQuery["must"] = []any{searchClause(
			query,
			[]string{"message", "process.command_line"},
			[]string{"destination.ip", "source.ip"},
			[]string{"process.name", "process.executable", "file.path", "file.name", "user.name"},
		)}
	}
	body := map[string]any{
		"from":             offset,
		"size":             limit,
		"track_total_hits": true,
		"sort":             []any{map[string]any{"@timestamp": map[string]any{"order": "desc"}}},
		"query":            map[string]any{"bool": boolQuery},
		"_source": []string{
			"@timestamp", "event.category", "event.action", "event.type", "event.code", "event.outcome",
			"process.name", "process.pid", "process.executable", "process.command_line",
			"file.path", "file.name",
			"destination.ip", "destination.port", "source.ip", "source.port",
			"network.protocol", "network.direction",
			"user.name", "message",
		},
	}
	var resp model.EventSearchResponse
	if err := esc.Search(context.Background(), "logs-endpoint.events.*", body, &resp); err != nil {
		return nil, err
	}
	page := &EventPage{Total: resp.Hits.Total.Value, Items: []EventItem{}}
	for _, hit := range resp.Hits.Hits {
		src := hit.Source
		raw, _ := json.Marshal(src)
		page.Items = append(page.Items, EventItem{
			ID:          hit.ID,
			Time:        src.Timestamp,
			Category:    strings.Join(src.Event.Category, ", "),
			Action:      strings.Join(src.Event.Action, ", "),
			ProcessName: src.Process.Name,
			ProcessPID:  src.Process.PID,
			ProcessPath: src.Process.Executable,
			FilePath:    src.File.Path,
			DestIP:      src.Destination.IP,
			DestPort:    src.Destination.Port,
			Protocol:    src.Network.Protocol,
			User:        src.User.Name,
			Message:     src.Message,
			Raw:         string(raw),
		})
	}
	return page, nil
}
