//go:build windows

package tray

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestIconIncludesWindowsDisplaySizes(t *testing.T) {
	if len(iconData) < 6 || binary.LittleEndian.Uint16(iconData[2:4]) != 1 {
		t.Fatal("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(iconData[4:6]))
	sizes := map[int]bool{}
	for i := range count {
		entry := iconData[6+i*16 : 6+(i+1)*16]
		size := int(entry[0])
		if size == 0 {
			size = 256
		}
		sizes[size] = true
	}
	for _, size := range []int{16, 20, 24, 32, 40, 48, 64, 96, 128, 256} {
		if !sizes[size] {
			t.Errorf("missing icon size %d", size)
		}
	}
}

func TestTrayIconRefreshesSameSizeCachedContent(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	path := ensureIconFile()
	if path == "" {
		t.Fatal("icon was not written")
	}
	stale := bytes.Repeat([]byte{0}, len(iconData))
	if err := os.WriteFile(path, stale, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ensureIconFile(); filepath.Clean(got) != filepath.Clean(path) {
		t.Fatal("cache path changed")
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, iconData) {
		t.Fatal("stale tray icon not replaced")
	}
}
