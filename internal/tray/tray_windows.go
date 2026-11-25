//go:build windows

package tray

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed icon.ico
var iconData []byte

type Actions struct {
	Show                func()
	Scan                func()
	Threats             func()
	NotificationDetails func(string)
	Quit                func()
}

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassEx      = user32.NewProc("RegisterClassExW")
	procCreateWindowEx       = user32.NewProc("CreateWindowExW")
	procDefWindowProc        = user32.NewProc("DefWindowProcW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procGetMessage           = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessage      = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procPostMessage          = user32.NewProc("PostMessageW")
	procRegisterWindowMsg    = user32.NewProc("RegisterWindowMessageW")
	procSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	procGetCursorPos         = user32.NewProc("GetCursorPos")
	procDrawText             = user32.NewProc("DrawTextW")
	procGetDC                = user32.NewProc("GetDC")
	procReleaseDC            = user32.NewProc("ReleaseDC")
	procFillRect             = user32.NewProc("FillRect")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procLoadImage            = user32.NewProc("LoadImageW")
	procGetDpi               = user32.NewProc("GetDpiForSystem")
	procSetBkMode            = gdi32.NewProc("SetBkMode")
	procSetTextColor         = gdi32.NewProc("SetTextColor")
	procCreateFont           = gdi32.NewProc("CreateFontW")
	procCreateSolidBrush     = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject         = gdi32.NewProc("DeleteObject")
	procSelectObject         = gdi32.NewProc("SelectObject")
	procGetTextExtentPoint32 = gdi32.NewProc("GetTextExtentPoint32W")
	procShellNotifyIcon      = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandle      = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmDestroy       = 0x0002
	wmClose         = 0x0010
	wmContextMenu   = 0x007B
	wmLButtonUp     = 0x0202
	wmRButtonUp     = 0x0205
	wmNull          = 0x0000
	trayCallbackMsg = 0x8000 + 1

	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	dtLeft          = 0x0000
	dtVCenter       = 0x0004
	dtSingleLine    = 0x0020
	dtNoPrefix      = 0x0800
	transparentMode = 1
	defaultCharset  = 1
	cleartype       = 5
	fwNormal        = 400
	imageIcon       = 1
	lrLoadFromFile  = 0x0010

	smCxSmIcon = 49
	smCySmIcon = 50

	idShow    = 1
	idScan    = 2
	idThreats = 3
	idQuit    = 4
)

var (
	hwnd       windows.HWND
	hwndMu     sync.Mutex
	actions    Actions
	themeDark  atomic.Bool
	menuFont   windows.Handle
	trayIcon   windows.Handle
	taskbarMsg uint32
	wndProcPtr uintptr
	iconPath   string
	dpiScale   = 1.0
)

func scaled(v int) int32 {
	return int32(float64(v) * dpiScale)
}

func init() {
	themeDark.Store(true)
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	HWnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             windows.HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     windows.Handle
}

type palette struct {
	bg     uint32
	hover  uint32
	text   uint32
	muted  uint32
	danger uint32
	border uint32
	accent uint32
}

func colors() palette {
	if themeDark.Load() {
		return palette{
			bg:     rgb(15, 19, 25),
			hover:  rgb(27, 34, 48),
			text:   rgb(227, 232, 240),
			muted:  rgb(152, 162, 179),
			danger: rgb(255, 138, 128),
			border: rgb(42, 52, 66),
			accent: rgb(106, 166, 255),
		}
	}
	return palette{
		bg:     rgb(247, 248, 250),
		hover:  rgb(230, 235, 242),
		text:   rgb(27, 34, 48),
		muted:  rgb(91, 102, 117),
		danger: rgb(180, 35, 24),
		border: rgb(215, 221, 229),
		accent: rgb(37, 99, 235),
	}
}

func rgb(r, g, b uint32) uint32 {
	return r | g<<8 | b<<16
}

func utf16Ptr(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}
	return p
}

func copyTip(dst *[128]uint16, s string) {
	src, err := windows.UTF16FromString(s)
	if err != nil {
		return
	}
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:], src[:n])
	if n > 0 {
		dst[n-1] = 0
	}
}

// Start creates the tray icon and themed popup menu on a dedicated message
// thread. It returns false if the icon window could not be created.
func Start(a Actions) bool {
	actions = a
	ready := make(chan bool, 1)

	go func() {
		// Win32 windows are bound to the thread that created them.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if !createWindow() {
			ready <- false
			return
		}
		ready <- true
		messageLoop()
	}()

	select {
	case ok := <-ready:
		return ok
	case <-time.After(3 * time.Second):
		return false
	}
}

// Quit closes the tray window and stops its message loop.
func Quit() {
	hwndMu.Lock()
	h := hwnd
	hwndMu.Unlock()
	if h != 0 {
		procPostMessage.Call(uintptr(h), wmClose, 0, 0)
	}
}

// SetTheme switches the popup menu palette. Safe to call from any goroutine.
func SetTheme(dark bool) {
	themeDark.Store(dark)
	if menuHwnd != 0 {
		procInvalidateRect.Call(uintptr(menuHwnd), 0, 1)
	}
}

func createWindow() bool {
	iconPath = ensureIconFile()
	trayIcon = loadIcon(iconPath)

	hInst, _, _ := procGetModuleHandle.Call(0)
	className := utf16Ptr("ElasticDefendTrayWindow")
	if className == nil {
		return false
	}

	wndProcPtr = windows.NewCallback(wndProc)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   wndProcPtr,
		HInstance:     windows.Handle(hInst),
		LpszClassName: className,
	}
	if r, _, _ := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return false
	}

	h, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("Elastic Defend"))),
		0,
		0, 0, 0, 0,
		0, 0,
		hInst,
		0,
	)
	if h == 0 {
		return false
	}

	hwndMu.Lock()
	hwnd = windows.HWND(h)
	hwndMu.Unlock()

	if !addIcon() {
		return false
	}

	height := int32(-13)
	if dpi, _, _ := procGetDpi.Call(); dpi >= 96 {
		dpiScale = float64(dpi) / 96.0
		height = -scaled(13)
	}
	font, _, _ := procCreateFont.Call(
		uintptr(height), 0, 0, 0, fwNormal,
		0, 0, 0, defaultCharset, 0, 0, cleartype, 0,
		uintptr(unsafe.Pointer(utf16Ptr("Microsoft YaHei UI"))),
	)
	menuFont = windows.Handle(font)
	if !createMenuPopup(hInst) {
		return false
	}
	if !createNotificationWindow(hInst) {
		return false
	}
	return true
}

func ensureIconFile() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "ElasticDefendConsole")
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "tray.ico")
	if data, err := os.ReadFile(path); err == nil && bytes.Equal(data, iconData) {
		return path
	}
	if err := os.WriteFile(path, iconData, 0o644); err != nil {
		return ""
	}
	return path
}

func loadIcon(path string) windows.Handle {
	if path != "" {
		w, _, _ := procGetSystemMetrics.Call(smCxSmIcon)
		h, _, _ := procGetSystemMetrics.Call(smCySmIcon)
		if w == 0 {
			w = 16
		}
		if h == 0 {
			h = 16
		}
		r, _, _ := procLoadImage.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr(path))),
			imageIcon,
			w, h,
			lrLoadFromFile,
		)
		if r != 0 {
			return windows.Handle(r)
		}
	}
	r, _, _ := procLoadImage.Call(0, 32512, imageIcon, 0, 0, 0) // IDI_APPLICATION
	return windows.Handle(r)
}

func iconDataStruct() notifyIconData {
	nid := notifyIconData{
		HWnd:             currentHwnd(),
		UID:              1,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: trayCallbackMsg,
		HIcon:            trayIcon,
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	copyTip(&nid.SzTip, "Elastic Defend 本机控制台")
	return nid
}

func currentHwnd() windows.HWND {
	hwndMu.Lock()
	defer hwndMu.Unlock()
	return hwnd
}

func addIcon() bool {
	if taskbarMsg == 0 {
		r, _, _ := procRegisterWindowMsg.Call(uintptr(unsafe.Pointer(utf16Ptr("TaskbarCreated"))))
		taskbarMsg = uint32(r)
	}
	nid := iconDataStruct()
	r, _, _ := procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	return r != 0
}

func messageLoop() {
	var m msg
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func wndProc(h windows.HWND, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case notificationMessage:
		receiveNotifications()
		return 0
	case clearNotificationsMessage:
		notificationQueue = []Notification{}
		if notificationVisible {
			closeNotification()
		}
		return 0
	case trayCallbackMsg:
		switch uint32(lParam) & 0xFFFF {
		case wmLButtonUp:
			run(actions.Show)
		case wmRButtonUp, wmContextMenu:
			showMenu(h)
		}
		return 0
	case wmClose:
		if notificationHwnd != 0 {
			procDestroyWindow.Call(uintptr(notificationHwnd))
			notificationHwnd = 0
		}
		if notificationTitleFont != 0 {
			procDeleteObject.Call(uintptr(notificationTitleFont))
			notificationTitleFont = 0
		}
		nid := iconDataStruct()
		procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
		if menuHwnd != 0 {
			procDestroyWindow.Call(uintptr(menuHwnd))
			menuHwnd = 0
		}
		procDestroyWindow.Call(uintptr(h))
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	default:
		if taskbarMsg != 0 && message == taskbarMsg {
			addIcon()
			return 0
		}
	}
	r, _, _ := procDefWindowProc.Call(uintptr(h), uintptr(message), wParam, lParam)
	return r
}

func run(fn func()) {
	if fn != nil {
		fn()
	}
}
