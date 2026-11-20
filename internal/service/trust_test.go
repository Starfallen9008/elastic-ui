package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elastic-ui/internal/auth"
	"elastic-ui/internal/config"
	"elastic-ui/internal/kibana"
)

func TestValidateTrustInput(t *testing.T) {
	tests := []struct {
		kind  string
		value string
		valid bool
	}{
		{kind: "path", value: `C:\App\app.exe`, valid: true},
		{kind: "path", value: `\\server\share\app.exe`, valid: true},
		{kind: "wildcard", value: `C:\Tools\*`, valid: true},
		{kind: "hash", value: strings.Repeat("a", 64), valid: true},
		{kind: "hash", value: strings.Repeat("z", 64)},
		{kind: "path", value: `relative.exe`},
		{kind: "path", value: `C:\Tools\*`},
		{kind: "path", value: `C:\Tools\`},
		{kind: "path", value: `C:\Tools\app.exe:stream`},
		{kind: "wildcard", value: `C:\app.exe`},
		{kind: "unknown", value: `C:\app.exe`},
	}
	for _, tt := range tests {
		if got := validateTrustInput(tt.kind, tt.value); (got == nil) != tt.valid {
			t.Errorf("validateTrustInput(%q, %q) = %v", tt.kind, tt.value, got)
		}
	}
}

func TestTrustMutationsRejectDuplicateAndGlobalDelete(t *testing.T) {
	writes := 0
	item := kibana.ExceptionListItem{
		ID: "global", OSTypes: []string{"windows"}, Tags: []string{"policy:local", "policy:all"},
		Entries: []kibana.ExceptionEntry{{
			Type: "match", Field: "process.executable.caseless", Value: `C:\App.exe`, Operator: "included",
		}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes++
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []kibana.ExceptionListItem{item}, "total": 1})
	}))
	defer server.Close()
	client, err := kibana.New(server.URL, auth.Auth{}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	settings := config.Default()
	settings.Profile.PackagePolicyID = "local"
	s := &Service{settings: settings, kb: client}
	if err := s.CreateTrust(TrustInput{Kind: "path", Value: `c:\APP.EXE`}); err == nil || !strings.Contains(err.Error(), "重复规则") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
	if err := s.DeleteTrust("global"); err == nil {
		t.Fatal("global item must be read-only even with a local tag")
	}
	items, err := s.TrustItems()
	if err != nil || len(items) != 1 || !items[0].ReadOnly {
		t.Fatalf("global scope projection: %v, %v", items, err)
	}
	settings.Profile.PackagePolicyID = ""
	if err := s.DeleteTrust("global"); err == nil {
		t.Fatal("unbound policy delete must fail")
	}
	if writes != 0 {
		t.Fatalf("unexpected writes: %d", writes)
	}
}

func TestTrustedAppMatchKey(t *testing.T) {
	entry := kibana.ExceptionEntry{Type: "match", Field: "process.executable.caseless", Value: `C:\App.exe`, Operator: "included"}
	item := kibana.ExceptionListItem{OSTypes: []string{"windows"}, Entries: []kibana.ExceptionEntry{entry}, Tags: []string{"policy:local"}}
	if got := trustedAppMatchKey(item); got != `path:c:\app.exe` {
		t.Fatal(got)
	}
	if !trustAppliesToPolicy(item, "local") || trustAppliesToPolicy(item, "other") {
		t.Fatal("incorrect scope")
	}
	item.Entries = append(item.Entries, entry)
	if trustedAppMatchKey(item) != "" {
		t.Fatal("compound rule is not equivalent")
	}
	item.Entries = []kibana.ExceptionEntry{entry}
	item.OSTypes = []string{"linux"}
	if trustedAppMatchKey(item) != "" {
		t.Fatal("non-Windows rule is not equivalent")
	}
	item.OSTypes = []string{"windows"}
	item.Entries[0].Operator = "excluded"
	if trustedAppMatchKey(item) != "" {
		t.Fatal("excluded condition is not equivalent")
	}
}

func TestTrustMutationsUseScopedKibanaRequests(t *testing.T) {
	// Keep audit output out of the user's live configuration directory.
	t.Setenv("APPDATA", t.TempDir())
	created := false
	deleted := false
	item := kibana.ExceptionListItem{ID: "local-item", Tags: []string{"policy:local"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows := []kibana.ExceptionListItem{}
			if created {
				rows = append(rows, item)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": rows, "total": len(rows)})
		case http.MethodPost:
			var input kibana.ExceptionListItem
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if input.ListID != "endpoint_trusted_apps" || len(input.Tags) != 1 || input.Tags[0] != "policy:local" {
				t.Errorf("incorrect list/scope: %+v", input)
			}
			if len(input.Entries) != 1 || input.Entries[0].Value != `C:\Test.exe` {
				t.Errorf("incorrect conditions: %+v", input.Entries)
			}
			created = true
			_ = json.NewEncoder(w).Encode(item)
		case http.MethodDelete:
			if r.URL.Query().Get("id") != item.ID {
				t.Error("wrong delete id")
			}
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	client, err := kibana.New(server.URL, auth.Auth{}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	settings := config.Default()
	settings.Profile.PackagePolicyID = "local"
	s := &Service{settings: settings, kb: client}
	if err := s.CreateTrust(TrustInput{Kind: "path", Value: `C:\Test.exe`, Description: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTrust(item.ID); err != nil {
		t.Fatal(err)
	}
	if !created || !deleted {
		t.Fatal("Kibana write methods not called")
	}
}
