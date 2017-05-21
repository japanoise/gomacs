package main

import (
	"strings"
	"unicode/utf8"
)

func indexEndOfBackwardWord() int {
	cx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	if icy >= Global.CurrentB.NumRows {
		return cx
	}
	pre := true
	for cx > 0 {
		r, rs :=
			utf8.DecodeLastRuneInString(Global.CurrentB.Rows[icy].Data[:cx])
		if r == ' ' && !pre {
			return cx
		} else {
			pre = false
		}
		cx -= rs
	}
	return cx
}

func indexEndOfForwardWord() int {
	cx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	if icy >= Global.CurrentB.NumRows {
		return cx
	}
	l := Global.CurrentB.Rows[icy].Size
	pre := true
	for cx < l {
		r, rs := utf8.DecodeRuneInString(Global.CurrentB.Rows[icy].Data[cx:])
		if r == ' ' && !pre {
			return cx
		} else {
			pre = false
		}
		cx += rs
	}
	return cx
}

func delBackWord() {
	icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	if icy >= Global.CurrentB.NumRows {
		return
	}
	ncx := indexEndOfBackwardWord()
	if ncx < icx {
		rowDelRange(Global.CurrentB.Rows[icy], ncx, icx, Global.CurrentB)
		Global.CurrentB.cx = ncx
	}
}

func moveBackWord() {
	if Global.CurrentB.cx == 0 {
		MoveCursor(-1, 0)
	}
	Global.CurrentB.cx = indexEndOfBackwardWord()
}

func delForwardWord() {
	icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	if icy >= Global.CurrentB.NumRows {
		return
	}
	ncx := indexEndOfForwardWord()
	if ncx > icx {
		rowDelRange(Global.CurrentB.Rows[icy], icx, ncx, Global.CurrentB)
	}
}

func moveForwardWord() {
	icy := Global.CurrentB.cy
	if icy >= Global.CurrentB.NumRows {
		return
	}
	if Global.CurrentB.cx == Global.CurrentB.Rows[icy].Size {
		MoveCursor(1, 0)
	}
	Global.CurrentB.cx = indexEndOfForwardWord()
}

func upcaseWord() {
	icx := Global.CurrentB.cx
	endc := indexEndOfForwardWord()
	if endc > icx {
		transposeRegion(Global.CurrentB, icx, endc, Global.CurrentB.cy, Global.CurrentB.cy, strings.ToUpper)
	}
}

func downcaseWord() {
	icx := Global.CurrentB.cx
	endc := indexEndOfForwardWord()
	if endc > icx {
		transposeRegion(Global.CurrentB, icx, endc, Global.CurrentB.cy, Global.CurrentB.cy, strings.ToLower)
	}
}
