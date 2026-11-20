package kibana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"elastic-ui/internal/auth"
)

func notificationPolicy() map[string]any {
	return map[string]any{
		"id": "defend", "name": "local", "namespace": "default", "enabled": true,
		"policy_ids": []any{"local-policy"}, "revision": float64(4), "updated_at": "server-only",
		"package": map[string]any{"name": "endpoint", "version": "9.3.0"},
		"inputs": []any{map[string]any{
			"type": "endpoint", "enabled": true, "streams": []any{},
			"config": map[string]any{
				"artifact_manifest": map[string]any{"value": map[string]any{"version": "latest"}},
				"policy": map[string]any{"value": map[string]any{
					"windows": map[string]any{
						"malware":  map[string]any{"mode": "prevent", "blocklist": true},
						"advanced": map[string]any{"unknown.setting": true},
						"popup": map[string]any{
							"malware":             map[string]any{"enabled": true, "message": "keep message"},
							"ransomware":          map[string]any{"enabled": true},
							"memory_protection":   map[string]any{"enabled": true},
							"behavior_protection": map[string]any{"enabled": true},
							"device_control":      map[string]any{"enabled": true},
						},
					},
					"mac": map[string]any{"popup": map[string]any{"malware": map[string]any{"enabled": true}}},
				}},
			},
		}},
	}
}

func TestNativeNotificationWritePreservesProtectionAndUnknownFields(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "ui", true: "native"}[enabled], func(t *testing.T) {
			original := notificationPolicy()
			expected := notificationPolicy()
			if err := editPopupFlags(expected, enabled); err != nil {
				t.Fatal(err)
			}
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/fleet/agents" {
					if r.URL.Query().Get("kuery") != `fleet-agents.policy_id:"local-policy"` {
						t.Error("missing scope filter")
					}
					json.NewEncoder(w).Encode(map[string]any{"total": 1, "items": []any{map[string]any{"id": "local-agent"}}})
					return
				}
				if r.Method == http.MethodGet {
					json.NewEncoder(w).Encode(map[string]any{"item": original})
					return
				}
				if r.Method != http.MethodPut {
					t.Error("unexpected method", r.Method)
				}
				writes++
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if _, exists := body["updated_at"]; exists {
					t.Error("server metadata included")
				}
				if !reflect.DeepEqual(body["inputs"], expected["inputs"]) {
					t.Error("protection or unknown settings changed")
				}
				json.NewEncoder(w).Encode(map[string]any{"item": map[string]any{"id": "defend", "revision": 5}})
			}))
			defer server.Close()
			client, _ := New(server.URL, auth.Auth{}, false, "")
			revision, err := client.SetNativeNotifications(context.Background(), "defend", "local-agent", enabled)
			if err != nil || revision != 5 || writes != 1 {
				t.Fatalf("revision=%d writes=%d err=%v", revision, writes, err)
			}
		})
	}
}

func TestNotificationSwitchRejectsSharedPolicyBeforeWrite(t *testing.T) {
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes++
			t.Error("shared policy mutated")
		}
		if r.URL.Path == "/api/fleet/agents" {
			json.NewEncoder(w).Encode(map[string]any{"total": 2})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"item": notificationPolicy()})
	}))
	defer server.Close()
	client, _ := New(server.URL, auth.Auth{}, false, "")
	_, err := client.SetNativeNotifications(context.Background(), "defend", "local-agent", false)
	if err == nil || writes != 0 {
		t.Fatalf("writes=%d err=%v", writes, err)
	}
}
