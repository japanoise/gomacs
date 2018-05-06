package main

import (
	"testing"
)

func selectTestRegion() {
	InitEditor()
	editorInsertStr("Test")
	editorInsertNewline(false)
	editorInsertStr("test2")
	setMark(Global.CurrentB)
	Global.CurrentB.cx = 0
	Global.CurrentB.cy = 0
}

func TestCopyRegion(t *testing.T) {
	selectTestRegion()
	doCopyRegion()
	if Global.Clipboard != "Test\ntest2" {
		t.Error("Clipboard contents was:", Global.Clipboard)
	}
}

func TestKillRegion(t *testing.T) {
	selectTestRegion()
	doKillRegion()
	if Global.Clipboard != "Test\ntest2" {
		t.Error("Clipboard contents was:", Global.Clipboard)
	}
	Global.CurrentB.FailIfBufferNe([]string{}, t)
}

func TestYankRegion(t *testing.T) {
	InitEditor()
	Global.Clipboard = "Test\ntest2"
	doYankRegion()
	Global.CurrentB.FailIfBufferNe([]string{"Test", "test2"}, t)
}

func TestToggleRegion(t *testing.T) {
	InitEditor()
	editorInsertStr("test")
	editorInsertNewline(false)
	editorInsertStr("test2")
	setMark(Global.CurrentB)
	setMark(Global.CurrentB)
	if Global.Input != "Mark deactivated." {
		t.Error("Mark was not deactivated")
	}
	setMark(Global.CurrentB)
	if Global.Input != "Mark activated." {
		t.Error("Mark was not activated")
	}
}

func TestUCRegion(t *testing.T) {
	selectTestRegion()
	doUCRegion()
	Global.CurrentB.FailIfBufferNe([]string{"TEST", "TEST2"}, t)
}

func TestLCRegion(t *testing.T) {
	selectTestRegion()
	doLCRegion()
	Global.CurrentB.FailIfBufferNe([]string{"test", "test2"}, t)
}

func TestFillRegion(t *testing.T) {
	InitEditor()
	Global.Fillcolumn = 80
	// It's cap'n jazz, dad, cap'n fucking jazz
	editorInsertStr("Burritos, Inspiration Point, Fork Balloon Sports, Cards in the Spokes, Automatic Biographies, Kites, Kung Fu, Trophies, Banana Peels We've Slipped On and Egg Shells We've Tippy Toed Over")
	doFillRegion()
	Global.CurrentB.FailIfBufferNe([]string{
		"Burritos, Inspiration Point, Fork Balloon Sports, Cards in the Spokes, Automatic",
		"Biographies, Kites, Kung Fu, Trophies, Banana Peels We've Slipped On and Egg",
		"Shells We've Tippy Toed Over",
	}, t)
}
