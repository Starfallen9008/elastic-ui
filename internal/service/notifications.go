package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type NotificationState struct {
	Mode    string `json:"mode"`
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

func (s *Service) SetNotificationMode(mode string) error {
	if mode != "native" && mode != "ui" {
		return errors.New("不支持的通知模式")
	}
	s.notificationMu.Lock()
	defer s.notificationMu.Unlock()
	s.mu.RLock()
	kb, agent := s.kb, s.agent
	profile := s.settings.Profile
	s.mu.RUnlock()
	if kb == nil || agent == nil || profile.PackagePolicyID == "" {
		return errors.New("请先连接并绑定本机")
	}
	s.mu.Lock()
	s.notificationSwitching = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.notificationSwitching = false; s.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	revision, err := kb.SetNativeNotifications(ctx, profile.PackagePolicyID, agent.ID, mode == "native")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Persist only after Fleet confirms the write; startup never edits policy.
	updated := *s.settings
	updated.NotificationMode = mode
	updated.NotificationRevision = revision
	updated.Notifications = mode == "ui"
	if err := updated.Save(); err != nil {
		return fmt.Errorf("远端通知策略已更新，但本地保存失败，请重新选择通知模式：%w", err)
	}
	*s.settings = updated
	return nil
}

func (s *Service) NotificationState() NotificationState {
	s.mu.RLock()
	mode := s.settings.NotificationMode
	revision := s.settings.NotificationRevision
	kb, agent := s.kb, s.agent
	policyID := s.settings.Profile.PackagePolicyID
	s.mu.RUnlock()
	state := NotificationState{Mode: mode}
	if mode != "ui" && revision == 0 {
		state.Mode = "native"
		state.Ready = true
		state.Message = "由 Elastic Agent 显示原生通知"
		return state
	}
	if kb == nil || agent == nil || revision <= 0 {
		state.Message = "请重新选择通知来源以同步策略"
		return state
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	meta, err := kb.GetEndpointMetadata(ctx, agent.ID)
	if err != nil {
		state.Message = "无法确认 Agent 通知策略：" + err.Error()
		return state
	}
	applied := meta.Endpoint.Policy.Applied
	version, _ := strconv.Atoi(applied.EndpointPolicyVersion)
	if applied.ID != policyID || version < revision || applied.Status != "success" {
		state.Message = "通知策略已下发，等待 Agent 应用"
		return state
	}
	// Recheck desired policy so changes made in Kibana cannot enable both channels.
	policy, err := kb.GetPackagePolicy(ctx, policyID)
	if err != nil {
		state.Message = "无法核实原生通知开关：" + err.Error()
		return state
	}
	value, err := policy.PolicyValue()
	if err != nil {
		state.Message = "无法解析通知策略"
		return state
	}
	if version < policy.Revision {
		state.Message = "等待 Agent 应用最新通知策略"
		return state
	}
	if len(value.Windows.Popup) == 0 {
		state.Message = "策略缺少原生通知状态"
		return state
	}
	for _, entry := range value.Windows.Popup {
		if entry.Enabled != (mode == "native") {
			state.Message = "原生通知开关与当前来源不一致，请重新同步通知模式"
			return state
		}
	}
	state.Ready = true
	state.Message = "由 Elastic UI 显示桌面弹窗；程序需保持运行"
	if mode == "native" {
		state.Message = "由 Elastic Agent 显示原生通知"
	}
	return state
}
