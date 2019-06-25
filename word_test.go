package main

import (
	"testing"
)

func TestDelForwardWord(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	Global.CurrentB.cx = 0
	delForwardWord()
	Global.CurrentB.FailIfBufferNe([]string{}, t)
}

func TestDelBackWord(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	delBackWord()
	Global.CurrentB.FailIfBufferNe([]string{}, t)
}

func TestMoveForwardWord(t *testing.T) {
	InitEditor()
	editorInsertStr("foo bar")
	Global.CurrentB.cx = 0
	moveForwardWord()
	if Global.CurrentB.cx != 3 {
		t.Error("Expected 3 for cx but was actually", Global.CurrentB.cx)
	}
}

func TestMoveBackWord(t *testing.T) {
	InitEditor()
	editorInsertStr("foo bar")
	moveBackWord()
	if Global.CurrentB.cx != 4 {
		t.Error("Expected 4 for cx but was actually", Global.CurrentB.cx)
	}
}

func TestUpcaseWord(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	Global.CurrentB.cx = 0
	upcaseWord()
	Global.CurrentB.FailIfBufferNe([]string{"TEST"}, t)
}

func TestDowncaseWord(t *testing.T) {
	InitEditor()
	editorInsertStr("TEST")
	Global.CurrentB.cx = 0
	downcaseWord()
	Global.CurrentB.FailIfBufferNe([]string{"test"}, t)
}

func TestCapitalizeWordWhenLowercase(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	Global.CurrentB.cx = 0
	capitalizeWord()
	Global.CurrentB.FailIfBufferNe([]string{"Test"}, t)
}

func TestCapitalizeWordWhenUppercase(t *testing.T) {
	InitEditor()
	editorInsertStr("TEST")
	Global.CurrentB.cx = 0
	capitalizeWord()
	Global.CurrentB.FailIfBufferNe([]string{"Test"}, t)
}

func TestForwardWordOverLines(t *testing.T) {
	InitEditor()
	editorInsertStr("test1")
	editorInsertNewline(false)
	editorInsertStr("test2")
	editorInsertNewline(false)
	editorInsertStr("test3")
	editorInsertNewline(false)
	Global.CurrentB.cx = 0
	Global.CurrentB.cy = 0
	moveForwardWord()
	if Global.CurrentB.cy != 0 {
		t.Error("Expected 0 for cy but was actually", Global.CurrentB.cy)
	}
	if Global.CurrentB.cx != 5 {
		t.Error("Expected 5 for cx but was actually", Global.CurrentB.cx)
	}
	moveForwardWord()
	if Global.CurrentB.cy != 1 {
		t.Error("Expected 1 for cy but was actually", Global.CurrentB.cy)
	}
}

func TestBackWordOverLines(t *testing.T) {
	InitEditor()
	editorInsertStr("test1")
	editorInsertNewline(false)
	editorInsertStr("test2")
	editorInsertNewline(false)
	editorInsertStr("test3")
	editorInsertNewline(false)
	moveBackWord()
	if Global.CurrentB.cy != 2 {
		t.Error("Expected 0 for cy but was actually", Global.CurrentB.cy)
	}
	if Global.CurrentB.cx != 0 {
		t.Error("Expected 0 for cx but was actually", Global.CurrentB.cx)
	}
	moveBackWord()
	if Global.CurrentB.cy != 1 {
		t.Error("Expected 1 for cy but was actually", Global.CurrentB.cy)
	}
}

func TestTransposeWordsBasic(t *testing.T) {
	InitEditor()
	editorInsertStr("test1 test2")
	moveBackWord()
	doTransposeWords()
	Global.CurrentB.FailIfBufferNe([]string{"test2 test1"}, t)
}

func TestTransposeWordsAtEof(t *testing.T) {
	InitEditor()
	editorInsertStr("test1 test2")
	doTransposeWords()
	Global.CurrentB.FailIfBufferNe([]string{"test1 test2"}, t)
	if Global.CurrentB.cx != 6 {
		t.Error("Expected 6 for cx but was actually", Global.CurrentB.cx)
	}
}
