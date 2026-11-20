package main

import (
	"fmt"
	"os"

	"elastic-ui/internal/service"
)

func main() {
	svc := service.New()
	if err := svc.ConnectError(); err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}
	procs, err := svc.Processes(true)
	if err != nil {
		fmt.Println("processes error:", err)
		os.Exit(1)
	}
	fmt.Printf("processes=%d\n", len(procs))
	for i, p := range procs {
		if i >= 12 {
			break
		}
		fmt.Printf("pid=%d user=%s entity=%s command=%s\n", p.PID, p.User, p.EntityID, p.Command)
	}
	audit, err := svc.GetAudit(10)
	if err != nil {
		fmt.Println("audit error:", err)
	} else {
		fmt.Printf("audit=%d\n", len(audit))
		for _, entry := range audit {
			fmt.Printf("%s | %s | %s | %s\n", entry.Time, entry.Action, entry.Result, entry.Target)
		}
	}
}
