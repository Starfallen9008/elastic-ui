package kibana

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type AgentPolicy struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Description       string           `json:"description"`
	Namespace         string           `json:"namespace"`
	InactivityTimeout int              `json:"inactivity_timeout"`
	Revision          int              `json:"revision"`
	Managed           bool             `json:"is_managed"`
	Packages          []map[string]any `json:"package_policies"`
}

func (c *Client) GetAgentPolicy(ctx context.Context, id string) (*AgentPolicy, error) {
	var out struct {
		Item AgentPolicy `json:"item"`
	}
	if err := c.Get(ctx, "/api/fleet/agent_policies/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out.Item, nil
}

func endpointPackage(policy *AgentPolicy) (map[string]any, error) {
	var endpoint map[string]any
	for _, p := range policy.Packages {
		pkg, _ := p["package"].(map[string]any)
		if pkg["name"] == "fleet_server" {
			return nil, errors.New("Fleet Server 策略请在 Kibana 管理")
		}
		if p["is_managed"] == true {
			return nil, errors.New("受管理的集成不能复制，请在 Kibana 管理")
		}
		// Fleet copy links shared integrations rather than copying them.
		ids, ok := p["policy_ids"].([]any)
		if !ok || len(ids) != 1 || ids[0] != policy.ID {
			return nil, errors.New("策略包含共享集成，请先在 Kibana 拆分共享集成")
		}
		if pkg["name"] == "endpoint" {
			if endpoint != nil {
				return nil, errors.New("存在多个 Defend 集成")
			}
			endpoint = p
		}
	}
	if endpoint == nil {
		return nil, errors.New("策略缺少 Elastic Defend 集成")
	}
	return endpoint, nil
}

func (c *Client) PolicyHostCount(ctx context.Context, policyID string) (*AgentsResponse, error) {
	var out AgentsResponse
	path := "/api/fleet/agents?perPage=2&kuery=" + url.QueryEscape(`fleet-agents.policy_id:"`+policyID+`"`)
	if err := c.Get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateIndependentPolicy copies every integration before reassigning just this agent.
// A failed/ambiguous reassign retains the copy for inspection and retry.
func (c *Client) CreateIndependentPolicy(ctx context.Context, agentID, name string) (*AgentPolicy, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 120 {
		return nil, errors.New("策略名称需为 1–120 个字符")
	}
	agent, err := c.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}
	source, err := c.GetAgentPolicy(ctx, agent.PolicyID)
	if err != nil {
		return nil, err
	}
	if source.Managed {
		return nil, errors.New("受管理策略请在 Kibana 管理")
	}
	original, err := endpointPackage(source)
	if err != nil {
		return nil, err
	}
	var out struct {
		Item AgentPolicy `json:"item"`
	}
	copyPath := "/api/fleet/agent_policies/" + url.PathEscape(source.ID) + "/copy"
	body := map[string]any{"name": name, "description": "Elastic UI 本机独立策略；Agent " + agentID}
	if err := c.Post(ctx, copyPath, body, &out); err != nil {
		return nil, fmt.Errorf("复制策略失败，请检查 Kibana 中是否已创建后再重试：%w", err)
	}
	copied, err := c.GetAgentPolicy(ctx, out.Item.ID)
	if err != nil {
		return nil, fmt.Errorf("策略 %s 已创建，读取失败：%w", out.Item.ID, err)
	}
	cloned, err := endpointPackage(copied)
	if err != nil {
		return nil, fmt.Errorf("策略 %s 未绑定本机：%w", copied.ID, err)
	}
	if copied.ID == "" || copied.ID == source.ID || cloned["id"] == original["id"] || len(copied.Packages) != len(source.Packages) {
		return nil, fmt.Errorf("策略 %s 复制结果不完整，尚未迁移本机", copied.ID)
	}
	current, err := c.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if current.PolicyID != source.ID {
		return nil, errors.New("本机策略已被其他操作修改，请刷新后重试")
	}
	if err := c.Post(ctx, "/api/fleet/agents/"+url.PathEscape(agentID)+"/reassign", map[string]any{"policy_id": copied.ID}, nil); err != nil {
		return nil, fmt.Errorf("独立策略 %s 已创建；本机迁移结果待确认，请刷新绑定：%w", copied.ID, err)
	}
	return copied, nil
}

func (c *Client) RenameIndependentPolicy(ctx context.Context, agentID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 120 {
		return errors.New("策略名称需为 1–120 个字符")
	}
	agent, err := c.GetAgent(ctx, agentID)
	if err != nil {
		return err
	}
	p, err := c.GetAgentPolicy(ctx, agent.PolicyID)
	if err != nil {
		return err
	}
	if p.Managed {
		return errors.New("受管理策略不能修改")
	}
	if _, err = endpointPackage(p); err != nil {
		return err
	}
	hosts, err := c.PolicyHostCount(ctx, p.ID)
	if err != nil {
		return err
	}
	if hosts.Total != 1 || len(hosts.Items) != 1 || hosts.Items[0].ID != agentID {
		return errors.New("仅允许修改本机独立策略")
	}
	path := "/api/fleet/agent_policies/" + url.PathEscape(p.ID)
	body := map[string]any{
		"name": name, "description": p.Description, "namespace": p.Namespace,
		"inactivity_timeout": p.InactivityTimeout,
	}
	return c.Put(ctx, path, body, nil)
}
