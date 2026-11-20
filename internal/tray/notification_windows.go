//go:build windows

package tray

import (
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Notification struct {
	ID          string
	Kind        string
	Outcome     string
	Object      string
	Rule        string
	MalwareName string
}

const (
	notificationMessage       = 0x8000 + 20
	clearNotificationsMessage = 0x8000 + 21
	wmTimer                   = 0x0113
	wsExLayered               = 0x00080000
	wsExNoActivate            = 0x08000000
)

var (
	procSetLayered               = user32.NewProc("SetLayeredWindowAttributes")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procSetTimer                 = user32.NewProc("SetTimer")
	procKillTimer                = user32.NewProc("KillTimer")
	procSystemParameters         = user32.NewProc("SystemParametersInfoW")
	notificationMu               sync.Mutex
	pendingNotifications         = []Notification{}
	notificationQueue            = []Notification{}
	notificationHwnd             windows.HWND
	notificationCallback         uintptr
	notificationTitleFont        windows.Handle
	notificationCurrent          Notification
	notificationStarted          time.Time
	notificationClosing          time.Time
	notificationVisible          bool
	notificationMotion           = true
	notificationW, notificationH int32
	lastNotificationID           string
	notificationObjectLines      []string
	notificationDetailLines      []string
	notificationDetailsRect      rect
)

// ShowNotification queues a custom topmost window, independent of the main UI.
func ShowNotification(value Notification) {
	notificationMu.Lock()
	pendingNotifications = append(pendingNotifications, value)
	notificationMu.Unlock()
	if h := currentHwnd(); h != 0 {
		procPostMessage.Call(uintptr(h), notificationMessage, 0, 0)
	}
}

func ClearNotifications() {
	notificationMu.Lock()
	pendingNotifications = []Notification{}
	notificationMu.Unlock()
	if h := currentHwnd(); h != 0 {
		procPostMessage.Call(uintptr(h), clearNotificationsMessage, 0, 0)
	}
}

func createNotificationWindow(instance uintptr) bool {
	className := utf16Ptr("ElasticUISecurityNotification")
	notificationCallback = windows.NewCallback(notificationWndProc)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   notificationCallback,
		HInstance:     windows.Handle(instance),
		LpszClassName: className,
	}
	if r, _, _ := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return false
	}
	h, _, _ := procCreateWindowEx.Call(
		wsExToolWindow|wsExTopmost|wsExLayered|wsExNoActivate,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("Elastic UI 安全通知"))),
		wsPopup,
		0, 0, 0, 0,
		0, 0, instance, 0,
	)
	notificationHwnd = windows.HWND(h)
	font, _, _ := procCreateFont.Call(
		uintptr(-scaled(20)), 0, 0, 0, 600,
		0, 0, 0, defaultCharset, 0, 0, cleartype, 0,
		uintptr(unsafe.Pointer(utf16Ptr("Microsoft YaHei UI"))),
	)
	notificationTitleFont = windows.Handle(font)
	return h != 0
}

func receiveNotifications() {
	notificationMu.Lock()
	notificationQueue = append(notificationQueue, pendingNotifications...)
	pendingNotifications = []Notification{}
	notificationMu.Unlock()
	if !notificationVisible {
		nextNotification()
	}
}

func nextNotification() {
	if len(notificationQueue) == 0 || notificationHwnd == 0 {
		return
	}
	notificationCurrent = notificationQueue[0]
	notificationQueue = notificationQueue[1:]
	notificationW, notificationH = scaled(420), scaled(284)
	// Use the taskbar's monitor work area, including non-bottom taskbars.
	var bounds rect
	procGetWindowRect.Call(uintptr(currentHwnd()), uintptr(unsafe.Pointer(&bounds)))
	mon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&bounds)), 1)
	work := monitorWork(mon)
	if notificationW > work.Right-work.Left-scaled(24) {
		notificationW = work.Right - work.Left - scaled(24)
	}
	layoutNotification()
	x := work.Right - notificationW - scaled(16)
	y := work.Bottom - notificationH - scaled(16)
	var animationEnabled int32 = 1
	procSystemParameters.Call(0x1042, 0, uintptr(unsafe.Pointer(&animationEnabled)), 0) // SPI_GETCLIENTAREAANIMATION
	notificationMotion = animationEnabled != 0
	notificationStarted = time.Now()
	notificationClosing = time.Time{}
	notificationVisible = true
	alpha := uintptr(0)
	if !notificationMotion {
		alpha = 255
	}
	procSetLayered.Call(uintptr(notificationHwnd), 0, alpha, 2)
	procSetWindowPos.Call(
		uintptr(notificationHwnd), ^uintptr(0),
		uintptr(x), uintptr(y), uintptr(notificationW), uintptr(notificationH),
		swpShowWindow|0x0010, // SWP_NOACTIVATE
	)
	procInvalidateRect.Call(uintptr(notificationHwnd), 0, 1)
	procSetTimer.Call(uintptr(notificationHwnd), 1, 16, 0)
}

func closeNotification() {
	if !notificationVisible || !notificationClosing.IsZero() {
		return
	}
	notificationClosing = time.Now()
}

func hideNotification() {
	procKillTimer.Call(uintptr(notificationHwnd), 1)
	procShowWindow.Call(uintptr(notificationHwnd), swHide)
	notificationVisible = false
	nextNotification()
}

func tickNotification() {
	if notificationClosing.IsZero() && time.Since(notificationStarted) >= 14*time.Second {
		closeNotification()
	}
	alpha := 255.0
	if !notificationClosing.IsZero() {
		if !notificationMotion || time.Since(notificationClosing) >= 180*time.Millisecond {
			hideNotification()
			return
		}
		alpha *= 1 - float64(time.Since(notificationClosing))/float64(180*time.Millisecond)
	} else if notificationMotion && time.Since(notificationStarted) < 220*time.Millisecond {
		alpha *= float64(time.Since(notificationStarted)) / float64(220*time.Millisecond)
	}
	procSetLayered.Call(uintptr(notificationHwnd), 0, uintptr(alpha), 2)
}

func notificationWndProc(h windows.HWND, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case 0x0318: // WM_PRINTCLIENT supports diagnostics without screen capture.
		paintNotification(windows.Handle(wParam))
		return 0
	case wmPaint:
		var ps paintStruct
		dc, _, _ := procBeginPaint.Call(uintptr(h), uintptr(unsafe.Pointer(&ps)))
		paintNotification(windows.Handle(dc))
		procEndPaint.Call(uintptr(h), uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmEraseBkgnd:
		return 1
	case wmTimer:
		tickNotification()
		return 0
	case wmLButtonUp:
		x, y := int32(int16(lParam&0xffff)), int32(int16((lParam>>16)&0xffff))
		if x >= notificationW-scaled(42) && y < scaled(42) {
			closeNotification()
			return 0
		}
		link := notificationDetailsRect
		if x >= link.Left && x <= link.Right && y >= link.Top && y <= link.Bottom {
			lastNotificationID = notificationCurrent.ID
			if actions.NotificationDetails != nil {
				id := lastNotificationID
				go actions.NotificationDetails(id)
			}
			closeNotification()
		}
		return 0
	case wmKeyDown:
		if wParam == vkeyEscape {
			closeNotification()
		}
		return 0
	case wmSetCursor:
		procSetCursor.Call(uintptr(menuHandCursor))
		return 1
	}
	r, _, _ := procDefWindowProc.Call(uintptr(h), uintptr(message), wParam, lParam)
	return r
}

// Wrap by measured characters so paths without spaces keep their complete filename.
func wrapNotificationText(dc windows.Handle, text string, width int32) []string {
	lines := []string{}
	line := ""
	for _, char := range strings.ReplaceAll(text, "\r", "") {
		if char == '\n' {
			lines = append(lines, line)
			line = ""
			continue
		}
		candidate := line + string(char)
		label, _ := windows.UTF16FromString(candidate)
		var extent struct{ X, Y int32 }
		procGetTextExtentPoint32.Call(uintptr(dc), uintptr(unsafe.Pointer(&label[0])),
			uintptr(len(label)-1), uintptr(unsafe.Pointer(&extent)))
		if extent.X > width && line != "" {
			lines = append(lines, line)
			line = string(char)
		} else {
			line = candidate
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func layoutNotification() {
	dc, _, _ := procGetDC.Call(uintptr(notificationHwnd))
	defer procReleaseDC.Call(uintptr(notificationHwnd), dc)
	font, _, _ := procSelectObject.Call(dc, uintptr(menuFont))
	defer procSelectObject.Call(dc, font)
	width := notificationW - scaled(48)
	notificationObjectLines = wrapNotificationText(windows.Handle(dc), notificationCurrent.Object, width)
	detail := notificationCurrent.Rule
	if notificationCurrent.MalwareName != "" {
		detail = notificationCurrent.MalwareName
	}
	notificationDetailLines = wrapNotificationText(windows.Handle(dc), detail, width)
	bottom := scaled(115) + int32(len(notificationObjectLines))*scaled(21)
	if len(notificationDetailLines) > 0 {
		bottom += scaled(12) + int32(len(notificationDetailLines))*scaled(21)
	}
	notificationH = bottom + scaled(58)
	if notificationH < scaled(248) {
		notificationH = scaled(248)
	}
	notificationDetailsRect = rect{
		Left: notificationW - scaled(108), Top: notificationH - scaled(44),
		Right: notificationW - scaled(24), Bottom: notificationH - scaled(12),
	}
}

func notificationText(dc windows.Handle, text string, area rect, color uint32, flags uintptr) {
	label, _ := windows.UTF16FromString(text)
	procSetTextColor.Call(uintptr(dc), uintptr(color))
	procDrawText.Call(
		uintptr(dc), uintptr(unsafe.Pointer(&label[0])), uintptr(len(label)-1),
		uintptr(unsafe.Pointer(&area)), flags|dtNoPrefix,
	)
}

func paintNotification(dst windows.Handle) {
	w, h := notificationW, notificationH
	mem, _, _ := procCreateCompatDC.Call(uintptr(dst))
	bmp, _, _ := procCreateCompatBmp.Call(uintptr(dst), uintptr(w), uintptr(h))
	old, _, _ := procSelectObject.Call(mem, bmp)
	defer func() { procSelectObject.Call(mem, old); procDeleteObject.Call(bmp); procDeleteDC.Call(mem) }()
	fillRect(mem, rect{Right: w, Bottom: h}, rgb(255, 255, 255))
	fillRect(mem, rect{Right: scaled(5), Bottom: h}, rgb(0, 132, 117))
	procSetBkMode.Call(mem, transparentMode)
	font, _, _ := procSelectObject.Call(mem, uintptr(menuFont))
	defer procSelectObject.Call(mem, font)
	dc := windows.Handle(mem)
	left, right := scaled(24), w-scaled(24)
	notificationText(dc, "Elastic UI", rect{Left: left, Top: scaled(16), Right: right - scaled(28), Bottom: scaled(37)}, rgb(68, 87, 87), dtSingleLine)
	notificationText(dc, "×", rect{Left: w - scaled(40), Top: scaled(10), Right: w - scaled(10), Bottom: scaled(36)}, rgb(86, 101, 101), dtSingleLine|1)
	if notificationTitleFont != 0 {
		procSelectObject.Call(mem, uintptr(notificationTitleFont))
	}
	notificationText(dc, "已拦截威胁", rect{Left: left, Top: scaled(48), Right: right, Bottom: scaled(75)}, rgb(0, 112, 99), dtSingleLine)
	procSelectObject.Call(mem, uintptr(menuFont))
	notificationText(dc, notificationCurrent.Kind, rect{Left: left, Top: scaled(83), Right: right, Bottom: scaled(107)}, rgb(37, 49, 49), dtSingleLine)
	y := scaled(115)
	for _, line := range notificationObjectLines {
		notificationText(dc, line, rect{Left: left, Top: y, Right: right, Bottom: y + scaled(21)}, rgb(37, 49, 49), dtSingleLine)
		y += scaled(21)
	}
	y += scaled(12)
	for _, line := range notificationDetailLines {
		notificationText(dc, line, rect{Left: left, Top: y, Right: right, Bottom: y + scaled(21)}, rgb(102, 116, 116), dtSingleLine)
		y += scaled(21)
	}
	notificationText(dc, "查看详情", notificationDetailsRect, rgb(0, 112, 99), dtSingleLine|dtVCenter|2)
	procBitBlt.Call(uintptr(dst), 0, 0, uintptr(w), uintptr(h), mem, 0, 0, srcCopy)
}
