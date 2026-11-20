package model

import (
	"encoding/json"
	"strings"
)

type Alert struct {
	Timestamp string `json:"@timestamp"`
	Message   string `json:"message"`
	Rule      struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Ruleset     string `json:"ruleset"`
		Description string `json:"description"`
	} `json:"rule"`
	Event struct {
		Dataset   string   `json:"dataset"`
		Action    string   `json:"action"`
		Code      string   `json:"code"`
		Category  []string `json:"category"`
		Severity  float64  `json:"severity"`
		RiskScore float64  `json:"risk_score"`
		Kind      string   `json:"kind"`
		Type      []string `json:"type"`
	} `json:"event"`
	DataStream struct {
		Dataset string `json:"dataset"`
	} `json:"data_stream"`
	Process struct {
		Name        string   `json:"name"`
		PID         int      `json:"pid"`
		EntityID    string   `json:"entity_id"`
		Executable  string   `json:"executable"`
		CommandLine string   `json:"command_line"`
		Args        []string `json:"args"`
		WorkingDir  string   `json:"working_directory"`
		Hash        struct {
			SHA256 string `json:"sha256"`
		} `json:"hash"`
		Parent struct {
			Name        string `json:"name"`
			PID         int    `json:"pid"`
			EntityID    string `json:"entity_id"`
			Executable  string `json:"executable"`
			CommandLine string `json:"command_line"`
		} `json:"parent"`
		Ext struct {
			Ancestry              []string        `json:"ancestry"`
			MalwareClassification json.RawMessage `json:"malware_classification"`
			MalwareSignature      json.RawMessage `json:"malware_signature"`
		} `json:"Ext"`
	} `json:"process"`
	File struct {
		Path string `json:"path"`
		Name string `json:"name"`
		Ext  struct {
			QuarantinePath        string          `json:"quarantine_path"`
			QuarantineResult      *bool           `json:"quarantine_result"`
			QuarantineMessage     string          `json:"quarantine_message"`
			MalwareClassification json.RawMessage `json:"malware_classification"`
			MalwareSignature      json.RawMessage `json:"malware_signature"`
		} `json:"Ext"`
		Hash struct {
			SHA256 string `json:"sha256"`
		} `json:"hash"`
	} `json:"file"`
	Host struct {
		Hostname string `json:"hostname"`
		OS       struct {
			Full string `json:"full"`
		} `json:"os"`
	} `json:"host"`
	Agent struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"agent"`
	Responses []struct {
		Timestamp string `json:"@timestamp"`
		Action    struct {
			Action string `json:"action"`
			Field  string `json:"field"`
			State  int    `json:"state"`
		} `json:"action"`
		Message string `json:"message"`
		Result  int    `json:"result"`
		Process struct {
			Name string `json:"name"`
			PID  int    `json:"pid"`
		} `json:"process"`
	} `json:"Responses"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	Threat []struct {
		Framework string `json:"framework"`
		Tactic    struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Reference string `json:"reference"`
		} `json:"tactic"`
		Technique []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Reference    string `json:"reference"`
			Subtechnique []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Reference string `json:"reference"`
			} `json:"subtechnique"`
		} `json:"technique"`
	} `json:"threat"`
}

func (a Alert) MalwareName() string {
	for _, raw := range []json.RawMessage{
		a.File.Ext.MalwareSignature, a.Process.Ext.MalwareSignature,
		a.File.Ext.MalwareClassification, a.Process.Ext.MalwareClassification,
	} {
		var value any
		if json.Unmarshal(raw, &value) != nil {
			continue
		}
		if name := malwareLabel(value); name != "" {
			return name
		}
	}
	return ""
}

func malwareLabel(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		names := []string{}
		for _, item := range v {
			if name := malwareLabel(item); name != "" {
				names = append(names, name)
			}
		}
		return strings.Join(names, ", ")
	case map[string]any:
		for _, key := range []string{"threat_name", "name", "all_names", "primary", "signature", "classification"} {
			if name := malwareLabel(v[key]); name != "" {
				return name
			}
		}
	}
	return ""
}

func (a Alert) ActionTaken() string {
	if a.File.Ext.QuarantineResult != nil && *a.File.Ext.QuarantineResult {
		return "quarantine"
	}
	for _, r := range a.Responses {
		if r.Action.Action != "" && (r.Result == 0 || r.Message == "Success") {
			return r.Action.Action
		}
	}
	return ""
}

func (a Alert) Blocked() bool {
	return a.ActionTaken() != ""
}

type AlertHit struct {
	ID     string `json:"_id"`
	Source Alert  `json:"_source"`
}

type SearchHits struct {
	Total struct {
		Value    int64  `json:"value"`
		Relation string `json:"relation"`
	} `json:"total"`
	Hits []AlertHit `json:"hits"`
}

type SearchResponse struct {
	Hits SearchHits `json:"hits"`
}
