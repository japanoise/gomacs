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

func doCopyRectangle() {
	if validMark(Global.CurrentB) {
		Global.Clipboard = Global.CurrentB.copyRect()
		Global.Input = "Copied rectangle to clipboard"
	} else {
		Global.Input = "Invalid mark position"
	}
}

func (buf *EditorBuffer) copyRect() string {
	var buffer bytes.Buffer
	rect := buf.getRectangle()
	for i := rect.TopLeftY; i <= rect.BotRightY && i < buf.NumRows; i++ {
		if i != rect.TopLeftY {
			buffer.WriteRune('\n')
		}
		row := buf.Rows[i]
		width := rect.BotRightX - rect.TopLeftX
		if rect.TopLeftX > row.Size {
			for i := 0; i < width; i++ {
				buffer.WriteRune(' ')
			}
		} else if rect.BotRightX > row.Size {
			buffer.WriteString(row.Data[rect.TopLeftX:])
			for i := row.Size; i < rect.BotRightX; i++ {
				buffer.WriteRune(' ')
			}
		} else {
			buffer.WriteString(row.Data[rect.TopLeftX:rect.BotRightX])
		}
	}
	return buffer.String()
}

func rectToRegister() {
	if validMark(Global.CurrentB) {
		_, regname := InteractiveGetRegister("Copy rectangle to register: ")
		reg := Global.Registers.getRegisterOrCreate(regname)
		reg.Text = Global.CurrentB.copyRect()
		reg.Type = RegisterText
		Global.Input = "Copied rectangle to register " + regname
	} else {
		Global.Input = "Invalid mark position"
	}
}

func doKillRectangle() {
	if validMark(Global.CurrentB) {
		Global.Clipboard = Global.CurrentB.copyRect()
		Global.CurrentB.stringRectangle("", Global.CurrentB.getRectangle())
		Global.Input = "Killed rectangle"
	} else {
		Global.Input = "Invalid mark position"
	}
}
