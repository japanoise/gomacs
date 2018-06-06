package main

import (
	"os/exec"

	"github.com/nsf/termbox-go"
)

const (
	GomacsMouseNone byte = iota
	GomacsMouseDragging
)

func (row *EditorRow) screenXtoCx(sx int) int {
	gut := 0
	if Global.CurrentB.hasMode("line-number-mode") {
		gut = GetGutterWidth(Global.CurrentB.NumRows)
	}
	rx := sx - gut + row.coloff
	return editorRowRxToCx(row, rx)
}

func screenYtoBufAndCy(sy int) (*EditorBuffer, int) {
	_, ssy := termbox.Size()
	bufheight := ssy / len(Global.Windows)
	buf := Global.Windows[sy/bufheight]
	return buf, buf.rowoff + sy%bufheight
}

func JumpToMousePoint() {
	var cy int
	Global.CurrentB, cy = screenYtoBufAndCy(Global.MouseY)
	if cy >= Global.CurrentB.NumRows {
		Global.CurrentB.cy = Global.CurrentB.NumRows - 1
		Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy].Size
	} else {
		Global.CurrentB.cy = cy
		Global.CurrentB.cx = Global.CurrentB.Rows[cy].screenXtoCx(Global.MouseX)
		Global.CurrentB.prefcx = Global.CurrentB.cx
	}
}

var mousestate byte = GomacsMouseNone

func MouseDragRegion() {
	buf := Global.CurrentB
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
	Global.CurrentB, _ = screenYtoBufAndCy(Global.MouseY)
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
	Global.CurrentB, _ = screenYtoBufAndCy(Global.MouseY)
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
