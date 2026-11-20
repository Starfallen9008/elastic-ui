package main

import (
	"fmt"
	"os"

	"elastic-ui/internal/service"
)

func main() {
	path := `D:\Works\elastic-ui\dev\getfile-target.txt`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	svc := service.New()
	if err := svc.ConnectError(); err != nil {
		fmt.Println("connect error:", err)
		os.Exit(1)
	}
	result, err := svc.GetFile(path)
	if err != nil {
		fmt.Println("get_file error:", err)
		os.Exit(1)
	}
	fmt.Printf("path=%s\nsaved=%s\nzip=%s\nsize=%d\nsha256=%s\nis_directory=%v\n", result.Path, result.SavedPath, result.ZipPath, result.Size, result.SHA256, result.IsDirectory)
	audit, err := svc.GetAudit(5)
	if err != nil {
		fmt.Println("audit error:", err)
		return
	}
	for _, entry := range audit {
		fmt.Printf("audit %s | %s | %s | %s\n", entry.Time, entry.Action, entry.Result, entry.Target)
	}
}
