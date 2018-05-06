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
