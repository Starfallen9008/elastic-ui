package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"elastic-ui/internal/agentstatus"
	"elastic-ui/internal/auth"
	"elastic-ui/internal/config"
	"elastic-ui/internal/es"
	"elastic-ui/internal/kibana"
	"elastic-ui/internal/model"
)

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type ConnectionInput struct {
	KibanaURL     string `json:"kibanaUrl"`
	ESURL         string `json:"esUrl"`
	AuthMethod    string `json:"authMethod"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	APIKey        string `json:"apiKey"`
	SkipTLSVerify bool   `json:"skipTlsVerify"`
}

type ConnectionResult struct {
	OK      bool      `json:"ok"`
	Checks  []Check   `json:"checks"`
	Agent   *AgentRef `json:"agent,omitempty"`
	Version string    `json:"version,omitempty"`
}

type AgentRef struct {
	ID         string `json:"id"`
	HostName   string `json:"hostName"`
	PolicyID   string `json:"policyId"`
	PolicyName string `json:"policyName"`
}

type Protection struct {
	Malware        string `json:"malware"`
	Ransomware     string `json:"ransomware"`
	Memory         string `json:"memory"`
	Behavior       string `json:"behavior"`
	DeviceControl  string `json:"deviceControl"`
	AVRegistration bool   `json:"avRegistration"`
}

type StatusView struct {
	Connected               bool               `json:"connected"`
	ConnectionError         string             `json:"connectionError"`
	ProtectionError         string             `json:"protectionError"`
	KibanaVersion           string             `json:"kibanaVersion"`
	License                 string             `json:"license"`
	HostName                string             `json:"hostName"`
	AgentID                 string             `json:"agentId"`
	AgentStatus             string             `json:"agentStatus"`
	LastCheckin             string             `json:"lastCheckin"`
	EndpointStatus          string             `json:"endpointStatus"`
	Isolation               bool               `json:"isolation"`
	TamperProtection        bool               `json:"tamperProtection"`
	AgentVersion            string             `json:"agentVersion"`
	EndpointVersion         string             `json:"endpointVersion"`
	PolicyName              string             `json:"policyName"`
	PolicyVersion           string             `json:"policyVersion"`
	EndpointPolicyVer       string             `json:"endpointPolicyVersion"`
	Protection              Protection         `json:"protection"`
	Alerts24h               int64              `json:"alerts24h"`
	Blocked24h              int64              `json:"blocked24h"`
	LastScan                *ScanTask          `json:"lastScan,omitempty"`
	Checks                  []Check            `json:"checks"`
	RefreshAt               string             `json:"refreshAt"`
	ScanCount               int                `json:"scanCount"`
	IsolationActionID       string             `json:"isolationActionId,omitempty"`
	IsolationPending        bool               `json:"isolationPending"`
	PackagePolicyID         string             `json:"packagePolicyId"`
	Capabilities            []string           `json:"capabilities"`
	ProtectionUpdateNote    string             `json:"protectionUpdateNote"`
	ProtectionUpdateVersion string             `json:"protectionUpdateVersion"`
	ProtectionUpdateAge     int                `json:"protectionUpdateAge"`
	AgentRuntime            agentstatus.Status `json:"agentRuntime"`
}

type Threat struct {
	ID            string  `json:"id"`
	Time          string  `json:"time"`
	Message       string  `json:"message"`
	RuleName      string  `json:"ruleName"`
	Severity      float64 `json:"severity"`
	RiskScore     float64 `json:"riskScore"`
	Blocked       bool    `json:"blocked"`
	Action        string  `json:"action"`
	ProcessName   string  `json:"processName"`
	ProcessPath   string  `json:"processPath"`
	FilePath      string  `json:"filePath"`
	MalwareName   string  `json:"malwareName"`
	SHA256        string  `json:"sha256"`
	Read          bool    `json:"read"`
	Category      string  `json:"category"`
	DetectionType string  `json:"detectionType"`
	ResponseTime  string  `json:"responseTime"`
}

type ThreatPage struct {
	Items []Threat `json:"items"`
	Total int64    `json:"total"`
}

type ThreatDetail struct {
	Threat
	QuarantineFile *ScanFile  `json:"quarantineFile,omitempty"`
	CommandLine    string     `json:"commandLine"`
	WorkingDir     string     `json:"workingDir"`
	ParentName     string     `json:"parentName"`
	ParentPath     string     `json:"parentPath"`
	ParentCmd      string     `json:"parentCmd"`
	User           string     `json:"user"`
	Responses      []Response `json:"responses"`
	Tactics        []string   `json:"tactics"`
	Techniques     []string   `json:"techniques"`
	HostOS         string     `json:"hostOs"`
}

type Response struct {
	Action    string `json:"action"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type ScanTask struct {
	ID          string `json:"id"`
	ActionID    string `json:"actionId"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt,omitempty"`
	Detail      string `json:"detail,omitempty"`
	Found       int64  `json:"found"`
}

type TrustItem struct {
	ID          string `json:"id"`
	ItemID      string `json:"itemId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	Value       string `json:"value"`
	OS          string `json:"os"`
	Scope       string `json:"scope"`
	ReadOnly    bool   `json:"readOnly"`
	CreatedBy   string `json:"createdBy"`
	UpdatedAt   string `json:"updatedAt"`
	MatchKey    string `json:"matchKey"`
}

type TrustInput struct {
	Kind        string `json:"kind"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type ProcessNode struct {
	WorkingDir     string   `json:"workingDir"`
	EntityID       string   `json:"entityId"`
	Name           string   `json:"name"`
	PID            int      `json:"pid"`
	ParentEntityID string   `json:"parentEntityId"`
	Executable     string   `json:"executable"`
	CommandLine    string   `json:"commandLine"`
	Args           []string `json:"args"`
	User           string   `json:"user"`
	Timestamp      string   `json:"timestamp"`
	Start          string   `json:"start"`
	End            string   `json:"end"`
	EventAction    string   `json:"eventAction"`
	Resolved       bool     `json:"resolved"`
	IsAlert        bool     `json:"isAlert"`
	Depth          int      `json:"depth"`
}

type ProcessChain struct {
	Notice    string        `json:"notice"`
	AlertID   string        `json:"alertId"`
	AlertName string        `json:"alertName"`
	AlertTime string        `json:"alertTime"`
	HostName  string        `json:"hostName"`
	Nodes     []ProcessNode `json:"nodes"`
	Children  []ProcessNode `json:"children"`
}

type Service struct {
	mu                    sync.RWMutex
	trustMu               sync.Mutex
	quarantineMu          sync.Mutex
	notificationMu        sync.Mutex
	notificationSwitching bool
	settings              *config.Settings
	kb                    *kibana.Client
	esc                   *es.Client
	agent                 *kibana.Agent
	meta                  *kibana.EndpointMetadata
	policy                *kibana.PackagePolicy
	policyV               *kibana.PolicyValue
	license               string
	scans                 []*ScanTask
	read                  map[string]bool
	isolate               string
	lastErr               error

	procsCache   []ProcessInfo
	procsCacheAt time.Time

	protectionVersion string
	protectionAge     int
	protectionAt      time.Time

	agentRuntimeCache agentstatus.Status
	agentRuntimeAt    time.Time
}

func New() *Service {
	s := &Service{settings: config.Default(), read: map[string]bool{}}
	if settings, err := config.Load(); err == nil {
		s.settings = settings
	}
	s.loadLocalState()
	s.lastErr = s.connect()
	return s
}

func (s *Service) ConnectError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastErr
}

func (s *Service) Settings() *config.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Service) SetNotifications(enabled bool) error {
	mode := "native"
	if enabled {
		mode = "ui"
	}
	return s.SetNotificationMode(mode)
}

func (s *Service) NotificationsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.NotificationMode == "ui" && !s.notificationSwitching
}

func (s *Service) SetCloseToTray(enabled bool) error {
	s.mu.Lock()
	s.settings.CloseToTray = enabled
	s.mu.Unlock()
	return s.settings.Save()
}

func (s *Service) CloseToTrayEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.CloseToTray
}

func (s *Service) Configure(in ConnectionInput) (*ConnectionResult, error) {
	result := &ConnectionResult{OK: false}
	add := func(name string, err error, okDetail string) bool {
		if err != nil {
			result.Checks = append(result.Checks, Check{Name: name, OK: false, Detail: err.Error()})
			return false
		}
		result.Checks = append(result.Checks, Check{Name: name, OK: true, Detail: okDetail})
		return true
	}

	method := config.AuthBasic
	if in.AuthMethod == string(config.AuthAPIKey) {
		method = config.AuthAPIKey
	}
	cred := auth.Auth{Method: method, Username: in.Username, Password: in.Password, APIKey: in.APIKey}
	if method == config.AuthBasic && in.Username == "" {
		return result, errors.New("用户名不能为空")
	}
	if method == config.AuthAPIKey && in.APIKey == "" {
		return result, errors.New("API Key 不能为空")
	}

	kb, err := kibana.New(in.KibanaURL, cred, in.SkipTLSVerify, "")
	if err != nil {
		add("Kibana 地址", err, "")
		return result, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := kb.LoadVersion(ctx); err != nil {
		add("Kibana 连接", err, "")
		return result, err
	}
	add("Kibana 连接", nil, "Kibana "+kb.Version)

	esc, err := es.New(in.ESURL, cred, in.SkipTLSVerify)
	if err != nil {
		add("Elasticsearch 地址", err, "")
		return result, err
	}
	var lic struct {
		License struct {
			Type string `json:"type"`
		} `json:"license"`
	}
	if err := esc.Get(ctx, "/_license", &lic); err != nil {
		add("Elasticsearch 连接", err, "")
		return result, err
	}
	add("Elasticsearch 连接", nil, "授权 "+lic.License.Type)

	agents, err := kb.ListAgents(ctx, 200)
	if err != nil {
		add("Fleet 权限", err, "")
		return result, err
	}
	add("Fleet 权限", nil, fmt.Sprintf("可读取 %d 个 agent", agents.Total))

	oldSettings := s.settings
	oldKB, oldES := s.kb, s.esc
	s.mu.Lock()
	s.kb, s.esc = kb, esc
	s.settings.Profile.KibanaURL = strings.TrimRight(strings.TrimSpace(in.KibanaURL), "/")
	s.settings.Profile.ESURL = strings.TrimRight(strings.TrimSpace(in.ESURL), "/")
	s.settings.Profile.AuthMethod = method
	s.settings.Profile.Username = in.Username
	s.settings.Profile.SkipTLSVerify = in.SkipTLSVerify
	s.settings.Profile.APIKeyHint = ""
	if method == config.AuthAPIKey && len(in.APIKey) > 8 {
		s.settings.Profile.APIKeyHint = in.APIKey[:4] + "..." + in.APIKey[len(in.APIKey)-4:]
	}
	s.license = lic.License.Type
	s.mu.Unlock()

	agent, ref, bindErr := s.bindLocal(ctx)
	if bindErr != nil {
		add("本机绑定", bindErr, "")
		s.mu.Lock()
		s.settings = oldSettings
		s.kb, s.esc = oldKB, oldES
		s.mu.Unlock()
		return result, bindErr
	}
	add("本机绑定", nil, ref.HostName+" / "+ref.PolicyName)

	if err := s.persistSecret(in); err != nil {
		add("凭证保存", err, "")
	}
	if err := s.settings.Save(); err != nil {
		return result, err
	}
	s.mu.Lock()
	s.agent = agent
	result.OK = true
	result.Agent = ref
	result.Version = kb.Version
	s.lastErr = nil
	s.mu.Unlock()
	return result, nil
}

func (s *Service) persistSecret(in ConnectionInput) error {
	if s.settings.Profile.AuthMethod == config.AuthAPIKey {
		_ = config.DeleteSecret("password")
		return config.SaveSecret("apikey", in.APIKey)
	}
	_ = config.DeleteSecret("apikey")
	return config.SaveSecret("password", in.Password)
}

func (s *Service) connect() error {
	profile := s.settings.Profile
	if !profile.Valid() {
		return errors.New("尚未配置连接")
	}
	cred := auth.Auth{Method: profile.AuthMethod, Username: profile.Username}
	if profile.AuthMethod == config.AuthAPIKey {
		key, err := config.LoadSecret("apikey")
		if err != nil {
			return fmt.Errorf("读取 API Key 失败: %w", err)
		}
		cred.APIKey = key
	} else {
		pass, err := config.LoadSecret("password")
		if err != nil {
			return fmt.Errorf("读取密码失败: %w", err)
		}
		cred.Password = pass
	}
	kb, err := kibana.New(profile.KibanaURL, cred, profile.SkipTLSVerify, profile.Space)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := kb.LoadVersion(ctx); err != nil {
		return err
	}
	esc, err := es.New(profile.ESURL, cred, profile.SkipTLSVerify)
	if err != nil {
		return err
	}
	var lic struct {
		License struct {
			Type string `json:"type"`
		} `json:"license"`
	}
	if err := esc.Get(ctx, "/_license", &lic); err == nil {
		s.license = lic.License.Type
	}
	s.mu.Lock()
	s.kb = kb
	s.esc = esc
	s.mu.Unlock()

	if profile.AgentID != "" {
		if agent, err := kb.GetAgent(ctx, profile.AgentID); err == nil {
			s.agent = agent
			if err := s.refreshEndpoint(ctx); err != nil {
				return err
			}
			return nil
		}
	}
	_, _, err = s.bindLocal(ctx)
	return err
}

func (s *Service) bindLocal(ctx context.Context) (*kibana.Agent, *AgentRef, error) {
	if s.kb == nil {
		return nil, nil, errors.New("尚未连接 Kibana")
	}
	agents, err := s.kb.ListAgents(ctx, 200)
	if err != nil {
		return nil, nil, err
	}
	hostname, _ := os.Hostname()
	var candidates []*kibana.Agent
	for i := range agents.Items {
		item := &agents.Items[i]
		if strings.EqualFold(item.LocalMetadata.Host.Name, hostname) {
			candidates = append(candidates, item)
		}
	}
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("Fleet 中没有找到主机名为 %s 的 agent", hostname)
	}
	localIPs := localIPSet()
	best := candidates[0]
	bestScore := -1
	for _, candidate := range candidates {
		score := 0
		if meta, err := s.kb.GetEndpointMetadata(ctx, candidate.ID); err == nil {
			for _, ip := range meta.Host.IP {
				if localIPs[ip] {
					score++
				}
			}
		}
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	meta, err := s.kb.GetEndpointMetadata(ctx, best.ID)
	if err != nil {
		return nil, nil, err
	}
	configured, err := s.kb.GetAgentPolicy(ctx, best.PolicyID)
	if err != nil {
		return nil, nil, err
	}
	packagePolicyID := ""
	for _, p := range configured.Packages {
		info, _ := p["package"].(map[string]any)
		if info["name"] == "endpoint" {
			packagePolicyID, _ = p["id"].(string)
		}
	}
	if packagePolicyID == "" {
		return nil, nil, errors.New("本机未启用 Elastic Defend 集成")
	}
	policy, err := s.kb.GetPackagePolicy(ctx, packagePolicyID)
	if err != nil {
		return nil, nil, err
	}
	policyV, err := policy.PolicyValue()
	if err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	s.agent = best
	s.meta = meta
	s.policy = policy
	s.policyV = policyV
	s.settings.Profile.AgentID = best.ID
	s.settings.Profile.HostName = meta.Host.Hostname
	s.settings.Profile.AgentPolicyID = best.PolicyID
	s.settings.Profile.PackagePolicyID = packagePolicyID
	s.settings.Profile.PolicyName = policy.Name
	s.settings.Profile.BoundAt = time.Now().Format(time.RFC3339)
	s.mu.Unlock()
	_ = s.settings.Save()
	return best, &AgentRef{ID: best.ID, HostName: meta.Host.Hostname, PolicyID: packagePolicyID, PolicyName: policy.Name}, nil
}

func (s *Service) refreshEndpoint(ctx context.Context) error {
	if s.kb == nil || s.agent == nil {
		return errors.New("尚未绑定本机")
	}
	agent, err := s.kb.GetAgent(ctx, s.settings.Profile.AgentID)
	if err != nil {
		return err
	}
	meta, err := s.kb.GetEndpointMetadata(ctx, agent.ID)
	if err != nil {
		return err
	}
	configured, err := s.kb.GetAgentPolicy(ctx, agent.PolicyID)
	if err != nil {
		return err
	}
	packagePolicyID := ""
	for _, p := range configured.Packages {
		info, _ := p["package"].(map[string]any)
		if info["name"] == "endpoint" {
			packagePolicyID, _ = p["id"].(string)
		}
	}
	var policy *kibana.PackagePolicy
	var policyV *kibana.PolicyValue
	if packagePolicyID != "" {
		policy, err = s.kb.GetPackagePolicy(ctx, packagePolicyID)
		if err != nil {
			return err
		}
		policyV, err = policy.PolicyValue()
		if err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.agent = agent
	s.meta = meta
	s.policy = policy
	s.policyV = policyV
	if s.settings.Profile.AgentPolicyID != agent.PolicyID || s.settings.Profile.PackagePolicyID != packagePolicyID {
		s.settings.Profile.AgentPolicyID = agent.PolicyID
		s.settings.Profile.PackagePolicyID = packagePolicyID
		if policy != nil {
			s.settings.Profile.PolicyName = policy.Name
		}
		if err := s.settings.Save(); err != nil {
			s.mu.Unlock()
			return err
		}
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) refreshProtection(ctx context.Context, agentID string) {
	if s.kb == nil || agentID == "" {
		return
	}
	s.mu.RLock()
	version, at := s.protectionVersion, s.protectionAt
	s.mu.RUnlock()
	if version != "" && time.Since(at) < 5*time.Minute {
		return
	}
	var resp struct {
		PolicyResponse struct {
			Endpoint struct {
				Policy struct {
					Applied struct {
						Artifacts struct {
							Global struct {
								Version   string `json:"version"`
								UpdateAge int    `json:"update_age"`
							} `json:"global"`
						} `json:"artifacts"`
					} `json:"applied"`
				} `json:"policy"`
			} `json:"Endpoint"`
		} `json:"policy_response"`
	}
	path := "/api/endpoint/policy_response?agentId=" + url.QueryEscape(agentID)
	if err := s.kb.Get(ctx, path, &resp); err != nil {
		return
	}
	s.mu.Lock()
	s.protectionVersion = resp.PolicyResponse.Endpoint.Policy.Applied.Artifacts.Global.Version
	s.protectionAge = resp.PolicyResponse.Endpoint.Policy.Applied.Artifacts.Global.UpdateAge
	s.protectionAt = time.Now()
	s.mu.Unlock()
}

func (s *Service) Status() (*StatusView, error) {
	if s.kb == nil {
		if err := s.connect(); err != nil {
			s.mu.Lock()
			s.lastErr = err
			s.mu.Unlock()
			return s.unavailableStatus(err), nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.refreshEndpoint(ctx); err != nil {
		return s.unavailableStatus(err), nil
	}
	s.refreshProtection(ctx, s.settings.Profile.AgentID)
	runtimeStatus := s.agentRuntimeStatus()
	s.mu.RLock()
	agent, meta, policy, policyV := s.agent, s.meta, s.policy, s.policyV
	license := s.license
	scans := append([]*ScanTask(nil), s.scans...)
	isolate := s.isolate
	protectionVersion := s.protectionVersion
	protectionAge := s.protectionAge
	s.mu.RUnlock()

	view := &StatusView{
		Connected:               true,
		KibanaVersion:           s.kb.Version,
		License:                 license,
		HostName:                meta.Host.Hostname,
		AgentID:                 agent.ID,
		AgentStatus:             agent.Status,
		LastCheckin:             agent.LastCheckin,
		EndpointStatus:          meta.Endpoint.Status,
		Isolation:               meta.Endpoint.State.Isolation,
		TamperProtection:        meta.Endpoint.State.TamperProtection,
		AgentVersion:            meta.Agent.Version,
		EndpointVersion:         meta.Agent.Version,
		PolicyName:              meta.Endpoint.Policy.Applied.Name,
		PolicyVersion:           meta.Endpoint.Policy.Applied.Version,
		EndpointPolicyVer:       meta.Endpoint.Policy.Applied.EndpointPolicyVersion,
		PackagePolicyID:         meta.Endpoint.Policy.Applied.ID,
		Capabilities:            meta.Endpoint.Capabilities,
		ProtectionUpdateVersion: protectionVersion,
		ProtectionUpdateAge:     protectionAge,
		AgentRuntime:            runtimeStatus,
		RefreshAt:               time.Now().Format(time.RFC3339),
		IsolationActionID:       isolate,
	}
	if policy != nil && policyV != nil {
		p := policyV.Windows
		view.Protection = Protection{
			Malware:        modeText(p.Malware.Mode),
			Ransomware:     modeText(p.Ransomware.Mode),
			Memory:         modeText(p.MemoryProtection.Mode),
			Behavior:       modeText(p.BehaviorProtection.Mode),
			DeviceControl:  deviceControlText(p.DeviceControl.Enabled, p.DeviceControl.USBStorage),
			AVRegistration: p.AntivirusRegistration.Enabled,
		}
	} else {
		view.ProtectionError = "未读取到本机 Windows 防护策略，无法确认模块状态"
	}
	if len(scans) > 0 {
		view.ScanCount = len(scans)
		view.LastScan = scans[0]
	}

	counts, err := s.alertCounts(ctx, agent.ID)
	if err == nil {
		view.Alerts24h = counts[0]
		view.Blocked24h = counts[1]
	}
	view.Checks = buildChecks(view)
	s.mu.Lock()
	s.lastErr = nil
	s.mu.Unlock()
	return view, nil
}

// Management connectivity and local daemon health are independent. Do not
// report zero-value remote policy fields as disabled when the read failed.
func (s *Service) unavailableStatus(err error) *StatusView {
	runtimeStatus := s.agentRuntimeStatus()
	s.mu.Lock()
	s.lastErr = err
	profile := s.settings.Profile
	s.mu.Unlock()
	view := &StatusView{
		Connected:       false,
		ConnectionError: err.Error(),
		ProtectionError: "管理连接失败，无法读取当前防护策略",
		HostName:        profile.HostName,
		AgentID:         profile.AgentID,
		PackagePolicyID: profile.PackagePolicyID,
		AgentRuntime:    runtimeStatus,
		RefreshAt:       time.Now().Format(time.RFC3339),
	}
	view.Checks = []Check{
		{Name: "管理连接", OK: false, Detail: view.ConnectionError},
		{Name: "本机 Agent", OK: runtimeStatus.State == "HEALTHY", Detail: runtimeStatus.State + " · " + runtimeStatus.Message},
		{Name: "防护策略", OK: false, Detail: view.ProtectionError},
	}
	return view
}

func (s *Service) alertCounts(ctx context.Context, agentID string) ([2]int64, error) {
	var out [2]int64
	if s.esc == nil {
		return out, errors.New("未连接 Elasticsearch")
	}
	query := map[string]any{
		"size": 0,
		"query": map[string]any{"bool": map[string]any{"filter": []any{
			map[string]any{"term": map[string]any{"agent.id": agentID}},
			map[string]any{"term": map[string]any{"event.kind": "alert"}},
			map[string]any{"range": map[string]any{"@timestamp": map[string]any{"gte": "now-24h"}}},
		}}},
		"aggs": map[string]any{
			"blocked": map[string]any{"filter": map[string]any{"exists": map[string]any{"field": "Responses"}}},
		},
	}
	var resp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
		} `json:"hits"`
		Aggregations struct {
			Blocked struct {
				DocCount int64 `json:"doc_count"`
			} `json:"blocked"`
		} `json:"aggregations"`
	}
	if err := s.esc.Search(ctx, "logs-endpoint.alerts-default", query, &resp); err != nil {
		return out, err
	}
	out[0] = resp.Hits.Total.Value
	out[1] = resp.Aggregations.Blocked.DocCount
	return out, nil
}

func (s *Service) Threats(offset, limit int, query string, rangeKey string) (*ThreatPage, error) {
	s.mu.RLock()
	agent, esc := s.agent, s.esc
	read := s.read
	s.mu.RUnlock()
	if agent == nil || esc == nil {
		return nil, errors.New("尚未绑定本机")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	filter := []any{
		map[string]any{"term": map[string]any{"agent.id": agent.ID}},
		map[string]any{"term": map[string]any{"event.kind": "alert"}},
	}
	switch rangeKey {
	case "24h", "7d", "30d":
		filter = append(filter, map[string]any{"range": map[string]any{"@timestamp": map[string]any{"gte": "now-" + rangeKey}}})
	}
	boolQuery := map[string]any{"filter": filter}
	if strings.TrimSpace(query) != "" {
		boolQuery["must"] = []any{searchClause(
			query,
			[]string{"message", "rule.name", "process.command_line"},
			nil,
			[]string{"process.name", "process.executable", "file.path", "process.hash.sha256", "file.hash.sha256"},
		)}
	}
	body := map[string]any{
		"from":             offset,
		"size":             limit,
		"track_total_hits": true,
		"sort":             []any{map[string]any{"@timestamp": map[string]any{"order": "desc"}}},
		"query":            map[string]any{"bool": boolQuery},
		"_source": []string{
			"@timestamp", "message", "rule.name", "event.*",
			"process.name", "process.pid", "process.executable", "process.hash.sha256",
			"file.path", "file.name", "file.hash.sha256", "file.Ext.malware_*", "file.Ext.quarantine_*", "process.Ext.malware_*",
			"Responses.action.action", "Responses.result", "Responses.message", "Responses.@timestamp",
			"host.hostname",
		},
	}
	var resp model.SearchResponse
	if err := esc.Search(context.Background(), "logs-endpoint.alerts-default", body, &resp); err != nil {
		return nil, err
	}
	page := &ThreatPage{Total: resp.Hits.Total.Value, Items: []Threat{}}
	for _, hit := range resp.Hits.Hits {
		src := hit.Source
		t := Threat{
			ID:            hit.ID,
			Time:          src.Timestamp,
			Message:       src.Message,
			RuleName:      src.Rule.Name,
			Severity:      src.Event.Severity,
			RiskScore:     src.Event.RiskScore,
			Blocked:       src.Blocked(),
			Action:        src.ActionTaken(),
			ProcessName:   src.Process.Name,
			ProcessPath:   src.Process.Executable,
			FilePath:      src.File.Path,
			MalwareName:   src.MalwareName(),
			SHA256:        src.Process.Hash.SHA256,
			Read:          read[hit.ID],
			Category:      strings.Join(src.Event.Category, ", "),
			DetectionType: src.Event.Code + " " + src.Rule.Ruleset,
		}
		if t.FilePath == "" {
			t.FilePath = src.File.Name
		}
		if t.SHA256 == "" {
			t.SHA256 = src.File.Hash.SHA256
		}
		if len(src.Responses) > 0 {
			t.ResponseTime = src.Responses[0].Timestamp
		}
		page.Items = append(page.Items, t)
	}
	return page, nil
}

func (s *Service) ThreatDetail(id string) (*ThreatDetail, error) {
	s.mu.RLock()
	esc := s.esc
	read := s.read
	s.mu.RUnlock()
	if esc == nil {
		return nil, errors.New("未连接 Elasticsearch")
	}
	src, err := s.alertSource(context.Background(), esc, id)
	if err != nil {
		return nil, err
	}
	d := &ThreatDetail{Threat: Threat{
		ID: id, Time: src.Timestamp, Message: src.Message, RuleName: src.Rule.Name,
		Severity: src.Event.Severity, RiskScore: src.Event.RiskScore, Blocked: src.Blocked(),
		Action: src.ActionTaken(), ProcessName: src.Process.Name, ProcessPath: src.Process.Executable,
		FilePath: src.File.Path, MalwareName: src.MalwareName(), SHA256: src.Process.Hash.SHA256, Read: read[id],
		Category:      strings.Join(src.Event.Category, ", "),
		DetectionType: src.Event.Code + " " + src.Rule.Ruleset,
	}}
	d.CommandLine = src.Process.CommandLine
	d.WorkingDir = src.Process.WorkingDir
	d.ParentName = src.Process.Parent.Name
	d.ParentPath = src.Process.Parent.Executable
	d.ParentCmd = src.Process.Parent.CommandLine
	d.User = src.User.Name
	d.HostOS = src.Host.OS.Full
	file := scanFile(model.AlertHit{ID: id, Source: *src})
	if file.QuarantinePath != "" || src.File.Ext.QuarantineResult != nil {
		d.QuarantineFile = &file
	}
	if d.SHA256 == "" {
		d.SHA256 = src.File.Hash.SHA256
	}
	for _, r := range src.Responses {
		d.Responses = append(d.Responses, Response{Action: r.Action.Action, Message: r.Message, Timestamp: r.Timestamp})
	}
	for _, th := range src.Threat {
		if th.Tactic.Name != "" {
			d.Tactics = append(d.Tactics, th.Tactic.Name)
		}
		for _, tech := range th.Technique {
			if tech.ID != "" {
				d.Techniques = append(d.Techniques, tech.ID+" "+tech.Name)
			}
			for _, sub := range tech.Subtechnique {
				d.Techniques = append(d.Techniques, sub.ID+" "+sub.Name)
			}
		}
	}
	return d, nil
}

func (s *Service) alertSource(ctx context.Context, esc *es.Client, id string) (*model.Alert, error) {
	if esc == nil {
		return nil, errors.New("未连接 Elasticsearch")
	}
	s.mu.RLock()
	agentID := s.settings.Profile.AgentID
	s.mu.RUnlock()
	if agentID == "" {
		return nil, errors.New("尚未绑定本机")
	}
	var resp model.SearchResponse
	body := map[string]any{
		"size": 1,
		"query": map[string]any{"bool": map[string]any{"filter": []any{
			map[string]any{"ids": map[string]any{"values": []string{id}}},
			map[string]any{"term": map[string]any{"agent.id": agentID}},
		}}},
	}
	if err := esc.Search(ctx, "logs-endpoint.alerts-*", body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Hits.Hits) == 0 {
		return nil, errors.New("告警不存在或已过期")
	}
	return &resp.Hits.Hits[0].Source, nil
}

func (s *Service) ProcessChain(alertID string) (*ProcessChain, error) {
	s.mu.RLock()
	esc := s.esc
	s.mu.RUnlock()
	if esc == nil {
		return nil, errors.New("未连接 Elasticsearch")
	}
	ctx := context.Background()
	alert, err := s.alertSource(ctx, esc, alertID)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	localAgent := s.settings.Profile.AgentID
	s.mu.RUnlock()
	if localAgent == "" || alert.Agent.ID != localAgent {
		return nil, errors.New("只能查看本机进程链")
	}
	entityID := alert.Process.EntityID
	if entityID == "" {
		return nil, errors.New("这条告警没有进程实体信息，无法还原进程链")
	}
	chain := &ProcessChain{
		AlertID:   alertID,
		AlertName: alert.Rule.Name,
		AlertTime: alert.Timestamp,
		HostName:  alert.Host.Hostname,
		Nodes:     []ProcessNode{},
		Children:  []ProcessNode{},
	}
	if chain.AlertName == "" {
		chain.AlertName = alert.Message
	}

	ids := make([]string, 0, len(alert.Process.Ext.Ancestry)+2)
	seen := map[string]bool{}
	addID := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range alert.Process.Ext.Ancestry {
		addID(id)
	}
	addID(entityID)
	addID(alert.Process.Parent.EntityID)

	events := map[string]model.ProcessEvent{}
	if len(ids) > 0 {
		query := map[string]any{
			"size": len(ids) * 4,
			"sort": []any{map[string]any{"@timestamp": map[string]any{"order": "asc"}}},
			"query": map[string]any{"bool": map[string]any{"filter": []any{
				map[string]any{"terms": map[string]any{"process.entity_id": ids}},
				map[string]any{"term": map[string]any{"agent.id": alert.Agent.ID}},
			}}},
			"_source": []string{
				"@timestamp", "event.action",
				"process.name", "process.pid", "process.entity_id", "process.parent.entity_id",
				"process.executable", "process.command_line", "process.args", "process.working_directory", "process.start", "process.end",
				"user.name",
			},
		}
		var resp model.ProcessSearchResponse
		if err := esc.Search(ctx, "logs-endpoint.events.process-*", query, &resp); err != nil {
			return nil, err
		}
		for _, hit := range resp.Hits.Hits {
			ev := hit.Source
			id := ev.Process.EntityID
			if id == "" {
				continue
			}
			current, ok := events[id]
			if !ok || processEventScore(ev) < processEventScore(current) {
				events[id] = ev
			}
		}
	}

	ordered := make([]string, 0, len(ids))
	orderedSeen := map[string]bool{}
	appendOrdered := func(id string) {
		if id != "" && !orderedSeen[id] {
			orderedSeen[id] = true
			ordered = append(ordered, id)
		}
	}
	for i := len(alert.Process.Ext.Ancestry) - 1; i >= 0; i-- {
		appendOrdered(alert.Process.Ext.Ancestry[i])
	}
	if len(ordered) == 0 {
		appendOrdered(alert.Process.Parent.EntityID)
	}
	appendOrdered(entityID)

	for i, id := range ordered {
		node := ProcessNode{EntityID: id, Depth: i}
		if ev, ok := events[id]; ok {
			node = nodeFromEvent(ev, i)
			node.Resolved = true
		} else if id == entityID {
			node.Name = alert.Process.Name
			node.PID = alert.Process.PID
			node.ParentEntityID = alert.Process.Parent.EntityID
			node.Executable = alert.Process.Executable
			node.CommandLine = alert.Process.CommandLine
			node.Args = alert.Process.Args
			node.WorkingDir = alert.Process.WorkingDir
			node.User = alert.User.Name
			node.Timestamp = alert.Timestamp
			node.Resolved = true
		} else if id == alert.Process.Parent.EntityID {
			node.Name = alert.Process.Parent.Name
			node.PID = alert.Process.Parent.PID
			node.Executable = alert.Process.Parent.Executable
			node.CommandLine = alert.Process.Parent.CommandLine
			node.Resolved = node.Name != ""
		}
		node.IsAlert = id == entityID
		chain.Nodes = append(chain.Nodes, node)
	}

	childQuery := map[string]any{
		"size": 30,
		"sort": []any{map[string]any{"@timestamp": map[string]any{"order": "desc"}}},
		"query": map[string]any{"bool": map[string]any{"filter": []any{
			map[string]any{"term": map[string]any{"process.parent.entity_id": entityID}},
			map[string]any{"term": map[string]any{"agent.id": alert.Agent.ID}},
		}}},
		"_source": []string{
			"@timestamp", "event.action",
			"process.name", "process.pid", "process.entity_id", "process.parent.entity_id",
			"process.executable", "process.command_line", "process.args", "process.working_directory", "process.start", "process.end",
			"user.name",
		},
	}
	var childResp model.ProcessSearchResponse
	if err := esc.Search(ctx, "logs-endpoint.events.process-*", childQuery, &childResp); err == nil {
		childSeen := map[string]bool{}
		for _, hit := range childResp.Hits.Hits {
			ev := hit.Source
			id := ev.Process.EntityID
			if id == "" || childSeen[id] {
				continue
			}
			childSeen[id] = true
			node := nodeFromEvent(ev, 0)
			node.Resolved = true
			chain.Children = append(chain.Children, node)
			if len(chain.Children) >= 12 {
				break
			}
		}
	} else {
		chain.Notice = "子进程查询失败：" + err.Error()
	}
	return chain, nil
}

func nodeFromEvent(ev model.ProcessEvent, depth int) ProcessNode {
	return ProcessNode{
		WorkingDir:     ev.Process.WorkingDir,
		EntityID:       ev.Process.EntityID,
		Name:           ev.Process.Name,
		PID:            ev.Process.PID,
		ParentEntityID: ev.Process.Parent.EntityID,
		Executable:     ev.Process.Executable,
		CommandLine:    ev.Process.CommandLine,
		Args:           ev.Process.Args,
		User:           ev.User.Name,
		Timestamp:      ev.Timestamp,
		Start:          ev.Process.Start,
		End:            ev.Process.End,
		EventAction:    strings.Join([]string(ev.Event.Action), ", "),
		Depth:          depth,
	}
}

func processEventScore(ev model.ProcessEvent) int {
	for _, action := range ev.Event.Action {
		switch action {
		case "start":
			return 0
		case "already_running":
			return 1
		case "end":
			return 3
		}
	}
	return 2
}

func (s *Service) MarkRead(ids []string) error {
	s.mu.Lock()
	for _, id := range ids {
		s.read[id] = true
	}
	s.mu.Unlock()
	return s.saveLocalState()
}

func (s *Service) StartScan(path string) (*ScanTask, error) {
	s.mu.RLock()
	agent, kb := s.agent, s.kb
	s.mu.RUnlock()
	if agent == nil || kb == nil {
		return nil, errors.New("尚未绑定本机")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("扫描路径不能为空")
	}
	if strings.HasPrefix(path, `\\`) {
		return nil, errors.New("Elastic Defend 无法扫描网络路径")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absolute); err != nil {
		return nil, fmt.Errorf("路径不可访问: %w", err)
	}
	action, err := kb.Scan(context.Background(), agent.ID, absolute, "本机控制台发起的扫描")
	if err != nil {
		if apiErr, ok := err.(*kibana.APIError); ok && apiErr.Kind == "waf" {
			s.audit("scan", absolute, "failed", "waf")
			return nil, errors.New("扫描请求被 WAF 拦截（403）。请检查路径是否包含 ../、/etc/passwd、win.ini 等被 WAF 标记的特征")
		}
		s.audit("scan", absolute, "failed", err.Error())
		return nil, err
	}
	task := &ScanTask{
		ID:        fmt.Sprintf("scan-%d", time.Now().UnixNano()),
		ActionID:  action.ID,
		Path:      absolute,
		Status:    action.Status,
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if action.StartedAt != "" {
		task.StartedAt = action.StartedAt
	}
	s.audit("scan", absolute, "submitted", action.ID)
	s.mu.Lock()
	s.scans = append([]*ScanTask{task}, s.scans...)
	if len(s.scans) > 50 {
		s.scans = s.scans[:50]
	}
	s.mu.Unlock()
	_ = s.saveLocalState()
	go s.pollScan(task.ID, action.ID)
	return task, nil
}

func (s *Service) pollScan(taskID, actionID string) {
	s.mu.RLock()
	kb := s.kb
	s.mu.RUnlock()
	if kb == nil {
		return
	}
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		action, err := kb.GetAction(context.Background(), actionID)
		s.mu.Lock()
		var task *ScanTask
		for _, t := range s.scans {
			if t.ID == taskID {
				task = t
				break
			}
		}
		if task == nil {
			s.mu.Unlock()
			return
		}
		if err == nil {
			task.Status = action.Status
			if action.Status == "successful" || action.Status == "failed" || action.IsExpired {
				completedAt := time.Now().Format(time.RFC3339)
				if action.CompletedAt != "" {
					completedAt = action.CompletedAt
				}
				task.CompletedAt = completedAt
				if action.IsExpired {
					task.Status = "expired"
				}
				path := task.Path
				result := task.Status
				s.mu.Unlock()

				_, _ = s.ScanResults(taskID, 0)
				_ = s.saveLocalState()
				s.audit("scan", path, result, actionID)
				return
			}
		} else {
			task.Detail = err.Error()
		}
		s.mu.Unlock()
		_ = s.saveLocalState()
	}
}

func (s *Service) Scans() []*ScanTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*ScanTask, 0, len(s.scans))
	for _, task := range s.scans {
		copy := *task
		items = append(items, &copy)
	}
	return items
}

func (s *Service) Isolate(confirmHost string) (*StatusView, error) {
	s.mu.RLock()
	agent, kb, hostName := s.agent, s.kb, s.settings.Profile.HostName
	s.mu.RUnlock()
	if agent == nil || kb == nil {
		return nil, errors.New("尚未绑定本机")
	}
	if !strings.EqualFold(strings.TrimSpace(confirmHost), hostName) {
		return nil, errors.New("主机名不匹配，未执行隔离")
	}
	action, err := kb.Isolate(context.Background(), agent.ID, "本机应急隔离")
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.isolate = action.ID
	s.mu.Unlock()
	deadline := time.Now().Add(300 * time.Second)
	status := "pending"
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		updated, err := kb.GetAction(context.Background(), action.ID)
		if err == nil {
			status = updated.Status
			if updated.Status == "successful" || updated.Status == "failed" || updated.IsExpired {
				break
			}
		}
	}
	s.audit("isolate", hostName, status, action.ID)
	_ = s.saveLocalState()
	return s.Status()
}

func (s *Service) TrustItems() ([]TrustItem, error) {
	s.mu.RLock()
	kb, packagePolicyID := s.kb, s.settings.Profile.PackagePolicyID
	s.mu.RUnlock()
	if kb == nil {
		return nil, errors.New("尚未连接 Kibana")
	}
	items, err := kb.FindExceptionItems(context.Background(), "endpoint_trusted_apps", "")
	if err != nil {
		return nil, err
	}
	policyTag := kibana.PolicyTag(packagePolicyID)
	out := []TrustItem{}
	for _, item := range items {
		scope := ""
		global := false
		for _, tag := range item.Tags {
			if tag == "policy:all" {
				global = true
			} else if tag == policyTag {
				scope = "本机策略"
			}
		}
		if global {
			scope = "全局"
		}
		if scope == "" {
			continue
		}
		trust := TrustItem{
			ID: item.ID, ItemID: item.ItemID, Name: item.Name, Description: item.Description,
			Scope: scope, ReadOnly: scope == "全局", CreatedBy: item.CreatedBy, UpdatedAt: item.UpdatedAt,
			OS: strings.Join(item.OSTypes, ", "),
		}
		trust.MatchKey = trustedAppMatchKey(item)
		if len(item.Entries) > 0 {
			entry := item.Entries[0]
			trust.Value = fmt.Sprint(entry.Value)
			if strings.Contains(entry.Field, "hash") {
				trust.Kind = "SHA256"
			} else if entry.Type == "wildcard" {
				trust.Kind = "路径通配"
			} else {
				trust.Kind = "路径"
			}
		}
		out = append(out, trust)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}

func (s *Service) CreateTrust(in TrustInput) error {
	s.trustMu.Lock()
	defer s.trustMu.Unlock()
	s.mu.RLock()
	kb, packagePolicyID := s.kb, s.settings.Profile.PackagePolicyID
	s.mu.RUnlock()
	if kb == nil {
		return errors.New("尚未连接 Kibana")
	}
	if packagePolicyID == "" {
		return errors.New("尚未绑定本机策略")
	}
	value := strings.TrimSpace(in.Value)
	if value == "" {
		return errors.New("条件不能为空")
	}
	if err := validateTrustInput(in.Kind, value); err != nil {
		return err
	}
	existing, err := kb.FindExceptionItems(context.Background(), "endpoint_trusted_apps", "")
	if err != nil {
		return fmt.Errorf("检查已有规则：%w", err)
	}
	key := trustRuleKey(in.Kind, value)
	for _, item := range existing {
		if trustAppliesToPolicy(item, packagePolicyID) && trustedAppMatchKey(item) == key {
			return errors.New("重复规则：本机策略或全局已存在相同条件")
		}
	}
	var entry kibana.ExceptionEntry
	switch in.Kind {
	case "hash":
		entry = kibana.ExceptionEntry{Type: "match", Field: "process.hash.sha256", Value: strings.ToLower(value), Operator: "included"}
	case "wildcard":
		entry = kibana.ExceptionEntry{Type: "wildcard", Field: "process.executable.caseless", Value: value, Operator: "included"}
	default:
		entry = kibana.ExceptionEntry{Type: "match", Field: "process.executable.caseless", Value: value, Operator: "included"}
	}
	name := "Trusted: " + value
	if len(name) > 120 {
		name = name[:120]
	}
	created, err := kb.CreateExceptionItem(
		context.Background(),
		"endpoint_trusted_apps",
		name,
		in.Description,
		[]kibana.ExceptionEntry{entry},
		[]string{"windows"},
		[]string{kibana.PolicyTag(packagePolicyID)},
	)
	if err != nil {
		s.audit("trust_create", value, "failed", err.Error())
		return err
	}
	itemID := ""
	if created != nil {
		itemID = created.ItemID
	}
	s.audit("trust_create", value, "success", itemID)
	return nil
}

func (s *Service) DeleteTrust(id string) error {
	s.trustMu.Lock()
	defer s.trustMu.Unlock()
	s.mu.RLock()
	kb, packagePolicyID := s.kb, s.settings.Profile.PackagePolicyID
	s.mu.RUnlock()
	if kb == nil {
		return errors.New("尚未连接 Kibana")
	}
	if packagePolicyID == "" {
		return errors.New("尚未绑定本机策略")
	}
	items, err := kb.FindExceptionItems(context.Background(), "endpoint_trusted_apps", "")
	if err != nil {
		return err
	}
	localTag := kibana.PolicyTag(packagePolicyID)
	for _, item := range items {
		if item.ID != id {
			continue
		}
		isLocal := false
		for _, tag := range item.Tags {
			if tag == "policy:all" {
				return errors.New("全局排除项来自 Kibana，本机控制台不能删除")
			}
			if tag == localTag {
				isLocal = true
			}
		}
		if !isLocal {
			return errors.New("全局排除项来自 Kibana，本机控制台不能删除")
		}
		if err := kb.DeleteExceptionItem(context.Background(), item); err != nil {
			s.audit("trust_delete", item.Name, "failed", err.Error())
			return err
		}
		s.audit("trust_delete", item.Name, "success", item.ItemID)
		return nil
	}
	return errors.New("排除项不存在或已过期")
}

func (s *Service) Disconnect() error {
	s.mu.Lock()
	s.kb = nil
	s.esc = nil
	s.agent = nil
	s.meta = nil
	s.policy = nil
	s.policyV = nil
	s.settings.Profile = config.Profile{}
	s.mu.Unlock()
	_ = config.DeleteSecret("password")
	_ = config.DeleteSecret("apikey")
	return s.settings.Save()
}

func (s *Service) localStatePath() string {
	dir, err := config.Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "local-state.json")
}

type localState struct {
	Scans []*ScanTask     `json:"scans"`
	Read  map[string]bool `json:"read"`
}

func (s *Service) loadLocalState() {
	path := s.localStatePath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var state localState
	if json.Unmarshal(data, &state) != nil {
		return
	}
	if state.Scans != nil {
		s.scans = state.Scans
	}
	if state.Read != nil {
		s.read = state.Read
	}
}

func (s *Service) saveLocalState() error {
	path := s.localStatePath()
	if path == "" {
		return nil
	}
	s.mu.RLock()
	state := localState{Scans: s.scans, Read: s.read}
	data, err := json.MarshalIndent(state, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *Service) Rebind() error {
	if s.kb == nil {
		return errors.New("尚未连接 Kibana")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, _, err := s.bindLocal(ctx)
	return err
}

func (s *Service) KibanaURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.Profile.KibanaURL
}

func (s *Service) agentRuntimeStatus() agentstatus.Status {
	s.mu.RLock()
	cache := s.agentRuntimeCache
	at := s.agentRuntimeAt
	s.mu.RUnlock()
	if !at.IsZero() && time.Since(at) < 10*time.Second {
		return cache
	}
	status := agentstatus.Check()
	s.mu.Lock()
	s.agentRuntimeCache = status
	s.agentRuntimeAt = time.Now()
	s.mu.Unlock()
	return status
}

func localIPSet() map[string]bool {
	out := map[string]bool{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				out[ipnet.IP.String()] = true
			}
		}
	}
	return out
}

func modeText(mode string) string {
	switch mode {
	case "prevent":
		return "阻止"
	case "detect":
		return "仅检测"
	case "off":
		return "关闭"
	case "":
		return "未知"
	default:
		return mode
	}
}

func deviceControlText(enabled bool, mode string) string {
	if !enabled {
		return "关闭"
	}
	if mode == "deny_all" {
		return "全部阻止"
	}
	return "已启用"
}

func buildChecks(v *StatusView) []Check {
	var checks []Check
	checks = append(checks, Check{Name: "Agent 在线", OK: v.AgentStatus == "online", Detail: v.AgentStatus})
	checks = append(checks, Check{Name: "Elastic Defend 已启用", OK: v.EndpointStatus == "enrolled", Detail: v.EndpointStatus})
	allPrevent := v.Protection.Malware == "阻止" && v.Protection.Ransomware == "阻止" && v.Protection.Memory == "阻止" && v.Protection.Behavior == "阻止"
	checks = append(checks, Check{Name: "四项防护处于阻止模式", OK: allPrevent, Detail: fmt.Sprintf("恶意软件 %s / 勒索 %s / 内存 %s / 行为 %s", v.Protection.Malware, v.Protection.Ransomware, v.Protection.Memory, v.Protection.Behavior)})
	checks = append(checks, Check{Name: "防篡改保护", OK: v.TamperProtection, Detail: boolText(v.TamperProtection, "已开启", "未开启")})
	updateDetail := v.ProtectionUpdateVersion
	if updateDetail == "" {
		updateDetail = "未读取到"
	}
	checks = append(checks, Check{Name: "防护更新", OK: v.ProtectionUpdateVersion != "" && v.ProtectionUpdateAge <= 48, Detail: updateDetail})
	checks = append(checks, Check{Name: "最近 24 小时有事件", OK: v.Alerts24h > 0, Detail: fmt.Sprintf("%d 条告警", v.Alerts24h)})
	return checks
}

func boolText(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}
