package main

import (
	"fmt"
	"os"
	"time"

	"elastic-ui/internal/service"
)

func main() {
	target := `D:\Works\elastic-ui\dev\scan-target.txt`
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	svc := service.New()
	if err := svc.ConnectError(); err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}
	task, err := svc.StartScan(target)
	if err != nil {
		fmt.Println("scan error:", err)
		os.Exit(1)
	}
	fmt.Printf("submitted id=%s action=%s path=%s\n", task.ID, task.ActionID, task.Path)

	final := task.Status
	deadline := time.Now().Add(4 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		for _, current := range svc.Scans() {
			if current.ID == task.ID {
				final = current.Status
				break
			}
		}
		if final == "successful" || final == "failed" || final == "expired" {
			break
		}
	}
	fmt.Println("final:", final)

	audit, err := svc.GetAudit(20)
	if err != nil {
		fmt.Println("audit error:", err)
		os.Exit(1)
	}
	fmt.Printf("audit=%d\n", len(audit))
	for _, entry := range audit {
		fmt.Printf("%s | %s | %s | %s | %s\n", entry.Time, entry.Action, entry.Result, entry.Target, entry.Detail)
	}
}
