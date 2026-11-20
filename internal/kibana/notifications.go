package kibana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

// SetNativeNotifications preserves the complete input configuration, including
// unknown protection fields. Only Windows popup enabled flags are edited.
func (c *Client) SetNativeNotifications(ctx context.Context, id, agentID string, enabled bool) (int, error) {
	path := "/api/fleet/package_policies/" + url.PathEscape(id)
	var envelope struct {
		Item map[string]any `json:"item"`
	}
	if err := c.Get(ctx, path, &envelope); err != nil {
		return 0, err
	}
	item := envelope.Item
	if item["is_managed"] == true {
		return 0, errors.New("此策略由管理员管理，请在 Kibana 中修改通知")
	}
	policyIDs := []string{}
	if ids, ok := item["policy_ids"].([]any); ok {
		for _, value := range ids {
			if id, ok := value.(string); ok {
				policyIDs = append(policyIDs, id)
			}
		}
	} else if policyID, ok := item["policy_id"].(string); ok {
		policyIDs = append(policyIDs, policyID)
	}
	if len(policyIDs) != 1 {
		return 0, errors.New("此 Defend 策略被多个 Agent 策略共享，请先在 Kibana 为本机分配独立策略")
	}
	quoted, _ := json.Marshal(policyIDs[0])
	query := url.Values{"perPage": {"2"}, "kuery": {"fleet-agents.policy_id:" + string(quoted)}}
	var agents AgentsResponse
	if err := c.Get(ctx, "/api/fleet/agents?"+query.Encode(), &agents); err != nil {
		return 0, err
	}
	if agents.Total != 1 || len(agents.Items) != 1 || agents.Items[0].ID != agentID {
		return 0, errors.New("此策略未独立绑定本机，请先在 Kibana 为本机分配独立策略后切换通知")
	}
	if err := editPopupFlags(item, enabled); err != nil {
		return 0, err
	}
	// Fleet update accepts configuration fields, not saved-object metadata.
	body := map[string]any{}
	for _, key := range []string{"name", "description", "namespace", "enabled", "policy_id", "policy_ids", "package", "inputs", "vars", "output_id", "version", "config", "condition", "global_data_tags", "spaceIds", "cloud_connector_id"} {
		if value, ok := item[key]; ok {
			body[key] = value
		}
	}
	for _, value := range body["inputs"].([]any) {
		if input, ok := value.(map[string]any); ok {
			delete(input, "compiled_input")
		}
	}
	var updated struct {
		Item PackagePolicy `json:"item"`
	}
	if err := c.Put(ctx, path, body, &updated); err != nil {
		return 0, fmt.Errorf("通知策略下发失败，未切换本地模式。若请求超时，请刷新策略后重试：%w", err)
	}
	if updated.Item.Revision <= 0 {
		return 0, errors.New("策略返回缺少版本号，请刷新后重试")
	}
	return updated.Item.Revision, nil
}

func editPopupFlags(item map[string]any, enabled bool) error {
	inputs, ok := item["inputs"].([]any)
	if !ok {
		return errors.New("策略输入格式不受支持")
	}
	for _, value := range inputs {
		input, _ := value.(map[string]any)
		if input["type"] != "endpoint" {
			continue
		}
		config, _ := input["config"].(map[string]any)
		policy, _ := config["policy"].(map[string]any)
		settings, _ := policy["value"].(map[string]any)
		windows, _ := settings["windows"].(map[string]any)
		popup, _ := windows["popup"].(map[string]any)
		for _, kind := range []string{"malware", "ransomware", "memory_protection", "behavior_protection"} {
			entry, ok := popup[kind].(map[string]any)
			if !ok {
				return fmt.Errorf("策略缺少 Windows %s 通知配置，未修改", kind)
			}
			if _, ok := entry["enabled"].(bool); !ok {
				return errors.New("通知开关格式不受支持，未修改")
			}
		}
		for _, value := range popup {
			entry, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := entry["enabled"].(bool); ok {
				entry["enabled"] = enabled
			}
		}
		return nil
	}
	return errors.New("没有找到 Elastic Defend Windows 通知配置")
}
