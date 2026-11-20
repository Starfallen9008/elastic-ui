package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"elastic-ui/internal/auth"
	"elastic-ui/internal/config"
	"elastic-ui/internal/kibana"
)

func TestUINotificationsWaitForAppliedPolicyAndSuppressNativeOverlap(t *testing.T) {
	for _, test := range []struct {
		name     string
		revision int
		status   string
		native   bool
		ready    bool
	}{
		{name: "waiting", revision: 4, status: "success"},
		{name: "failure", revision: 5, status: "failure"},
		{name: "applied", revision: 5, status: "success", ready: true},
		{name: "native reopened", revision: 5, status: "success", native: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/endpoint/metadata/local" {
					json.NewEncoder(w).Encode(map[string]any{"metadata": map[string]any{"Endpoint": map[string]any{"policy": map[string]any{"applied": map[string]any{
						"id": "defend", "version": 3, "endpoint_policy_version": test.revision, "status": test.status,
					}}}}})
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"item": map[string]any{"inputs": []any{map[string]any{"config": map[string]any{"policy": map[string]any{"value": map[string]any{
					"windows": map[string]any{"popup": map[string]any{"malware": map[string]any{"enabled": test.native}}},
				}}}}}}})
			}))
			defer server.Close()
			kb, _ := kibana.New(server.URL, auth.Auth{}, false, "")
			settings := config.Default()
			settings.NotificationMode = "ui"
			settings.NotificationRevision = 5
			settings.Profile.PackagePolicyID = "defend"
			s := &Service{settings: settings, kb: kb, agent: &kibana.Agent{ID: "local"}}
			state := s.NotificationState()
			if state.Ready != test.ready || state.Message == "" {
				t.Fatalf("unexpected state: %+v", state)
			}
		})
	}
}
