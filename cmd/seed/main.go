package main

import (
	"encoding/json"
	"fmt"
	"os"

	"elastic-ui/internal/service"
)

type devConfig struct {
	KibanaURL     string `json:"kibanaUrl"`
	ESURL         string `json:"esUrl"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	SkipTLSVerify bool   `json:"skipTlsVerify"`
}

func main() {
	data, err := os.ReadFile("dev/local.json")
	if err != nil {
		fail("read dev config: %v", err)
	}
	var dc devConfig
	if err := json.Unmarshal(data, &dc); err != nil {
		fail("parse dev config: %v", err)
	}
	if dc.Password == "" {
		dc.Password = os.Getenv("ELASTIC_UI_PASSWORD")
	}

	svc := service.New()
	result, err := svc.Configure(service.ConnectionInput{
		KibanaURL:     dc.KibanaURL,
		ESURL:         dc.ESURL,
		AuthMethod:    "basic",
		Username:      dc.Username,
		Password:      dc.Password,
		SkipTLSVerify: dc.SkipTLSVerify,
	})
	if err != nil {
		for _, c := range result.Checks {
			fmt.Printf("[%v] %s: %s\n", c.OK, c.Name, c.Detail)
		}
		fail("configure: %v", err)
	}
	fmt.Println("configured ok")
	for _, c := range result.Checks {
		fmt.Printf("[%v] %s: %s\n", c.OK, c.Name, c.Detail)
	}
	if result.Agent != nil {
		fmt.Printf("agent: %s host=%s policy=%s\n", result.Agent.ID, result.Agent.HostName, result.Agent.PolicyName)
	}
	status, err := svc.Status()
	if err != nil {
		fail("status: %v", err)
	}
	fmt.Printf("status: host=%s agent=%s policy=%s isolation=%v\n",
		status.HostName, status.AgentStatus, status.PolicyName, status.Isolation)
	fmt.Printf("protection: malware=%s ransomware=%s memory=%s behavior=%s\n",
		status.Protection.Malware, status.Protection.Ransomware, status.Protection.Memory, status.Protection.Behavior)
	fmt.Printf("alerts24h=%d blocked24h=%d\n", status.Alerts24h, status.Blocked24h)

	threats, err := svc.Threats(0, 5, "", "30d")
	if err != nil {
		fmt.Println("threats error:", err)
	} else {
		fmt.Printf("threats total=%d first page=%d\n", threats.Total, len(threats.Items))
		for i, t := range threats.Items {
			if i >= 3 {
				break
			}
			fmt.Printf(" - %s | %s | blocked=%v | %s\n", t.Time, t.RuleName, t.Blocked, t.ProcessName)
		}
		if len(threats.Items) > 0 {
			detail, err := svc.ThreatDetail(threats.Items[0].ID)
			if err != nil {
				fmt.Println("detail error:", err)
			} else {
				fmt.Printf("detail: actions=%d tactics=%v\n", len(detail.Responses), detail.Tactics)
			}
		}
	}

	trust, err := svc.TrustItems()
	if err != nil {
		fmt.Println("trust items error:", err)
	} else {
		fmt.Printf("trust items=%d\n", len(trust))
		for i, item := range trust {
			if i >= 3 {
				break
			}
			fmt.Printf(" - %s | %s | %s\n", item.Kind, item.Scope, item.Value)
		}
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
