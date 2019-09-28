package main

import (
	"os/exec"

	termbox "github.com/nsf/termbox-go"
)

const (
	GomacsMouseNone byte = iota
	GomacsMouseDragging
)

func (t *winTree) mouseInBuffer(x, y, wx, wy, mx, my int) (int, int, *EditorBuffer) {
	if !(x <= mx && mx <= x+wx && y <= my && my <= y+wx) {
		// Do nothing
		return 0, 0, nil
	} else if t.split {
		if t.hor {
			rx, ry, rbuf := t.childLT.mouseInBuffer(x, y, wx/2, wy, mx, my)

			if rbuf != nil {
				return rx, ry, rbuf
			}

			return t.childRB.mouseInBuffer(x+(wx/2)+1, y, (wx / 2),
				wy, mx, my)
		}
		rx, ry, rbuf := t.childLT.mouseInBuffer(x, y, wx, wy/2, mx, my)

		if rbuf != nil {
			return rx, ry, rbuf
		}

		return t.childRB.mouseInBuffer(x, y+1+(wy/2), wx, (wy/2)-1, mx, my)
	}

	cy := t.buf.rowoff + my%wy

	if cy >= Global.CurrentB.NumRows {
		return 0, cy, t.buf
	}

	row := Global.CurrentB.Rows[cy]
	gut := 0
	if Global.CurrentB.hasMode("line-number-mode") {
		gut = GetGutterWidth(Global.CurrentB.NumRows)
	}
	rx := mx - x - gut + row.coloff

	Global.WindowTree.mapTree(func(wt *winTree) { wt.focused = false })
	t.setFocus()

	return editorRowRxToCx(row, rx), cy, t.buf
}

func getMousePoint(mx, my int) (int, int, *EditorBuffer) {
	sx, sy := termbox.Size()
	return Global.WindowTree.mouseInBuffer(0, 0, sx, sy-2, mx, my)
}

func JumpToMousePoint() {
	if Global.CurrentB.NumRows <= 0 {
		return
	}
	var cx, cy int
	cx, cy, Global.CurrentB = getMousePoint(Global.MouseX, Global.MouseY)
	if cy >= Global.CurrentB.NumRows {
		Global.CurrentB.cy = Global.CurrentB.NumRows - 1
		Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy].Size
	} else {
		Global.CurrentB.cy = cy
		Global.CurrentB.cx = cx
		Global.CurrentB.prefcx = cx
	}
}

var mousestate byte = GomacsMouseNone

func MouseDragRegion() {
	buf := Global.CurrentB
	if buf.NumRows <= 0 {
		return
	}
	cachedcx, cachedcy := buf.cx, buf.cy
	JumpToMousePoint()
	if mousestate == GomacsMouseDragging && buf == Global.CurrentB && (cachedcx != buf.cx || cachedcy != buf.cy) {
		if !buf.regionActive {
			buf.MarkX = cachedcx
			buf.MarkY = cachedcy
			buf.regionActive = true
		}
		buf.recalcRegion()
	}
	mousestate = GomacsMouseDragging
}

func MouseScrollUp() {
	_, _, Global.CurrentB = getMousePoint(Global.MouseX, Global.MouseY)
	if Global.CurrentB.rowoff > 0 {
		Global.CurrentB.rowoff--
	} else {
		Global.Input = "Beginning of buffer"
	}
	if Global.CurrentB.cy >= Global.CurrentB.rowoff+Global.CurrentBHeight-1 {
		Global.CurrentB.MoveCursorUp()
	}
}

func MouseScrollDown() {
	_, _, Global.CurrentB = getMousePoint(Global.MouseX, Global.MouseY)
	if Global.CurrentB.rowoff < Global.CurrentB.NumRows {
		Global.CurrentB.rowoff++
	} else {
		Global.Input = "End of buffer"
	}
	if Global.CurrentB.cy < Global.CurrentB.rowoff {
		Global.CurrentB.MoveCursorDown()
	}
}

func MouseRelease() {
	Global.Input = ""
	mousestate = GomacsMouseNone
}

func MouseYankXsel() {
	prog, args, err := getXsel()
	if err != nil {
		Global.Input = "Can't find xsel or xclip in your PATH"
		AddErrorMessage(Global.Input)
		return
	}
	var out string
	out, err = shellCmd(prog, args)
	if err != nil {
		Global.Input = err.Error()
		AddErrorMessage(Global.Input)
		return
	}
	Global.Clipboard = out
	if Global.CurrentB.hasMode("xsel-jump-to-cursor-mode") {
		JumpToMousePoint()
	}
	doYankRegion()
	Global.Input = "Yanked X selection."
}

func getXsel() (string, []string, error) {
	args := []string{}
	ret, err := exec.LookPath("xsel")
	if err != nil {
		ret, err = exec.LookPath("xclip")
		if err == nil {
			args = []string{"-o", "-selection", "primary"}
		}
	}
	return ret, args, err
}
