package main

import "bytes"

type rectangle struct {
	TopLeftX, TopLeftY   int
	BotRightX, BotRightY int
}

func doStringRectangle() {
	if validMark(Global.CurrentB) {
		rect := Global.CurrentB.getRectangle()
		Global.CurrentB.stringRectangle(editorPrompt("String rectangle", nil), rect)
	} else {
		Global.Input = "Invalid mark position."
	}
}

func (buf *EditorBuffer) stringRectangle(rep string, rect rectangle) {
	addRectUndo(false, buf, rect)
	for i := rect.TopLeftY; i <= rect.BotRightY && i < buf.NumRows; i++ {
		rectReplace(rect.TopLeftX, rect.BotRightX, buf.Rows[i], buf, rep)
	}
	if rect.TopLeftX+len(rep) != rect.BotRightX {
		rect.BotRightX = rect.TopLeftX + len(rep)
	}
	buf.cx, buf.cy = rect.BotRightX, rect.BotRightY
	addRectUndo(true, buf, rect)
}

func addRectUndo(ins bool, buf *EditorBuffer, rect rectangle) {
	startc, endc, startl, endl := rectToRegion(buf, rect)
	editorAddRegionUndo(ins, startc, endc,
		startl, endl, getRegionText(buf, startc, endc, startl, endl))
}

// HACK: Horrid signature. I need a region struct, but I'm too lazy
func rectToRegion(buf *EditorBuffer, rect rectangle) (int, int, int, int) {
	startc, endc, startl, endl := rect.TopLeftX, rect.BotRightX, rect.TopLeftY, rect.BotRightY
	if endc > buf.Rows[endl].Size {
		endc = buf.Rows[endl].Size
	}
	return startc, endc, startl, endl
}

func rectReplace(TopLeftX, BotRightX int, row *EditorRow, buf *EditorBuffer, s string) {
	if TopLeftX > row.Size {
		var buffer bytes.Buffer
		buffer.WriteString(row.Data)
		for i := row.Size; i < TopLeftX; i++ {
			buffer.WriteRune(' ')
		}
		buffer.WriteString(s)
		editorMutateRow(row, buf, buffer.String())
	} else if BotRightX > row.Size {
		editorMutateRow(row, buf, row.Data[:TopLeftX]+s)
	} else {
		editorMutateRow(row, buf, row.Data[:TopLeftX]+s+row.Data[BotRightX:])
	}
}

func (buf *EditorBuffer) getRectangle() rectangle {
	ret := rectangle{}
	if buf.cx < buf.MarkX {
		ret.TopLeftX = buf.cx
		ret.BotRightX = buf.MarkX
	} else {
		ret.TopLeftX = buf.MarkX
		ret.BotRightX = buf.cx
	}
	if buf.cy < buf.MarkY {
		ret.TopLeftY = buf.cy
		ret.BotRightY = buf.MarkY
	} else {
		ret.TopLeftY = buf.MarkY
		ret.BotRightY = buf.cy
	}
	return ret
}
