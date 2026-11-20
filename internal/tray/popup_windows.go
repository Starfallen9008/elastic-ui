//go:build windows

package tray

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procShowWindow      = user32.NewProc("ShowWindow")
	procSetWindowPos    = user32.NewProc("SetWindowPos")
	procSetCapture      = user32.NewProc("SetCapture")
	procReleaseCapture  = user32.NewProc("ReleaseCapture")
	procSetFocus        = user32.NewProc("SetFocus")
	procSetWindowRgn    = user32.NewProc("SetWindowRgn")
	procInvalidateRect  = user32.NewProc("InvalidateRect")
	procBeginPaint      = user32.NewProc("BeginPaint")
	procEndPaint        = user32.NewProc("EndPaint")
	procLoadCursor      = user32.NewProc("LoadCursorW")
	procSetCursor       = user32.NewProc("SetCursor")
	procMonitorFromRect = user32.NewProc("MonitorFromRect")
	procGetMonitorInfo  = user32.NewProc("GetMonitorInfoW")
	procCreateRoundRect = gdi32.NewProc("CreateRoundRectRgn")
	procRoundRect       = gdi32.NewProc("RoundRect")
	procCreatePen       = gdi32.NewProc("CreatePen")
	procGetStockObject  = gdi32.NewProc("GetStockObject")
	procCreateCompatDC  = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatBmp = gdi32.NewProc("CreateCompatibleBitmap")
	procBitBlt          = gdi32.NewProc("BitBlt")
	procDeleteDC        = gdi32.NewProc("DeleteDC")
	menuWndProcPtr      uintptr
	menuHwnd            windows.HWND
	menuVisible         bool
	menuRows            []menuRow
	menuHover           = -1
	menuW, menuH        int32
	menuHandCursor      windows.Handle
	menuArrowCursor     windows.Handle
)

const (
	wmPaint          = 0x000F
	wmEraseBkgnd     = 0x0014
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmRButtonDown    = 0x0204
	wmKeyDown        = 0x0100
	wmMouseWheel     = 0x020A
	wmActivate       = 0x0006
	wmKillFocus      = 0x0008
	wmCaptureChanged = 0x0215
	wmSetCursor      = 0x0020

	wsPopup        = 0x80000000
	wsExToolWindow = 0x00000080
	wsExTopmost    = 0x00000008

	swHide = 0

	swpShowWindow = 0x0040

	waInactive = 0
	htClient   = 1

	vkeyEscape = 0x1B
	vkeyUp     = 0x26
	vkeyDown   = 0x28
	vkeyReturn = 0x0D

	idiArrow = 32512
	idiHand  = 32649

	psSolid   = 0
	nullBrush = 5
	nullPen   = 8

	srcCopy = 0x00CC0020
)

type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

type paintStruct struct {
	Hdc         windows.Handle
	FErase      int32
	RcPaint     rect
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type menuDef struct {
	id    int
	label string
	sep   bool
}

type menuRow struct {
	id    int
	label string
	sep   bool
	r     rect
}

var menuDefs = []menuDef{
	{id: idShow, label: "显示主窗口"},
	{id: idScan, label: "打开扫描"},
	{id: idThreats, label: "威胁历史"},
	{sep: true},
	{id: idQuit, label: "退出"},
}

func createMenuPopup(hInst uintptr) bool {
	className := utf16Ptr("ElasticDefendTrayMenu")
	if className == nil {
		return false
	}
	menuWndProcPtr = windows.NewCallback(menuWndProc)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   menuWndProcPtr,
		HInstance:     windows.Handle(hInst),
		LpszClassName: className,
	}
	if r, _, _ := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return false
	}
	h, _, _ := procCreateWindowEx.Call(
		wsExToolWindow|wsExTopmost,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(""))),
		wsPopup,
		0, 0, 0, 0,
		0, 0, hInst, 0,
	)
	if h == 0 {
		return false
	}
	menuHwnd = windows.HWND(h)
	h, _, _ = procLoadCursor.Call(0, idiHand)
	menuHandCursor = windows.Handle(h)
	a, _, _ := procLoadCursor.Call(0, idiArrow)
	menuArrowCursor = windows.Handle(a)
	return true
}

func showMenu(anchor windows.HWND) {
	if menuHwnd == 0 {
		return
	}
	buildMenuRows()

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	mon := monitorAt(pt)
	work := monitorWork(mon)
	w, h := menuW, menuH
	margin := scaled(8)
	x := pt.X - w
	y := pt.Y - h
	if x < work.Left+margin {
		x = work.Left + margin
	}
	if y < work.Top+margin {
		y = work.Top + margin
	}
	if x+w > work.Right-margin {
		x = work.Right - w - margin
	}
	if y+h > work.Bottom-margin {
		y = work.Bottom - h - margin
	}

	rgn, _, _ := procCreateRoundRect.Call(
		0, 0, uintptr(w+1), uintptr(h+1),
		uintptr(scaled(24)), uintptr(scaled(24)),
	)
	if rgn != 0 {
		if ok, _, _ := procSetWindowRgn.Call(uintptr(menuHwnd), rgn, 0); ok == 0 {
			procDeleteObject.Call(rgn)
		}
	}

	procSetWindowPos.Call(
		uintptr(menuHwnd),
		^uintptr(0), // HWND_TOPMOST
		uintptr(x), uintptr(y),
		uintptr(w), uintptr(h),
		swpShowWindow,
	)
	menuVisible = true
	procSetForegroundWindow.Call(uintptr(menuHwnd))
	procSetFocus.Call(uintptr(menuHwnd))
	procSetCapture.Call(uintptr(menuHwnd))
	menuHover = -1
	procInvalidateRect.Call(uintptr(menuHwnd), 0, 1)
}

func hideMenu() {
	if !menuVisible {
		return
	}
	menuVisible = false
	menuHover = -1
	procReleaseCapture.Call()
	if menuHwnd != 0 {
		procShowWindow.Call(uintptr(menuHwnd), swHide)
	}
}

func buildMenuRows() {
	pad := scaled(5)
	itemH := scaled(36)
	sepH := scaled(7)
	w := scaled(210)
	y := pad
	rows := make([]menuRow, 0, len(menuDefs))
	for _, def := range menuDefs {
		rh := itemH
		if def.sep {
			rh = sepH
		}
		rows = append(rows, menuRow{
			id:    def.id,
			label: def.label,
			sep:   def.sep,
			r:     rect{Left: 0, Top: y, Right: w, Bottom: y + rh},
		})
		y += rh
	}
	menuRows = rows
	menuW = w
	menuH = y + pad
}

func monitorAt(pt point) uintptr {
	r := rect{Left: pt.X, Top: pt.Y, Right: pt.X + 1, Bottom: pt.Y + 1}
	mon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&r)), 2)
	return mon
}

func monitorWork(mon uintptr) rect {
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
	if mi.RcWork.Right == 0 && mi.RcWork.Bottom == 0 {
		return rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	}
	return mi.RcWork
}

func menuWndProc(h windows.HWND, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case wmEraseBkgnd:
		return 1
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(uintptr(h), uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			paintMenu(windows.Handle(hdc))
			procEndPaint.Call(uintptr(h), uintptr(unsafe.Pointer(&ps)))
		}
		return 0
	case wmMouseMove:
		x, y := mousePoint(lParam)
		idx := menuHit(x, y)
		if idx != menuHover {
			menuHover = idx
			procInvalidateRect.Call(uintptr(h), 0, 0)
		}
		if menuHandCursor != 0 {
			if idx >= 0 {
				procSetCursor.Call(uintptr(menuHandCursor))
			} else {
				procSetCursor.Call(uintptr(menuArrowCursor))
			}
		}
		return 0
	case wmSetCursor:
		if uint32(lParam)&0xFFFF == htClient {
			if menuHover >= 0 && menuHandCursor != 0 {
				procSetCursor.Call(uintptr(menuHandCursor))
			} else if menuArrowCursor != 0 {
				procSetCursor.Call(uintptr(menuArrowCursor))
			}
			return 1
		}
	case wmLButtonUp:
		x, y := mousePoint(lParam)
		idx := menuHit(x, y)
		hideMenu()
		if idx >= 0 {
			runMenu(menuRows[idx].id)
		}
		return 0
	case wmRButtonUp:
		hideMenu()
		return 0
	case wmKeyDown:
		switch wParam {
		case vkeyEscape:
			hideMenu()
		case vkeyUp:
			moveHover(-1)
		case vkeyDown:
			moveHover(1)
		case vkeyReturn:
			if menuHover >= 0 {
				id := menuRows[menuHover].id
				hideMenu()
				runMenu(id)
			}
		}
		return 0
	case wmMouseWheel:
		if int16(wParam>>16) > 0 {
			moveHover(-1)
		} else {
			moveHover(1)
		}
		return 0
	case wmCaptureChanged:
		hideMenu()
		return 0
	case wmClose:
		hideMenu()
		return 0
	}
	r, _, _ := procDefWindowProc.Call(uintptr(h), uintptr(message), wParam, lParam)
	return r
}

func mousePoint(lParam uintptr) (int32, int32) {
	return int32(lParam & 0xFFFF), int32((lParam >> 16) & 0xFFFF)
}

func menuHit(x, y int32) int {
	for i, row := range menuRows {
		if row.sep {
			continue
		}
		if x >= row.r.Left && x < row.r.Right && y >= row.r.Top && y < row.r.Bottom {
			return i
		}
	}
	return -1
}

func moveHover(dir int) {
	if len(menuRows) == 0 {
		return
	}
	next := menuHover
	if next < 0 {
		if dir > 0 {
			next = -1
		} else {
			next = len(menuRows)
		}
	}
	for i := 0; i < len(menuRows); i++ {
		next += dir
		if next < 0 {
			next = len(menuRows) - 1
		}
		if next >= len(menuRows) {
			next = 0
		}
		if !menuRows[next].sep {
			menuHover = next
			procInvalidateRect.Call(uintptr(menuHwnd), 0, 0)
			return
		}
	}
}

func runMenu(id int) {
	switch id {
	case idShow:
		run(actions.Show)
	case idScan:
		run(actions.Scan)
	case idThreats:
		run(actions.Threats)
	case idQuit:
		run(actions.Quit)
	}
}

func paintMenu(dst windows.Handle) {
	p := colors()
	w, h := menuW, menuH
	if w <= 0 || h <= 0 {
		return
	}
	mem, _, _ := procCreateCompatDC.Call(uintptr(dst))
	if mem == 0 {
		return
	}
	bmp, _, _ := procCreateCompatBmp.Call(uintptr(dst), uintptr(w), uintptr(h))
	if bmp == 0 {
		procDeleteDC.Call(mem)
		return
	}
	oldBmp, _, _ := procSelectObject.Call(mem, bmp)
	hollowBrush, _, _ := procGetStockObject.Call(nullBrush)
	hollowPen, _, _ := procGetStockObject.Call(nullPen)

	fillRect(mem, rect{Left: 0, Top: 0, Right: w, Bottom: h}, p.bg)

	// Rounded outline, inset by one pixel so it survives the window region.
	pen, _, _ := procCreatePen.Call(psSolid, 1, uintptr(p.border))
	oldPen, _, _ := procSelectObject.Call(mem, pen)
	oldBrush, _, _ := procSelectObject.Call(mem, hollowBrush)
	procRoundRect.Call(
		mem, 1, 1, uintptr(w-1), uintptr(h-1),
		uintptr(scaled(22)), uintptr(scaled(22)),
	)
	procSelectObject.Call(mem, oldPen)
	procSelectObject.Call(mem, oldBrush)
	procDeleteObject.Call(pen)

	for i, row := range menuRows {
		if row.sep {
			mid := (row.r.Top + row.r.Bottom) / 2
			fillRect(mem, rect{
				Left:   row.r.Left + scaled(14),
				Top:    mid,
				Right:  row.r.Right - scaled(14),
				Bottom: mid + 1,
			}, p.border)
			continue
		}
		if i == menuHover {
			hbrush, _, _ := procCreateSolidBrush.Call(uintptr(p.hover))
			old, _, _ := procSelectObject.Call(mem, hbrush)
			oldP, _, _ := procSelectObject.Call(mem, hollowPen)
			procRoundRect.Call(
				mem,
				uintptr(row.r.Left+scaled(4)), uintptr(row.r.Top+scaled(3)),
				uintptr(row.r.Right-scaled(4)), uintptr(row.r.Bottom-scaled(3)),
				uintptr(scaled(16)), uintptr(scaled(16)),
			)
			procSelectObject.Call(mem, old)
			procSelectObject.Call(mem, oldP)
			procDeleteObject.Call(hbrush)
			fillRect(mem, rect{
				Left:   row.r.Left + scaled(9),
				Top:    row.r.Top + scaled(10),
				Right:  row.r.Left + scaled(11),
				Bottom: row.r.Bottom - scaled(10),
			}, p.accent)
		}

		color := p.text
		if row.id == idQuit {
			color = p.danger
		}
		oldFont, _, _ := procSelectObject.Call(mem, uintptr(menuFont))
		procSetBkMode.Call(mem, transparentMode)
		procSetTextColor.Call(mem, uintptr(color))
		text := rect{
			Left:   row.r.Left + scaled(24),
			Top:    row.r.Top,
			Right:  row.r.Right - scaled(14),
			Bottom: row.r.Bottom,
		}
		label, _ := windows.UTF16FromString(row.label)
		procDrawText.Call(
			mem,
			uintptr(unsafe.Pointer(&label[0])),
			uintptr(len(label)-1),
			uintptr(unsafe.Pointer(&text)),
			dtLeft|dtVCenter|dtSingleLine|dtNoPrefix,
		)
		procSelectObject.Call(mem, oldFont)
	}

	procBitBlt.Call(
		uintptr(dst), 0, 0, uintptr(w), uintptr(h),
		mem, 0, 0, srcCopy,
	)
	procSelectObject.Call(mem, oldBmp)
	procDeleteObject.Call(bmp)
	procDeleteDC.Call(mem)
}

func fillRect(hdc uintptr, r rect, color uint32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(color))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brush)
	procDeleteObject.Call(brush)
}
