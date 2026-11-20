package kibana

import (
	"context"
	"elastic-ui/internal/auth"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIndependentPolicyCopiesBeforeReassigningOnlyLocalAgent(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(map[bool]string{false: "independent-copy", true: "shared-rejected"}[shared], func(t *testing.T) {
			copies, reassigns := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/api/fleet/agents/local/reassign":
					reassigns++
					if copies != 1 {
						t.Error("reassigned before copy")
					}
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					if body["policy_id"] != "copy" {
						t.Error(body)
					}
					w.Write([]byte(`{}`))
				case r.URL.Path == "/api/fleet/agents/local":
					json.NewEncoder(w).Encode(map[string]any{"item": map[string]any{"id": "local", "policy_id": "original"}})
				case strings.HasSuffix(r.URL.Path, "/copy") && r.Method == http.MethodPost:
					copies++
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					if body["name"] != "My PC" {
						t.Error(body)
					}
					w.Write([]byte(`{"item":{"id":"copy"}}`))
				default:
					id, pkgID := "original", "old-endpoint"
					if r.URL.Path == "/api/fleet/agent_policies/copy" {
						id, pkgID = "copy", "new-endpoint"
					}
					ids := []string{id}
					if shared {
						ids = append(ids, "another")
					}
					json.NewEncoder(w).Encode(map[string]any{"item": map[string]any{"id": id, "package_policies": []any{map[string]any{"id": pkgID, "policy_ids": ids, "package": map[string]any{"name": "endpoint"}}, map[string]any{"id": "system-" + id, "policy_ids": []string{id}, "package": map[string]any{"name": "system"}}}}})
				}
			}))
			defer server.Close()
			c, _ := New(server.URL, auth.Auth{}, false, "")
			policy, err := c.CreateIndependentPolicy(context.Background(), "local", "My PC")
			if shared {
				if err == nil || copies != 0 || reassigns != 0 {
					t.Fatalf("err=%v copies=%d reassigns=%d", err, copies, reassigns)
				}
			} else if err != nil || policy.ID != "copy" || reassigns != 1 {
				t.Fatalf("policy=%+v err=%v reassigns=%d", policy, err, reassigns)
			}
		})
	}
}

func TestRenameIndependentPolicyPreservesNamespaceAndTimeout(t *testing.T) {
	updated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/fleet/agents/local":
			w.Write([]byte(`{"item":{"id":"local","policy_id":"private"}}`))
		case r.URL.Path == "/api/fleet/agents":
			w.Write([]byte(`{"items":[{"id":"local"}],"total":1}`))
		case r.Method == http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["name"] != "New name" || body["namespace"] != "custom" || body["inactivity_timeout"] != float64(86400) {
				t.Errorf("unexpected update: %+v", body)
			}
			updated = true
			w.Write([]byte(`{}`))
		default:
			w.Write([]byte(`{"item":{"id":"private","name":"Original","namespace":"custom","inactivity_timeout":86400,"package_policies":[{"id":"defend","policy_ids":["private"],"package":{"name":"endpoint"}}]}}`))
		}
	}))
	defer server.Close()
	c, err := New(server.URL, auth.Auth{}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RenameIndependentPolicy(context.Background(), "local", "New name"); err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("policy was not renamed")
	}
}
