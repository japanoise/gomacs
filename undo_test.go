package main

import (
	"bytes"
	"strconv"
	"testing"
)

func (b *EditorBuffer) FailIfBufferNe(lines []string, t *testing.T) {
	if !b.TestIs(lines) {
		t.Error("Expected buffer", lines, "but was actually:\n", b.PrintBuf())
	}
}

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

func (b *EditorBuffer) TestIs(lines []string) bool {
	// Special case - one empty line is functionally equivalent to an empty buffer
	if len(lines) == 0 && (b.LinesNum() == 1 && b.Rows[0].Data == "") {
		return true
	}
	if len(lines) != len(b.Rows) {
		return false
	}
	for i, line := range lines {
		if b.Rows[i].Data != line {
			return false
		}
	}
	return true
}

func TestAppendUndo(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	editorUndoAction()
	Global.CurrentB.FailIfBufferNe([]string{}, t)
}

func TestDeleteUndo(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	editorDelChar()
	editorUndoAction()
	Global.CurrentB.FailIfBufferNe([]string{"test"}, t)
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
	Global.CurrentB.FailIfBufferNe([]string{}, t)
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
	Global.CurrentB.FailIfBufferNe([]string{"test1", "test2", "test3"}, t)
}

func TestLastLineUndo(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	Global.CurrentB.cx = 0
	Global.CurrentB.cy = 0
	editorInsertStr("a ")
	Global.CurrentB.cy = Global.CurrentB.NumRows
	Global.CurrentB.cx = 0
	editorInsertStr("foo")
	editorUndoAction()
	Global.CurrentB.FailIfBufferNe([]string{"a test", ""}, t)
}

func TestDoUndoWithNothing(t *testing.T) {
	InitEditor()
	editorUndoAction() // If this doesn't crash, then test for a message
	if Global.Input == "" {
		t.Error("No message to the user was printed")
	}
}

func TestDoRedoWithNothing(t *testing.T) {
	InitEditor()
	doOneRedo(nil) // If this doesn't crash, then test for a message
	if Global.Input == "" {
		t.Error("No message to the user was printed")
	}
}

func TestAppendRedo(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	editorUndoAction()
	doOneRedo(nil)
	Global.CurrentB.FailIfBufferNe([]string{"test"}, t)
}

func TestDeleteRedo(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	editorDelChar()
	editorUndoAction()
	doOneRedo(nil)
	Global.CurrentB.FailIfBufferNe([]string{"tes"}, t)
}

func TestMultilineInsertRedo(t *testing.T) {
	InitEditor()
	editorInsertStr("test1")
	editorInsertNewline(false)
	editorInsertStr("test2")
	editorInsertNewline(false)
	editorInsertStr("test3")
	editorUndoAction()
	doOneRedo(nil)
	Global.CurrentB.FailIfBufferNe([]string{"test1", "test2", "test3"}, t)
}

func TestMultilineDeleteRedo(t *testing.T) {
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
	doOneRedo(nil)
	Global.CurrentB.FailIfBufferNe([]string{}, t)
}
