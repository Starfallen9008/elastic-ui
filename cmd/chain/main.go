package main

import (
	"fmt"
	"os"

	"elastic-ui/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: chain <alert-id>")
		os.Exit(1)
	}
	svc := service.New()
	if err := svc.ConnectError(); err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}
	chain, err := svc.ProcessChain(os.Args[1])
	if err != nil {
		fmt.Println("chain error:", err)
		os.Exit(1)
	}
	fmt.Printf("alert=%s host=%s nodes=%d children=%d\n", chain.AlertName, chain.HostName, len(chain.Nodes), len(chain.Children))
	for _, node := range chain.Nodes {
		fmt.Printf("[%d] %s pid=%d user=%s resolved=%v alert=%v action=%s\n    %s\n",
			node.Depth, node.Name, node.PID, node.User, node.Resolved, node.IsAlert, node.EventAction, node.CommandLine)
	}
	for _, child := range chain.Children {
		fmt.Printf("child: %s pid=%d %s\n", child.Name, child.PID, child.CommandLine)
	}
}
