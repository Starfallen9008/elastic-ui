package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elastic-ui/internal/agentstatus"
	"elastic-ui/internal/auth"
	"elastic-ui/internal/config"
	"elastic-ui/internal/kibana"
)

func disconnectedTestService() *Service {
	settings := config.Default()
	settings.Profile.HostName = "local-host"
	return &Service{
		settings:          settings,
		agentRuntimeCache: agentstatus.Status{State: "HEALTHY", Message: "Running", Raw: "local runtime output"},
		agentRuntimeAt:    time.Now(),
	}
}

func TestDisconnectedStatusKeepsErrorAndLocalHealth(t *testing.T) {
	s := disconnectedTestService()
	view, err := s.Status()
	if err != nil {
		t.Fatal(err)
	}
	if view.Connected || view.ConnectionError == "" || view.ProtectionError == "" {
		t.Fatalf("missing failure details: %+v", view)
	}
	if view.HostName != "local-host" || view.AgentRuntime.State != "HEALTHY" || view.AgentRuntime.Raw == "" {
		t.Fatalf("local state lost: %+v", view)
	}
	if len(view.Checks) != 3 || view.Checks[0].OK || !view.Checks[1].OK || view.Checks[2].OK {
		t.Fatalf("misleading checks: %+v", view.Checks)
	}
}

func TestDetectOnlyRequiresAllFourModulesInDetectMode(t *testing.T) {
	var policy kibana.OSProtection
	policy.Malware.Mode = "detect"
	policy.Ransomware.Mode = "detect"
	policy.MemoryProtection.Mode = "detect"
	policy.BehaviorProtection.Mode = "detect"
	if protection, detectOnly := protectionFromPolicy(policy); !detectOnly || protection.Malware != "仅检测" {
		t.Fatalf("four detect modes not recognized: %+v %v", protection, detectOnly)
	}
	policy.BehaviorProtection.Mode = "prevent"
	if _, detectOnly := protectionFromPolicy(policy); detectOnly {
		t.Fatal("mixed detect/prevent reported as detect-only")
	}
	policy.BehaviorProtection.Mode = ""
	if _, detectOnly := protectionFromPolicy(policy); detectOnly {
		t.Fatal("unknown mode reported as detect-only")
	}
}

func TestStatusRefreshFailureDoesNotLeaveHealthyRemoteSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"fixture unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, err := kibana.New(server.URL, auth.Auth{}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	s := disconnectedTestService()
	s.kb = client
	s.agent = &kibana.Agent{ID: "local"}
	s.settings.Profile.AgentID = "local"
	view, err := s.Status()
	if err != nil {
		t.Fatal(err)
	}
	if view.Connected || !strings.Contains(view.ConnectionError, "fixture unavailable") {
		t.Fatalf("refresh failure hidden: %+v", view)
	}
	if view.AgentRuntime.State != "HEALTHY" || s.ConnectError() == nil {
		t.Fatal("missing local health or stored error")
	}
}
