package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type IndependentPolicyState struct {
	Independent bool   `json:"independent"`
	Applied     bool   `json:"applied"`
	Name        string `json:"name"`
	PolicyID    string `json:"policyId"`
	PackageID   string `json:"packageId"`
	HostCount   int    `json:"hostCount"`
	Message     string `json:"message"`
}

func (s *Service) IndependentPolicyState() (*IndependentPolicyState, error) {
	s.mu.RLock()
	kb, agent := s.kb, s.agent
	s.mu.RUnlock()
	if kb == nil || agent == nil {
		return nil, errors.New("请先连接并绑定本机")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	current, err := kb.GetAgent(ctx, agent.ID)
	if err != nil {
		return nil, err
	}
	p, err := kb.GetAgentPolicy(ctx, current.PolicyID)
	if err != nil {
		return nil, err
	}
	hosts, err := kb.PolicyHostCount(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	state := &IndependentPolicyState{
		Name: p.Name, PolicyID: p.ID, HostCount: hosts.Total,
		Message: "当前策略未独立绑定本机，可创建独立副本",
	}
	for _, pkg := range p.Packages {
		// Inspect only Defend membership, leaving non-Defend administration in Kibana.
		info, _ := pkg["package"].(map[string]any)
		if info["name"] == "endpoint" {
			state.PackageID, _ = pkg["id"].(string)
			ids, _ := pkg["policy_ids"].([]any)
			localOnly := hosts.Total == 1 && len(hosts.Items) == 1 && hosts.Items[0].ID == agent.ID
			privatePackage := len(ids) == 1 && ids[0] == p.ID
			state.Independent = localOnly && privatePackage && !p.Managed && pkg["is_managed"] != true
		}
	}
	if state.Independent {
		state.Message = "独立策略已绑定，等待 Agent 应用"
		meta, err := kb.GetEndpointMetadata(ctx, agent.ID)
		if err == nil && meta != nil {
			applied := meta.Endpoint.Policy.Applied
			state.Applied = current.PolicyRevision >= p.Revision &&
				applied.ID == state.PackageID && applied.Status == "success"
		}
		if state.Applied {
			state.Message = "本机独立策略已生效，可单独设置通知和排除规则"
		}
	}
	return state, nil
}

func (s *Service) CreateIndependentPolicy(name string) (*IndependentPolicyState, error) {
	s.notificationMu.Lock()
	defer s.notificationMu.Unlock()
	s.trustMu.Lock()
	defer s.trustMu.Unlock()
	s.mu.RLock()
	kb, agent := s.kb, s.agent
	s.mu.RUnlock()
	if kb == nil || agent == nil {
		return nil, errors.New("请先连接并绑定本机")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s.mu.Lock()
	s.notificationSwitching = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.notificationSwitching = false; s.mu.Unlock() }()
	p, err := kb.CreateIndependentPolicy(ctx, agent.ID, name)
	if err != nil {
		return nil, err
	}
	var packageID, packageName string
	for _, pkg := range p.Packages {
		info, _ := pkg["package"].(map[string]any)
		if info["name"] == "endpoint" {
			packageID, _ = pkg["id"].(string)
			packageName, _ = pkg["name"].(string)
		}
	}
	s.mu.Lock()
	updated := *s.settings
	updated.Profile.AgentPolicyID = p.ID
	updated.Profile.PackagePolicyID = packageID
	updated.Profile.PolicyName = packageName
	updated.Profile.BoundAt = time.Now().Format(time.RFC3339)
	updated.NotificationRevision = 0
	err = updated.Save()
	*s.settings = updated
	s.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("本机已迁移至 %s，本地保存失败：%w", p.ID, err)
	}
	// Re-read the actual agent assignment rather than trusting old Endpoint metadata.
	if err = s.Rebind(); err != nil {
		return nil, fmt.Errorf("本机已迁移，请刷新绑定：%w", err)
	}
	return s.IndependentPolicyState()
}

func (s *Service) RenameIndependentPolicy(name string) (*IndependentPolicyState, error) {
	s.notificationMu.Lock()
	defer s.notificationMu.Unlock()
	s.mu.RLock()
	kb, agent := s.kb, s.agent
	s.mu.RUnlock()
	if kb == nil || agent == nil {
		return nil, errors.New("请先连接并绑定本机")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := kb.RenameIndependentPolicy(ctx, agent.ID, name); err != nil {
		return nil, err
	}
	return s.IndependentPolicyState()
}

func (s *Service) CompletePolicySetup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	updated := *s.settings
	updated.PolicySetupComplete = true
	if err := updated.Save(); err != nil {
		return err
	}
	*s.settings = updated
	return nil
}
