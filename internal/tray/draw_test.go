//go:build windows

package tray

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

var (
	testCreateDC  = gdi32.NewProc("CreateCompatibleDC")
	testCreateBmp = gdi32.NewProc("CreateCompatibleBitmap")
	testSelect    = gdi32.NewProc("SelectObject")
	testGetPixel  = gdi32.NewProc("GetPixel")
	testStockFont = gdi32.NewProc("GetStockObject")
	testDeleteDC  = gdi32.NewProc("DeleteDC")
	testDelete    = gdi32.NewProc("DeleteObject")
)

type drawResult struct {
	bg     uint32
	hasInk bool
}

func TestNotificationPathKeepsEveryCharacter(t *testing.T) {
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	font, _, _ := testStockFont.Call(17)
	old, _, _ := procSelectObject.Call(screen, font)
	defer procSelectObject.Call(screen, old)
	paths := []string{
		`C:\Users\19393\AppData\Local\Temp\opencode\notification-preview\example-threat-with-complete-filename.exe`,
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		`C:\Users\19393\Documents\测试目录\完整文件名.docx`,
	}
	for _, path := range paths {
		lines := wrapNotificationText(windows.Handle(screen), path, 180)
		if len(lines) < 2 || strings.Join(lines, "") != path {
			t.Fatalf("path was truncated: %q", lines)
		}
	}
}

func paintSample(t *testing.T, dark bool, hover int) drawResult {
	t.Helper()
	themeDark.Store(dark)
	buildMenuRows()
	screen, _, _ := procGetDC.Call(0)
	mem, _, _ := testCreateDC.Call(screen)
	defer testDeleteDC.Call(mem)
	bmp, _, _ := testCreateBmp.Call(screen, uintptr(menuW), uintptr(menuH))
	defer testDelete.Call(bmp)
	testSelect.Call(mem, bmp)
	font, _, _ := testStockFont.Call(17)
	menuFont = windows.Handle(font)
	menuHover = hover
	paintMenu(windows.Handle(mem))

	rowMid := (menuRows[0].r.Top + menuRows[0].r.Bottom) / 2
	clear, _, _ := testGetPixel.Call(mem, uintptr(menuW-12), uintptr(rowMid))
	bgv := uint32(clear)

	ink := false
	for x := int32(24); x < menuW-30; x += 2 {
		for y := menuRows[0].r.Top + 6; y < menuRows[0].r.Bottom-6; y += 2 {
			p, _, _ := testGetPixel.Call(mem, uintptr(x), uintptr(y))
			if uint32(p) != bgv {
				ink = true
			}
		}
	}
	procReleaseDC.Call(0, screen)
	return drawResult{bg: bgv, hasInk: ink}
}

func TestPaintMenuColors(t *testing.T) {
	result := paintSample(t, false, -1)
	r, g, b := result.bg&0xFF, (result.bg>>8)&0xFF, (result.bg>>16)&0xFF
	if r != 247 || g != 248 || b != 250 {
		t.Fatalf("light background = %d,%d,%d", r, g, b)
	}
	if !result.hasInk {
		t.Fatalf("no text pixels drawn")
	}

	hover := paintSample(t, false, 0)
	r, g, b = hover.bg&0xFF, (hover.bg>>8)&0xFF, (hover.bg>>16)&0xFF
	if r != 230 || g != 235 || b != 242 {
		t.Fatalf("light hover = %d,%d,%d", r, g, b)
	}

	darkResult := paintSample(t, true, -1)
	r, g, b = darkResult.bg&0xFF, (darkResult.bg>>8)&0xFF, (darkResult.bg>>16)&0xFF
	if r != 15 || g != 19 || b != 25 {
		t.Fatalf("dark background = %d,%d,%d", r, g, b)
	}
}
