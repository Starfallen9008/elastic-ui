package kibana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type EndpointMetadata struct {
	Timestamp string `json:"@timestamp"`
	Endpoint  struct {
		Capabilities []string `json:"capabilities"`
		State        struct {
			Isolation        bool `json:"isolation"`
			TamperProtection bool `json:"tamper_protection"`
			Orphaned         bool `json:"orphaned"`
		} `json:"state"`
		Policy struct {
			Applied struct {
				Name                  string `json:"name"`
				ID                    string `json:"id"`
				Version               string `json:"version"`
				EndpointPolicyVersion string `json:"endpoint_policy_version"`
				Status                string `json:"status"`
			} `json:"applied"`
		} `json:"policy"`
		Status string `json:"status"`
	} `json:"Endpoint"`
	Agent struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Type    string `json:"type"`
	} `json:"agent"`
	Host struct {
		Hostname string `json:"hostname"`
		OS       struct {
			Name     string `json:"name"`
			Full     string `json:"full"`
			Platform string `json:"platform"`
		} `json:"os"`
		IP []string `json:"ip"`
	} `json:"host"`
	HostStatus string `json:"host_status"`
	PolicyInfo struct {
		Agent struct {
			Applied struct {
				Revision int    `json:"revision"`
				ID       string `json:"id"`
			} `json:"applied"`
			Configured struct {
				Revision int    `json:"revision"`
				ID       string `json:"id"`
			} `json:"configured"`
		} `json:"agent"`
		Endpoint struct {
			Revision int    `json:"revision"`
			ID       string `json:"id"`
		} `json:"endpoint"`
	} `json:"policy_info"`
	LastCheckin string `json:"last_checkin"`
}

// Endpoint versions have been returned as both JSON numbers and strings.
func (m *EndpointMetadata) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	endpoint, _ := raw["Endpoint"].(map[string]any)
	policy, _ := endpoint["policy"].(map[string]any)
	applied, _ := policy["applied"].(map[string]any)
	for _, key := range []string{"version", "endpoint_policy_version"} {
		if value, ok := applied[key].(float64); ok {
			applied[key] = fmt.Sprintf("%.0f", value)
		}
	}
	normalized, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	type plain EndpointMetadata
	return json.Unmarshal(normalized, (*plain)(m))
}

func (c *Client) GetEndpointMetadata(ctx context.Context, agentID string) (*EndpointMetadata, error) {
	var out struct {
		Metadata EndpointMetadata `json:"metadata"`
	}
	path := "/api/endpoint/metadata/" + url.PathEscape(agentID)
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out.Metadata, nil
}

type PackagePolicy struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	PolicyID string `json:"policy_id"`
	Revision int    `json:"revision"`
	Enabled  bool   `json:"enabled"`
	Package  struct {
		Name    string `json:"name"`
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"package"`
	Inputs []struct {
		Type    string `json:"type"`
		Enabled bool   `json:"enabled"`
		Config  struct {
			Policy struct {
				Value json.RawMessage `json:"value"`
			} `json:"policy"`
		} `json:"config"`
	} `json:"inputs"`
}

func (c *Client) GetPackagePolicy(ctx context.Context, id string) (*PackagePolicy, error) {
	var out struct {
		Item PackagePolicy `json:"item"`
	}
	path := "/api/fleet/package_policies/" + url.PathEscape(id)
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out.Item, nil
}

type ProtectionMode struct {
	Mode        string `json:"mode"`
	Blocklist   bool   `json:"blocklist"`
	OnWriteScan bool   `json:"on_write_scan"`
	Supported   bool   `json:"supported"`
}

type OSProtection struct {
	Popup map[string]struct {
		Enabled bool `json:"enabled"`
	} `json:"popup"`
	Events     map[string]bool `json:"events"`
	Malware    ProtectionMode  `json:"malware"`
	Ransomware struct {
		Mode      string `json:"mode"`
		Supported bool   `json:"supported"`
	} `json:"ransomware"`
	MemoryProtection struct {
		Mode      string `json:"mode"`
		Supported bool   `json:"supported"`
	} `json:"memory_protection"`
	BehaviorProtection struct {
		Mode              string `json:"mode"`
		ReputationService bool   `json:"reputation_service"`
		Supported         bool   `json:"supported"`
	} `json:"behavior_protection"`
	DeviceControl struct {
		Enabled    bool   `json:"enabled"`
		USBStorage string `json:"usb_storage"`
	} `json:"device_control"`
	AntivirusRegistration struct {
		Enabled bool   `json:"enabled"`
		Mode    string `json:"mode"`
	} `json:"antivirus_registration"`
	AttackSurfaceReduction struct {
		CredentialHardening struct {
			Enabled bool `json:"enabled"`
		} `json:"credential_hardening"`
	} `json:"attack_surface_reduction"`
	Logging struct {
		File string `json:"file"`
	} `json:"logging"`
}

type PolicyValue struct {
	GlobalManifestVersion string       `json:"global_manifest_version"`
	Windows               OSProtection `json:"windows"`
	Mac                   OSProtection `json:"mac"`
	Linux                 OSProtection `json:"linux"`
	Meta                  struct {
		License string `json:"license"`
	} `json:"meta"`
}

func (p *PackagePolicy) PolicyValue() (*PolicyValue, error) {
	if len(p.Inputs) == 0 {
		return nil, fmt.Errorf("package policy has no inputs")
	}
	var value PolicyValue
	if err := json.Unmarshal(p.Inputs[0].Config.Policy.Value, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

type ActionRequest struct {
	EndpointIDs []string       `json:"endpoint_ids"`
	AgentType   string         `json:"agent_type"`
	Comment     string         `json:"comment,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type Action struct {
	ID            string                     `json:"id"`
	AgentType     string                     `json:"agentType"`
	Agents        []string                   `json:"agents"`
	Command       string                     `json:"command"`
	Status        string                     `json:"status"`
	IsCompleted   bool                       `json:"isCompleted"`
	WasSuccessful bool                       `json:"wasSuccessful"`
	WasCanceled   bool                       `json:"wasCanceled"`
	IsExpired     bool                       `json:"isExpired"`
	StartedAt     string                     `json:"startedAt"`
	CompletedAt   string                     `json:"completedAt"`
	CreatedBy     string                     `json:"createdBy"`
	Parameters    map[string]any             `json:"parameters"`
	Outputs       map[string]json.RawMessage `json:"outputs"`
	Errors        []string                   `json:"errors"`
}

type actionEnvelope struct {
	Data Action `json:"data"`
}

func decodeAction(data []byte) (*Action, error) {
	var env actionEnvelope
	if err := json.Unmarshal(data, &env); err == nil && env.Data.ID != "" {
		return &env.Data, nil
	}
	var action Action
	if err := json.Unmarshal(data, &action); err != nil {
		return nil, err
	}
	return &action, nil
}

func (c *Client) SendAction(ctx context.Context, command string, req ActionRequest) (*Action, error) {
	if req.AgentType == "" {
		req.AgentType = "endpoint"
	}
	var raw json.RawMessage
	path := "/api/endpoint/action/" + url.PathEscape(command)
	if err := c.Post(ctx, path, req, &raw); err != nil {
		return nil, err
	}
	return decodeAction(raw)
}

func (c *Client) GetAction(ctx context.Context, id string) (*Action, error) {
	var raw json.RawMessage
	path := "/api/endpoint/action/" + url.PathEscape(id)
	if err := c.Get(ctx, path, &raw); err != nil {
		return nil, err
	}
	return decodeAction(raw)
}

type ActionsResponse struct {
	Data  []Action `json:"data"`
	Total int      `json:"total"`
}

func (c *Client) ListActions(ctx context.Context, agentID string, pageSize int) (*ActionsResponse, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	var out ActionsResponse
	path := fmt.Sprintf("/api/endpoint/action?page=1&pageSize=%d&agentIds=%s", pageSize, url.QueryEscape(agentID))
	if err := c.Get(ctx, path, &out); err != nil {
		// 动作索引尚未创建时 Kibana 返回 404 index_not_found_exception
		if IsNotFound(err) {
			return &ActionsResponse{}, nil
		}
		return nil, err
	}
	return &out, nil
}

func (c *Client) Scan(ctx context.Context, agentID, path, comment string) (*Action, error) {
	return c.SendAction(ctx, "scan", ActionRequest{
		EndpointIDs: []string{agentID},
		Comment:     comment,
		Parameters:  map[string]any{"path": path},
	})
}

func (c *Client) Isolate(ctx context.Context, agentID, comment string) (*Action, error) {
	return c.SendAction(ctx, "isolate", ActionRequest{
		EndpointIDs: []string{agentID},
		Comment:     comment,
	})
}

func (c *Client) RunningProcesses(ctx context.Context, agentID string) (*Action, error) {
	return c.SendAction(ctx, "running_procs", ActionRequest{
		EndpointIDs: []string{agentID},
	})
}

func (c *Client) KillProcess(ctx context.Context, agentID string, pid int, entityID string) (*Action, error) {
	params := map[string]any{}
	if pid > 0 {
		params["pid"] = pid
	}
	if entityID != "" {
		params["entity_id"] = entityID
	}
	return c.SendAction(ctx, "kill_process", ActionRequest{
		EndpointIDs: []string{agentID},
		Parameters:  params,
	})
}

func (c *Client) SuspendProcess(ctx context.Context, agentID string, pid int, entityID string) (*Action, error) {
	params := map[string]any{}
	if pid > 0 {
		params["pid"] = pid
	}
	if entityID != "" {
		params["entity_id"] = entityID
	}
	return c.SendAction(ctx, "suspend_process", ActionRequest{
		EndpointIDs: []string{agentID},
		Parameters:  params,
	})
}

func (c *Client) GetFile(ctx context.Context, agentID, path string) (*Action, error) {
	return c.SendAction(ctx, "get_file", ActionRequest{
		EndpointIDs: []string{agentID},
		Parameters:  map[string]any{"path": path},
	})
}
