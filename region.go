package main

import (
	"bytes"
	"strings"
)

func setMark(buf *EditorBuffer) {
	buf.MarkX = buf.cx
	buf.MarkY = buf.cy
	Global.Input = "Mark set."
}

func validMark(buf *EditorBuffer) bool {
	return buf.MarkY < buf.NumRows && buf.MarkX <= len(buf.Rows[buf.MarkY].Data)
}

func rowDelRange(row *EditorRow, startc, endc int, buf *EditorBuffer) {
	editorAddUndo(false, startc, endc,
		row.idx, row.idx, row.Data[startc:endc])
	Global.Clipboard = row.Data[startc:endc]
	editorRowDelChar(row, startc, endc-startc)
}

func bufKillRegion(buf *EditorBuffer, startc, endc, startl, endl int) {
	if startl == endl {
		rowDelRange(buf.Rows[startl], startc, endc, buf)
	} else {
		var bb bytes.Buffer
		row := buf.Rows[startl]
		rowDelRange(row, startc, row.Size, buf)
		editorPopUndo()
		bb.WriteString(Global.Clipboard)
		bb.WriteRune('\n')
		for i := startl + 1; i < endl; i++ {
			row = buf.Rows[startl+1] //it deletes as they go!
			hasdata := false
			if row.Size > 0 {
				hasdata = true
				rowDelRange(row, 0, row.Size, buf)
				editorPopUndo()
			}
			buf.cy = startl + 1
			buf.cx = 0
			editorDelChar()
			editorPopUndo()
			if hasdata {
				bb.WriteString(Global.Clipboard)
			}
			bb.WriteRune('\n')
		}
		row = buf.Rows[startl+1]
		rowDelRange(row, 0, endc, buf)
		editorPopUndo()
		buf.cy = startl + 1
		buf.cx = 0
		editorDelChar()
		editorPopUndo()
		bb.WriteString(Global.Clipboard)
		Global.Clipboard = bb.String()
		updateLineIndexes()
	}
	editorAddRegionUndo(false, startc, endc,
		startl, endl, Global.Clipboard)
	buf.cx = startc
	buf.cy = startl
}

func bufCopyRegion(buf *EditorBuffer, startc, endc, startl, endl int) {
	if startl == endl {
		Global.Clipboard = buf.Rows[startl].Data[startc:endc]
	} else {
		var bb bytes.Buffer
		row := buf.Rows[startl]
		bb.WriteString(row.Data[startc:])
		bb.WriteRune('\n')
		for i := startl + 1; i < endl; i++ {
			row = buf.Rows[i]
			bb.WriteString(row.Data)
			bb.WriteRune('\n')
		}
		row = buf.Rows[endl]
		bb.WriteString(row.Data[:endc])
		Global.Clipboard = bb.String()
	}
}

func markAhead(buf *EditorBuffer) bool {
	if buf.MarkY == buf.cy {
		return buf.MarkX > buf.cx
	} else {
		return buf.MarkY > buf.cy
	}
}

func regionCmd(c func(*EditorBuffer, int, int, int, int)) {
	buf := Global.CurrentB
	if !validMark(buf) {
		Global.Input = "Invalid mark position"
		return
	}
	if markAhead(buf) {
		c(buf, buf.cx, buf.MarkX, buf.cy, buf.MarkY)
	} else {
		c(buf, buf.MarkX, buf.cx, buf.MarkY, buf.cy)
	}
}

func doKillRegion() {
	regionCmd(bufKillRegion)
}

func doCopyRegion() {
	regionCmd(bufCopyRegion)
}

func spitRegion(cx, cy int, region string) {
	Global.CurrentB.cx = cx
	Global.CurrentB.cy = cy
	clipLines := strings.Split(region, "\n")
	editorInsertStr(clipLines[0])
	editorPopUndo()
	if len(clipLines) > 1 {
		// Insert more lines...
		for i := 1; i < len(clipLines); i++ {
			editorInsertNewline()
			editorPopUndo()
			editorInsertStr(clipLines[i])
			editorPopUndo()
		}
	}
	editorAddRegionUndo(true, cx, Global.CurrentB.cx,
		cy, Global.CurrentB.cy, region)
}

func doYankRegion() {
	spitRegion(Global.CurrentB.cx, Global.CurrentB.cy, Global.Clipboard)
}

func killToEol() {
	cx := Global.CurrentB.cx
	cy := Global.CurrentB.cy
	if cy == Global.CurrentB.NumRows {
		return
	}
	if cx >= Global.CurrentB.Rows[cy].Size {
		MoveCursor(1, 0)
		editorDelChar()
	} else {
		rowDelRange(Global.CurrentB.Rows[cy], cx, Global.CurrentB.Rows[cy].Size, Global.CurrentB)
	}
}

func transposeRegion(buf *EditorBuffer, startc, endc, startl, endl int, trans func(string) string) {
	clip := Global.Clipboard
	bufKillRegion(buf, startc, endc, startl, endl)
	spitRegion(startc, startl, trans(Global.Clipboard))
	Global.Clipboard = clip
}

func transposeRegionCmd(trans func(string) string) {
	regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) {
		transposeRegion(buf, startc, endc, startl, endl, trans)
	})
}

func doUCRegion() {
	transposeRegionCmd(strings.ToUpper)
}

func doLCRegion() {
	transposeRegionCmd(strings.ToLower)
}
