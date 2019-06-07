package main

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/japanoise/termbox-util"
	glisp "github.com/zhemao/glisp/interpreter"
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
		if !termutil.WordCharacter(r) && !pre {
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
		if !termutil.WordCharacter(r) && !pre {
			return cx
		} else {
			pre = false
		}
		cx += rs
	}
	return cx
}

func delBackWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
		if icy >= Global.CurrentB.NumRows {
			return
		}
		ncx := indexEndOfBackwardWord()
		if ncx < icx {
			Global.Clipboard = rowDelRange(Global.CurrentB.Rows[icy], ncx, icx, Global.CurrentB)
			Global.CurrentB.cx = ncx
		}
	}
}

func moveBackWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		if Global.CurrentB.cx == 0 {
			Global.CurrentB.MoveCursorLeft()
		}
		Global.CurrentB.cx = indexEndOfBackwardWord()
		Global.CurrentB.prefcx = Global.CurrentB.cx
	}
}

func delForwardWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
		if icy >= Global.CurrentB.NumRows {
			return
		}
		ncx := indexEndOfForwardWord()
		if ncx > icx {
			Global.Clipboard = rowDelRange(Global.CurrentB.Rows[icy], icx, ncx, Global.CurrentB)
		}
	}
}

func moveForwardWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icy := Global.CurrentB.cy
		if icy >= Global.CurrentB.NumRows {
			return
		}
		if Global.CurrentB.cx == Global.CurrentB.Rows[icy].Size {
			Global.CurrentB.MoveCursorRight()
		}
		Global.CurrentB.cx = indexEndOfForwardWord()
		Global.CurrentB.prefcx = Global.CurrentB.cx
	}
}

func upcaseWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icx := Global.CurrentB.cx
		endc := indexEndOfForwardWord()
		if endc > icx {
			transposeRegion(Global.CurrentB, icx, endc, Global.CurrentB.cy, Global.CurrentB.cy, strings.ToUpper)
		}
	}
}

func downcaseWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icx := Global.CurrentB.cx
		endc := indexEndOfForwardWord()
		if endc > icx {
			transposeRegion(Global.CurrentB, icx, endc, Global.CurrentB.cy, Global.CurrentB.cy, strings.ToLower)
		}
	}
}

func capitalizeWord() {
	times := getRepeatTimes()
	icx := Global.CurrentB.cx
	endc := icx
	for i := 0; i < times; i++ {
		endc = indexEndOfForwardWord()
	}
	transposeRegion(Global.CurrentB, icx, endc, Global.CurrentB.cy, Global.CurrentB.cy, func(s string) string { return strings.Title(strings.ToLower(s)) })
}

func indexOfLastWord(s string) int {
	for i := len(s) - 1; i > 0; i-- {
		ru, _ := utf8.DecodeLastRuneInString(s[:i])
		if !termutil.WordCharacter(ru) {
			return i
		}
	}
	return 0
}

func indexOfFirstWord(s string) int {
	for i, ru := range s {
		if !termutil.WordCharacter(ru) {
			return i
		}
	}
	return len(s)
}

func getBackwardWord() string {
	return Global.CurrentB.Rows[Global.CurrentB.cy].
		Data[indexEndOfBackwardWord():Global.CurrentB.cx]
}

func autoComplete(env *glisp.Glisp) {
	word := getBackwardWord()
	re, err := regexp.Compile(`\b` + regexp.QuoteMeta(word) + `(\w+)`)
	if err != nil {
		Global.Input = err.Error()
	}

	matches := []string{}
	for _, buf := range Global.Buffers {
		for _, row := range buf.Rows {
			somematches := re.FindAllStringSubmatch(row.Data, -1)
			if len(somematches) > 0 {
			MATCH:
				for _, match := range somematches {
					for _, pmatch := range matches {
						if pmatch == match[1] {
							continue MATCH
						}
					}
					matches = append(matches, match[1])
				}
			}
		}
	}
	lm := len(matches)

	if lm == 0 {
		Global.Input = "No matches for " + word
		return
	} else if lm == 1 {
		editorInsertStr(matches[0])
		return
	}

	index := 0
	first := true
	ocx, ocy := Global.CurrentB.cx, Global.CurrentB.cy
	micromode("M-/", "Press M-/ again to cycle through complete candidates",
		env, func(*glisp.Glisp) {
			if first {
				first = false
			} else {
				bufKillRegion(Global.CurrentB, ocx,
					Global.CurrentB.cx, ocy, ocy)
			}
			editorRowInsertStr(Global.CurrentB.Rows[ocy],
				Global.CurrentB, ocx, matches[index])
			Global.CurrentB.cx = ocx + len(matches[index])

			index++
			if index >= lm {
				index = 0
			}
		})

	editorAddInsertUndo(ocx, ocy, matches[index])
}
