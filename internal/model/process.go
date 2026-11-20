package model

import "encoding/json"

// StringOrArray 处理 event.action 这类既可能是字符串也可能是数组的字段。
type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*s = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

type ProcessEvent struct {
	Timestamp string `json:"@timestamp"`
	Event     struct {
		Action StringOrArray `json:"action"`
	} `json:"event"`
	Process struct {
		Name        string   `json:"name"`
		PID         int      `json:"pid"`
		EntityID    string   `json:"entity_id"`
		Executable  string   `json:"executable"`
		CommandLine string   `json:"command_line"`
		WorkingDir  string   `json:"working_directory"`
		Args        []string `json:"args"`
		Start       string   `json:"start"`
		End         string   `json:"end"`
		Parent      struct {
			EntityID string `json:"entity_id"`
		} `json:"parent"`
	} `json:"process"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
}

type ProcessEventHit struct {
	ID     string       `json:"_id"`
	Source ProcessEvent `json:"_source"`
}

type ProcessSearchResponse struct {
	Hits struct {
		Total struct {
			Value    int64  `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		Hits []ProcessEventHit `json:"hits"`
	} `json:"hits"`
}
