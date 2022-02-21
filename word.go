package main

import (
	"regexp"
	"strings"
	"unicode/utf8"

	termutil "github.com/japanoise/termbox-util"
	glisp "github.com/glycerine/zygomys/zygo"
)

func indexEndOfBackwardWord() (int, int) {
	cx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	if icy >= Global.CurrentB.NumRows {
		return cx, icy
	}
	pre := true
	for cy := icy; cy >= 0; cy-- {
		if cy != icy {
			cx = Global.CurrentB.Rows[cy].Size
		}
		for cx > 0 {
			r, rs :=
				utf8.DecodeLastRuneInString(Global.CurrentB.Rows[cy].Data[:cx])
			if !termutil.WordCharacter(r) && !pre {
				return cx, cy
			} else if termutil.WordCharacter(r) {
				pre = false
			}
			cx -= rs
		}
		if !pre {
			return cx, cy
		}
	}
	return cx, 0
}

func indexEndOfForwardWord() (int, int) {
	cx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	if icy >= Global.CurrentB.NumRows {
		return cx, icy
	}
	pre := true
	for cy := icy; cy < Global.CurrentB.NumRows; cy++ {
		l := Global.CurrentB.Rows[cy].Size
		for cx < l {
			r, rs := utf8.DecodeRuneInString(Global.CurrentB.Rows[cy].Data[cx:])
			if !termutil.WordCharacter(r) && !pre {
				return cx, cy
			} else if termutil.WordCharacter(r) {
				pre = false
			}
			cx += rs
		}
		if !pre {
			return cx, cy
		}
		cx = 0
	}
	return cx, icy
}

func delBackWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
		if icy >= Global.CurrentB.NumRows {
			return
		}
		ncx, ncy := indexEndOfBackwardWord()
		if ncx < icx || ncy != icy {
			ret := bufKillRegion(Global.CurrentB, ncx, icx, ncy, icy)
			editorAddRegionUndo(false, ncx, icx, ncy, icy, ret)
			Global.Clipboard = ret
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
		Global.CurrentB.cx, Global.CurrentB.cy = indexEndOfBackwardWord()
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
		ncx, ncy := indexEndOfForwardWord()
		if ncx > icx || ncy != icy {
			ret := bufKillRegion(Global.CurrentB, icx, ncx, icy, ncy)
			editorAddRegionUndo(false, icx, ncx, icy, ncy, ret)
			Global.Clipboard = ret
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
		Global.CurrentB.cx, Global.CurrentB.cy = indexEndOfForwardWord()
		Global.CurrentB.prefcx = Global.CurrentB.cx
	}
}

func upcaseWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
		endc, endl := indexEndOfForwardWord()
		if endc > icx {
			transposeRegion(Global.CurrentB, icx, endc, icy, endl, strings.ToUpper)
		}
	}
}

func downcaseWord() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
		endc, endl := indexEndOfForwardWord()
		if endc > icx {
			transposeRegion(Global.CurrentB, icx, endc, icy, endl, strings.ToLower)
		}
	}
}

func capitalizeWord() {
	times := getRepeatTimes()
	icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	endc, endl := icx, icy
	for i := 0; i < times; i++ {
		endc, endl = indexEndOfForwardWord()
	}
	transposeRegion(Global.CurrentB, icx, endc, icy, endl, func(s string) string { return strings.Title(strings.ToLower(s)) })
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
	bwx, bwy := indexEndOfBackwardWord()
	return getRegionText(Global.CurrentB, bwx, Global.CurrentB.cx, bwy, Global.CurrentB.cy)
}

func autoComplete(env *glisp.Zlisp) {
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
		env, func(*glisp.Zlisp) {
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
