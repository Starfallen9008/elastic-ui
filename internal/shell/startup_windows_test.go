//go:build windows

package shell

import (
	"os"
	"testing"
)

// Opt-in integration check: a user task is created briefly and removed again.
func TestStartupLifecycle(t *testing.T) {
	if os.Getenv("ELASTIC_UI_TEST_STARTUP") != "1" {
		t.Skip("requires an elevated, opt-in integration run")
	}
	enabled, err := StartupEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Skip("preserve existing startup task")
	}
	name, err := startupTaskName()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := taskCommand("/Query", "/TN", name); err == nil {
		t.Skip("preserve an existing disabled task")
	}
	t.Cleanup(func() { _, _ = taskCommand("/Delete", "/TN", name, "/F") })
	if err := SetStartupEnabled(true); err != nil {
		t.Fatal(err)
	}
	if enabled, err := StartupEnabled(); err != nil || !enabled {
		t.Fatalf("task not enabled: %v %v", enabled, err)
	}
	if err := SetStartupEnabled(false); err != nil {
		t.Fatal(err)
	}
	if enabled, err := StartupEnabled(); err != nil || enabled {
		t.Fatalf("task not removed: %v %v", enabled, err)
	}
}
