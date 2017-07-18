package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/japanoise/termbox-util"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

func editorRowCxToRx(row *EditorRow) int {
	rx := 0
	cx := Global.CurrentB.cx
	for i, rv := range row.Data {
		if i >= cx {
			break
		}
		if rv == '\t' {
			rx += Global.Tabsize
		} else {
			rx += termutil.Runewidth(rv)
		}
	}
	return rx
}

func editorRowRxToCx(row *EditorRow, rx int) int {
	cur_rx := 0
	var cx int
	for cx = 0; cx < row.Size; cx++ {
		if row.Data[cx] == '\t' {
			cur_rx += Global.Tabsize
		} else {
			cur_rx++
		}
		if cur_rx > rx {
			return cx
		}
	}
	return cx
}

func editorRefreshScreen() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	x, y := termbox.Size()
	yrows := y - 2
	numwin := len(Global.Windows)
	winheight := yrows / numwin
	for i, win := range Global.Windows {
		gutter := 0
		if win.hasMode("line-number-mode") && win.NumRows > 0 {
			gutter = GetGutterWidth(win.NumRows)
		}
		starth := 0
		if i >= 1 {
			starth = 1 + winheight*i
			editorDrawStatusLine(x, winheight*i, Global.Windows[i-1])
			editorScroll(x-gutter, winheight-1)
		} else {
			editorScroll(x-gutter, winheight)
		}
		if win == Global.CurrentB {
			Global.CurrentBHeight = winheight
			termbox.SetCursor(Global.CurrentB.rx-Global.CurrentB.coloff+gutter, starth+Global.CurrentB.cy-Global.CurrentB.rowoff)
		}
		editorDrawRows(starth, winheight*(i+1)+1, win, gutter)
	}
	editorDrawStatusLine(x, y-2, Global.Windows[numwin-1])
	editorDrawPrompt(y)
	termbox.Flush()
}

func trimString(s string, coloff int) (string, int) {
	if coloff == 0 {
		return s, 0
	}
	sr := []rune(s)
	if coloff < len(sr) {
		ret := string(sr[coloff:])
		return ret, strings.Index(s, ret)
	} else {
		return "", 0
	}
}

func editorDrawRows(starty, sy int, buf *EditorBuffer, gutsize int) {
	for y := starty; y < sy; y++ {
		filerow := (y - starty) + buf.rowoff
		if filerow >= buf.NumRows {
			if buf.coloff == 0 && buf.hasMode("tilde-mode") {
				termbox.SetCell(gutsize, y, '~', termbox.ColorBlue, termbox.ColorDefault)
			}
		} else {
			if gutsize > 0 {
				if buf.hasMode("gdi") {
					termutil.Printstring(string(buf.Rows[filerow].idx), 0, y)
				} else {
					termutil.Printstring(runewidth.FillLeft(LineNrToString(buf.Rows[filerow].idx+1), gutsize-2), 0, y)
				}
				termutil.PrintRune(gutsize-2, y, '│', termbox.ColorDefault)
				if buf.coloff > 0 {
					termutil.PrintRune(gutsize-1, y, '←', termbox.ColorDefault)
				}
			}
			row := buf.Rows[filerow]
			if buf.coloff < row.RenderSize {
				ts, off := trimString(row.Render, buf.coloff)
				if Global.NoSyntax || buf.Highlighter == nil {
					termutil.Printstring(ts, gutsize, y)
				} else {
					row.HlPrint(gutsize, y, buf.coloff, off, ts)
				}
			}
		}
	}
}

func editorUpdateStatus(buf *EditorBuffer) string {
	fn := buf.getRenderName()
	dc := '-'
	if buf.Dirty {
		dc = '*'
	}
	return fmt.Sprintf("-%c %s - (%s) %d:%d", dc, fn, buf.MajorMode,
		buf.cy+1, buf.cx)
}

func GetScreenSize() (int, int) {
	x, _ := termbox.Size()
	return x, Global.CurrentBHeight
}

func editorDrawStatusLine(x, y int, buf *EditorBuffer) {
	line := editorUpdateStatus(buf)
	if buf == Global.CurrentB && buf.hasMode("terminal-title-mode") {
		fmt.Printf("\033]0;%s - gomacs\a", buf.getRenderName())
	}
	var ru rune
	rx := 0
	for _, ru = range line {
		termbox.SetCell(rx, y, ru, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		rx += termutil.Runewidth(ru)
	}
	termbox.SetCell(rx, y, ' ', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
	var ix int
	for ix = rx + 1; ix < x-7; ix++ {
		if buf == Global.CurrentB {
			termbox.SetCell(ix, y, '-', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		} else {
			termbox.SetCell(ix, y, ' ', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		}
	}
	el := calcEndLabel(buf)
	for _, ru := range el {
		termbox.SetCell(ix, y, ru, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		ix++
	}
	for ix < x {
		if buf == Global.CurrentB {
			termbox.SetCell(ix, y, '-', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		} else {
			termbox.SetCell(ix, y, ' ', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		}
		ix++
	}
}

func calcEndLabel(buf *EditorBuffer) string {
	if buf.NumRows == 0 {
		return " Emp "
	} else if Global.CurrentBHeight >= buf.NumRows {
		return " All "
	} else if buf.rowoff+Global.CurrentBHeight >= buf.NumRows {
		return " Bot "
	} else if buf.rowoff == 0 {
		return " Top "
	} else {
		perc := float64(buf.rowoff) / float64(buf.NumRows)
		return fmt.Sprintf(" %2d%% ", int(perc*100))
	}
}

func editorDrawPrompt(y int) {
	termutil.Printstring(Global.Prompt+"-> "+Global.Input, 0, y-1)
}

func NumStrWidth(num int) int {
	return int(math.Log10(float64(num))) + 1
}

func GetGutterWidth(NumRows int) int {
	return NumStrWidth(NumRows) + 2
}

func LineNrToString(num int) string {
	return strconv.Itoa(num)
}
