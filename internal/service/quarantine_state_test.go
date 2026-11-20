package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"elastic-ui/internal/auth"
	"elastic-ui/internal/es"
	"elastic-ui/internal/kibana"
)

func TestQuarantineRecoverySurvivesRestartAndRetry(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	s := disconnectedTestService()
	if err := s.recordQuarantineRecovery("local", `C:\.equarantine\fixture`, `D:\recovered.zip`, nil); err != nil {
		t.Fatal(err)
	}
	s = disconnectedTestService()
	if err := s.recordQuarantineRecovery("local", `C:\.equarantine\fixture`, "", errors.New("retry failed")); err != nil {
		t.Fatal(err)
	}
	state := readQuarantineState()[quarantineKey("local", `c:\.EQUARANTINE\FIXTURE`)]
	if state.SavedPath != `D:\recovered.zip` || state.RecoveredAt == "" || state.LastError != "retry failed" {
		t.Fatalf("lost recovery: %+v", state)
	}
	if _, exists := readQuarantineState()[quarantineKey("other", `C:\.equarantine\fixture`)]; exists {
		t.Fatal("shared recovery across hosts")
	}
	for _, path := range []string{`C:\Windows\test.exe`, `\\server\share\.equarantine\test`, `C:\.equarantine\..\Windows\test.exe`, `relative`} {
		if quarantineAvailability(path) != "unknown" {
			t.Fatalf("inspected outside quarantine: %s", path)
		}
	}
}

func TestQuarantineCollapsesFilesAndRefreshesTrust(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "_search") {
			var query map[string]any
			_ = json.NewDecoder(r.Body).Decode(&query)
			if query["collapse"].(map[string]any)["field"] != "file.Ext.quarantine_path" {
				t.Error("missing file collapse")
			}
			_, _ = w.Write([]byte(`{"hits":{"total":{"value":8},"hits":[{"_id":"latest","_source":{"file":{"hash":{"sha256":"` + strings.Repeat("a", 64) + `"},"Ext":{"quarantine_result":true,"quarantine_path":"C:\\.equarantine\\fixture"}}}}]},"aggregations":{"unique_files":{"value":1}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"trusted","os_types":["windows"],"tags":["policy:local-policy"],"entries":[{"field":"process.hash.sha256","type":"match","operator":"included","value":"` + strings.Repeat("a", 64) + `"}]}],"total":1}`))
	}))
	defer server.Close()
	s := disconnectedTestService()
	s.agent = &kibana.Agent{ID: "local"}
	s.settings.Profile.PackagePolicyID = "local-policy"
	s.esc, _ = es.New(server.URL, auth.Auth{}, false)
	s.kb, _ = kibana.New(server.URL, auth.Auth{}, false, "")
	r, err := s.Quarantine(0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Total != 1 || len(r.Items) != 1 || r.Items[0].TrustStatus != "configured" {
		t.Fatalf("wrong live status: %+v", r)
	}
}
