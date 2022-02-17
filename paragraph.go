package main

import "unicode/utf8"

func indexPreviousBlankLine() int {
	if Global.CurrentB.cy == 0 {
		Global.Input = "Beginning of buffer"
		return 0
	}
	for i := Global.CurrentB.cy - 1; 0 < i; i-- {
		if Global.CurrentB.Rows[i].Size == 0 {
			return i
		}
	}
	return 0
}

func indexNextBlankLine() int {
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		Global.Input = "End of buffer"
		return Global.CurrentB.NumRows
	} else if Global.CurrentB.cy == Global.CurrentB.NumRows-1 {
		return Global.CurrentB.NumRows
	}
	for i := Global.CurrentB.cy + 1; i < Global.CurrentB.NumRows; i++ {
		if Global.CurrentB.Rows[i].Size == 0 {
			return i
		}
	}
	return Global.CurrentB.NumRows
}

func backwardParagraph() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		cy := indexPreviousBlankLine()
		Global.CurrentB.cy = cy
		Global.CurrentB.cx = 0
	}
}

func forwardParagraph() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		cy := indexNextBlankLine()
		Global.CurrentB.cy = cy
		Global.CurrentB.cx = 0
	}
}

func doFillParagraph() {
	startl := indexPreviousBlankLine()
	endl := indexNextBlankLine() - 1
	if Global.CurrentB.NumRows == 0 {
		return
	} else {
		runeidx, space := savePointBeforeFill(startl, endl)
		transposeRegion(Global.CurrentB, 0, Global.CurrentB.Rows[endl].Size, startl, endl, FillString)
		restorePointAfterFill(runeidx, space)
	}
}

func savePointBeforeFill(startl, endl int) (int, bool) {
	buf := Global.CurrentB
	runeidx := 0
	space := false
rowloop:
	for cy := startl; cy <= endl; cy++ {
		for cx, rv := range buf.Rows[cy].Data {
			// Spaces excluded from the count because they can be added or
			// removed by the fill.
			if rv != ' ' {
				runeidx++
			}
			if cy == buf.cy && cx == buf.cx {
				// Need to know if point is on a space so that it can be put
				// back there. However, spaces before first non-spaces don't
				// count because they are deleted.
				space = rv == ' ' && runeidx > 0
				break rowloop
			}
		}
		if cy == buf.cy {
			// Handle edge case where cx wasn't reached because point is at end
			// of line. This is treated as a virtual space.
			space = true
			break
		}
	}
	return runeidx, space
}

func restorePointAfterFill(runeidx int, space bool) {
	buf := Global.CurrentB
	startl := indexPreviousBlankLine()
	endl := buf.cy // transposeRegion leaves point at end
	cur_runeidx := 0
	for cy := startl; cy <= endl; cy++ {
		for cx, rv := range buf.Rows[cy].Data {
			if rv != ' ' {
				cur_runeidx++
			}
			// '>=' not '==' to handle edge case where the original paragraph
			// started with a space that was deleted.
			if cur_runeidx >= runeidx {
				buf.cy = cy
				buf.cx = cx
				if space {
					buf.cx += utf8.RuneLen(rv)
				}
				return
			}
		}
	}
}
