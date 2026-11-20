package model

type EventSource struct {
	Timestamp string `json:"@timestamp"`
	Event     struct {
		Category StringOrArray `json:"category"`
		Action   StringOrArray `json:"action"`
		Type     StringOrArray `json:"type"`
		Code     string        `json:"code"`
		Outcome  string        `json:"outcome"`
	} `json:"event"`
	Process struct {
		Name        string `json:"name"`
		PID         int    `json:"pid"`
		Executable  string `json:"executable"`
		CommandLine string `json:"command_line"`
	} `json:"process"`
	File struct {
		Path string `json:"path"`
		Name string `json:"name"`
	} `json:"file"`
	Destination struct {
		IP   string `json:"ip"`
		Port int    `json:"port"`
	} `json:"destination"`
	Source struct {
		IP   string `json:"ip"`
		Port int    `json:"port"`
	} `json:"source"`
	Network struct {
		Protocol  string `json:"protocol"`
		Direction string `json:"direction"`
	} `json:"network"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	Message string `json:"message"`
}

type EventHit struct {
	ID     string      `json:"_id"`
	Source EventSource `json:"_source"`
}

type EventSearchResponse struct {
	Hits struct {
		Total struct {
			Value    int64  `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		Hits []EventHit `json:"hits"`
	} `json:"hits"`
}
