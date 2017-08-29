package main

import "github.com/nsf/termbox-go"

func (row *EditorRow) screenXtoCx(sx int) int {
	gut := 0
	if Global.CurrentB.hasMode("line-number-mode") {
		gut = GetGutterWidth(Global.CurrentB.NumRows)
	}
	rx := sx - gut - Global.CurrentB.coloff
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
		Global.CurrentB.cy = Global.CurrentB.NumRows
		Global.CurrentB.cx = 0
	} else {
		Global.CurrentB.cy = cy
		Global.CurrentB.cx = Global.CurrentB.Rows[cy].screenXtoCx(Global.MouseX)
		Global.CurrentB.prefcx = Global.CurrentB.cx
	}
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

func ClearInput() {
	Global.Input = ""
}
