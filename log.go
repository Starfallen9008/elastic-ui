package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"elastic-ui/internal/config"
)

func logf(format string, args ...any) {
	dir, err := config.Dir()
	if err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	fmt.Fprintf(file, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}
