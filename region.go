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
	buf.Clipboard = row.Data[startc:endc]
	editorRowDelChar(row, startc, endc-startc)
}

func bufKillRegion(buf *EditorBuffer, startc, endc, startl, endl int) {
	if startl == endl {
		rowDelRange(buf.Rows[startl], startc, endc, buf)
	} else {
		var bb bytes.Buffer
		row := buf.Rows[startl]
		rowDelRange(row, startc, row.Size, buf)
		bb.WriteString(buf.Clipboard)
		bb.WriteRune('\n')
		for i := startl + 1; i < endl; i++ {
			row = buf.Rows[startl+1] //it deletes as they go!
			if row.Size > 0 {
				rowDelRange(row, 0, row.Size, buf)
			}
			buf.cy = startl + 1
			buf.cx = 0
			editorDelChar()
			bb.WriteString(buf.Clipboard)
			bb.WriteRune('\n')
		}
		row = buf.Rows[startl+1]
		rowDelRange(row, 0, endc, buf)
		buf.cy = startl + 1
		buf.cx = 0
		editorDelChar()
		bb.WriteString(buf.Clipboard)
		buf.Clipboard = bb.String()
		updateLineIndexes()
	}
	buf.cx = startc
	buf.cy = startl
}

func bufCopyRegion(buf *EditorBuffer, startc, endc, startl, endl int) {
	if startl == endl {
		buf.Clipboard = buf.Rows[startl].Data[startc:endc]
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
		buf.Clipboard = bb.String()
	}
}

func markAhead(buf *EditorBuffer) bool {
	if buf.MarkY == buf.cy {
		return buf.MarkX > buf.cx
	} else {
		return buf.MarkY > buf.cy
	}
}

func doKillRegion() {
	buf := Global.CurrentB
	if !validMark(buf) {
		Global.Input = "Invalid mark position"
		return
	}
	if markAhead(buf) {
		bufKillRegion(buf, buf.cx, buf.MarkX, buf.cy, buf.MarkY)
	} else {
		bufKillRegion(buf, buf.MarkX, buf.cx, buf.MarkY, buf.cy)
	}
}

func doCopyRegion() {
	buf := Global.CurrentB
	if !validMark(buf) {
		Global.Input = "Invalid mark position"
		return
	}
	if markAhead(buf) {
		bufCopyRegion(buf, buf.cx, buf.MarkX, buf.cy, buf.MarkY)
	} else {
		bufCopyRegion(buf, buf.MarkX, buf.cx, buf.MarkY, buf.cy)
	}
}

func doYankRegion() {
	clipLines := strings.Split(Global.CurrentB.Clipboard, "\n")
	editorInsertStr(clipLines[0])
	if len(clipLines) > 1 {
		// Insert more lines...
		for i := 1; i < len(clipLines); i++ {
			editorInsertNewline()
			editorInsertStr(clipLines[i])
		}
	}
}