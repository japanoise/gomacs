package main

import (
	"bytes"
	"strconv"
	"testing"
)

func (b *EditorBuffer) PrintBuf() string {
	buf := bytes.Buffer{}
	for i, row := range b.Rows {
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString(": ")
		buf.WriteString(row.Data)
		buf.WriteRune('\n')
	}
	return buf.String()
}

func TestAppendUndo(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	editorUndoAction()
	if Global.CurrentB.Rows[0].Data != "" {
		t.Error("expected empty buffer, but buffer's content is:\n", Global.CurrentB.PrintBuf())
	}
}

func TestDeleteUndo(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	editorDelChar()
	editorUndoAction()
	if Global.CurrentB.Rows[0].Data != "test" {
		t.Error("expected buffer's content to be \"test\", but buffer's content is:\n", Global.CurrentB.PrintBuf())
	}
}

func TestMultilineInsertUndo(t *testing.T) {
	InitEditor()
	editorInsertStr("test1")
	editorInsertNewline(false)
	editorInsertStr("test2")
	editorInsertNewline(false)
	editorInsertStr("test3")
	editorInsertNewline(false)
	editorUndoAction()
	if Global.CurrentB.LinesNum() != 1 || Global.CurrentB.Rows[0].Data != "" {
		t.Error("expected empty buffer, but buffer's content is:\n", Global.CurrentB.PrintBuf())
	}
}

func TestMultilineDeleteUndo(t *testing.T) {
	InitEditor()
	editorInsertStr("test1")
	editorInsertNewline(false)
	editorInsertStr("test2")
	editorInsertNewline(false)
	editorInsertStr("test3")
	setMark(Global.CurrentB)
	Global.CurrentB.cx = 0
	Global.CurrentB.cy = 0
	doKillRegion()
	editorUndoAction()
	if Global.CurrentB.LinesNum() != 3 || Global.CurrentB.Rows[0].Data != "test1" || Global.CurrentB.Rows[1].Data != "test2" || Global.CurrentB.Rows[2].Data != "test3" {
		t.Error("expected buffer {\"test1\",\"test2\",\"test3\"}, but buffer's content is:\n", Global.CurrentB.PrintBuf())
	}
}
