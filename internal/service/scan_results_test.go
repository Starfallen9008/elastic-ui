package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"elastic-ui/internal/auth"
	"elastic-ui/internal/es"
	"elastic-ui/internal/kibana"
	"elastic-ui/internal/model"
	"github.com/yeka/zip"
)

func TestScanFileUsesQuarantineResultAndNestedSignature(t *testing.T) {
	for _, tc := range []struct {
		raw, outcome string
		recoverable  bool
	}{
		{`{"file":{"path":"C:\\bad.exe","Ext":{"quarantine_result":true,"quarantine_path":"C:\\.equarantine\\id","malware_signature":{"primary":{"signature":{"name":"Windows.VulnDriver.ComputerZ"}}}}}}`, "已隔离", true},
		{`{"file":{"Ext":{"quarantine_result":false,"quarantine_path":"C:\\.equarantine\\id"}},"event":{"type":["denied"]}}`, "隔离失败", false},
		{`{"file":{"Ext":{"quarantine_path":"C:\\.equarantine\\id"}}}`, "仅检测", false},
		{`{"Responses":[{"result":-1,"action":{"action":"delete_file"}}]}`, "仅检测", false},
		{`{"Responses":[{"result":0,"action":{"action":"delete_file"}}]}`, "已删除", false},
	} {
		var hit model.AlertHit
		if err := json.Unmarshal([]byte(tc.raw), &hit.Source); err != nil {
			t.Fatal(err)
		}
		got := scanFile(hit)
		if got.Outcome != tc.outcome || got.Recoverable != tc.recoverable {
			t.Fatalf("%s: %+v", tc.raw, got)
		}
		if tc.recoverable && got.MalwareName != "Windows.VulnDriver.ComputerZ" {
			t.Fatalf("missing nested signature: %+v", got)
		}
	}
}

func TestScanResultsFilterHostDemandTimeAndEscapedPath(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var query map[string]any
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			t.Error(err)
		}
		data, _ := json.Marshal(query)
		for _, want := range []string{`"agent.id":"local"`, `"event.action":"demand"`, `"gte":"2026-11-19T11:00:00Z"`, `"lte":"2026-11-19T11:30:00Z"`, `"from":50`, `"case_insensitive":true`} {
			if !strings.Contains(string(data), want) {
				t.Errorf("missing %s in %s", want, data)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"total":{"value":51},"hits":[{"_id":"file","_source":{"file":{"path":"C:\\Downloads\\a.exe"}}}]}}`))
	}))
	defer server.Close()
	client, _ := es.New(server.URL, auth.Auth{}, false)
	s := disconnectedTestService()
	s.esc = client
	s.agent = &kibana.Agent{ID: "local"}
	s.scans = []*ScanTask{{ID: "scan", Path: `C:\Downloads`, StartedAt: "2026-11-19T11:00:00Z", CompletedAt: "2026-11-19T11:30:00Z"}}
	r, err := s.ScanResults("scan", 50)
	if err != nil {
		t.Fatal(err)
	}
	if r.Total != 51 || len(r.Items) != 1 || s.scans[0].Found != 51 {
		t.Fatalf("%+v", r)
	}
	if _, err := s.ScanResults("missing", 0); err == nil {
		t.Fatal("accepted unknown scan")
	}
	filter := scanPathFilter(`C:\foo*?`)["bool"].(map[string]any)["should"].([]any)
	pattern := filter[1].(map[string]any)["wildcard"].(map[string]any)["file.path"].(map[string]any)["value"]
	if pattern != `C:\\foo\*\?\\*` {
		t.Fatalf("unescaped path: %s", pattern)
	}
}

func TestRecoveredArchiveValidatesOriginalHash(t *testing.T) {
	var data bytes.Buffer
	w := zip.NewWriter(&data)
	f, err := w.Encrypt("file.exe", "elastic", zip.StandardEncryption)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("recovery fixture"))
	_ = w.Close()
	digest := sha256.Sum256([]byte("recovery fixture"))
	if err := validateRecoveredArchive(data.Bytes(), hex.EncodeToString(digest[:])); err != nil {
		t.Fatal(err)
	}
	if err := validateRecoveredArchive(data.Bytes(), strings.Repeat("0", 64)); err == nil {
		t.Fatal("accepted mismatched hash")
	}
	if err := validateRecoveredArchive([]byte("not a zip"), ""); err == nil {
		t.Fatal("accepted invalid zip")
	}
}

func TestRecoveryRejectsOtherHostBeforeResponseAction(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	actions := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "_search") {
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"agent":{"id":"other-host"},"file":{"Ext":{"quarantine_result":true,"quarantine_path":"C:\\.equarantine\\id"}}}}]}}`))
			return
		}
		actions++
		http.Error(w, "unexpected action", 400)
	}))
	defer server.Close()
	s := disconnectedTestService()
	s.esc, _ = es.New(server.URL, auth.Auth{}, false)
	s.kb, _ = kibana.New(server.URL, auth.Auth{}, false, "")
	s.agent = &kibana.Agent{ID: "local"}
	s.settings.Profile.AgentID = "local"
	if err := s.SaveQuarantinedFile("alert", "unused.zip"); err == nil || !strings.Contains(err.Error(), "本机") {
		t.Fatalf("%v", err)
	}
	if actions != 0 {
		t.Fatal("sent remote response action")
	}
}

func TestRecoveryDownloadsValidatedArchiveWithoutExtractingOrTrusting(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Encrypt("original.exe", "elastic", zip.StandardEncryption)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("inert recovery fixture"))
	_ = writer.Close()
	digest := sha256.Sum256([]byte("inert recovery fixture"))
	var submitted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "_search"):
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"agent":{"id":"local"},"file":{"path":"C:\\Downloads\\original.exe","hash":{"sha256":"` + hex.EncodeToString(digest[:]) + `"},"Ext":{"quarantine_result":true,"quarantine_path":"C:\\.equarantine\\fixture"}}}}]}}`))
		case r.URL.Path == "/api/endpoint/action/get_file":
			_ = json.NewDecoder(r.Body).Decode(&submitted)
			_, _ = w.Write([]byte(`{"data":{"id":"recover-action","status":"pending"}}`))
		case r.URL.Path == "/api/endpoint/action/recover-action":
			_, _ = w.Write([]byte(`{"data":{"id":"recover-action","status":"successful","outputs":{"local":{"content":{"downloadUri":"/fixture.zip"}}}}}`))
		case r.URL.Path == "/fixture.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(archive.Bytes())
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.Error(w, "unexpected", 400)
		}
	}))
	defer server.Close()
	s := disconnectedTestService()
	s.settings.Profile.AgentID = "local"
	s.agent = &kibana.Agent{ID: "local"}
	s.esc, _ = es.New(server.URL, auth.Auth{}, false)
	s.kb, _ = kibana.New(server.URL, auth.Auth{}, false, "")
	destination := filepath.Join(t.TempDir(), "recovered.zip")
	if err := s.SaveQuarantinedFile("alert", destination); err != nil {
		t.Fatal(err)
	}
	if submitted["parameters"].(map[string]any)["path"] != `C:\.equarantine\fixture` {
		t.Fatalf("wrong path: %+v", submitted)
	}
	contents, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(contents, archive.Bytes()) {
		t.Fatalf("archive changed: %v", err)
	}
	files, _ := os.ReadDir(filepath.Dir(destination))
	if len(files) != 1 {
		t.Fatal("unexpected extracted files")
	}
	batch := s.SaveQuarantinedFiles([]string{"alert", "alert", "another"}, filepath.Dir(destination))
	if len(batch) != 2 || batch[0].Error != "" || batch[1].Error != "" || batch[0].Path == batch[1].Path {
		t.Fatalf("batch: %+v", batch)
	}
	preserved, _ := os.ReadFile(destination)
	if !bytes.Equal(preserved, contents) {
		t.Fatal("batch overwrote original archive")
	}
}

func TestQuarantineTrustResolvesHashAndPolicyOnBackend(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	agentID := "local"
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "_search"):
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"agent":{"id":"` + agentID + `"},"file":{"hash":{"sha256":"` + strings.Repeat("a", 64) + `"},"Ext":{"quarantine_result":true,"quarantine_path":"C:\\.equarantine\\fixture"}}}}]}}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[],"total":0}`))
		case r.Method == http.MethodPost:
			var item kibana.ExceptionListItem
			_ = json.NewDecoder(r.Body).Decode(&item)
			if len(item.Tags) != 1 || item.Tags[0] != "policy:local-policy" || len(item.Entries) != 1 || item.Entries[0].Field != "process.hash.sha256" || item.Entries[0].Value != strings.Repeat("a", 64) {
				t.Errorf("unscoped or wrong hash: %+v", item)
			}
			writes++
			_, _ = w.Write([]byte(`{"id":"trusted"}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	s := disconnectedTestService()
	s.settings.Profile.AgentID = "local"
	s.settings.Profile.PackagePolicyID = "local-policy"
	s.agent = &kibana.Agent{ID: "local"}
	s.esc, _ = es.New(server.URL, auth.Auth{}, false)
	s.kb, _ = kibana.New(server.URL, auth.Auth{}, false, "")
	if err := s.TrustQuarantinedFile("alert"); err != nil {
		t.Fatal(err)
	}
	agentID = "another-host"
	if err := s.TrustQuarantinedFile("alert"); err == nil {
		t.Fatal("accepted another host")
	}
	if writes != 1 {
		t.Fatalf("unexpected trust writes: %d", writes)
	}
}
