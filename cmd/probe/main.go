package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"elastic-ui/internal/auth"
	"elastic-ui/internal/config"
	"elastic-ui/internal/es"
	"elastic-ui/internal/kibana"
	"elastic-ui/internal/model"
)

type devConfig struct {
	KibanaURL     string `json:"kibanaUrl"`
	ESURL         string `json:"esUrl"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	APIKey        string `json:"apiKey"`
	SkipTLSVerify bool   `json:"skipTlsVerify"`
	HostName      string `json:"hostName"`
}

func main() {
	cfgPath := flag.String("config", "dev/local.json", "path to dev config")
	doAction := flag.Bool("action", false, "run running_procs action")
	writeTest := flag.Bool("write", false, "create a trusted application for the dev build path")
	cleanup := flag.Bool("cleanup", false, "delete the dev trusted applications")
	flag.Parse()

	data, err := os.ReadFile(*cfgPath)
	if err != nil {
		fail("read config: %v", err)
	}
	var dc devConfig
	if err := json.Unmarshal(data, &dc); err != nil {
		fail("parse config: %v", err)
	}
	if dc.Password == "" {
		dc.Password = os.Getenv("ELASTIC_UI_PASSWORD")
	}

	a := auth.Auth{Method: config.AuthBasic, Username: dc.Username, Password: dc.Password}
	if dc.APIKey != "" {
		a = auth.Auth{Method: config.AuthAPIKey, APIKey: dc.APIKey}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	kb, err := kibana.New(dc.KibanaURL, a, dc.SkipTLSVerify, "")
	if err != nil {
		fail("kibana client: %v", err)
	}
	if err := kb.LoadVersion(ctx); err != nil {
		fail("kibana status: %v", err)
	}
	fmt.Println("kibana version:", kb.Version)

	agents, err := kb.ListAgents(ctx, 200)
	if err != nil {
		fail("list agents: %v", err)
	}
	fmt.Println("agents:", agents.Total)
	var local *kibana.Agent
	for i := range agents.Items {
		item := &agents.Items[i]
		marker := " "
		if strings.EqualFold(item.LocalMetadata.Host.Name, dc.HostName) {
			local = item
			marker = "*"
		}
		fmt.Printf(" %s %s  %s  policy=%s rev=%d\n", marker, item.ID, item.LocalMetadata.Host.Name, item.PolicyID, item.PolicyRevisionIdx)
	}
	if local == nil {
		fail("local agent not found for host %q", dc.HostName)
	}

	meta, err := kb.GetEndpointMetadata(ctx, local.ID)
	if err != nil {
		fail("endpoint metadata: %v", err)
	}
	fmt.Printf("endpoint: status=%s host_status=%s isolation=%v tamper=%v version=%s\n",
		meta.Endpoint.Status, meta.HostStatus, meta.Endpoint.State.Isolation,
		meta.Endpoint.State.TamperProtection, meta.Agent.Version)
	fmt.Printf("endpoint policy: id=%s name=%s version=%s endpoint_version=%s\n",
		meta.Endpoint.Policy.Applied.ID, meta.Endpoint.Policy.Applied.Name,
		meta.Endpoint.Policy.Applied.Version, meta.Endpoint.Policy.Applied.EndpointPolicyVersion)
	fmt.Printf("capabilities: %s\n", strings.Join(meta.Endpoint.Capabilities, ", "))

	pp, err := kb.GetPackagePolicy(ctx, meta.Endpoint.Policy.Applied.ID)
	if err != nil {
		fail("package policy: %v", err)
	}
	pv, err := pp.PolicyValue()
	if err != nil {
		fail("policy value: %v", err)
	}
	fmt.Printf("package: %s %s  policy=%s revision=%d\n", pp.Package.Name, pp.Package.Version, pp.Name, pp.Revision)
	fmt.Printf("protection: malware=%s ransomware=%s memory=%s behavior=%s av_reg=%v device_control=%v/%s\n",
		pv.Windows.Malware.Mode, pv.Windows.Ransomware.Mode, pv.Windows.MemoryProtection.Mode,
		pv.Windows.BehaviorProtection.Mode, pv.Windows.AntivirusRegistration.Enabled,
		pv.Windows.DeviceControl.Enabled, pv.Windows.DeviceControl.USBStorage)

	esc, err := es.New(dc.ESURL, a, dc.SkipTLSVerify)
	if err != nil {
		fail("es client: %v", err)
	}
	query := map[string]any{
		"size":  5,
		"sort":  []any{map[string]any{"@timestamp": map[string]any{"order": "desc"}}},
		"query": map[string]any{"bool": map[string]any{"filter": []any{map[string]any{"term": map[string]any{"agent.id": local.ID}}}}},
		"_source": []string{
			"@timestamp", "message", "rule.name", "event.*",
			"process.name", "process.pid", "process.executable", "process.hash.sha256",
			"file.path", "file.hash.sha256",
			"Responses.action.action", "Responses.message", "host.hostname",
		},
	}
	var hits model.SearchResponse
	if err := esc.Search(ctx, "logs-endpoint.alerts-default", query, &hits); err != nil {
		fail("search alerts: %v", err)
	}
	fmt.Println("alerts total:", hits.Hits.Total.Value)
	for _, h := range hits.Hits.Hits {
		a := h.Source
		fmt.Printf(" - %s  %s  %s  action=%q  proc=%s\n",
			a.Timestamp, a.Rule.Name, a.Message, a.ActionTaken(), a.Process.Name)
	}

	lists, err := kb.FindExceptionLists(ctx, "endpoint_trusted_apps")
	if err != nil {
		fmt.Println("exception lists error:", err)
	} else {
		fmt.Println("trusted app lists:", len(lists))
		for _, l := range lists {
			fmt.Printf(" - %s list_id=%s name=%s\n", l.ID, l.ListID, l.Name)
		}
	}

	allLists, err := kb.FindExceptionLists(ctx, "")
	if err != nil {
		fmt.Println("all exception lists error:", err)
	} else {
		fmt.Println("all exception lists:", len(allLists))
		for _, l := range allLists {
			fmt.Printf(" - id=%s list_id=%s name=%s type=%s\n", l.ID, l.ListID, l.Name, l.Type)
		}
	}

	items, err := kb.FindEndpointListItems(ctx)
	if err != nil {
		fmt.Println("endpoint list error:", err)
	} else {
		fmt.Println("endpoint list items:", len(items))
	}

	for _, p := range []string{
		"/api/trusted_apps",
		"/api/event_filters",
		"/api/blocklist",
		"/api/isolation_exceptions",
		"/api/endpoint_exceptions",
		"/api/endpoint/artifacts",
	} {
		var raw json.RawMessage
		if err := kb.Get(ctx, p, &raw); err != nil {
			fmt.Printf("probe %s -> %v\n", p, err)
			continue
		}
		text := string(raw)
		if len(text) > 300 {
			text = text[:300]
		}
		fmt.Printf("probe %s -> 200 %s\n", p, text)
	}

	var cat []map[string]any
	if err := esc.Get(ctx, "/_cat/indices/.lists*,.items*?format=json&h=index,docs.count", &cat); err != nil {
		fmt.Println("lists indices error:", err)
	} else {
		fmt.Println("lists indices:", cat)
	}

	if *writeTest {
		tryCreate := func(label string, entry kibana.ExceptionEntry) {
			item, err := kb.CreateExceptionItem(ctx, "endpoint_trusted_apps",
				"elastic-ui dev build "+label,
				"Trusted build output for the Elastic Defend local console development.",
				[]kibana.ExceptionEntry{entry},
				[]string{"windows"}, []string{kibana.PolicyTag(meta.Endpoint.Policy.Applied.ID)})
			if err != nil {
				fmt.Printf("create trusted app (%s) error: %v\n", label, err)
				return
			}
			fmt.Printf("created trusted app (%s): id=%s item_id=%s list_id=%s tags=%v\n",
				label, item.ID, item.ItemID, item.ListID, item.Tags)
		}
		tryCreate("match", kibana.ExceptionEntry{
			Type:     "match",
			Field:    "process.executable.caseless",
			Value:    `D:\Works\elastic-ui\build\probe.exe`,
			Operator: "included",
		})
		tryCreate("wildcard", kibana.ExceptionEntry{
			Type:     "wildcard",
			Field:    "process.executable.caseless",
			Value:    `D:\Works\elastic-ui\build\*`,
			Operator: "included",
		})
	}

	trusted, err := kb.FindExceptionItems(ctx, "endpoint_trusted_apps", "")
	if err != nil {
		fmt.Println("list trusted apps error:", err)
	} else {
		fmt.Println("trusted apps:", len(trusted))
		for _, it := range trusted {
			if strings.HasPrefix(it.Name, "elastic-ui dev build") {
				if *cleanup {
					if err := kb.DeleteExceptionItem(ctx, it); err != nil {
						fmt.Printf("delete %s error: %v\n", it.Name, err)
					} else {
						fmt.Printf("deleted %s (%s)\n", it.Name, it.ID)
					}
					continue
				}
			}
			fmt.Printf(" - id=%s item_id=%s name=%s tags=%v entries=%v\n", it.ID, it.ItemID, it.Name, it.Tags, it.Entries)
		}
	}

	if *doAction {
		action, err := kb.RunningProcesses(ctx, local.ID)
		if err != nil {
			fail("running_procs: %v", err)
		}
		fmt.Printf("action submitted: id=%s status=%s method=%s\n", action.ID, action.Status, action.Command)
		raw, _ := json.Marshal(action)
		fmt.Printf("action payload: %s\n", string(raw))
		for i := 0; i < 20; i++ {
			time.Sleep(2 * time.Second)
			updated, err := kb.GetAction(ctx, action.ID)
			if err != nil {
				fail("get action: %v", err)
			}
			fmt.Printf("poll %d: status=%s expired=%v outputs=%d\n", i+1, updated.Status, updated.IsExpired, len(updated.Outputs))
			if updated.Status == "successful" || updated.Status == "failed" || updated.IsExpired {
				for agentID, output := range updated.Outputs {
					text := string(output)
					if len(text) > 2000 {
						text = text[:2000]
					}
					fmt.Printf("output[%s]: %s\n", agentID, text)
				}
				break
			}
		}
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

